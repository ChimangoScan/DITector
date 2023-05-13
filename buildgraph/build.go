package buildgraph

import "runtime"

var (
	// chanLimitMainGoroutine 限制goroutine数量
	chanLimitMainGoroutine chan struct{}
)

var (
	chanRepository     = make(chan *Repository, runtime.NumCPU())
	chanTag            = make(chan *Tag, runtime.NumCPU())
	chanImage          = make(chan *Image, runtime.NumCPU())
	chanDoneRepository = make(chan struct{})
	chanDoneTag        = make(chan struct{})
	chanDoneImage      = make(chan struct{})
)

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
