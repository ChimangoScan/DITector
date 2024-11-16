package myutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

type MyES struct {
	Client    *elasticsearch.Client
	IndexName string
}

type MyESImgResult struct {
	Total       int
	UniqueField []ImgResultUniqueField
}

func NewESGlobalConfig() (*MyES, error) {
	return NewES(GlobalConfig.ESConfig.ESURI, GlobalConfig.ESConfig.ESUsername,
		GlobalConfig.ESConfig.ESPassword, GlobalConfig.ESConfig.ESIndexName)
}

func NewES(esURI, esUser, esPwd, indexName string) (*MyES, error) {
	myES := new(MyES)
	var err error

	cfg := elasticsearch.Config{
		Addresses: []string{
			esURI,
		},
		Username: esUser,
		Password: esPwd,
	}
	myES.Client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// 以下ping检查方法来自官方文档
	// https://pkg.go.dev/github.com/elastic/go-elasticsearch/v8@v8.0.0#section-readme
	res, err := myES.Client.Info()
	if err != nil {
		Logger.Error("connect to es failed, get err:", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		Logger.Error("connect to es failed, get err:", res.String())
		return nil, fmt.Errorf("es get info err: %s", res.String())
	}

	myES.IndexName = indexName

	return myES, err
}

func (es *MyES) FindImgResultByTextPaged(search string, page, pageSize int64) (*MyESImgResult, error) {
	ret := new(MyESImgResult)

	query, err := genQueryPagedBody(search, page, pageSize)
	if err != nil {
		return nil, err
	}

	res, err := es.Client.Search(
		es.Client.Search.WithContext(context.Background()),
		es.Client.Search.WithIndex(es.IndexName),
		es.Client.Search.WithBody(query),
		es.Client.Search.WithTrackTotalHits(true),
		es.Client.Search.WithPretty(),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
			Logger.Error(fmt.Sprintf("json decode res of es search for keyword %s, failed with err: %s", search, err))
			return nil, err
		} else {
			Logger.Error(fmt.Sprintf("es search for keyword %s page %d pageSize %d, failed got [%s] %s: %s",
				search, page, pageSize,
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"]))

			return nil, fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"])
		}
	}

	var r map[string]interface{}
	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		Logger.Error(fmt.Sprintf("json decode res of es search for keyword %s, failed with err: %s", search, err))
		return nil, err
	}

	ret.Total = int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		hitSrc := hit.(map[string]interface{})["_source"].(map[string]interface{})
		uniqueField := ImgResultUniqueField{
			Registry:  hitSrc["registry"].(string),
			Namespace: hitSrc["namespace"].(string),
			RepoName:  hitSrc["repository_name"].(string),
			TagName:   hitSrc["tag_name"].(string),
			Digest:    hitSrc["digest"].(string),
		}

		ret.UniqueField = append(ret.UniqueField, uniqueField)
	}

	return ret, nil
}

func genQueryPagedBody(keyword string, page, pageSize int64) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	query := getQueryMap(keyword)
	query["from"] = (page - 1) * pageSize
	query["size"] = pageSize
	query["_source"] = []string{"registry", "namespace", "repository_name", "tag_name", "digest"}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	return &buf, nil
}

func getQueryMap(search string) map[string]interface{} {
	// TODO 这里如果后面有更精细的需求可以转用es的nested类型，主要是为文档内的列表建立顺序对应关系
	// 原因参考：https://blog.csdn.net/laoyang360/article/details/82950393
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"match": map[string]string{
							"metadata_result.installed_contents.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"metadata_result.installed_contents.source": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"metadata_result.secret_leakages.match": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"metadata_result.secret_leakages.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"metadata_result.sensitive_params.match": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"configuration_result.secret_leakages.match": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"configuration_result.secret_leakages.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.components.file_md5": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.components.file_sha1": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.components.filename": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.malicious_files.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.malicious_files.sha256": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.misconfigurations.app_name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.secret_leakages.match": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.secret_leakages.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.vulnerabilities.name": search,
						},
					},
					map[string]interface{}{
						"match": map[string]string{
							"content_result.vulnerabilities.product_name": search,
						},
					},
				},
			},
		},
	}
}
