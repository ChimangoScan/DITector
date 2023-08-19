package myutils

import (
	"fmt"
	"log"
	"os"
	"path"
)

func init() {
	// 初始化日志文件
	var err error
	logFilepath := path.Join("/data/docker-crawler", "docker-crawler.log")
	fileLogger, err = os.OpenFile(logFilepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0744)
	if err != nil {
		log.Fatalf("[ERROR] Open %s failed with: %s\n", logFilepath, err)
	} else {
		fmt.Println("[+] Open log file succeed: ", logFilepath)
	}
}
