package main

import (
	"crawler"
	"fmt"
)

func main() {
	fmt.Println(crawler.GetRegURL("mongo", "community", "1", "4"))
	fmt.Println(crawler.GetNamespaceURL("aa281916", "1", "4"))
	fmt.Println(crawler.GetRepoMetaURL("aa281916", "getting-started"))
	fmt.Println(crawler.GetRepoTagsURL("aa281916", "getting-started", "1", "4"))
	fmt.Println(crawler.GetImageMetaURL("aa281916", "getting-started", "latest"))
	fmt.Println(crawler.GetImageHistoryURL("aa281916", "getting-started", "latest"))
	// 访问地址
	//for _, i := range []string{"1"} {
	//	c.Visit(strings.Replace(RegURLTemplate, "{PAGE}", i, 1))
	//}
}
