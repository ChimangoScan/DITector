package buildgraph

func Build(format string) {
	config(format)

	switch format {
	case "json":
		BuildFromJSON()
	case "mysql":
		BuildFromMysql()
	}
}

// BuildFromJSON 根据crawler爬到的json内容建立信息库
func BuildFromJSON() {
	StartFromJSON()
}

// BuildFromMysql 根据crawler爬到的mysql内容建立信息库
func BuildFromMysql() {

}
