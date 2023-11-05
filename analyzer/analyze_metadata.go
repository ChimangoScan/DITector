package analyzer

import "github.com/Musso12138/dockercrawler/myutils"

func (analyzer *ImageAnalyzer) analyzeMetadata(ci *CurrentImage) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)

	rmi, err := analyzer.analyzeRepoMetadata(ci)
	if err != nil {
		return nil, err
	}
	myutils.AddIssue(res, rmi...)

	return res, nil
}

func (analyzer *ImageAnalyzer) analyzeRepoMetadata(ci *CurrentImage) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)

	// 分析敏感参数
	// full_description中推荐的`docker run`
	for _, recCmd := range ci.recommendedCmd {
		is := analyzer.scanSensitiveParamInString(recCmd)
		for _, i := range is {
			i.Part = myutils.IssuePart.RepoMetadata
			i.Path = "full_description"
		}

		myutils.AddIssue(res, is...)
	}

	return res, nil
}

func (analyzer *ImageAnalyzer) analyzeImageMetadata(ci *CurrentImage) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)

	// 分析隐私泄露
	// 扫描layers.instruction
	for i, layer := range ci.metadata.imageMetadata.Layers {
		
	}

	return res, nil
}
