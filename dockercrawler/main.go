package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"strconv"
	"sync"
	"time"
)

// done 用于标识整个爬虫结束
var done = make(chan struct{})

func main() {
	//// 递归[]string{"00"-"zz"}, 不停尝试直到RepoList.Count < 9500, 只需要制定一个轮换规则, 记录当前状态即可
	//
	//// 当找到关键字使RepoList.Count < 9000时，遍历每一页，爬取仓库信息
	//go crawler.ScrapeRegRepoListRecursive("mongo", "community")
	//
	//// 处理Repo list，对每个Repo递归找Tag
	//time.Sleep(time.Second * 5)
	//for r := range crawler.ChannelRegRepoList {
	//	fmt.Println(r)
	//}

	//var Repo crawler.Repository__
	//c := crawler.GetRepoMetadataCollector(Repo)
	//c.Visit(crawler.GetRepoMetaURL("library", "mongo"))
	//var TagR crawler.TagReceiver__
	//c2 := crawler.GetRepoTagsCollector(&TagR)
	//c2.Visit(crawler.GetRepoTagsURL("library", "mongo", "1", "4"))
	//fmt.Println(TagR)
	//time.Sleep(time.Second * 3)
	//c3 := crawler.GetImageHistoryCollector(&TagR.Results[0].Archs)
	//c3.Visit(crawler.GetImageHistoryURL("library", "mongo", "latest"))
	//fmt.Println(TagR)

	//fmt.Println(crawler.GetNamespaceURL("aa281916", "1", "4"))
	//fmt.Println(crawler.GetRepoMetaURL("aa281916", "getting-started"))
	//fmt.Println(crawler.GetRepoTagsURL("aa281916", "getting-started", "1", "4"))
	//fmt.Println(crawler.GetImageMetaURL("aa281916", "getting-started", "latest"))
	//fmt.Println(crawler.GetImageHistoryURL("aa281916", "getting-started", "latest"))
	// 访问地址
	//for _, i := range []string{"1"} {
	//	c.Visit(strings.Replace(RegURLTemplate, "{PAGE}", i, 1))
	//}
	//c := crawler.GetDockerHubCollector()
	//fmt.Println(c)

	//sem := semaphore.NewWeighted(3)
	//var wg sync.WaitGroup
	//ctx := context.Background()
	//for i := 0; i < 10; i++ {
	//	sem.Acquire(ctx, 1)
	//	wg.Add(1)
	//	go func(j int) {
	//		time.Sleep(3 * time.Second)
	//		fmt.Println("From: ", j)
	//		defer func() {
	//			sem.Release(1)
	//			wg.Done()
	//		}()
	//	}(i)
	//}
	//wg.Wait()
	//go func() { time.Sleep(time.Second * 3); done <- struct{}{} }()
	//// 退出程序
	//<-done

	c := colly.NewCollector()
	fmt.Println(&c)
	if err := c.Limit(&colly.LimitRule{
		DomainGlob: "learnku.com",
		Delay:      5 * time.Second,
	}); err != nil {
		fmt.Println(err)
	}
	c.OnRequest(func(request *colly.Request) {
		fmt.Println("FROM: ", request.URL)
	})
	wg := sync.WaitGroup{}
	for j := 0; j < 10; j++ {
		wg.Add(1)
		go func(j int) {
			if err := c.Request("GET", "https://learnku.com/docs/the-way-to-go/106-method/"+strconv.Itoa(j), nil, nil, nil); err != nil {
				fmt.Println(err)
			}
			wg.Done()
		}(j)
	}

	wg.Wait()

	time.Sleep(time.Second)
}
