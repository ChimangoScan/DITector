package analyzer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

func TestAsky(t *testing.T) {
	scaResFilepath := "/Users/musso/workshop/docker-projects/test/sca.json"
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           nil,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	task, err := postCreateAskYTask(client, scaResFilepath)
	if err != nil {
		log.Fatalln("postCreateAskYTask got err:", err)
	}

	// 获取检测报告
	report, err := checkGetAskYReport(client, task)
	if err != nil {
		log.Fatalln("checkGetAskYReport got err:", err)
	}

	fmt.Println(report)
}

func TestExecWithTimeout(t *testing.T) {
	log.Println("start")
	timeout := 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", "sleep 10")
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		log.Println("finish with timeout")
	} else if err != nil {
		log.Println("exec failed with:", err)
	} else {
		log.Println("finish")
	}
}
