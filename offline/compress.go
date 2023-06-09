package offline

import (
	"os"
)

// 反射链的声明和控制

const ImagePullPolicyLink = "Spec.Template.Spec.Containers" // 修改镜像的拉取方式

// UpdatedAnnotations 修改为离线模式后进行标识以便还原
// 推荐使用标签而不是注解，因为注解可能会为空导致反射异常，而标签能快速进行选择
const UpdatedAnnotations = "Spec.Template.Labels"
const UpdateMarkKey = "OfflineSetAt" // 离线模式标识key

type DiskStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
}

type CompressImage struct {
}

// FreeDiskSpace 返回磁盘剩余空间 - MB Size
//func (cs *CompressImage) FreeDiskSpace(driverID string) uint64 {
//	diskUsage := func(driverID string) uint64 {
//		fs := syscall.Statfs_t{}
//		err := syscall.Statfs(driverID, &fs)
//		if err != nil {
//			log.Printf("init driver syscall error : %s", err)
//			return 0
//		}
//		return fs.Bfree * uint64(fs.Bsize)
//	}
//	return diskUsage(driverID) / bytes.MB
//}

func (cs *CompressImage) Compress() {

}

func (cs *CompressImage) UnCompress() {

}

func (cs *CompressImage) SaveOf(sPath string) *os.File {
	return nil
}
