package bigger

import "sync"

var globalThreadPoolDic = struct {
	FileShaID sync.Map
}{}

type ThreadSharePool struct {
	ID         string
	MaxLen     int
	CurrentLen int

	ShareDataInfo []ShareDataInfo
}

func NewThreadSharePool(id string) *ThreadSharePool {
	const maxLen = 8
	return &ThreadSharePool{
		ID:            id,
		MaxLen:        maxLen, // 8 * 10M (Single)
		CurrentLen:    0,
		ShareDataInfo: make([]ShareDataInfo, maxLen),
	}
}

func (tsp *ThreadSharePool) AddTask(si ShareDataInfo) bool {
	if tsp.MaxLen <= tsp.CurrentLen+1 {
		return false
	}
	tsp.CurrentLen += 1
	tsp.ShareDataInfo[tsp.CurrentLen] = si
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
