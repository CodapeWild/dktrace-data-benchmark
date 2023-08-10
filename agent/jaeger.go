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
	"log"
	"net/http"
)

var _ Amplifier = (*JgAmplifier)(nil)

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

		switch version {
		case jgV01:
		default:
		}
	}
}

type JgAmplifier struct {
}

func (jgamp *JgAmplifier) SendData(value any) error {
	return nil
}

func (jgamp *JgAmplifier) StartThreads(ctx context.Context) (finish chan struct{}, err error) {
	return
}

func (jgamp *JgAmplifier) Close() {

}

func newJgAmplifier(ip string, port int, path string, expectedSpansCount, threads, repeat int) *JgAmplifier {
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
