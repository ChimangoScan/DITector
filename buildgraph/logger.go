package buildgraph

import (
	"os"
	"strings"
	"sync"
	"time"
)

// logger.go 记录build日志

var (
	fileBuilderLogger     *os.File
	lockFileBuilderLogger = sync.Mutex{}
)

func logBuilderString(s ...string) {
	lockFileBuilderLogger.Lock()
	defer lockFileBuilderLogger.Unlock()
	tmp := strings.Join(s, " ")
	fileBuilderLogger.WriteString(time.Now().Add(8*time.Hour).Format(time.DateTime) + " " + tmp + "\n")
}
