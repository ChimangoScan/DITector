// 此文件用于补充对镜像的分析，用于北京项目的结题

package scripts

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/Musso12138/docker-scan/analyzer"
	"github.com/Musso12138/docker-scan/myutils"
)

func SupplementImageAnalysis(page int64, pageSize int64, tagCnt int, partial bool) error {
	// 配置线程数
	//maxThreads := runtime.NumCPU()
	//if myutils.GlobalConfig.MaxThread > 0 && myutils.GlobalConfig.MaxThread < maxThreads {
	//	maxThreads = myutils.GlobalConfig.MaxThread
	//	runtime.GOMAXPROCS(maxThreads)
	//}
	myutils.Logger.Debug(fmt.Sprintf("analyze-all start with threads: %d", myutils.GlobalConfig.MaxThread))

	// 初始化控制并发线程数的管道
	jobCh := make(chan job)
	wg := sync.WaitGroup{}

	for w := 1; w <= myutils.GlobalConfig.MaxThread; w++ {
		wg.Add(1)
		go supplementWorker(w, jobCh, &wg)
	}

	wg.Add(1)
	go jobGeneratorSupplement(page, pageSize, tagCnt, partial, jobCh, &wg)

	wg.Wait()

	return nil
}

// jobGeneratorSupplement 从MongoDB读取数据
// 最重要的就是快！所以只对repo、tag、image metadata都在本地Mongo的数据生成检测任务
func jobGeneratorSupplement(page int64, pageSize int64, tagCnt int, partial bool, jobCh chan<- job, wg *sync.WaitGroup) {
	defer close(jobCh)
	defer wg.Done()
	if !myutils.GlobalDBClient.MongoFlag {
		log.Fatalln("jobGeneratorAll got error: MongoDB not online")
	}

	generatedWorks := 0
	var repoPage int64 = page
	// var pageSize int64 = 5
	for {
		repoDocs, err := myutils.GlobalDBClient.Mongo.FindRepositoriesByKeywordPaged(nil, repoPage, pageSize)
		if err != nil {
			myutils.Logger.Error(fmt.Sprintf("find repository in MongoDB page: %d, pagesize: %d, got error: %s", repoPage, pageSize, err))
			continue
		}
		// 进程结束标志: mongodb中没有更多repo
		if len(repoDocs) == 0 {
			break
		}

		// 根据tag生成任务
		for _, repoDoc := range repoDocs {
			tagDocs, err := myutils.GlobalDBClient.Mongo.FindTagsByRepoNamePaged(repoDoc.Namespace, repoDoc.Name, 1, int64(tagCnt))
			if err != nil {
				myutils.Logger.Error(fmt.Sprintf("find tags for repository %s/%s in MongoDB page: %d, pagesize: %d, got error: %s", repoDoc.Namespace, repoDoc.Name, 1, tagCnt, err))
				continue
			}

			// 生产任务
			// tag全是从Mongo获取的
			for _, tagDoc := range tagDocs {
				// tag已经检查过，跳过当前tag
				if _, err := myutils.GlobalDBClient.Mongo.FindImgResultByName(tagDoc.RepositoryNamespace, tagDoc.RepositoryName, tagDoc.Name, ""); err == nil {
					myutils.Logger.Warn("repo:tag already analyzed before:", tagDoc.RepositoryNamespace, "/", tagDoc.RepositoryName, ":", tagDoc.Name)
					continue
				}

				digest := ""

				// 对每个tag至多取一个image
				for _, img := range tagDoc.Images {
					// 对应的digest已经检查过，跳过
					if _, err := myutils.GlobalDBClient.Mongo.FindImgResultByDigest(img.Digest); err == nil {
						continue
					}
					// 对应的image在数据库
					if _, err := myutils.GlobalDBClient.Mongo.FindImageByDigest(img.Digest); err == nil {
						digest = img.Digest
						break
					}
					// 根据arch大概选一个digest
					if img.Architecture == "amd64" || img.Architecture == "arm64" || img.Architecture == "unknown" || img.Architecture == "" {
						digest = img.Digest
					}
				}

				if digest != "" {
					jobCh <- job{
						name:    fmt.Sprintf("%s/%s:%s@%s", repoDoc.Namespace, repoDoc.Name, tagDoc.Name, digest),
						partial: partial,
					}
					generatedWorks++
				}
			}
		}

		fmt.Printf("[%s] generatied all job for repo page: %d, page_size: %d, generated works: %d\n", myutils.GetLocalNowTimeStr(), repoPage, pageSize, generatedWorks)
		repoPage++
	}
}

// supplementWorker 用于补充检测镜像的worker
func supplementWorker(workerId int, jobCh <-chan job, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobCh {
		if j.partial {
			_, err := analyzer.AnalyzeImagePartialByName(j.name)
			if err != nil {
				myutils.Logger.Error("analyzeAllWorker", strconv.Itoa(workerId), "analyze partial image", j.name, "failed with:", err.Error())
			} else {
				myutils.Logger.Debug("analyzeAllWorker", strconv.Itoa(workerId), "analyze partial image", j.name, "succeeded")
			}
		} else {
			_, err := analyzer.AnalyzeImageByName(j.name, true)
			if err != nil {
				myutils.Logger.Error("analyzeAllWorker", strconv.Itoa(workerId), "analyze image", j.name, "failed with:", err.Error())
			} else {
				myutils.Logger.Debug("analyzeAllWorker", strconv.Itoa(workerId), "analyze partial image", j.name, "succeeded")
			}
		}
	}
}
