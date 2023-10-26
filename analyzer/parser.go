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
	architecture   string
	variant        string
	os             string
	osVersion      string
	digest         string

	localFlag    bool
	downloadFlag bool

	// metadata of the repository, the tag and the image
	metadata *metadata
	// configuration of the image
	configuration *types.ImageInspect
	// content of the image
	layerWithContentList []string
	layerInfoMap         map[string]layerInfo

	Results *myutils.ImageResult
}

type metadata struct {
	repositoryMetadata *myutils.Repository
	tagMetadata        *myutils.Tag
	imageMetadata      *myutils.Image
}

type layerInfo struct {
	size        int64
	instruction string
	digest      string
	// localFilePath of the layer
	localFilePath string
}

// Parse TODO: 解析指定镜像的元数据、配置信息，下载镜像，定位镜像的各个层
func (currI *CurrentImage) Parse() error {
	// 解析镜像基本信息
	currI.parseName()

	// 获取当前平台
	if err := currI.getServerPlatform(); err != nil {
		myutils.Logger.Error("get Docker server platform failed with:", err.Error())
	}

	// 获取元数据
	if err := currI.parseMetadata(); err != nil {
		myutils.Logger.Error("parse metadata of image", currI.name, "failed with:", err.Error())
		return err
	}

	// 解析配置信息
	// 检查image是否位于本地Docker环境中，如果不存在则下载镜像
	if err := currI.parseConfigurationFromDockerEnv(); err != nil {
		myutils.Logger.Error("inspect image", currI.name, "failed with:", err.Error())

		// 将镜像下载到本地
		// TODO: 目前是异步的，下面可能还是读不了
		rc, err := currI.dockerClient.ImagePull(context.TODO(), currI.name, types.ImagePullOptions{})
		myutils.Logger.Debug("pulling image", currI.name)
		if err != nil {
			myutils.Logger.Error("pull image", currI.name, "failed with:", err.Error())
		} else {
			// 下载成功
			defer rc.Close()
			currI.downloadFlag = true
		}
	} else {
		currI.localFlag = true
	}

	// 下载后尝试解析镜像信息
	if currI.downloadFlag {
		if err := currI.parseConfigurationFromDockerEnv(); err != nil {
			myutils.Logger.Error("inspect image", currI.name, "failed with:", err.Error())
		} else {
			currI.localFlag = true
		}
	}

	//

	return nil
}

// ParsePartial TODO: 解析指定镜像的元数据
func (currI *CurrentImage) ParsePartial() {

}

// parseName parses registry, namespace, repository, tag of the image according to name.
func (currI *CurrentImage) parseName() {
	currI.registry, currI.namespace, currI.repositoryName, currI.tagName = myutils.DivideImageName(currI.name)
}

// getServerPlatform gets platform of the host with Docker client.
func (currI *CurrentImage) getServerPlatform() error {
	if plf, err := currI.dockerClient.ServerVersion(context.TODO()); err != nil {
		return err
	} else {
		currI.architecture, currI.os = plf.Arch, plf.Os
	}

	return nil
}

// parseMetadata loads metadata of repository
func (currI *CurrentImage) parseMetadata() error {
	var err error
	currI.metadata = new(metadata)

	if currI.metadata.repositoryMetadata, err = currI.getRepositoryMetadata(); err != nil {
		return err
	}

	if err := currI.getTagMetadata(); err != nil {
		return err
	}

	if err := currI.getImageMetadata(); err != nil {
		return err
	}

	return nil
}

// getRepositoryMetadata gets repository metadata from local MongoDB,
// if repository not maintained in MongoDB or disconnected from MongoDB,
// try to get metadata from Docker Hub API and store metadata to MongoDB.
func (currI *CurrentImage) getRepositoryMetadata() (*myutils.Repository, error) {
	// 数据库在线
	if myutils.GlobalDBClient.MongoFlag {
		if
	}
	return nil
}

// getTagMetadata gets tag metadata from local MongoDB, if tag not maintained
// in MongoDB or disconnected from MongoDB, try to get metadata from Docker
// Hub API and store metadata to MongoDB.
func (currI *CurrentImage) getTagMetadata() (*myutils.Tag, error) {
	return nil
}

// getImageMetadata gets image metadata from local MongoDB, if image not
// maintained in MongoDB or disconnected from MongoDB, try to get
// metadata from Docker Hub API and store metadata to MongoDB.
func (currI *CurrentImage) getImageMetadata() (*myutils.Image, error) {
	return nil
}

// parseConfigurationFromDockerEnv tries to inspect image from local env, with results
// stored to currI.Configuration, formatted like `docker image inspect`.
//
// returns:
//
//	bool: whether image has been stored in local Docker env.
func (currI *CurrentImage) parseConfigurationFromDockerEnv() error {
	// 从本地inspect读取镜像配置信息
	if conf, _, err := currI.dockerClient.ImageInspectWithRaw(context.TODO(), currI.name); err != nil {
		return err
	} else {
		currI.configuration = &conf
	}

	// TODO: 解析镜像的配置信息

	return nil
}
