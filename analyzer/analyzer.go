package analyzer

import (
	"github.com/Musso12138/dockercrawler/myutils"
	"strconv"
)

type ImageAnalyzer struct {
	rules *ImageAnalyzerRules
}

// NewImageAnalyzerGlobalConfig creates a new ImageAnalyzer configured based on config.json
func NewImageAnalyzerGlobalConfig() (*ImageAnalyzer, error) {
	return NewImageAnalyzer(myutils.GlobalConfig.RulesConfig.SecretRulesFile,
		myutils.GlobalConfig.RulesConfig.SensitiveParamRulesFile)
}

// NewImageAnalyzer returns a configured ImageAnalyzer
//
// Parameters:
//
//	secretFile: file path containing rules for matching secrets
//	sensParamFile: file path containing rules for matching sensitive parameters
func NewImageAnalyzer(secretFile, sensParamFile string) (*ImageAnalyzer, error) {
	analyzer := new(ImageAnalyzer)
	var err error

	// 初始化成员变量
	analyzer.rules = newImageAnalyzerRules()

	// 配置隐私泄露规则
	if err = analyzer.rules.loadSecretsFromYAMLFile(secretFile); err != nil {
		return nil, err
	}
	analyzer.rules.compileSecretsRegex()

	// 配置敏感参数规则
	if err = analyzer.rules.loadSensitiveParamsFromYAMLFile(secretFile); err != nil {
		return nil, err
	}

	return analyzer, nil
}

// AnalyzerImagePartialByName analyzes image partially by name, including only the metadata.
//
// This will never pull the layers of the image to local env.
func (analyzer *ImageAnalyzer) AnalyzerImagePartialByName(name string) {

}

// AnalyzeMetadata analyzes metadata of repository, tag and image.
func (analyzer *ImageAnalyzer) AnalyzeMetadata() {

}

// AnalyzeImageMetadata analyze instruction of layers to
func (analyzer *ImageAnalyzer) AnalyzeImageMetadata(image *myutils.Image) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)

	for index, layer := range image.Layers {
		digest := ""
		if layer.Size != 0 {
			digest = layer.Digest
		}
		results := analyzer.scanSecretsInString(layer.Instruction)

		for _, result := range results {
			result.Type = "in-dockerfile-command"
			result.Path = "layer[" + strconv.Itoa(index) + "].instruction"
			result.LayerDigest = digest
		}
		res = append(res, results...)
	}

	return res, nil
}

func (analyzer *ImageAnalyzer) scanSecretsInString(s string) []*myutils.Issue {
	res := make([]*myutils.Issue, 0)

	for _, secret := range analyzer.rules.SecretRules {
		if secret.CompiledRegex == nil {
			continue
		}
		matches := secret.CompiledRegex.FindAllString(s, -1)
		for _, match := range matches {
			tmp := &myutils.Issue{
				Type:          myutils.IssueType.SecretLeakage,
				RuleName:      secret.Name,
				Match:         match,
				Description:   secret.Description,
				Severity:      secret.Severity,
				SeverityScore: secret.SeverityScore,
			}
			res = append(res, tmp)
		}
	}

	return res
}

func (analyzer *ImageAnalyzer) scanSecretsInBytes(b []byte) []*myutils.Issue {
	res := make([]*myutils.Issue, 0)

	for _, secret := range analyzer.rules.SecretRules {
		if secret.CompiledRegex == nil {
			continue
		}
		matches := secret.CompiledRegex.FindAll(b, -1)
		for _, match := range matches {
			tmp := &myutils.Issue{
				Type:          myutils.IssueType.SecretLeakage,
				RuleName:      secret.Name,
				Match:         string(match),
				Description:   secret.Description,
				Severity:      secret.Severity,
				SeverityScore: secret.SeverityScore,
			}
			res = append(res, tmp)
		}
	}

	return res
}

func (analyzer *ImageAnalyzer) scanSensitiveParamInString(s string) []*myutils.Issue {
	res := make([]*myutils.Issue, 0)

	for _, sensitive := range analyzer.rules.SensitiveParamRules {
		matches := sensitive.CompiledRegex.FindAllString(s, -1)
		for _, match := range matches {
			tmp := &myutils.Issue{
				Type:          myutils.IssueType.SensitiveParam,
				RuleName:      sensitive.Name,
				Match:         match,
				Description:   sensitive.Description,
				Severity:      sensitive.Severity,
				SeverityScore: sensitive.SeverityScore,
			}
			res = append(res, tmp)
		}
	}

	return res
}
