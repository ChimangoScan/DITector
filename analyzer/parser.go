package analyzer

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"os"
	"path"
	"strings"
)

type CurrentImage struct {
	dockerClient *client.Client
	tarFilepath  string
	filepath     string
	name         string

	registry     string
	namespace    string
	repoName     string
	tagName      string
	architecture string
	variant      string
	os           string
	osVersion    string
	digest       string

	// metadata of the repository, the tag and the image
	metadata *metadata
	// configuration of the image
	configuration *types.ImageInspect
	// content of the image
	layerWithContentList []string
	layerInfoMap         map[string]layerInfo
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

func NewCurrentImage(imgName string) (*CurrentImage, error) {
	currI := new(CurrentImage)
	var err error

	currI.dockerClient, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	currI.name = imgName
	currI.parseName()

	currI.metadata = new(metadata)
	currI.layerWithContentList = make([]string, 0)
	currI.layerInfoMap = make(map[string]layerInfo)

	return currI, nil
}

type ImagePullEvent struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
}

type downloadFinish struct {
	tarFilepath string // filepath for the tar archive
	filepath    string // filepath for the extracted result dir
	err         error
}

// pullSaveExtractImage pulls Docker image to local Docker env, saves it
// to a tar archive, and extracts all tar archive(including image and each layer).
func (currI *CurrentImage) pullSaveExtractImage(targetDir string, finish chan downloadFinish) {
	var tarFilepath string
	var filepath string
	var err error

	defer func() {
		finish <- downloadFinish{tarFilepath: tarFilepath, filepath: filepath, err: err}
	}()

	myutils.Logger.Debug("start pulling image", currI.name)

	// 同步下载镜像
	if err = currI.pullImage(); err != nil {
		myutils.Logger.Error("pull image", currI.name, "failed with:", err.Error())
		return
	}

	// 保存镜像
	targetTarFilename := fmt.Sprintf("%s-%s-%s.tar", currI.namespace, currI.repoName, currI.tagName)
	tarFilepath = path.Join(targetDir, targetTarFilename)
	if err = currI.saveImage(tarFilepath); err != nil {
		myutils.Logger.Error("save image", currI.name, "to filepath", tarFilepath, "failed with:", err.Error())
		return
	}

	// 解压镜像
	targetDirname := fmt.Sprintf("%s-%s-%s", currI.namespace, currI.repoName, currI.tagName)
	filepath = path.Join(targetDir, targetDirname)
	if err = extractImage(tarFilepath, filepath); err != nil {
		myutils.Logger.Error("extract image", currI.name, "from file", tarFilepath, "failed with:", err.Error())
		return
	}

	return
}

// pullImage calls client.Client.ImagePull to download image.
// It turns ImagePull progress from async to sync with a non-buffered chan.
func (currI *CurrentImage) pullImage() error {
	rc, err := currI.dockerClient.ImagePull(context.TODO(), currI.name, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()

	success := false

	decoder := json.NewDecoder(rc)
	for {
		event := new(ImagePullEvent)
		if err = decoder.Decode(event); err != nil {
			if err == io.EOF {
				break
			}
			myutils.Logger.Error("decode JSON when pulling image", currI.name, "failed with:", err.Error())
		}

		if strings.Contains(event.Status, "Downloaded newer image for") ||
			strings.Contains(event.Status, "Image is up to date") {
			success = true
		}
	}

	if success {
		return nil
	} else {
		return fmt.Errorf("not catch download success signal in ImagePull events")
	}
}

// saveImage calls client.Client.ImageSave to save image to tar archive.
func (currI *CurrentImage) saveImage(filepath string) error {
	imageRC, err := currI.dockerClient.ImageSave(context.TODO(), []string{currI.name})
	if err != nil {
		return err
	}
	defer imageRC.Close()

	tarFile, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	_, err = io.Copy(tarFile, imageRC)
	if err != nil {
		return err
	}

	return nil
}

// extractImage extracts source image tar archive to dest dir,
// including image tar and all layer tar.
func extractImage(imgTar, dstDir string) error {
	// 解压image tar
	if err := extractTar(imgTar, dstDir); err != nil {
		return err
	}

	// 逐个解压layer tar

	return nil
}

// extractTar extracts tar file to specific dst dir,
// creating recursively when dir not exists.
func extractTar(src, dst string) error {
	// 打开tar文件
	tarFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	// 创建目标文件夹
	if err = os.MkdirAll(dst, 0750); err != nil {
		return err
	}

	// 创建Tar读取器
	tr := tar.NewReader(tarFile)

	// 逐个解压文件
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // 所有文件已解压
		}
		if err != nil {
			return err
		}

		// 创建目标文件
		targetFile := path.Join(dst, header.Name)
		info := header.FileInfo()

		// 如果是文件夹，创建目录
		if info.IsDir() {
			if err = os.MkdirAll(targetFile, info.Mode()); err != nil {
				return err
			}
			continue
		}

		// 如果是文件，创建文件并写入数据
		file, err := os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, tr)
		if err != nil {
			return err
		}
	}

	return nil
}
