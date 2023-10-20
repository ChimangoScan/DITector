package analyzer

import (
	"fmt"
	"myutils"
)

var imageAnalyzer, imageAnalyzerE = NewImageAnalyzerGlobalConfig()

// AnalyzeImage analyzes the image totally, including metadata, configuration, content of the image
func AnalyzeImage(name string) (*myutils.ImageResult, error) {
	if imageAnalyzerE != nil {
		return nil, fmt.Errorf("create ImageAnalyzer failed with: %s", imageAnalyzerE)
	}

	var err error
	res := new(myutils.ImageResult)

	return res, err
}

// AnalyzeImagePartial analyzes partial information of the image, currently only metadata
func AnalyzeImagePartial(name string) (*myutils.ImageResult, error) {
	if imageAnalyzerE != nil {
		return nil, fmt.Errorf("create ImageAnalyzer failed with: %s", imageAnalyzerE)
	}

	var err error
	res := new(myutils.ImageResult)

	return res, err
}
