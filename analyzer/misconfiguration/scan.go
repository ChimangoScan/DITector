package misconfiguration

import "github.com/Musso12138/dockercrawler/myutils"

const (
	AppMongo   = "mongo"
	AppCouchDB = "couchdb"
)

func ScanFileMisconfiguration(filepath string, app string) ([]*myutils.Misconfiguration, bool, error) {
	res := make([]*myutils.Misconfiguration, 0)

	switch app {
	case AppMongo:
		return
	}

	return res, true, nil
}

// FileNeedScan 判断Linux文件系统下的文件是否需要检测
func FileNeedScan(filepath string) (bool, string) {

	if isMongoConfFile(filepath) {
		return true, AppMongo
	}

	return false, ""
}
