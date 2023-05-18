package bigger

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"github.com/labstack/echo/v4"
	"github.com/sxueck/k8sodep/model"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
)

func DecompressData(compressedData []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	decompressedData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	defer func(r *gzip.Reader) {
		err := r.Close()
		if err != nil {
			log.Println("reader.Close err:", err)
		}
	}(reader)
	return decompressedData, nil
}

func ComputeMD5HashString(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// WriteBytesToFile debug
func WriteBytesToFile(filename string, data []byte) error {
	err := os.WriteFile(filename, data, 0644)
	return err
}

func StartRecvUploadHandle() echo.MiddlewareFunc {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取文件名和分片编号
		log.Println("r.Header:", r.Header)
		fn := r.Header.Get("File-Name")
		fn = path.Base(fn)
		fileSize, _ := strconv.ParseInt(r.Header.Get("Content-Range"), 10, 64)
		partNumber, _ := strconv.Atoi(r.Header.Get("Part-Number"))

		isEnd := r.Header.Get("Last-Part")
		chunkSize, _ := strconv.ParseInt(r.Header.Get("Origin-Size"), 10, 64)

		tDir, _ := os.MkdirTemp("", "images")
		fileName := path.Join(tDir, fn)

		// 以读写模式打开文件
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if partNumber == 0 {
			log.Println("the first slice")
			// 如果是第一片，则创建一个新文件
			err := os.Truncate(fileName, fileSize)
			if err != nil {
				log.Println(err)
				return
			}
		}
		defer file.Close()

		// 将文件指针移动到指定位置
		offset := int64(partNumber) * chunkSize
		// 如果为最后一片，则chunkSize为非标准大小
		// 则使用part*size为offset会导致不正常的覆盖写入

		if len(isEnd) != 0 {
			log.Println("the last slice")
			// 注意这里，如果直接想以文件结尾追加写入
			// 需要注意协程的执行并不是随机的，所以可能会导致文件内容不完整

			_, err = file.Seek(func() int64 { // 对小文件的适配
				if offset == 0 {
					return 0
				}
				return fileSize - chunkSize
			}(), io.SeekStart)
		} else {
			_, err = file.Seek(offset, io.SeekStart)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var bs []byte
		bs, err = io.ReadAll(r.Body)
		if err != nil {
			log.Println("io.ReadAll err:", err)
			return
		}

		log.Println(len(bs), "bytes received.")

		m5 := r.Header.Get("Md5")
		if ComputeMD5HashString(bs) != m5 {
			log.Printf("Share MD5 %s not match，it could be a network anomaly", m5)
			return
		}

		//dbs, err := DecompressData(bs)
		if err != nil {
			log.Println("Share decompressData err:", err)
			return
		}
		// 写入文件内容
		_, err = io.Copy(file, bytes.NewReader(bs))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("slice write to file successful : %s", m5)
	})

	m := echo.WrapMiddleware(func(handler http.Handler) http.Handler {
		return h
	})

	return m
}

func RegisterUploadTaskToDaemon(c echo.Context) error {
	task := &model.ReCallDeployInfo{}
	err := c.Bind(task)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	if task.AccessToken != os.Getenv("ACCESS_TOKEN") || task.AccessToken == "" {
		return c.String(http.StatusForbidden, "forbidden")
	}

	imageUploadDaemon[task.Images] = *task
	return c.String(http.StatusOK, "ok")
}
