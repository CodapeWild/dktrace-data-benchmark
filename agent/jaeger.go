/*
 *   Copyright (c) 2023 CodapeWild
 *   All rights reserved.

 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at

 *   http://www.apache.org/licenses/LICENSE-2.0

 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package agent

import (
	"bytes"
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"

	"github.com/CodapeWild/devkit/comerr"
	dkhttp "github.com/CodapeWild/devkit/net/http"
	"github.com/uber/jaeger-client-go/thrift"
	"github.com/uber/jaeger-client-go/thrift-gen/jaeger"
)

var (
	jgV01            = "v01"
	jgPatternVersion = map[string]string{
		"/apis/traces": jgV01,
	}
)

type JgAgent struct {
	http.ServeMux
}

func (jga *JgAgent) Start(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, jga); err != nil {
			log.Fatalln(err.Error())
		}
	}()
}

func newJgAgent(amp *jgAmplifier) *JgAgent {
	if amp == nil {
		log.Fatalln("traces amplifier for jaeger agent can not be nil")
	}

	agent := &JgAgent{}
	for p, v := range jgPatternVersion {
		agent.HandleFunc(p, handleJgTracesWrapper(p, v, amp))
	}

	return agent
}

func handleJgTracesWrapper(pattern, version string, amp *jgAmplifier) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Println("jg: received http headers")
		for k, v := range req.Header {
			log.Printf("%s: %v", k, v)
		}

		var (
			batch *jaeger.Batch
			err   error
		)
		switch version {
		case jgV01:
			batch, err = decodeJgBinaryProtocol(req.Body)
		default:
			err = comerr.ErrUnrecognizedParameters(version)
		}

		if err != nil {
			log.Println(err.Error())

			return
		} else if len(batch.Spans) == 0 {
			log.Println("jg: empty trace")

			return
		}

		amp.AppendTrace(&jgReqWrapper{header: req.Header, batch: batch})
	}
}

func decodeJgBinaryProtocol(r io.Reader) (*jaeger.Batch, error) {
	tmbuf := thrift.NewTMemoryBuffer()
	_, err := tmbuf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	var (
		trans = thrift.NewTBinaryProtocolConf(tmbuf, &thrift.TConfiguration{})
		batch = &jaeger.Batch{}
	)

	return batch, batch.Read(context.Background(), trans)
}

func encodeJgBinaryProtocol(batch *jaeger.Batch) ([]byte, error) {
	if batch == nil {
		return nil, comerr.ErrInvalidParameters
	}

	tmbuf := thrift.NewTMemoryBuffer()
	trans := thrift.NewTBinaryProtocolConf(tmbuf, &thrift.TConfiguration{})
	err := batch.Write(context.Background(), trans)
	if err != nil {
		return nil, err
	}

	return tmbuf.Bytes(), nil
}

type jgReqWrapper struct {
	header http.Header
	batch  *jaeger.Batch
}

type jgAmplifier struct {
	*GeneralAmplifier
	expectedSpansCount, receivedSpansCount int
	header                                 http.Header
	batch                                  *jaeger.Batch
	ready                                  chan any
}

func (jgamp *jgAmplifier) AppendTrace(jgreq *jgReqWrapper) {
	if jgreq.batch == nil || len(jgreq.batch.Spans) == 0 {
		return
	}

	jgamp.header = dkhttp.MergeHeaders(jgamp.header, jgreq.header)
	if jgamp.batch == nil {
		jgamp.batch = jgreq.batch
	} else {
		jgamp.batch.Spans = append(jgamp.batch.Spans, jgreq.batch.Spans...)
	}
	jgamp.receivedSpansCount += len(jgreq.batch.Spans)
	if jgamp.receivedSpansCount >= jgamp.expectedSpansCount {
		jgamp.ready <- &jgReqWrapper{header: jgamp.header, batch: jgamp.batch}
	}
}

func (jgamp *jgAmplifier) StartThreads(ctx context.Context, endpoint string) (finish chan struct{}, err error) {
	return jgamp.GeneralAmplifier.StartThreads(ctx, endpoint, jgamp.ready)
}

func (jgamp *jgAmplifier) Close() {
	select {
	case <-jgamp.close:
	default:
		close(jgamp.close)
	}
}

func jgAmplifierThread(ID int, ctx context.Context, endpoint string, repeat int, trace any, threadDown chan int) error {
	jgreq, ok := trace.(*jgReqWrapper)
	if !ok {
		return comerr.ErrAssertFailed
	}

	var (
		client  = &http.Client{Transport: newSingleHostTransport()}
		replica = duplicateJgBatch(jgreq.batch)
	)
	for i := 1; i <= repeat; i++ {
		if buf, err := encodeJgBinaryProtocol(replica); err != nil {
			log.Println(err.Error())
		} else {
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(buf))
			if err != nil {
				log.Fatalln(err)
			}
			req.Header = jgreq.header
			resp, err := client.Do(req)
			if err != nil {
				log.Println(err.Error())
			} else {
				log.Printf("thread %d send %d times status: %s", ID, i, resp.Status)
				resp.Body.Close()
			}
		}
		changeJgTraceIDs(replica)
	}
	threadDown <- ID

	return nil
}

func duplicateJgBatch(batch *jaeger.Batch) *jaeger.Batch {
	buf, err := encodeJgBinaryProtocol(batch)
	if err != nil {
		log.Fatalln(err.Error())
	}
	dupli := &jaeger.Batch{}
	if dupli, err = decodeJgBinaryProtocol(bytes.NewBuffer(buf)); err != nil {
		log.Fatalln(err.Error())
	}

	return dupli
}

func changeJgTraceIDs(batch *jaeger.Batch) {
	var (
		newtidh = rand.Int63()
		newtidl = rand.Int63()
	)
	for _, span := range batch.Spans {
		span.TraceIdHigh = newtidh
		span.TraceIdLow = newtidl

		var (
			newsid = rand.Int63()
			oldsid = span.SpanId
		)
		span.SpanId = newsid
		for _, s := range batch.Spans {
			if s.ParentSpanId == oldsid {
				span.ParentSpanId = newsid
				break
			}
		}
	}
}

func newJgAmplifier(endpointAddress string, expectedSpansCount, threads, repeat int) *jgAmplifier {
	return &jgAmplifier{
		GeneralAmplifier:   NewGeneralAmplifier("jaeger", threads, repeat, jgAmplifierThread),
		expectedSpansCount: expectedSpansCount,
		ready:              make(chan any),
	}
}

func StartJgAgent(agentAddress, endpointAddress string, expectedSpansCount, threads, repeat int) (context.CancelFunc, chan struct{}, error) {
	ctx, canceler := context.WithCancel(context.TODO())

	ampf := newJgAmplifier(endpointAddress, expectedSpansCount, threads, repeat)
	finish, err := ampf.StartThreads(ctx, endpointAddress)
	if err != nil {
		return nil, nil, err
	}

	agent := newJgAgent(ampf)
	agent.Start(agentAddress)

	return canceler, finish, nil
}
