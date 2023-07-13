package main

import (
	"context"
	"log"

	"github.com/CodapeWild/trace-data-demo/agent"
)

var (
	ddtrace       = "ddtrace"
	jaeger        = "jaeger"
	opentelemetry = "open-telemetry"
	pinpoint      = "pinpoint"
	skywalking    = "sky-walking"
	zipkin        = "zipkin"
)

func main() {
	Execute()

	if demoConf == nil || len(demoConf.Tracers) == 0 {
		log.Println("trace-data-demo not configurated properly")

		return
	}

	for _, v := range demoConf.Tracers {
		task, err := newTaskFromJSONFile(v.TaskConfig)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		var cancler context.CancelFunc
		switch v.Tracer {
		case ddtrace:
			tr := task.createTree(&ddtracerwrapper{})
			agentAddress := agent.NewRandomPortWithLocalHost()
			cancler = agent.BuildDDAgentForWork(agentAddress, v.CollectorIP, v.CollectorPort, v.CollectorPath, tr.count(), v.SendThreads, v.SendTimesPerThread)
			tr.spawn(agentAddress)
		case jaeger:
		case opentelemetry:
		case pinpoint:
		case skywalking:
		case zipkin:
		default:
			log.Printf("unrecognized tracer %s\n", v.Tracer)
		}
		if cancler != nil {
			cancler()
		}
	}
}
