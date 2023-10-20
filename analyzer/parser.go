package analyzer

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"myutils"
)

type CurrentImage struct {
	DockerClient *client.Client `json:"-"`
	Filepath     string         `json:"filepath,omitempty"`

	Name           string `json:"name"`
	Registry       string `json:"registry"`
	Namespace      string `json:"namespace"`
	RepositoryName string `json:"repository_name"`
	TagName        string `json:"tag_name"`
	Digest         string `json:"digest"`

	Metadata      *ImageMetadata      `json:"metadata"`
	Configuration *types.ImageInspect `json:"configuration"`

	LayerLocalFileMap map[string]string `json:"layer_local_file_map"`
}

type ImageMetadata struct {
	RepositoryMetadata *myutils.Repository `json:"repository_metadata"`
	TagMetadata        *myutils.Tag        `json:"tag_metadata"`
	ImageMetadata      *myutils.Image      `json:"image_metadata"`
}

// Parse TODO: 解析指定镜像的元数据、配置信息，下载镜像，定位镜像的各个层
func (curr *CurrentImage) Parse() {

}

func (curr *CurrentImage) ParsePartial() {

}

func (curr *CurrentImage) parseConfigurationFromDockerEnv() {

}
