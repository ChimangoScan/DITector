package server

import (
	"fmt"
	"log"
	"myutils"
)

var (
	myMongo       *myutils.MyMongo
	myNeo4jDriver *myutils.MyNeo4j

	// totalCnt is the number of documents in each collection,
	// used for calculate table pages
	totalRepositoriesCnt int64
	totalImagesCnt       int64
)

func configServer(initFlag bool) {
	var err error
	myMongo, err = myutils.ConfigMongoClient(initFlag)
	if err != nil {
		log.Fatalln("[ERROR] connect to and config MongoDB failed with err: ", err)
	}
	fmt.Println("[+] Connect to MongoDB succeed")

	myNeo4jDriver, err = myutils.ConfigNewNeo4jDriverWithContext("neo4j://localhost:7687", "neo4j", "qazwsxedc")
	if err != nil {
		log.Fatalln("[ERROR] Connect to neo4j failed with:", err)
	}
	fmt.Println("[+] Connect to Neo4j succeed")
}

func updateImageCnt() {

}

func updateRepositoriesCnt() {

}
