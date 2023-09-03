package myutils

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

// CalSha256 对字符串计算sha256，并返回string
func CalSha256(s string) string {
	tmpHash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(tmpHash[:])
}

// StrLegalForRepository check whether string s is legal for repository search
func StrLegalForRepository(s string) bool {
	match, _ := regexp.MatchString(`^[a-zA-Z0-9:\-]*$`, s)
	return match
}

// StrLegalForImage check whether string s is legal for image search
func StrLegalForImage(s string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9:]*$", s)
	return match
}
