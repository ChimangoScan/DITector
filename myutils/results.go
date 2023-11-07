package myutils

// ImageResult is used to store analysis result
type ImageResult struct {
	Name         string `json:"name"`
	Registry     string `json:"registry"`
	Namespace    string `json:"namespace"`
	RepoName     string `json:"repository_name"`
	TagName      string `json:"tag_name"`
	Digest       string `json:"digest"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`

	LastAnalyzed string `json:"last_analyzed"`
	TotalTime    string `json:"total_time"`
	AnalyzeTime  string `json:"analyze_time"`

	MetadataAnalyzed bool     `json:"metadata_analyzed"`
	MetadataIssues   []*Issue `json:"metadata_issues"`

	ConfigurationAnalyzed bool     `json:"configuration_analyzed"`
	ConfigurationIssues   []*Issue `json:"configuration_issues"`

	ContentAnalyzed bool `json:"content_analyzed"`
	// Layers: [ layer-id1, layer-id2, ... ], from bottom to top
	Layers []string `json:"layers"`
	// LayerResults: layer-id -> LayerResult
	LayerResults map[string]*LayerResult `json:"layer_results"`
	// FileIssues: filepath -> []*Issue, issues in the file system after mounting by UnionFS
	FileIssues    map[string][]*Issue `json:"-"`
	ContentIssues []*Issue            `json:"content_issues"`
}

type LayerResult struct {
	Instruction   string   `json:"instruction"`
	Size          int64    `json:"size"`
	Digest        string   `json:"digest"`
	AnalyzedFiles []string `json:"analyzed_files"`
	// FileIssues: filepath -> []*Issue
	FileIssues map[string][]*Issue `json:"file_issues"`
}

func NewImageResult() *ImageResult {
	ir := new(ImageResult)

	ir.MetadataIssues = make([]*Issue, 0)
	ir.ConfigurationIssues = make([]*Issue, 0)
	ir.Layers = make([]string, 0)
	ir.LayerResults = make(map[string]*LayerResult)
	ir.FileIssues = make(map[string][]*Issue)
	ir.ContentIssues = make([]*Issue, 0)

	return ir
}

// Issue 表示一条发现的问题
// TODO: 需要考虑怎么统一所有检测的结果
type Issue struct {
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Part          string  `json:"part"` // part of image: metadata, configuration, content
	Path          string  `json:"path"`
	Sha256        string  `json:"sha256,omitempty"`  // sha256 of file, only for malicious file
	Version       string  `json:"version,omitempty"` // version of the product, only for vulnerability
	Match         string  `json:"match,omitempty"`
	Description   string  `json:"description"`
	Severity      string  `json:"severity"`
	SeverityScore float64 `json:"severity_score"`
	LayerDigest   string  `json:"layer_digest,omitempty"`
}

var IssueType = struct {
	SecretLeakage    string
	SensitiveParam   string
	Vulnerability    string
	Misconfiguration string
	MaliciousFile    string
}{
	"secret-leakage",
	"sensitive-parameter",
	"vulnerability",
	"misconfiguration",
	"malicious-file",
}

var IssuePart = struct {
	RepoMetadata  string
	TagMetadata   string
	ImageMetadata string
	Configuration string
	Content       string
}{
	"repository-metadata",
	"tag-metadata",
	"image-metadata",
	"configuration",
	"content",
}

func AddIssue(dest []*Issue, src ...*Issue) {
	for _, i := range src {
		// 隐私泄露不存在覆盖
		if i.Type != IssueType.SecretLeakage {
			for _, j := range dest {
				if *i == *j {
					break
				}
			}
		}
		dest = append(dest, i)
	}
}
