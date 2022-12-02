//go:build !windows

package bigger

import (
	"fmt"
	"golang.org/x/net/context"
	"log"
	"os"
	"sync"
)

type DownloadsMetaData struct {
	Downloader chan DownloadFraMeta
	DLMutex    sync.RWMutex // 控制总文件的读写锁
	FPoint     *os.File     // 文件句柄，用以规避分片数量过多导致频繁 IO 开销
	Ctx        context.Context
}

// DownloadFraMeta 包含分片序号与分片内容，用以与当前队列进行匹配
type DownloadFraMeta struct {
	Bytes        *[]byte // 传递分片文件内容
	SerialNumber int
}

func TruncatePlaceholder(id string, size int64) *os.File {
	fn := fmt.Sprintf("%s.cache", id)
	_, err := os.Stat(fn)
	if !os.IsNotExist(err) {
		log.Printf("the file exists but cannot be loaded ： %s", err)
		return nil
	}

	f, err := os.Create(fn)
	if err != nil {
		log.Println(err)
		return nil
	}

	if err = f.Truncate(size); err != nil {
		log.Printf("unable to create placeholder file, maybe insufficient disk space ? - %s", err)
	}

	return f
}

// Downloader 该下载器实现了有序写入文件，与计划汇总
func Downloader(dl *DownloadsMetaData, si <-chan ShareDataInfo) chan error {
	var errChan = make(chan error, 1)
	for {
		select {
		case v := <-si:
			// 0 means relative to the origin of the file
			// 1 means relative to the current offset
			// 2 means relative to the end
			dl.DLMutex.Lock()
			if _, err := dl.FPoint.Seek(v.SeekStart, 1); err != nil {
				log.Printf("error writing file try again ： %s", err)
				errChan <- err
			}
			dl.DLMutex.Unlock()
		case <-dl.Ctx.Done():
			dl.DLMutex.Unlock()
			log.Println("the service is aborted and exiting the process")
		}
	}
}
