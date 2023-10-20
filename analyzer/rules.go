package analyzer

import (
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
)

type ImageAnalyzerRules struct {
	SecretRules         []*SecretRule         `yaml:"secrets"`
	SensitiveParamRules []*SensitiveParamRule `yaml:"sensitive_params"`
}

type SecretRule struct {
	Name          string         `yaml:"name" json:"name"`
	Regex         string         `yaml:"regex" json:"regex"`
	RegexType     string         `yaml:"regex_type"`
	CompiledRegex *regexp.Regexp `yaml:"-" json:"-"`
	Severity      string         `yaml:"severity"`
	SeverityScore float64        `yaml:"severity_score"`
}

type SensitiveParamRule struct {
}

func newImageAnalyzerRules() *ImageAnalyzerRules {
	rules := new(ImageAnalyzerRules)
	rules.SecretRules = make([]*SecretRule, 0)
	rules.SensitiveParamRules = make([]*SensitiveParamRule, 0)
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
