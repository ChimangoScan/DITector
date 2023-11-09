package misconfiguration

import "regexp"

var mongoConfFileRe = regexp.MustCompile("mongo(d|db)?.conf$")

func isMongoConfFile(filepath string) bool {
	return mongoConfFileRe.MatchString(filepath)
}
