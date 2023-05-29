package agent

import (
	"bytes"
	"context"
	"log"
	"mime"
	"net"
	"net/http"
	"sync"
	"time"
)

type Amplifier interface {
	StartThreads(ctx context.Context)
	Close()
}

var bufpool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

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

func newSingleHostTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
		WriteBufferSize:       10 * 1024,
	}
}
