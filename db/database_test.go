package db

import (
	"fmt"
	"testing"
)

func TestNewDockerDB(t *testing.T) {
	db, err := NewDockerDB("docker:docker@tcp(localhost:3306)/dockerhub")
	defer db.db.Close()
	if err != nil {
		t.Fatal("[ERROR] Getting DockerDB: ", err)
	}
	err = db.db.Ping()
	if err != nil {
		t.Fatal("[ERROR] Ping DockerDB.db failed with: ", err)
	}
	fmt.Println("[+] TestNewDockerDB Pass!")
}
