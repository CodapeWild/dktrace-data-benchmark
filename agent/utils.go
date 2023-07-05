package agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
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

// generate a random port ranging from 6000 to 9000
func NewRandomPortWithLocalHost() string {
	return fmt.Sprintf("127.0.0.1:%d", rand.Intn(3000)+6000)
}
