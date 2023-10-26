package myutils

import (
	"os"
	"strings"
	"sync"
)

type MyLogger struct {
	logger   *os.File
	lock     sync.Mutex
	logLevel int
}

var Logger = new(MyLogger)

var LogLevel = struct {
	Critical int
	Error    int
	Warn     int
	Info     int
	Debug    int
}{
	5,
	4,
	3,
	2,
	1,
}

var LogLevelStr = struct {
	Critical string
	Error    string
	Warn     string
	Info     string
	Debug    string
}{
	"[Critical]",
	"[ERROR]",
	"[WARN]",
	"[INFO]",
	"[DEBUG]",
}

// configLogger 用于初始化日志模块，打开日志文件，配置日志级别
func configLogger(logFilepath string, logLevel int) error {
	var err error
	Logger.logger, err = os.OpenFile(logFilepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0744)
	Logger.lock = sync.Mutex{}
	Logger.logLevel = logLevel
	return err
}

func (l *MyLogger) Critical(s ...string) {
	if l.logLevel <= LogLevel.Critical {
		l.logString(LogLevelStr.Critical, s...)
	}
}

func (l *MyLogger) Error(s ...string) {
	if l.logLevel <= LogLevel.Error {
		l.logString(LogLevelStr.Error, s...)
	}
}

func (l *MyLogger) Warn(s ...string) {
	if l.logLevel <= LogLevel.Warn {
		l.logString(LogLevelStr.Warn, s...)
	}
}

func (l *MyLogger) Info(s ...string) {
	if l.logLevel <= LogLevel.Info {
		l.logString(LogLevelStr.Info, s...)
	}
}

func (l *MyLogger) Debug(s ...string) {
	if l.logLevel <= LogLevel.Debug {
		l.logString(LogLevelStr.Debug, s...)
	}
}

func (l *MyLogger) logString(levelStr string, s ...string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	tmp := strings.Join(s, " ")
	l.logger.WriteString(GetLocalNowTime() + " " + levelStr + " " + tmp + "\n")
}

func (l *MyLogger) Close() error {
	return l.logger.Close()
}
