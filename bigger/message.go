package bigger

import (
	"encoding/xml"
	"log"
)

var messageStatus = struct {
	Send   int
	Added  int
	Failed int
}{1, 2, 3}

type ShareDataInfo struct {
	XMLName   xml.Name `xml:"xml"`
	Text      string   `xml:",chardata"`
	ID        string   `xml:"id"`
	MD5       string   `xml:"md5"`
	FileName  string   `xml:"file-name"`
	Size      string   `xml:"size"`
	Time      string   `xml:"time"`
	SeekStart int32    `xml:"seek-start"`
	SeekEnd   int32    `xml:"seek-end"`
	Status    int      `xml:"status"`
}

func shareMessageMarshal(si *ShareDataInfo) []byte {
	x, err := xml.Marshal(si)
	if err != nil {
		log.Printf("xml marshal error : %s", err)
		return nil
	}
	return x
}

func shareMessageUnmarshal(bs []byte) *ShareDataInfo {
	si := &ShareDataInfo{}
	err := xml.Unmarshal(bs, si)
	if err != nil {
		log.Printf("xml unmarshal error : %s", err)
		return nil
	}
	return si
}
