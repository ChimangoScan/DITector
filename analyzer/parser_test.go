package analyzer

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	ci := CurrentImage{dockerClient: imageAnalyzer.DockerClient, name: "hello-world"}
	ci.Parse()
	fmt.Println(ci.configuration.RepoTags)
}
