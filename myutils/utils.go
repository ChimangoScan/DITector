package myutils

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
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

// DivideImageName 拆解镜像名称，[registry/][namespace/]repository[:tag][@digest]
func DivideImageName(name string) (registry, namespace, repository, tag, digest string) {
	// obtain digest by splitting by "@"
	digestParts := strings.Split(name, "@")
	if len(digestParts) == 2 {
		digest = digestParts[1]
	}

	// obtain tag by splitting by ":"
	nameParts := strings.Split(digestParts[0], ":")
	switch len(nameParts) {
	case 1:
		tag = "latest"
	case 2:
		tag = nameParts[1]
	}

	// obtain registry, namespace, repository by splitting by "/"
	repoParts := strings.Split(nameParts[0], "/")
	switch len(repoParts) {
	case 1:
		registry = "docker.io"
		namespace = "library"
		repository = repoParts[0]
	case 2:
		registry = "docker.io"
		namespace = repoParts[0]
		repository = repoParts[1]
	case 3:
		registry = repoParts[0]
		namespace = repoParts[1]
		repository = repoParts[2]
	}

	return
}
