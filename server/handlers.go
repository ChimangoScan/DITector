package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Musso12138/docker-scan/myutils"
	"github.com/gin-gonic/gin"
)

func handleRepositoriesSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST")

		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil || pageSize < 1 {
			pageSize = 10
		}

		var totalCnt int64
		var results []*myutils.Repository

		// search允许是带/的名称
		if search == "" {
			// 使用stats获取集合元素数量
			totalCnt = totalRepoCnt
			results, err = myutils.GlobalDBClient.Mongo.FindRepositoriesByKeywordPaged(map[string]any{}, page, pageSize)
		} else {
			// 没有/的时候通过$text匹配
			if !strings.Contains(search, "/") {
				totalCnt, _ = myutils.GlobalDBClient.Mongo.CountRepoByText(search)
				results, err = myutils.GlobalDBClient.Mongo.FindRepositoriesByText(search, page, pageSize)
			} else {
				registry, namespace, name, _, _ := myutils.DivideImageName(search)
				switch registry {
				case "docker.io":
					// namespace和name都不是空
					if namespace != "" && name != "" {
						totalCnt, _ = myutils.GlobalDBClient.Mongo.CountRepoByKeyword(map[string]any{
							"namespace": namespace,
							"name":      name,
						})
						results, err = myutils.GlobalDBClient.Mongo.FindRepositoriesByKeywordPaged(map[string]any{
							"namespace": namespace,
							"name":      name,
						}, page, pageSize)
					}
				}
			}
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"msg": err.Error(),
			})
		} else {
			// used to handle CORS requests
			c.JSON(http.StatusOK, gin.H{
				"count":     totalCnt,
				"page":      page,
				"page_size": pageSize,
				"results":   results,
			})
		}
	}
}

func handleTagsSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST")

		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil || pageSize < 1 {
			pageSize = 10
		}

		var totalCnt int64
		var results []*myutils.Tag

		// search允许是带/的名称
		if search == "" {
			// 使用stats获取集合元素数量
			totalCnt = totalTagCnt
			results, err = myutils.GlobalDBClient.Mongo.FindTagByKeywordPaged(map[string]any{}, page, pageSize)
		} else {
			// 输入的字符串是个SHA256哈希值
			if strings.HasPrefix(search, "sha256:") && len(search) == 71 {
				results, err = myutils.GlobalDBClient.Mongo.FindTagByImgDigestPaged(search, page, pageSize)
			} else if !strings.Contains(search, "/") && !strings.Contains(search, ":") {
				// 没有/和:的时候通过text匹配
				totalCnt, _ = myutils.GlobalDBClient.Mongo.CountTagByText(search)
				results, err = myutils.GlobalDBClient.Mongo.FindTagByTextPaged(search, page, pageSize)
			} else {
				registry, repoNamespace, repoName, tagName, _ := myutils.DivideImageName(search)
				switch registry {
				case "docker.io":
					if !strings.Contains(search, ":") {
						totalCnt, _ = myutils.GlobalDBClient.Mongo.CountTagByKeyword(map[string]any{
							"repositories_namespace": repoNamespace,
							"repositories_name":      repoName,
						})
						results, err = myutils.GlobalDBClient.Mongo.FindTagByKeywordPaged(map[string]any{
							"repositories_namespace": repoNamespace,
							"repositories_name":      repoName,
						}, page, pageSize)
					} else {
						totalCnt, _ = myutils.GlobalDBClient.Mongo.CountTagByKeyword(map[string]any{
							"repositories_namespace": repoNamespace,
							"repositories_name":      repoName,
							"name":                   tagName,
						})
						results, err = myutils.GlobalDBClient.Mongo.FindTagByKeywordPaged(map[string]any{
							"repositories_namespace": repoNamespace,
							"repositories_name":      repoName,
							"name":                   tagName,
						}, page, pageSize)
					}
				}
			}
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"msg": err.Error(),
			})
		} else {
			// used to handle CORS requests
			c.JSON(http.StatusOK, gin.H{
				"count":     totalCnt,
				"page":      page,
				"page_size": pageSize,
				"results":   results,
			})
		}
	}
}

func handleImagesSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.ParseInt(pageStr, 10, 64)
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
		if err != nil || pageSize < 1 {
			pageSize = 10
		}

		var totalCnt int64
		var results []*myutils.Image

		if search == "" {
			// 使用stats获取集合元素数量
			totalCnt = totalImgCnt
			results, err = myutils.GlobalDBClient.Mongo.FindImageByKeywordPaged(map[string]any{}, page, pageSize)
		} else {
			if len(search) != 71 || !strings.HasPrefix(search, "sha256:") {
				c.JSON(http.StatusBadRequest, gin.H{
					"msg": "invalid input string for search, need to be a valid digest start with sha256:",
				})
				return
			} else {
				totalCnt, _ = myutils.GlobalDBClient.Mongo.CountImageByKeyword(map[string]any{
					"digest": search,
				})
				results, err = myutils.GlobalDBClient.Mongo.FindImageByKeywordPaged(map[string]any{
					"digest": search,
				}, page, pageSize)
			}
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"msg": err.Error(),
			})
		} else {
			// used to handle CORS requests
			c.Header("Access-Control-Allow-Origin", "*")
			c.JSON(http.StatusOK, gin.H{
				"count":     totalCnt,
				"page":      page,
				"page_size": pageSize,
				"results":   results,
			})
		}
	}
}

func handleResultsSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		totalCnt := totalImgCnt
		if search != "" {
			// time costs too much
			totalCnt, _ = myutils.GlobalDBClient.Mongo.CountImgResByText(search)
		}
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			pageSize = 10
		}

		results, err := myutils.GlobalDBClient.Mongo.FindImgResultByText(search, int64(page), int64(pageSize))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"msg": err.Error(),
			})
		} else {
			// used to handle CORS requests
			c.Header("Access-Control-Allow-Origin", "*")
			c.JSON(http.StatusOK, gin.H{
				"count":     totalCnt,
				"page":      page,
				"page_size": pageSize,
				"results":   ResultsToImagesWithResults(results),
			})
		}
	}
}

func handleResultSearch() func(c *gin.Context) {
	return func(c *gin.Context) {
		search := c.DefaultQuery("search", "")
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("page_size", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		totalCnt := totalImgCnt
		if search != "" {
			// time costs too much
			totalCnt, _ = myutils.GlobalDBClient.Mongo.CountImgResByText(search)
		}
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			pageSize = 10
		}

		results, err := myutils.GlobalDBClient.Mongo.FindImgResultByText(search, int64(page), int64(pageSize))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"msg": err.Error(),
			})
		} else {
			// used to handle CORS requests
			c.Header("Access-Control-Allow-Origin", "*")
			c.JSON(http.StatusOK, gin.H{
				"count":     totalCnt,
				"page":      page,
				"page_size": pageSize,
				"results":   ResultsToImagesWithResults(results),
			})
		}
	}
}
