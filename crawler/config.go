package crawler

import (
	"encoding/json"
	"fmt"
	"os"
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
	fmt.Println(ConfigCrawler)
}
