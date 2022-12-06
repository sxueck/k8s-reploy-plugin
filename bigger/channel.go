package bigger

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"hash"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// HmacKey 摘要算法强制要求KEY，由于只涉及文件校验，写死在代码里也可以
const HmacKey = "The-quick-brown-fox-jumps-over-the-lazy-dog"

// BackgroundFileThread 每开启一个文件上传任务
// 就建立一个守护进程，这样即使ws连接断了也不影响总体进程
//func BackgroundFileThread(id string, timeout time.Duration, pool chan *ThreadSharePool) {
//	// id 是通过 MD5 与文件名相交计算而来，用于落盘进度文件名等，这样即使服务重启也能借凭该ID文件恢复数据
//	log.Printf("%s 守护线程启动", id)
//	ticker := time.NewTicker(timeout)
//	for {
//		select {
//		case <-ticker.C:
//			// 超时后结束任务并且落盘相关的进度信息
//			log.Println("file upload timed out")
//			close(pool)
//		case p := <-pool:
//			globalThreadPoolDic.FileShaID.Store(id, &p)
//			return
//		}
//	}
//}

func UniqueIDCalculation(filename string, md5 string) string {
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(HmacKey))
	_, _ = io.WriteString(h, fmt.Sprintf("%s:%s", filename, md5))
	return hex.EncodeToString(h.Sum(nil))
}

// TODO: 关闭Websocket连接这边需要做优化处理

// ImagesInfoHandler 初始化文件信息沟通接口
func ImagesInfoHandler(c echo.Context) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.String(http.StatusServiceUnavailable, err.Error())
	}
	defer func(ws *websocket.Conn) {
		err = ws.Close()
		if err != nil {
			log.Printf("warn - ws close error : %s ", err)
		}
	}(ws)

	var tsp chan ShareDataInfo
	var end chan struct{}
	var writeChan chan []byte

	// Read
	go func() {
		for {
			var msg []byte
			_, msg, err = ws.ReadMessage()
			if err != nil {
				log.Println(err)
			}
			// 代表分片信息已经成功加入就绪队列
			// 如果拒绝，Client 等待 x 个间隔后重新发送查询
			si := parsePack(msg)
			if si.Status == MessageStatus.Init {
				oid := UniqueIDCalculation(si.FileName, si.MD5)
				si.ID = -1
				si.CommitID = oid
				tsp, end, writeChan = GETThreadSharePool(*si)
				si.Status = MessageStatus.Added
			}

			// 尽量不要直接对接出口直接发送，由一层转发层执行
			tsp <- *si
		}

	}()

	// Write method
	for {
		select {
		case bs := <-writeChan:
			err = ws.WriteMessage(websocket.TextMessage, bs)
			if err != nil {
				log.Println(err)
				return c.String(http.StatusBadGateway, "send failed")
			}
		case <-end:
			return c.String(http.StatusOK, "process done")
		}
	}
}

// ShareColumnImagesUploadHandler 文件分片上传接口
func ShareColumnImagesUploadHandler(c echo.Context) error {
	var PostHeaderColumn = struct {
		CommitID string
		MD5      string
	}{}

	reErr := func(err error) error {
		return c.String(http.StatusBadRequest, fmt.Sprintf("submit an exception : %s", err))
	}

	// id 字符串-分片序号：读取后需要拆分开
	commitId := c.Get(PostHeaderColumn.CommitID)
	checkoutMd5 := c.Get(PostHeaderColumn.MD5)

	if commitId == "" || checkoutMd5 == "" {
		return reErr(errors.New("header does not have checksum field, this submission is rejected"))
	}

	x := strings.Split(commitId.(string), "-")
	cid, fra := x[0], x[1]
	frai, err := strconv.Atoi(fra)
	if err != nil {
		return reErr(fmt.Errorf("%s The field format is incorrect, "+
			"does not contain the fragment sequence number, "+
			"or the sequence number is incorrect", err))
	}

	postFrom, err := c.FormFile("fragment")
	if err != nil {
		return reErr(err)
	}

	fp, err := postFrom.Open()
	if err != nil {
		return reErr(err)
	}

	var bs = make([]byte, postFrom.Size)
	_, err = io.ReadFull(fp, bs)
	if err != nil {
		return reErr(err)
	}

	// 拿到自己唯一 ID 的池子，往里塞入本次上传的文件内容
	curPool, ok := globalThreadPoolDic.FileShaID.Load(cid)
	if ok {
		p := curPool.(*ThreadSharePool)

		p.DLMeta.Downloader <- DownloadFraMeta{
			Bytes:        &bs,
			SerialNumber: frai,
		}
	}

	return c.String(http.StatusOK, "the task has been committed and is awaiting asynchronous processing")
}

func parsePack(msg []byte) *ShareDataInfo {
	fatalReturn := &ShareDataInfo{Status: MessageStatus.Failed}
	si := shareMessageUnmarshal(msg)
	if si == nil {
		log.Println("fatal - incorrect interface submission content")
		return fatalReturn
	}

	if si.Status != MessageStatus.Send && si.Status != MessageStatus.Init {
		log.Println("not a normal request message")
		si.Status = MessageStatus.Failed
		return fatalReturn
	}

	return si
}
