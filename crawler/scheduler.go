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

// CrawlDockerHubStaged 划分阶段进行整个DockerHub的爬取。
// 未必进行实现，在实际实现中仍考虑任务分发全过程并发，而非分阶段在阶段内并发。
func CrawlDockerHubStaged() {
	// Stage1 发现所有可用关键字
	// Stage2 根据可用关键字爬取keyword->Repo List，发现全部Repo Name
	// Stage3 根据Repo Name爬取Repo的全部Tags
	// Stage4(可并入Stage3) 根据Tag爬取Tag对应的Arch History
}

// DistributeKeywordToScrapeRegRepoList 负责具体将Repo count<9000的keyword分发给ScrapeRegRepoListRecursive。
func DistributeKeywordToScrapeRegRepoList(kc chan string) {
	for k := range kc {
		// 尝试拿到
		fmt.Println(k)
	}
}
