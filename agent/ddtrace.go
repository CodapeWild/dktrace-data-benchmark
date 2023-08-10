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
	dkhttp "github.com/CodapeWild/devkit/net/http"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

var _ Amplifier = (*ddAmplifier)(nil)

var (
	ddV01            = "v01"
	ddV02            = "v02"
	ddV03            = "v03"
	ddV04            = "v04"
	ddV05            = "v05"
	ddV07            = "v07"
	ddPatternVersion = map[string]string{
		"/spans": ddV01, "/v0.1/spans": ddV01,
		"/v0.2/traces": ddV02,
		"/v0.3/traces": ddV03,
		"/v0.4/traces": ddV04,
		"/v0.5/traces": ddV05,
		"/v0.7/traces": ddV07,
	}
)

type ddReqWrapper struct {
	header http.Header
	traces pb.Traces
}

type DDAgent struct {
	http.ServeMux
}

func (ddagt *DDAgent) Start(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, ddagt); err != nil {
			log.Fatalln(err.Error())
		}
	}()
}

func newDDAgent(amp Amplifier) *DDAgent {
	if amp == nil {
		log.Fatalln("traces amplifier for ddtrace agent can not be nil")
	}

	agent := &DDAgent{}
	for p, v := range ddPatternVersion {
		agent.HandleFunc(p, handleDDTracesWrapper(p, v, amp))
	}

	return agent
}

func handleDDTracesWrapper(pattern, version string, amp Amplifier) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("dd: received http headers:")
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
		case ddV01:
			var spans []*pb.Span
			if err = json.NewDecoder(req.Body).Decode(&spans); err == nil {
				traces = append(traces, pb.Trace(spans))
			}
		case ddV02, ddV03, ddV04:
			traces, err = decodeDDRequest(req)
		case ddV05:
			bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
				if _, err = io.Copy(buf, req.Body); err == nil {
					err = traces.UnmarshalMsgDictionary(buf.Bytes())
				}
			})
		case ddV07:
			bufpool.MakeUseOfBuffer(func(buf *bytes.Buffer) {
				if _, err = io.Copy(buf, req.Body); err == nil {
					_, err = traces.UnmarshalMsg(buf.Bytes())
				}
			})
		default:
			err = comerr.ErrUnrecognizedParameters(version)
		}

		// reply ok or error based on parameter err
		reply(pattern, version, resp, err)
		if err != nil {
			log.Println(err.Error())

			return
		} else if len(traces) == 0 {
			log.Println("dd: empty trace")

			return
		}

		amp.AppendData(&ddReqWrapper{header: req.Header, traces: traces})
	}
}

func reply(pattern, version string, resp http.ResponseWriter, err error) {
	if err == nil {
		resp.WriteHeader(http.StatusOK)
		switch version {
		case ddV01, ddV02, ddV03:
			io.WriteString(resp, "OK\n")
		default:
			resp.Header().Set("Content-Type", "application/json")
			resp.Write([]byte("{}"))
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
	}
}

func decodeDDRequest(req *http.Request) (pb.Traces, error) {
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

type ddAmplifier struct {
	colAddr            string
	threads, repeat    int
	expectedSpansCount int
	receivedSpansCount int
	header             http.Header
	traces             pb.Traces
	ddReqChan          chan *ddReqWrapper
	ready              chan struct{}
	finish             chan int
	closer             chan struct{}
}

func (ddamp *ddAmplifier) AppendData(value any) error {
	ddreq, ok := value.(*ddReqWrapper)
	if !ok {
		return comerr.ErrAssertFailed
	}

	ddamp.header = dkhttp.MergeHeaders(ddamp.header, ddreq.header)
	ddamp.traces = append(ddamp.traces, ddreq.traces...)
	for _, trace := range ddreq.traces {
		ddamp.receivedSpansCount += len(trace)
	}
	if ddamp.receivedSpansCount >= ddamp.expectedSpansCount {
		ddamp.ready <- struct{}{}
	}

	return nil
}

func (ddamp *ddAmplifier) StartThreads(ctx context.Context) (finish chan struct{}, err error) {
	if err = ctx.Err(); err != nil {
		return
	}

	finish = make(chan struct{})
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
			case <-ddamp.closer:
				log.Println("ddtrace amplifier closed")

				return
			case <-ddamp.ready:
				ddamp.runThreads(&ddReqWrapper{header: ddamp.header, traces: ddamp.traces})
			case thID := <-ddamp.finish:
				log.Printf("thread id: %d accomplished", thID)
				if finished++; finished == ddamp.threads {
					finish <- struct{}{}

					return
				}
			}
		}
	}()

	return
}

func (ddamp *ddAmplifier) Close() {
	select {
	case <-ddamp.closer:
		return
	default:
		close(ddamp.closer)
	}
}

func (ddamp *ddAmplifier) runThreads(ddreq *ddReqWrapper) {
	client := http.Client{Transport: newSingleHostTransport()}
	for i := 1; i <= ddamp.threads; i++ {
		dupli := duplicateDDTraces(ddreq.traces)
		go func(traces pb.Traces, i int) {
			for j := 1; j <= ddamp.repeat; j++ {
				if bts, err := traces.MarshalMsg(nil); err != nil {
					log.Println(err.Error())
				} else {
					req, err := http.NewRequest(http.MethodPost, ddamp.colAddr, bytes.NewBuffer(bts))
					if err != nil {
						log.Fatalln(err)
					}
					req.Header = ddreq.header
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
			ddamp.finish <- i
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

func newDDAmplifier(ip string, port int, path string, expectedSpansCount, threads, repeat int) *ddAmplifier {
	return &ddAmplifier{
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

	ampf := newDDAmplifier(endpointIP, endpointPort, endpointPath, expectedSpansCount, threads, repeat)
	finish, err := ampf.StartThreads(ctx)
	if err != nil {
		return nil, nil, err
	}

	agent := newDDAgent(ampf)
	agent.Start(agentAddress)

	return canceler, finish, nil
}
