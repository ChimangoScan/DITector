package analyzer

import (
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"time"
)

var imageAnalyzer, imageAnalyzerE = NewImageAnalyzerGlobalConfig()

// AnalyzeImageByName analyzes image totally, including metadata, configuration, content of the image.
func AnalyzeImageByName(name string) (*myutils.ImageResult, error) {
	if imageAnalyzerE != nil {
		return nil, fmt.Errorf("create ImageAnalyzer failed with: %s", imageAnalyzerE)
	}

	return imageAnalyzer.AnalyzeImageByName(name)
}

// AnalyzeImagePartialByName analyzes image partially, currently only metadata.
func AnalyzeImagePartialByName(name string) (*myutils.ImageResult, error) {
	if imageAnalyzerE != nil {
		return nil, fmt.Errorf("create ImageAnalyzer failed with: %s", imageAnalyzerE)
	}

	var err error
	res := new(myutils.ImageResult)

	return res, err
}

// AnalyzeImageByName analyzes image totally by name, including analyzing metadata,
// configuration, content of the image.
//
// Image needs to be stored in the local Docker environment.
func (analyzer *ImageAnalyzer) AnalyzeImageByName(name string) (*myutils.ImageResult, error) {
	// 解析镜像信息
	ci, err := NewCurrentImage(name)
	if err != nil {
		myutils.Logger.Error("create CurrentImage for image", name, "failed with:", err.Error())
		return nil, err
	}
	beginTime := time.Now()
	beginTimeStr := myutils.GetLocalNowTime()
	if err = ci.ParseFromFile(); err != nil {
		myutils.Logger.Error("parse image", name, "failed with:", err.Error())
		return nil, err
	}

	// 创建扫描结果对象
	ir := CurrentImageToImageResult(ci)
	ir.LastAnalyzed = beginTimeStr

	// 分析镜像
	// 分析镜像元数据
	metaIs, err := analyzer.analyzeMetadata(ci)
	if err != nil {
		return nil, err
	}
	ir.MetadataAnalyzed = true
	ir.MetadataIssues = metaIs

	// 分析镜像配置信息
	configIs, err := analyzer.analyzeConfiguration(ci)
	if err != nil {
		return nil, err
	}
	ir.ConfigurationAnalyzed = true
	ir.ConfigurationIssues = configIs

	// 分析镜像内容信息
	contentIs, err := analyzer.analyzeContent(ci, ir)
	if err != nil {
		return nil, err
	}
	ir.ContentAnalyzed = true
	ir.ContentIssues = contentIs

	// 收尾赋值工作
	totalTime := time.Since(beginTime).String()
	ir.TotalTime = totalTime

	return ir, nil
}

func CurrentImageToImageResult(ci *CurrentImage) *myutils.ImageResult {
	ir := myutils.NewImageResult()

	ir.Name = ci.name
	ir.Registry = ci.registry
	ir.Namespace = ci.namespace
	ir.RepoName = ci.repoName
	ir.TagName = ci.tagName
	ir.Digest = ci.digest
	ir.Architecture = ci.architecture
	ir.Variant = ci.variant
	ir.OS = ci.os
	ir.OSVersion = ci.osVersion

	ir.Layers = ci.layerWithContentList
	for _, digest := range ci.layerWithContentList {
		ir.LayerResults[digest] = &myutils.LayerResult{
			Instruction: ci.layerInfoMap[digest].instruction,
			Size:        ci.layerInfoMap[digest].size,
			Digest:      digest,
		}
	}

	return ir
}
