package analyzer

import (
	"fmt"
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	ci := CurrentImage{dockerClient: imageAnalyzer.DockerClient, name: "curlimages/curl:latest"}
	ci.Parse()

	// 查看系统平台
	fmt.Println(ci.architecture, ci.os)

	// 查看元数据信息
	fmt.Println(ci.metadata.repositoryMetadata)

	// 查看配置信息
	fmt.Println(ci.configuration.RepoTags, ci.configuration.Architecture, ci.configuration.Variant)

	// 查看内容信息
	fmt.Println(ci.layerInfoMap[ci.layerWithContentList[0]])
}

func TestParseMetadata(t *testing.T) {
	ci := CurrentImage{dockerClient: imageAnalyzer.DockerClient, name: "curlimages/curl:latest"}
	ci.parseName()
	ci.getServerPlatform()

	if err := ci.parseMetadata(true); err != nil {
		log.Fatalln("parse metadata failed with:", err)
	}

	fmt.Println(ci.architecture, ci.os)

	fmt.Println(ci.metadata.repositoryMetadata.Namespace, ci.metadata.repositoryMetadata.Name)

	fmt.Println(ci.metadata.tagMetadata.Name, ci.metadata.tagMetadata.LastUpdated)

	fmt.Println(ci.metadata.imageMetadata.Digest)
}
