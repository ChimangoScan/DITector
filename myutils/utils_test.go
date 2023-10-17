package myutils

import (
	"fmt"
	"testing"
)

func TestDivideImageName(t *testing.T) {
	fmt.Println(DivideImageName("hello-world"))
	fmt.Println(DivideImageName("minio/minio"))
	fmt.Println(DivideImageName("docker.io/library/mongo"))
	fmt.Println(DivideImageName("hello-world@sha256:88ec0acaa3ec199d3b7eaf73588f4518c25f9d34f58ce9a0df68429c5af48e8d"))
	fmt.Println(DivideImageName("library/hello-world@sha256:88ec0acaa3ec199d3b7eaf73588f4518c25f9d34f58ce9a0df68429c5af48e8d"))
	fmt.Println(DivideImageName("hello-world:latest@sha256:88ec0acaa3ec199d3b7eaf73588f4518c25f9d34f58ce9a0df68429c5af48e8d"))
	fmt.Println(DivideImageName("library/hello-world:latest@sha256:88ec0acaa3ec199d3b7eaf73588f4518c25f9d34f58ce9a0df68429c5af48e8d"))
	fmt.Println(DivideImageName("docker.io/library/hello-world:latest@sha256:88ec0acaa3ec199d3b7eaf73588f4518c25f9d34f58ce9a0df68429c5af48e8d"))
}
