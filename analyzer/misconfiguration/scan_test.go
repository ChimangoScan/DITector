package misconfiguration

import (
	"fmt"
	"testing"
)

func TestFileNeedScan(t *testing.T) {
	fmt.Println(FileNeedScan("/etc/mongo/mongo.confa"))
}
