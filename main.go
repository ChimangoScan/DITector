package main

import (
	"github.com/Musso12138/docker-scan/cmd"
	"log"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatalln("execute cobra cmd failed with:", err)
	}
}
