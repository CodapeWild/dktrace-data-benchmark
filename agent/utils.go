package agent

import (
	"bytes"
	"log"
	"mime"
	"net/http"
	"sync"
)

var bufpool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func getBuffer() *bytes.Buffer {
	return bufpool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	bufpool.Put(buf)
}

func getMetaType(req *http.Request, def string) string {
	mt, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("get meta type failed: %s", err.Error())

		return def
	}

	return mt
}
