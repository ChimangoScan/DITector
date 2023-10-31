package myutils

// ImageResult is used to store analysis result
type ImageResult struct {
	Name string `json:"name"`

	Registry   string `json:"registry"`
	Namespace  string `json:"namespace"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`

	LastAnalyzedTime      string `json:"last_analyzed_time"`
	MetadataAnalyzed      bool   `json:"metadata_analyzed"`
	ConfigurationAnalyzed bool   `json:"configuration_analyzed"`
	ContentAnalyzed       bool   `json:"content_analyzed"`

	// Layers: [ layer-id1, layer-id2, ... ], from bottom to top
	Layers []string `json:"layers"`
	// LayerResults: layer-id ->
	LayerResults map[string]LayerResult `json:"layer_results"`

	Results []*Issue `json:"results"`
}

type LayerResult struct {
	Instruction         string              `json:"instruction"`
	Size                int64               `json:"size"`
	Digest              string              `json:"digest"`
	AnalyzedFiles       []string            `json:"analyzed_files"`
	AnalyzedFileResults map[string][]*Issue `json:"analyzed_file_results"`
}

func NewImageResult() *ImageResult {
	ir := new(ImageResult)

	ir.LayerResults = make(map[string]LayerResult)

	return ir
}

// Issue 表示一条发现的问题
// TODO: 需要考虑怎么统一所有检测的结果
type Issue struct {
	Type          string  `json:"type"`
	Part          string  `json:"part"` // part of image: metadata, configuration, content
	Path          string  `json:"path"`
	Rule          any     `json:"rule"`
	Match         string  `json:"match"`
	Severity      string  `json:"severity"`
	SeverityScore float64 `json:"severity_score"`
	LayerDigest   string  `json:"layer_digest"`
}

var IssueType = struct {
	SecretLeakage     string
	SensitiveParam    string
	Vulnerability     string
	Misconfiguration  string
	MaliciousSoftware string
}{
	"secret-leakage",
	"sensitive-parameter",
	"vulnerability",
	"misconfiguration",
	"malicious-software",
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
