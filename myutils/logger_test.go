package myutils

import (
	"fmt"
	"testing"
)

func TestLogDockerCrawlerString(t *testing.T) {
	LogDockerCrawlerString(LogLevel.Error, "this is error")
	LogDockerCrawlerString(LogLevel.Warn, "this is warn")
	LogDockerCrawlerString(LogLevel.Info, "this is info")
	LogDockerCrawlerString(LogLevel.Debug, "this is debug")
}

func TestGetLocalNowTime(t *testing.T) {
	fmt.Println(GetLocalNowTime())
}
