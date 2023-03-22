package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

var ConfigCrawler struct {
	MaxThread int    `json:"max_thread"`
	ProxyFile string `json:"proxy_file"`
}

func init() {
	fb, err := os.ReadFile("crawler/config.json")
	if err != nil {
		fmt.Println("[ERROR] Failed to load crawler/config.json")
	}
	if err := json.Unmarshal(fb, &ConfigCrawler); err != nil {
		fmt.Println("[ERROR] Json failed to unmarshal crawler/config.json with err: ", err)
	}
	// 默认情况下，允许启动的核心goroutine数为系统可调内核数
	if ConfigCrawler.MaxThread == 0 {
		ConfigCrawler.MaxThread = runtime.GOMAXPROCS(runtime.NumCPU())
	}
	ChanLimitMainGoroutine = make(chan struct{}, ConfigCrawler.MaxThread)
	fmt.Println(ConfigCrawler)
}
