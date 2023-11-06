package analyzer

import (
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"strings"
)

func (analyzer *ImageAnalyzer) analyzeConfiguration(ci *CurrentImage) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)

	repoMetaIs, err := analyzer.analyzeRepoMetadata(ci)
	if err != nil {
		return nil, err
	}
	myutils.AddIssue(res, repoMetaIs...)

	imgMetaIs, err := analyzer.analyzeImageMetadata(ci)
	if err != nil {
		return nil, err
	}
	myutils.AddIssue(res, imgMetaIs...)

	return res, nil
}

func (analyzer *ImageAnalyzer) analyzeEnvConfig(ci *CurrentImage) []*myutils.Issue {
	res := make([]*myutils.Issue, 0)

	// 分析隐私泄露
	// 扫描镜像环境变量
	for _, env := range ci.configuration.Config.Env {
		is := analyzer.scanSensitiveParamInString(env)
		for _, i := range is {
			i.Part = myutils.IssuePart.Configuration
			i.Path = fmt.Sprintf("Env[%s]", strings.Split(env, "=")[0])
		}

		myutils.AddIssue(res, is...)
	}

	return res
}
