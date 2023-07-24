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

package main

import (
	"context"
	"log"
	"os"

	"github.com/CodapeWild/dktrace-data-benchmark/agent"
)

func main() {
	Execute()

	var err error
	benchConf, err = buildBenchmarkConfig()
	if err != nil {
		log.Fatalln(err.Error())
	}
	benchConf.Print()

	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
	if benchConf.DisableLog {
		log.Println("log disabled")
		log.SetOutput(nil)
	}

	if benchConf == nil || len(benchConf.Tracers) == 0 {
		log.Println("dktrace-data-benchmark not configurated properly")

		return
	}

	for _, v := range benchConf.Tracers {
		task, err := newTaskFromJSONFile(v.TaskConfig)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		var (
			canceler context.CancelFunc
			finish   chan struct{}
		)
		switch v.Tracer {
		case dd:
			canceler, finish, err = benchDDTraceCollector(v, task)
		case jg:
		case otel:
		case pp:
		case sky:
		case zpk:
		default:
			log.Printf("unrecognized tracer %s\n", v.Tracer)
		}
		if err != nil {
			canceler()
			log.Println(err.Error())
			continue
		}
		if finish != nil {
			<-finish
		}
	}
}

func benchDDTraceCollector(tconf *tracerConfig, task task) (canceler context.CancelFunc, finish chan struct{}, err error) {
	tr := task.createTree(&ddtracerwrapper{})
	agentAddress := agent.NewRandomPortWithLocalHost()
	canceler, finish, err = agent.BuildDDAgentForWork(agentAddress, tconf.CollectorIP, tconf.CollectorPort, tconf.CollectorPath, tr.count(), tconf.SendThreads, tconf.SendTimesPerThread)
	tr.spawn(context.TODO(), agentAddress)

	return
}
