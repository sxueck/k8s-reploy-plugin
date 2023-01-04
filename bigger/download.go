package bigger

import (
	"errors"
	"fmt"
	"github.com/labstack/gommon/bytes"
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
	// 如果业务需要，可以适当放宽这里的限制值
	const limitLarge int64 = 4
	if size > limitLarge*bytes.GB {
		log.Printf("The file size is too large."+
			" Only files smaller than %d GB are allowed to be uploaded", limitLarge)
		return nil
	}

	fn := fmt.Sprintf("%s.cache", id)
	_, err := os.Stat(fn)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("the file exists but cannot be loaded ： %s", err)
			return nil
		}
		log.Println("the file does not exist and is being created")
	}

	f, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Println(err)
		return nil
	}

	if err = f.Truncate(size); err != nil {
		log.Printf("unable to create placeholder file, maybe insufficient disk space ? - %s", err)
		return nil
	}

	log.Printf("%s file created successfully", fn)

	return f
}

// Downloader 该下载器实现了有序写入文件，与计划汇总
func Downloader(dl *DownloadsMetaData, si <-chan ShareDataInfo) chan error {
	var errChan = make(chan error, 1)
	for {
		select {
		case v := <-si:
			if v.ID == -1 {
				log.Println("break signal")
				continue
			}
			// 0 means relative to the origin of the file
			// 1 means relative to the current offset
			// 2 means relative to the end
			dl.DLMutex.Lock()
			if _, err := dl.FPoint.Seek(v.Seek, 1); err != nil {
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
