package crawler

import "fmt"

// 负责整个爬虫的核心调度，启动goroutine等。

// 定义一系列控制goroutine数量的通道

var (
	// ChanLimitRepoListScraper 限制核心调度器goroutine数量，即并发ScrapeRegRepoListRecursive的数量
	ChanLimitRepoListScraper = make(chan struct{}, 1)
	// ChanLimitRepoInfoScraper 限制
	ChanLimitRepoInfoScraper = make(chan struct{}, 1)
)

// DistributeKeywordToScrapeRegRepoList 负责具体将Repo count<9000的keyword分发给ScrapeRegRepoListRecursive。
func DistributeKeywordToScrapeRegRepoList(kc chan string) {
	for k := range kc {
		// 尝试拿到
		fmt.Println(k)
	}
}
