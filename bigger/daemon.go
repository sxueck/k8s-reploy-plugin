package bigger

import (
	"context"
	"errors"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/sxueck/k8sodep/model"
	"io"
	"log"
	"os"
)

type readCounter struct {
	io.Reader
	N int
}

// ImageUploadDaemon map[image_name]image_info
var imageUploadDaemon = make(map[string]model.ReCallDeployInfo)

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

// ImportImageToCluster 将镜像信息导入到集群 这里可以根据需要使用docker或者containerd的sdk进行替换
func ImportImageToCluster(fn string, si model.ReCallDeployInfo) error {
	cri, err :=
		containerd.New("/run/containerd/containerd.sock")

	if err != nil {
		return err
	}

	fp, err := os.Open(fn)
	if err != nil {
		return err
	}

	err = loadImage(
		namespaces.WithNamespace(context.Background(), "k8s.io"),
		cri, fp, fmt.Sprintf("%s:%s", si.Images, si.Tag))
	if err != nil {
		return err
	}
	return nil
}

func loadImage(ctx context.Context, client *containerd.Client, in io.Reader, imageName string) error {
	// In addition to passing WithImagePlatform() to client.Import(),
	// we also need to pass WithDefaultPlatform() to NewClient().
	// Otherwise unpacking may fail.
	r := &readCounter{Reader: in}
	imgs, err := client.Import(ctx, r,
		containerd.WithIndexName(imageName),
		containerd.WithAllPlatforms(true),
		containerd.WithSkipDigestRef(func(name string) bool { return name != "" }))
	if err != nil {
		if r.N == 0 {
			// Avoid confusing "unrecognized image format"
			return errors.New("no image was built")
		}
		return err
	}

	log.Println(imgs)
	return nil
}
