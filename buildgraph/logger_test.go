package buildgraph

import "testing"

func TestLogBuilderString(t *testing.T) {
	config("json")
	logBuilderString("This is for test")
}
