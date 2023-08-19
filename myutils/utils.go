package myutils

import (
	"crypto/sha256"
	"encoding/hex"
)

// CalSha256 对字符串计算sha256，并返回string
func CalSha256(s string) string {
	tmpHash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(tmpHash[:])
}
