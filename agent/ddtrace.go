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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/CodapeWild/devkit/bufpool"
	"github.com/CodapeWild/devkit/comerr"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

var (
	V01            = "v01"
	V02            = "v02"
	V03            = "v03"
	V04            = "v04"
	V05            = "v05"
	V07            = "v07"
	PatternVersion = map[string]string{
		"/spans": V01, "/v0.1/spans": V01,
		"/v0.2/traces": V02,
		"/v0.3/traces": V03,
		"/v0.4/traces": V04,
		"/v0.5/traces": V05,
		"/v0.7/traces": V07,
	}
)

type ddReqWrapper struct {
	headers http.Header
	traces  pb.Traces
}

type DDAgent struct {
	http.ServeMux
}

func (dd *DDAgent) Start(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, dd); err != nil {
			log.Fatalln(err.Error())
		}
	}()
}

func NewDDAgent(ampf *DDAmplifier) *DDAgent {
	if ampf == nil {
		log.Fatalln("traces amplifier for ddtrace agent can not be nil")
	}

	dd := &DDAgent{}
	for pattern, version := range PatternVersion {
		dd.HandleFunc(pattern, handleTracesWrapper(version, ampf))
	}

	return dd
}

func handleTracesWrapper(version string, ampf *DDAmplifier) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("received ddtrace headers:")
		for k, v := range req.Header {
			log.Printf("%s: %v", k, v)
		}

		tc := countTraces(req)
		if tc == 0 {
			resp.WriteHeader(http.StatusOK)

			return
		}

		var (
			traces pb.Traces
			err    error
		)
		switch version {
		case V01:
			var spans []*pb.Span
			if err = json.NewDecoder(req.Body).Decode(&spans); err == nil {
				traces = append(traces, pb.Trace(spans))
			}
		case V02, V03, V04:
			traces, err = decodeRequest(req)
		case V05:
			bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
				if _, err = io.Copy(buf, req.Body); err == nil {
					err = traces.UnmarshalMsgDictionary(buf.Bytes())
				}
			})
		case V07:
			bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
				if _, err = io.Copy(buf, req.Body); err == nil {
					_, err = traces.UnmarshalMsg(buf.Bytes())
				}
			})
		}

		// reply ok or error based on parameter err
		reply(version, resp, err)
		if err != nil {
			log.Println(err.Error())

			return
		} else if len(traces) == 0 {
			log.Println("empty traces")

			return
		}

		log.Printf("%v", traces)

		ampf.SendData(&ddReqWrapper{headers: req.Header, traces: traces})
	}
}

func reply(version string, resp http.ResponseWriter, err error) {
	if err == nil {
		resp.WriteHeader(http.StatusOK)
		switch version {
		case V01, V02, V03:
			io.WriteString(resp, "OK\n")
		default:
			resp.Header().Set("Content-Type", "application/json")
			resp.Write([]byte("{}"))
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
	}
}

func decodeRequest(req *http.Request) (pb.Traces, error) {
	var (
		traces pb.Traces
		err    error
	)
	switch mt := getMetaType(req, "application/json"); mt {
	case "application/msgpack":
		bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
			if _, err = io.Copy(buf, req.Body); err == nil {
				_, err = traces.UnmarshalMsg(buf.Bytes())
			}
		})
	case "application/json", "test/json", "":
		err = json.NewDecoder(req.Body).Decode(&traces)
	default:
		err = fmt.Errorf("unrecognized media type: %s", mt)
	}

	return traces, err
}

func countTraces(req *http.Request) int {
	v := req.Header.Get("X-Datadog-Trace-Count")
	if v == "" || v == "0" {
		return 0
	}
	if c, err := strconv.Atoi(v); err != nil {
		return 0
	} else {
		return c
	}
}

type DDAmplifier struct {
	colAddr            string
	threads, repeat    int
	expectedSpansCount int
	receivedSpansCount int
	traces             pb.Traces
	ddReqChan          chan *ddReqWrapper
	finish             chan int
	closer             chan struct{}
}

func (ddampf *DDAmplifier) SendData(value any) error {
	ddreq, ok := value.(*ddReqWrapper)
	if !ok {
		return comerr.ErrAssertFailed
	}

	ddampf.traces = append(ddampf.traces, ddreq.traces...)
	for _, trace := range ddreq.traces {
		ddampf.receivedSpansCount += len(trace)
	}
	if ddampf.receivedSpansCount >= ddampf.expectedSpansCount {
		ddampf.ddReqChan <- ddreq
	}

	return nil
}

func (ddampf *DDAmplifier) StartThreads(ctx context.Context) (chan struct{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	finish := make(chan struct{})
	go func() {
		var finished int
		for {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("error: %s", err.Error())
				} else {
					log.Println("ddtrace amplifier context done")
				}

				return
			case <-ddampf.closer:
				log.Println("ddtrace amplifier closed")

				return
			case traces := <-ddampf.ddReqChan:
				ddampf.runThreads(traces)
			case thID := <-ddampf.finish:
				log.Printf("thread id: %d accomplished", thID)
				if finished++; finished == ddampf.threads {
					finish <- struct{}{}

					return
				}
			}
		}
	}()

	return finish, nil
}

func (ddampf *DDAmplifier) Close() {
	select {
	case <-ddampf.closer:
		return
	default:
		close(ddampf.closer)
	}
}

func (ddampf *DDAmplifier) runThreads(ddreq *ddReqWrapper) {
	client := http.Client{Transport: newSingleHostTransport()}
	for i := 1; i <= ddampf.threads; i++ {
		dupli := duplicateDDTraces(ddreq.traces)
		go func(traces pb.Traces, i int) {
			for j := 1; j <= ddampf.repeat; j++ {
				if bts, err := traces.MarshalMsg(nil); err != nil {
					log.Println(err.Error())
				} else {
					req, err := http.NewRequest(http.MethodPost, ddampf.colAddr, bytes.NewBuffer(bts))
					if err != nil {
						log.Fatalln(err)
					}
					req.Header = ddreq.headers
					resp, err := client.Do(req)
					if err != nil {
						log.Println(err.Error())
					} else {
						log.Printf("thread %d send %d times status: %s", i, j, resp.Status)
						resp.Body.Close()
					}
				}
				changeIDs(traces)
			}
			ddampf.finish <- i
		}(dupli, i)
	}
}

func duplicateDDTraces(traces pb.Traces) pb.Traces {
	var dupli *pb.Traces = &pb.Traces{}
	bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
		if err := json.NewEncoder(buf).Encode(traces); err != nil {
			log.Fatalln(err.Error())
		}
		if err := json.NewDecoder(buf).Decode(dupli); err != nil {
			log.Fatalln(err.Error())
		}
	})

	return *dupli
}

func changeIDs(traces pb.Traces) {
	for i := range traces {
		var newtid = rand.Uint64()
		for j := range traces[i] {
			traces[i][j].TraceID = newtid
			var (
				newcid = rand.Uint64()
				oldcid = traces[i][j].SpanID
			)
			traces[i][j].SpanID = newcid
			for k := j + 1; k < len(traces[i]); k++ {
				if traces[i][k].ParentID == oldcid {
					traces[i][k].ParentID = newcid
					break
				}
			}
		}
	}
}

func NewDDAmplifier(ip string, port int, path string, expectedSpansCount, threads, repeat int) *DDAmplifier {
	return &DDAmplifier{
		colAddr:            fmt.Sprintf("http://%s:%d%s", ip, port, path),
		expectedSpansCount: expectedSpansCount,
		threads:            threads,
		repeat:             repeat,
		ddReqChan:          make(chan *ddReqWrapper),
		finish:             make(chan int),
		closer:             make(chan struct{}),
	}
}

func BuildDDAgentForWork(agentAddress string, endpointIP string, endpointPort int, endpointPath string, expectedSpansCount, threads, repeat int) (context.CancelFunc, chan struct{}, error) {
	ctx, canceler := context.WithCancel(context.TODO())

	ampf := NewDDAmplifier(endpointIP, endpointPort, endpointPath, expectedSpansCount, threads, repeat)
	finish, err := ampf.StartThreads(ctx)
	if err != nil {
		return nil, nil, err
	}

	agent := NewDDAgent(ampf)
	agent.Start(agentAddress)

	return canceler, finish, nil
}
