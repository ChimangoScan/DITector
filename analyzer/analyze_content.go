package analyzer

import (
	"encoding/json"
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"io"
	"net/http"
	"sync"
)

type AskYReport struct {
	Data AskYData `json:"data"`
}

type AskYData struct {
	ReportData AskYReportData `json:"reportData"`
}

type AskYReportData struct {
	Total        int            `json:"total"`
	ComponentNum int            `json:"componentNum"`
	VulnInfo     []AskYVulnInfo `json:"vulnInfo"`
}

type AskYVulnInfo struct {
	CVEID       string  `json:"cveID"`
	FileName    string  `json:"fileName"`
	ProductName string  `json:"productName"`
	VendorName  string  `json:"vendorName"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	CVSSScore   float64 `json:"cvssScore"`
	FilePath    string  `json:"filePath"`
}

type FileReputation struct {
	Sha256          string  `json:"sha256"`
	Level           int     `json:"level"`
	MalwareName     string  `json:"malware_name"`
	MalwareTypeName string  `json:"malware_type_name"`
	FileDesc        string  `json:"file_desc"`
	Describe        string  `json:"describe"`
	MaliciousFamily string  `json:"malicious_family"`
	SandboxScore    float64 `json:"sandbox_score"`
}

func (analyzer *ImageAnalyzer) analyzeContent(ci *CurrentImage, ir *myutils.ImageResult) ([]*myutils.Issue, error) {
	res := make([]*myutils.Issue, 0)
	wg := sync.WaitGroup{}

	// 逐层分析layer内容，写入对应LayerResult
	for _, ld := range ir.Layers {
		wg.Add(1)
		go func(digest, layerDir string) {
			defer wg.Done()
			layerRes, fromMongo, err := analyzer.analyzeLayer(digest, layerDir)
			if err != nil {
				myutils.Logger.Error("analyze layer", digest, "failed with:", err.Error())
				return
			}
			ir.LayerResults[digest] = layerRes

			if !fromMongo {
				if myutils.GlobalDBClient.MongoFlag {
					go func(layerRes *myutils.LayerResult) {
						if e := myutils.GlobalDBClient.Mongo.UpdateLayerResult(layerRes); e != nil {
							myutils.Logger.Error("update LayerResult", layerRes.Digest, "failed with:", e.Error())
						}
					}(layerRes)
				}
			}
		}(ld, ci.layerInfoMap[ld].localFilePath)
	}

	// 等待各层分析结束
	wg.Wait()

	// 遍历各层结果，存入全局表中（当前状态）
	for _, ld := range ir.Layers {
		for filepath, fileIs := range ir.LayerResults[ld].FileIssues {
			// 如果下层中扫过filepath，将其中的隐私信息泄露问题加进来
			tmpIs := make([]*myutils.Issue, len(fileIs))
			copy(tmpIs, fileIs)
			if preIs, ok := ir.FileIssues[filepath]; ok {
				for _, preI := range preIs {
					if preI.Type == myutils.IssueType.SecretLeakage {
						myutils.AddIssue(tmpIs, preI)
					}
				}
			}

			ir.FileIssues[filepath] = tmpIs
		}
	}

	// 汇总各层file，形成最终结果
	for _, fileIs := range ir.FileIssues {
		myutils.AddIssue(res, fileIs...)
	}

	return res, nil
}

// analyzeLayer TODO: traverses and analyzes files under inputted layerDir,
// and writes results directly to layerResult.
func (analyzer *ImageAnalyzer) analyzeLayer(digest, layerDir string) (*myutils.LayerResult, bool, error) {
	// 数据库在线，检查是否已被分析
	if myutils.GlobalDBClient.MongoFlag {
		if lr, err := myutils.GlobalDBClient.Mongo.FindLayerResultByDigest(digest); err == nil {
			return lr, true, nil
		}
	}

	res := new(myutils.LayerResult)
	wg := sync.WaitGroup{}

	// SCA: 调用asky对本地层文件做
	return nil
}

// scaVul TODO: 对层文件进行SCA并进行漏洞匹配
func scaVul(layerDir string) {

}

// scanFileMalicious 利用奇安信云查接口检查文件是否恶意
func scanFileMalicious(filepath string) (*myutils.Issue, bool, error) {
	reputation, err := getFileReputation(filepath)
	if err != nil {
		return nil, false, err
	}

	if reputation.MalwareName == "" {
		return nil, false, nil
	}

	i := new(myutils.Issue)
	i.Type = myutils.IssueType.MaliciousFile
	i.Part = myutils.IssuePart.Content
	i.Sha256 = reputation.Sha256
	i.Description = reputation.Describe
	i.SeverityScore = reputation.SandboxScore

	return i, true, nil
}

func getFileReputation(filepath string) (*FileReputation, error) {
	h, err := myutils.Sha256File(filepath)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(fmt.Sprintf("https://tqs.qianxin-inc.cn/file/v1/files/%s/reputation", h))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := new(FileReputation)
	err = json.Unmarshal(body, res)

	return res, err
}
