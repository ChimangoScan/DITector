package myutils

import (
	"os"
	"strings"
	"sync"
	"time"
)

// logger.go 记录build日志

var (
	fileLogger     *os.File
	lockFileLogger = sync.Mutex{}
)

var LogLevel = struct {
	Error string
	Warn  string
	Info  string
	Debug string
}{
	"[ERROR]",
	"[WARN]",
	"[INFO]",
	"[DEBUG]",
}

func LogDockerCrawlerString(s ...string) {
	lockFileLogger.Lock()
	defer lockFileLogger.Unlock()
	tmp := strings.Join(s, " ")
	fileLogger.WriteString(time.Now().Add(8*time.Hour).Format(time.DateTime) + " " + tmp + "\n")
}
