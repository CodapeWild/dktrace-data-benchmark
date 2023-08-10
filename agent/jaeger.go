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
	"context"
	"io"
	"log"
	"net/http"

	"github.com/CodapeWild/devkit/comerr"
	"github.com/uber/jaeger-client-go/thrift"
	"github.com/uber/jaeger-client-go/thrift-gen/jaeger"
)

var _ Amplifier = (*jgAmplifier)(nil)

var (
	jgV01            = "v01"
	jgPatternVersion = map[string]string{
		"/apis/traces": jgV01,
	}
)

type jgReqWrapper struct {
	headers http.Header
}

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

func newJgAgent(amp Amplifier) *JgAgent {
	if amp == nil {
		log.Fatalln("traces amplifier for jaeger agent can not be nil")
	}

	agent := &JgAgent{}
	for p, v := range jgPatternVersion {
		agent.HandleFunc(p, handleJgTracesWrapper(p, v, amp))
	}

	return agent
}

func handleJgTracesWrapper(pattern, version string, amp Amplifier) http.HandlerFunc {
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

type jgAmplifier struct {
}

func (jgamp *jgAmplifier) AppendData(value any) error {
	return nil
}

func (jgamp *jgAmplifier) StartThreads(ctx context.Context) (finish chan struct{}, err error) {
	return
}

func (jgamp *jgAmplifier) Close() {

}

func newJgAmplifier(ip string, port int, path string, expectedSpansCount, threads, repeat int) *jgAmplifier {
	return nil
}

func BuildJgAgentForWork(agentAddress string, endpointIP string, endpointPort int, endpointPath string, expectedSpansCount, threads, repeat int) (context.CancelFunc, chan struct{}, error) {
	ctx, canceler := context.WithCancel(context.TODO())

	ampf := newJgAmplifier(endpointIP, endpointPort, endpointPath, expectedSpansCount, threads, repeat)
	finish, err := ampf.StartThreads(ctx)
	if err != nil {
		return nil, nil, err
	}

	agent := newJgAgent(ampf)
	agent.Start(agentAddress)

	return canceler, finish, nil
}
