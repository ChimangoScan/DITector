package main

import (
	"db"
	"fmt"
	"time"
)

func main() {

	//// 开始爬虫
	//crawler.StartRecursive()

	d, _ := db.NewDockerDB("docker:docker@/dockerhub")
	r, err := d.InsertTag("xmrig2021", "r2021", "latest", "", "", "",
		"", "", "")
	i, j := r.RowsAffected()
	fmt.Println(i, j, "\n", err)
	time.Sleep(time.Second)
}
