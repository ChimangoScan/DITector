package analyzer

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"myutils"
)

type CurrentImage struct {
	dockerClient *client.Client
	filepath     string
	name         string

	registry       string
	namespace      string
	repositoryName string
	tagName        string
	digest         string

	metadata          *metadata
	configuration     *types.ImageInspect
	layerLocalFileMap map[string]string

	Results *myutils.ImageResult
}

type metadata struct {
	repositoryMetadata *myutils.Repository
	tagMetadata        *myutils.Tag
	imageMetadata      *myutils.Image
}

// Parse TODO: 解析指定镜像的元数据、配置信息，下载镜像，定位镜像的各个层
func (currI *CurrentImage) Parse() {
	// 解析镜像配置信息，顺便检查image是否位于本地Docker环境中
	if !currI.parseConfigurationFromDockerEnv() {
		// 将镜像下载到本地
		rc, err := currI.dockerClient.ImagePull(context.TODO(), currI.name, types.ImagePullOptions{})
		if err != nil {
			myutils.LogDockerCrawlerString(myutils.LogLevel.Error, "pull image", currI.name, "failed with:", err.Error())
		} else {
			defer rc.Close()
		}
	}
}

// ParsePartial TODO: 解析指定镜像的元数据
func (currI *CurrentImage) ParsePartial() {

}

func (currI *CurrentImage) parseMetadata() {
	// 数据库在线时，尝试从数据库中读取
}

// parseConfigurationFromDockerEnv tries to inspect image from local env, with results
// stored to currI.Configuration, formatted like `docker image inspect`.
//
// returns:
//
//	bool: whether image has been stored in local Docker env.
func (currI *CurrentImage) parseConfigurationFromDockerEnv() bool {
	if conf, _, err := currI.dockerClient.ImageInspectWithRaw(context.TODO(), currI.name); err != nil {
		return false
	} else {
		currI.configuration = &conf
		return true
	}
}
