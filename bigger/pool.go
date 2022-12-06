package bigger

import (
	"context"
	"log"
	"sync"
)

var globalThreadPoolDic = struct {
	FileShaID sync.Map
}{}

type ThreadSharePool struct {
	CommitID   string
	MaxLen     int
	CurrentLen int

	DLMeta DownloadsMetaData

	FragmentRetry chan struct{}
	ShareDataInfo []ShareDataInfo
}

func GETThreadSharePool(initRs ShareDataInfo) (chan ShareDataInfo, chan struct{}, chan []byte) {
	backendUpdateShareDescribe := make(chan ShareDataInfo, 1)
	const maxLen = 8

	end := make(chan struct{}, 1)
	var writeChan = make(chan []byte, 1)
	var runtime *ThreadSharePool
	ctx, cancel := context.WithCancel(context.Background())

	if rsi := RestoreSharePoolFromStorage(); rsi != nil {
		runtime = rsi
	} else {
		runtime = &ThreadSharePool{
			CommitID:      initRs.CommitID,
			MaxLen:        maxLen, // 8 * 10M (Single)
			CurrentLen:    0,
			FragmentRetry: make(chan struct{}, 1),
			DLMeta: DownloadsMetaData{
				Downloader: make(chan DownloadFraMeta, 1),
				DLMutex:    sync.RWMutex{},
				Ctx:        ctx,
			},
			ShareDataInfo: make([]ShareDataInfo, maxLen),
		}

		fp := TruncatePlaceholder(initRs.CommitID, initRs.Size)
		runtime.DLMeta.FPoint = fp

		go func() {
			if fp == nil {
				end <- struct{}{}
			}
		}()
	}

	go func() {
		r := 0
		for {
			select {
			case <-runtime.FragmentRetry:
				r += 1
				if r >= 10 {
					log.Printf(
						"%s Too many retries of task sharding, the network may be abnormal, triggering a fuse",
						runtime.CommitID)
					end <- struct{}{}
				}
			case <-end:
				return
			}
		}
	}()

	// 出口转发层，也划定了全部的状态处理逻辑
	go func(cancel context.CancelFunc) {
		for {
			select {
			// 这里不止对出口进行审查，也对内部包进行一定操作
			case v := <-backendUpdateShareDescribe:
				log.Println(v)
				switch v.Status {
				case MessageStatus.Failed:
					// retry, constant
					runtime.FragmentRetry <- struct{}{}
					writeChan <- shareMessageMarshal(&v)
				case MessageStatus.Send:
					if runtime.AddTask(v) {
						v.Status = MessageStatus.Added
					} else {
						v.Status = MessageStatus.Failed
					}
					backendUpdateShareDescribe <- v
				case MessageStatus.Added:
					writeChan <- shareMessageMarshal(&v)
				case MessageStatus.End:
					end <- struct{}{}
				}
			case <-end:
				err := SaveSharePoolToStorage()
				cancel()
				if err != nil {
					log.Println(err)
				}
				return
			}
		}
	}(cancel)

	go Downloader(&runtime.DLMeta, backendUpdateShareDescribe)
	return backendUpdateShareDescribe, end, writeChan
}

func SaveSharePoolToStorage() error {
	return nil
}

func RestoreSharePoolFromStorage() *ThreadSharePool {
	return nil
}

func (tsp *ThreadSharePool) AddTask(si ShareDataInfo) bool {
	log.Printf("new tasks are added :%+v",si)
	if tsp.MaxLen <= tsp.CurrentLen+1 {
		return false
	}
	tsp.ShareDataInfo[tsp.CurrentLen] = si
	tsp.CurrentLen += len(tsp.ShareDataInfo) - 1
	return true
}

func (tsp *ThreadSharePool) DoneTask(si ShareDataInfo) {
	log.Printf("new tasks are deleted :%+v",si)
	taskID := si.ID
	if tsp.CurrentLen > 0 {
		tsp.CurrentLen -= 1
	}
	for i, v := range tsp.ShareDataInfo {
		if v.ID == taskID {
			x := &tsp.ShareDataInfo
			(*x)[i], (*x)[tsp.CurrentLen] = (*x)[tsp.CurrentLen], ShareDataInfo{}
		}
	}
}
