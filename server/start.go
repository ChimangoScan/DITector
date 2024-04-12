package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func StartServer(port string) {
	configServer()

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.GET("/repositories", handleRepositoriesSearch())
	router.GET("/tags", handleTagsSearch())
	router.GET("/images", handleImagesSearch())
	router.GET("/results", handleResultsSearch())
	router.GET("/result", handleResultSearch())

	router.Run(":" + port)
}
