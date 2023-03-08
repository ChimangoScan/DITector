package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
)

// GetCollector 用于配置一个适用于本项目的colly.Collector
func GetCollector() *colly.Collector {
	// 创建新的Collector
	c := colly.NewCollector(
		colly.AllowedDomains("hub.docker.com"),
	)

	// 配置Collector
	// 配置代理池
	//if p, err := proxy.RoundRobinProxySwitcher(
	//	"https://127.0.0.1:8080",
	//	"https://127.0.0.1:8081",
	//	"https://127.0.0.1:8082",
	//); err == nil {
	//	c.SetProxyFunc(p)
	//}

	return c
}

// GetRegisterCollector 为爬取指定Register的Repo list的Collector绑定回调函数
func GetRegisterCollector() *colly.Collector {
	c := GetCollector()

	// 绑定回调函数
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("FROM OnRequest-----------------------")
		fmt.Println("Visiting: ", r.URL)
		// 查看request时使用的proxy
		fmt.Println("Proxy: ", r.ProxyURL)
		// 查看Cookie，如果有要清除，否则容易封号
		fmt.Println("Cookie: ", r.Headers.Get("Cookie"))
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("FROM OnResponse-----------------------")
		fmt.Println("Status Code", r.StatusCode)
		var content map[string]interface{}
		json.NewDecoder(bytes.NewReader(r.Body)).Decode(&content)
		fmt.Println(content)
	})

	return c
}
