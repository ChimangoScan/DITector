package myutils

import (
	"os"
	"strings"
	"sync"
)

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

// configLogger 用于初始化日志模块，打开日志文件
func configLogger(logFilepath string) error {
	var err error
	fileLogger, err = os.OpenFile(logFilepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0744)
	return err
}

func LogDockerCrawlerString(s ...string) {
	lockFileLogger.Lock()
	defer lockFileLogger.Unlock()
	tmp := strings.Join(s, " ")
	fileLogger.WriteString(GetLocalNowTime() + " " + tmp + "\n")
}

func CloseLogger() error {
	return fileLogger.Close()
}
