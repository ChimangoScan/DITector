package analyzer

import (
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"log"
	"testing"
)

func TestPullSaveExtractImage(t *testing.T) {
	ci, err := NewCurrentImage("mongo:latest")
	if err != nil {
		log.Fatalln("create new current image got error:", err)
	}

	finish := make(chan downloadFinish)

	go ci.pullSaveExtractImage(myutils.GlobalConfig.TmpDir, finish)

	f := <-finish
	fmt.Println(f.imgTarPath)
	fmt.Println(f.imgDirPath)
	fmt.Println(f.err)

	fmt.Println(ci.manifest.Config)
	fmt.Println(ci.manifest.RepoTags)
	fmt.Println(ci.manifest.Layers)

	fmt.Println(ci.layerLocalFilepathList)
}

func TestParse(t *testing.T) {
	ci, err := NewCurrentImage("hello-world:latest")
	if err != nil {
		log.Fatalln("create new current image got error:", err)
	}
	ci.ParseFromDockerEnv()

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
	ci, err := NewCurrentImage("hello-world:latest")
	if err != nil {
		log.Fatalln("create new current image got error:", err)
	}
	ci.parseName()
	ci.parseServerPlatform()

	if err := ci.parseMetadata(true); err != nil {
		log.Fatalln("parse metadata failed with:", err)
	}

	fmt.Println(ci.architecture, ci.os)

	fmt.Println(ci.metadata.repositoryMetadata.Namespace, ci.metadata.repositoryMetadata.Name)

	fmt.Println(ci.metadata.tagMetadata.Name, ci.metadata.tagMetadata.LastUpdated)

	fmt.Println(ci.metadata.imageMetadata.Digest)
}
