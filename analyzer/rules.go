package analyzer

import (
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
)

type ImageAnalyzerRules struct {
	SecretRules         []*SecretRule         `yaml:"secrets"`
	SensitiveParamRules []*SensitiveParamRule `yaml:"sensitive_params"`
	MisConfigRules      []*MisConfigRule      `yaml:"mis_config"`
}

type SecretRule struct {
	Name          string         `yaml:"name" json:"name"`
	Description   string         `yaml:"description" json:"description"`
	Regex         string         `yaml:"regex" json:"regex"`
	RegexType     string         `yaml:"regex_type"`
	CompiledRegex *regexp.Regexp `yaml:"-" json:"-"`
	Severity      string         `yaml:"severity"`
	SeverityScore float64        `yaml:"severity_score"`
}

type SensitiveParamRule struct {
	Name          string         `yaml:"name" json:"name"`
	Description   string         `yaml:"description"`
	Regex         string         `yaml:"regex" json:"regex"`
	RegexType     string         `yaml:"regex_type"`
	CompiledRegex *regexp.Regexp `yaml:"-" json:"-"`
	Severity      string         `yaml:"severity"`
	SeverityScore float64        `yaml:"severity_score"`
}

type MisConfigRule struct {
	Name               string           `yaml:"name" json:"name"`
	Description        string           `yaml:"description"`
	FileRegex          string           `yaml:"file_regex"` // 文件路径特征
	CompiledFileRegex  *regexp.Regexp   `yaml:"-"`
	CheckRegex         []string         `yaml:"check_regex"` // 检查是否为有效配置文件
	CompliedCheckRegex []*regexp.Regexp `yaml:"-"`
	Necessary          bool             `yaml:"necessary"` // true/false -> 包含/不包含Regex时为误配置
	Regex              string           `yaml:"regex" json:"regex"`
	RegexType          string           `yaml:"regex_type"`
	CompiledRegex      *regexp.Regexp   `yaml:"-" json:"-"`
	Severity           string           `yaml:"severity"`
	SeverityScore      float64          `yaml:"severity_score"`
}

func newImageAnalyzerRules() *ImageAnalyzerRules {
	rules := new(ImageAnalyzerRules)
	rules.SecretRules = make([]*SecretRule, 0)
	rules.SensitiveParamRules = make([]*SensitiveParamRule, 0)
	rules.MisConfigRules = make([]*MisConfigRule, 0)
	return rules
}

func (rs *ImageAnalyzerRules) loadSecretsFromYAMLFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(content, &rs.SecretRules); err != nil {
		return err
	}

	return nil
}

func (rs *ImageAnalyzerRules) compileSecretsRegex() {
	for _, secret := range rs.SecretRules {
		secret.CompiledRegex, _ = regexp.Compile(secret.Regex)
	}
}

func (rs *ImageAnalyzerRules) loadSensitiveParamsFromYAMLFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(content, &rs.SensitiveParamRules); err != nil {
		return err
	}

	return nil
}

func (rs *ImageAnalyzerRules) compileSensitiveParamRegex() {
	for _, sensitive := range rs.SensitiveParamRules {
		sensitive.CompiledRegex, _ = regexp.Compile(sensitive.Regex)
	}
}

func (rs *ImageAnalyzerRules) loadMisConfigFromYAMLFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(content, &rs.MisConfigRules); err != nil {
		return err
	}

	return nil
}

func (rs *ImageAnalyzerRules) compileMisConfigRegex() {
	for _, misconf := range rs.MisConfigRules {
		misconf.CompiledFileRegex, _ = regexp.Compile(misconf.FileRegex)
		for _, check := range misconf.CheckRegex {
			checkR, _ := regexp.Compile(check)
			misconf.CompliedCheckRegex = append(misconf.CompliedCheckRegex, checkR)
		}
		misconf.CompiledRegex, _ = regexp.Compile(misconf.Regex)
	}
}
