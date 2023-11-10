package analyzer

import (
	"fmt"
	"testing"
)

func TestFileNeedScanSecrets(t *testing.T) {
	fmt.Println(FileNeedScanSecrets("/sbin/md5"))
}

func TestScanSecretsInFile(t *testing.T) {
	fmt.Println(imageAnalyzer.scanSecretsInFile("/Users/musso/workshop/docker-projects/test/secrets.txt"))
}
