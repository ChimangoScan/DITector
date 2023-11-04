package analyzer

import (
	"fmt"
	"github.com/Musso12138/dockercrawler/myutils"
	"log"
	"testing"
)

func TestNewImageAnalyzerGlobalConfig(t *testing.T) {

}

func TestAnalyzeImageMetadata(t *testing.T) {
	targetImages := make([]*myutils.Image, 0)
	targetImages = append(targetImages, &myutils.Image{
		Layers: []myutils.Layer{
			myutils.Layer{},
			myutils.Layer{Digest: "123456", Instruction: "-----BEGIN RSA PRIVATE KEYsk_test_000011112222333344445555", Size: 10},
		},
	})
	for _, targetImage := range targetImages {
		results, _ := imageAnalyzer.AnalyzeImageMetadata(targetImage)
		for _, result := range results {
			fmt.Println(result)
		}
	}
}

func TestScanSecretsInString(t *testing.T) {
	if imageAnalyzerE != nil {
		log.Fatalln(imageAnalyzerE)
	}

	secrets, _ := imageAnalyzer.scanSecretsInString("-----BEGIN RSA PRIVATE KEYsk_test_000011112222333344445555")
	for _, secret := range secrets {
		fmt.Println(secret)
	}
}
