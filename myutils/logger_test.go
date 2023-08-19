package myutils

import (
	"testing"
)

func TestLogDockerCrawlerString(t *testing.T) {
	LogDockerCrawlerString(LogLevel.Error, "this is error")
	LogDockerCrawlerString(LogLevel.Warn, "this is warn")
	LogDockerCrawlerString(LogLevel.Info, "this is info")
	LogDockerCrawlerString(LogLevel.Debug, "this is debug")
}
