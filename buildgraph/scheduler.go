package buildgraph

import "fmt"

// scheduler.go 负责分发任务，从chan中获取内容并组织为合适的形式存入数据库

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
		fmt.Println(repo)
	}
}

func StoreTagScheduler() {

}

func StoreImageScheduler() {

}
