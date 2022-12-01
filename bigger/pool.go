package bigger

import (
	"log"
	"sync"
)

var globalThreadPoolDic = struct {
	FileShaID sync.Map
}{}

type ThreadSharePool struct {
	ID         string
	MaxLen     int
	CurrentLen int

	FragmentRetry chan struct{}
	ShareDataInfo []ShareDataInfo
}

func GETThreadSharePool(id string) (chan ShareDataInfo, chan struct{}, chan []byte) {
	backendUpdateShareDescribe := make(chan ShareDataInfo, 1)
	const maxLen = 8

	var writeChan = make(chan []byte, 1)
	var runtime *ThreadSharePool
	if rsi := RestoreSharePoolFromStorage(); rsi != nil {
		runtime = rsi
	} else {
		runtime = &ThreadSharePool{
			ID:            id,
			MaxLen:        maxLen, // 8 * 10M (Single)
			CurrentLen:    0,
			FragmentRetry: make(chan struct{}, 1),
			ShareDataInfo: make([]ShareDataInfo, maxLen),
		}
	}

	end := make(chan struct{}, 1)
	go func() {
		r := 0
		for {
			select {
			case <-runtime.FragmentRetry:
				r += 1
				if r >= 10 {
					log.Printf(
						"%s Too many retries of task sharding, the network may be abnormal, triggering a fuse",
						runtime.ID)
					end <- struct{}{}
				}
			case <-end:
				return
			}
		}
	}()

	// 出口转发层，也划定了全部的状态处理逻辑
	go func() {
		for {
			select {
			// 这里不止对出口进行审查，也对内部包进行一定操作
			case v := <-backendUpdateShareDescribe:
				switch v.Status {
				case messageStatus.Failed:
					// retry, constant
					runtime.FragmentRetry <- struct{}{}
					writeChan <- shareMessageMarshal(&v)
				case messageStatus.Send:
					if runtime.AddTask(v) {
						v.Status = messageStatus.Added
					} else {
						v.Status = messageStatus.Failed
					}
					backendUpdateShareDescribe <- v
				case messageStatus.Added:
					writeChan <- shareMessageMarshal(&v)
				case messageStatus.End:
					end <- struct{}{}
				}
			case <-end:
				err := SaveSharePoolToStorage()
				if err != nil {
					log.Println(err)
				}
				return
			}
		}
	}()

	return backendUpdateShareDescribe, end, writeChan
}

func SaveSharePoolToStorage() error {
	return nil
}

func RestoreSharePoolFromStorage() *ThreadSharePool {
	return nil
}

func (tsp *ThreadSharePool) AddTask(si ShareDataInfo) bool {
	if tsp.MaxLen <= tsp.CurrentLen+1 {
		return false
	}
	tsp.ShareDataInfo[tsp.CurrentLen] = si
	tsp.CurrentLen += len(tsp.ShareDataInfo) - 1
	return true
}

func (tsp *ThreadSharePool) DoneTask(si ShareDataInfo) {
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
