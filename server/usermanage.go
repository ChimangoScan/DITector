package server

import (
	"fmt"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/Musso12138/docker-scan/myutils"
	"github.com/gin-gonic/gin"
)

type UserLogin struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func verifyLicense() (bool, error) {
	cmd := exec.Command(myutils.GlobalConfig.NSSLLicenseConfig.Filepath, "license", "verify")
	_, err := cmd.CombinedOutput()
	if err != nil {
		myutils.Logger.Error(fmt.Sprintf("verify nssl license failed, got error: %s", err))
		return false, err
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return false, fmt.Errorf("got status: %d", status.ExitStatus())
		}
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
	return true, nil
}

func handleUserLogin() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 验证许可证授权情况
		licVerified, err := verifyLicense()
		if !licVerified || err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": fmt.Sprintf("许可证验证失败, got err: %s", err)})
			return
		}

		// 验证用户名密码
		var u UserLogin
		err = c.ShouldBind(&u)
		if err != nil {
			// 这里返回200然后在body内响应500是合理的，把系统错误和业务错误区分开
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
			return
		}
		fmt.Println("received login req, username:", u.Username, ", password:", u.Password)

		backU, err := myutils.GlobalDBClient.Mongo.FindUserByKeyword(map[string]any{"username": u.Username})
		if err != nil {
			fmt.Println("err:", err)
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "用户名或密码错误"})
			return
		}

		hashedPass := myutils.Sha256Str(u.Password)
		if hashedPass != backU.Password {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "用户名或密码错误"})
			return
		}

		// 到这用户名密码都对了，生成token以及JWT响应给客户端
		jwtToken, err := generateToken(u.Username, 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
			return
		}

		err = myutils.GlobalDBClient.Mongo.UpdateUserLogin(map[string]any{"username": u.Username}, myutils.GetLocalNowTime())
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "login success", "token": jwtToken})
	}
}

type UserRegister struct {
	Username  string `form:"username" binding:"required"`
	Password  string `form:"password" binding:"required"`
	Lastname  string `form:"lastname"`
	Firstname string `form:"firstname"`
	Email     string `form:"email"`
	Phone     string `form:"phone"`
}

func handleUserRegister() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 验证许可证授权情况
		licVerified, err := verifyLicense()
		if !licVerified || err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": fmt.Sprintf("许可证验证失败, got err: %s", err)})
			return
		}

		// 查询用户存在情况
		var u UserRegister
		err = c.ShouldBind(&u)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": err.Error()})
			return
		}
		fmt.Println("received register req, username:", u.Username, ", password:", u.Password)

		backU, err := myutils.GlobalDBClient.Mongo.FindUserByKeyword(map[string]any{"username": u.Username})
		if err == nil && backU != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "用户已存在"})
			return
		}

		// 密码进行sha256哈希存储
		hashedPass := myutils.Sha256Str(u.Password)
		err = myutils.GlobalDBClient.Mongo.InsertUser(u.Username, hashedPass, u.Lastname, u.Firstname, u.Email, u.Phone)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": fmt.Sprintf("创建用户失败, got err: %s", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "register success"})
	}
}
