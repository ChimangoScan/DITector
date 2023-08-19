package mymysql

import "regexp"

var (
	escapeRegexp = regexp.MustCompile(`[\0\t\x1a\n\r\"\'\\]`)

	//see href='https://dev.mysql.com/doc/refman/8.0/en/string-literals.html#character-escape-sequences'
	characterEscapeMap = map[string]string{
		"\\0":  `\\0`, //ASCII NULL
		"\b":   `\\b`, //backspace
		"\t":   `\\t`, //tab
		"\x1a": `\\Z`, //ASCII 26 (Control+Z);
		"\n":   `\\n`, //newline character
		"\r":   `\\r`, //return character
		"\"":   `\"`,  //quote (")
		"'":    `\'`,  //quote (')
		"\\":   `\\`,  //backslash (\)
		//"\\%":  `\\%`,  //% character
		//"\\_":  `\\_`,  //_ character
	}
)

// EscapeString 将mysql输入字符串转义为安全的字符串
func EscapeString(val string) string {
	return escapeRegexp.ReplaceAllStringFunc(val, func(s string) string {

		mVal, ok := characterEscapeMap[s]
		if ok {
			return mVal
		}
		return s
	})
}
