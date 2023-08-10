package main

import (
	"buildgraph"
	"crawler"
	"flag"
	"os"
)

func main() {
	var (
		crawl       string // 指定要爬的镜像仓库，比如dockerhub
		libraryFlag bool   // 爬虫是否爬官方镜像

		buildGraph bool // 是否要建信息库

		format string // 爬虫存储格式/信息库从什么格式中取内容，json、mysql
	)

	flag.StringVar(&crawl, "crawl", "", "crawl the register if not nil, e.g. dockerhub")
	flag.BoolVar(&libraryFlag, "official", false, "true for crawling official images; false for crawling community images")
	flag.BoolVar(&buildGraph, "build-graph", false, "true for building graph based on crawler results")
	flag.StringVar(&format, "format", "json", "format for crawling or building graph, e.g. json, mysql, clear")
	flag.Parse()

	if crawl != "" {
		if crawl == "dockerhub" {
			crawler.StartRecursive(format, libraryFlag)
		}
	} else if buildGraph {
		buildgraph.Build(format)
	} else {
		flag.Usage()
		os.Exit(-1)
	}
}
