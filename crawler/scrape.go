package crawler

import "github.com/gocolly/colly"

// 实现一些统一的遍历爬取

// ScrapeRegRepoListRecursive 在已经确定source下q=keyword时，匹配条目数count<9500时，
// 递归遍历该关键字的repo结果，拿到全部的repo名。
func ScrapeRegRepoListRecursive(c *colly.Collector, keyword, source string) {
	for _, i := range []string{"1", "2", "3"} {
		if err := c.Visit(GetRegURL(keyword, source, i, "4")); err != nil {
			continue
		}
	}
	close(ChannelRegRepoList)
}

// ScrapeRepoInfo 用于爬取指定repo的metadata和全部tag的信息。
// 考虑在内部进一步将metadata和tag持久化。
func ScrapeRepoInfo(namespace, repo string) {
	// 思路1-------------------------
	// 建立有效管道每阶段都在传数据
	// ch1 := make(chan Repository__)
	// ScrapeRepoMetadata 爬Metadata，结果传进ch1

	// 读ch1
	// ch2 := make(chan TagReceiver__ 收tag list)
	// ScrapeRepoTagsRecursive爬tag list传进ch2

	// 思路2-------------------------
	// GetCollector时候传入&Repository__
	// ch := make(chan TagReceiver__ 收tag list)
	// ScrapeRepoTagsRecursive爬tag list传进ch
	// 读ch
	// for t := range ch {
	// 	for _, tag := range t.Results {
	// 		进一步爬每个tag的Archs
	//
	//	}
	// }
	// 后续都在这个基础上
}

// ScrapeRepoMetadata 用于爬取指定repo的metadata，返回一个。
func ScrapeRepoMetadata(namespace, repo string) {

}

// ScrapeRepoTagsRecursive 递归爬取指定Repo的全部Tag记录。
func ScrapeRepoTagsRecursive(c *colly.Collector, namespace, repo string) {
	for _, i := range []string{"1"} {
		if err := c.Visit(GetRepoTagsURL(namespace, repo, i, "100")); err != nil {
			continue
		}
	}
}
