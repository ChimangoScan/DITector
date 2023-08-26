package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// handleImageSearch return a function used for images
// search API exported by gin framework
//
// URI arguments:
// search: keyword
func handleImageSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil {
			page = 1
		} else if page < 1 {
			page = 1
		}

		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			pageSize = 10
		} else if pageSize < 1 {
			pageSize = 10
		}

		results, err := myMongo.FindImagesByText(search, int64(page), int64(pageSize))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"msg": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"page":      page,
				"page_size": pageSize,
				"results":   results,
			})
		}
	}
}
