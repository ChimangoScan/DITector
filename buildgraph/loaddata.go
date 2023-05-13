package buildgraph

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// 面向JSON数据源的接口

var (
	fileRepository *os.File
	fileTags       *os.File
	fileImages     *os.File
)

// ReadFileRepositoryByLine 用于逐行读取fileRepository，并将结果转换为Repository
func ReadFileRepositoryByLine() {
	fmt.Println("[INFO] Begin to read fileRepository")

	scanner := bufio.NewReader(fileRepository)
	for i := 0; i < 10; i++ {
		b, err := scanner.ReadBytes('\n')
		if err != nil {
			// 读到fileRepository结尾，退出
			if err == io.EOF {
				fileRepository.Close()
				fmt.Println("[INFO] Read fileRepository done")
				close(chanRepository)
				break
			}
			fmt.Println("[ERROR] Fail to ReadLine in ReadFileRepositoryByLine: Line ", i, ", err: ", err)
			break
		}

		var repo = new(Repository)
		err = json.Unmarshal(b, repo)
		if err != nil {
			fmt.Println("[ERROR] json.Unmarshal failed with: ", err)
			continue
		}
		chanRepository <- repo
	}
}

// ReadFileTagsByLine 用于逐行读取fileTags，并将结果转换为Tag
func ReadFileTagsByLine() {
	fmt.Println("[INFO] Begin to read fileTags")

	scanner := bufio.NewReader(fileTags)
	for i := 0; i < 10; i++ {
		b, err := scanner.ReadBytes('\n')
		if err != nil {
			// 读到fileTags结尾，退出
			if err == io.EOF {
				fileTags.Close()
				fmt.Println("[INFO] Read fileTags done")
				close(chanTag)
				break
			}
			fmt.Println("[ERROR] Fail to ReadLine in ReadFileTagsByLine: Line ", i, ", err: ", err)
			break
		}

		var tag = new(Tag)
		err = json.Unmarshal(b, tag)
		if err != nil {
			fmt.Println("[ERROR] json.Unmarshal failed with: ", err)
			continue
		}
		chanTag <- tag
	}
}

// ReadFileImagesByLine 用于逐行读取fileImages，并将结果转换为Image
func ReadFileImagesByLine() {
	fmt.Println("[INFO] Begin to read fileImages")

	scanner := bufio.NewReader(fileImages)
	for i := 0; i < 10; i++ {
		b, err := scanner.ReadBytes('\n')
		if err != nil {
			// 读到文件结尾，退出
			if err == io.EOF {
				fileImages.Close()
				fmt.Println("[INFO] Read fileImages done")
				close(chanImage)
				break
			}
			fmt.Println("[ERROR] Fail to ReadLine in ReadFileRepositoryByLine: Line ", i, ", err: ", err)
			break
		}

		var image = new(Image)
		err = json.Unmarshal(b, image)
		if err != nil {
			fmt.Println("[ERROR] json.Unmarshal failed with: ", err)
			continue
		}
		chanImage <- image
	}
}
