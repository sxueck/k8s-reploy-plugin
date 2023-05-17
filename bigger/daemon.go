package bigger

import (
	"github.com/sxueck/k8sodep/model"
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
