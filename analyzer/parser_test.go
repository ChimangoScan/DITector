package analyzer

import (
	"fmt"
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
	ci.parseMetadata()
}
