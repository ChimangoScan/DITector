package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func StartServer() {
	configServer(false)

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.GET("/images", handleImagesSearch())
	router.GET("/repositories", handleRepositoriesSearch())
	router.GET("/results", handleResultSearch())

	router.Run(":23434")
}
