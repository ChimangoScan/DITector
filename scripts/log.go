package scripts

import (
	"myutils"
	"os"
	"strings"
)

var selflogger, _ = os.OpenFile("/data/docker-crawler/results/dependent-weights-top100.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0744)

func logself(s ...string) {
	tmp := strings.Join(s, " ")
	selflogger.WriteString(myutils.GetLocalNowTime() + " " + tmp + "\n")
}
