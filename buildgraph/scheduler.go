package buildgraph

import (
	"fmt"
	"runtime"
)

// scheduler.go 负责分发任务，从chan中获取内容并组织为合适的形式存入数据库

var (
	// chanLimitMainGoroutine 限制goroutine数量
	chanLimitMainGoroutine chan struct{}
)

var (
	chanRepository     = make(chan *Repository, runtime.NumCPU())
	chanTag            = make(chan *Tag, runtime.NumCPU())
	chanImage          = make(chan *Image, runtime.NumCPU())
	chanDoneRepository = make(chan struct{})
	chanDoneTag        = make(chan struct{})
	chanDoneImage      = make(chan struct{})
)

// StartFromJSON 启动以JSON文件为数据源的信息库建设过程
func StartFromJSON() {
	go StoreRepositoryScheduler()
	ReadFileRepositoryByLine()
	go StoreTagScheduler()
	ReadFileTagsByLine()
	go StoreImageScheduler()
	ReadFileImagesByLine()

	<-chanDoneRepository
	<-chanDoneTag
	<-chanDoneImage
}

func StoreRepositoryScheduler() {
	for repo := range chanRepository {
		fmt.Println(repo.Namespace, repo.Name)
	}
}

func StoreTagScheduler() {
	for tag := range chanTag {
		fmt.Println(tag.Namespace, tag.Repository, tag.Name)
	}
}

func StoreImageScheduler() {
	for image := range chanImage {
		fmt.Println(image.Namespace, image.Repository, image.Arch.Digest)
	}
}
