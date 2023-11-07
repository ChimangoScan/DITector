package myutils

import (
	"fmt"
	"log"
	"testing"
)

func TestFindImgResultByDigest(t *testing.T) {
	if GlobalDBClient.MongoFlag {
		ir, err := GlobalDBClient.Mongo.FindImgResultByDigest("allfortest")
		if err != nil {
			log.Fatalln("got err", err)
		}
		fmt.Println(ir)
	} else {
		log.Fatalln("mongo not connected")
	}
	return
}
