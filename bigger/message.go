package bigger

import (
	"encoding/xml"
	"log"
)

var MessageStatus = struct {
	Init   int // Client 发送，连接后第一个发送的包，用于创建Thread
	Send   int // Client 发送，代表这是一个新分片事务
	Added  int // Server 发送，代表事务已经被加入队列
	Failed int // Server 发送，代表事务被拒绝，可能是队列已满或者系统错误
	End    int // 双边发送，代表任务被结束，可能是用户截断或者网络异常
}{1, 2, 3, 4, 5}

type ShareDataInfo struct {
	XMLName  xml.Name `xml:"xml"`
	Text     string   `xml:",chardata"`
	ID       int      `xml:"id"` // 16 进制标识
	CommitID string   `xml:"commit-id"`
	MD5      string   `xml:"md5"`
	FileName string   `xml:"file-name"`
	Size     int64    `xml:"size"`
	Time     string   `xml:"time"`
	Seek     int64    `xml:"seek"`
	Status   int      `xml:"status"`
}

func ShareMessageMarshal(si *ShareDataInfo) []byte {
	x, err := xml.Marshal(si)
	if err != nil {
		log.Printf("xml marshal error : %s", err)
		return nil
	}
	return x
}

func ShareMessageUnmarshal(bs []byte) *ShareDataInfo {
	si := &ShareDataInfo{}
	err := xml.Unmarshal(bs, si)
	if err != nil {
		log.Printf("xml unmarshal error : %s", err)
		return nil
	}
	return si
}
