package buildgraph

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// neo4j.go 用于操作neo4j

// neo4jDriver 相当于neo4j connector
var neo4jDriver neo4j.DriverWithContext

// InsertImageToNeo4j 将
func InsertImageToNeo4j(image *ImageSource) {
	// 创建一个neo4j session
	session := neo4jDriver.NewSession(context.TODO(), neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(context.TODO())

	// 用于堆1、1-2、1-2-5，方便直接计算hash
	accumulateLayerID := ""

	for i, _ := range image.Image.Layers {
		// 跳过没有文件内容的层
		if image.Image.Layers[i].Size == 0 {
			continue
		}

		// 计算hash(1-2-5)，转成string类型
		accumulateLayerID += image.Image.Layers[i].Digest[7:]
		tmpHash := sha256.Sum256([]byte(accumulateLayerID))
		accumulateHash := string(tmpHash[:])
		fmt.Println(accumulateLayerID)
		fmt.Println(accumulateHash)

		// TODO: 实现neo4j插入
	}
}
