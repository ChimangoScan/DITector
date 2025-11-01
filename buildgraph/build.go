package buildgraph

func Build(format string, page int64, pageSize int64, tagCnt int, pullCountThreshold int64) {
	config(format)

	switch format {
	case "mongo":
		BuildFromMongo(page, pageSize, tagCnt, pullCountThreshold)
	}
}

// BuildFromMongo 根据crawler爬到的mysql内容建立信息库
func BuildFromMongo(page int64, pageSize int64, tagCnt int, pullCountThreshold int64) {
	StartFromMongo(page, pageSize, tagCnt, pullCountThreshold)
}
