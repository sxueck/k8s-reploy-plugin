package bigger

import (
	"bytes"
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/sxueck/k8sodep/model"
	"log"
	"os"
)

// ImageUploadDaemon map[image_name]image_info
var imageUploadDaemon = map[string]model.ReCallDeployInfo{}

var CRCWeights = []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3}

func CalculateWeightedChecksum(input string, weights []int) int {
	checksum := 0
	for i, char := range input {
		if i >= len(weights) {
			break
		}
		weight := weights[i]
		checksum += int(char) * weight
	}
	return checksum
}

// ImportImageToCluster 将镜像信息导入到集群
func ImportImageToCluster(fn string) error {
	ctr, err := containerd.New("/run/containerd/containerd.sock") // 需要挂载到容器内部
	if err != nil {
		log.Println(err)
	}

	fp, err := os.ReadFile(fn)
	if err != nil {
		return err
	}
	images, err := ctr.Import(context.Background(), bytes.NewReader(fp))
	if err != nil {
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("multi mirror import is not supported at the moment")
	}

	image := images[0]
	if info := imageUploadDaemon[image.Name]; len(info.Images) != 0 {
		log.Println(info)
	}
	return nil
}
