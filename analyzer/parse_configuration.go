package analyzer

import (
	"context"
	"os"
)

// parseConfigurationFromFile TODO: loads image config from file <digest>.json (CurrentImage.manifest.Config).
func (currI *CurrentImage) parseConfigurationFromFile() error {
	manifestFile, err := os.ReadFile(currI.manifest.Config)
	if err != nil {
		return err
	}

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
