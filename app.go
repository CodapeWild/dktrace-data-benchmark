package main

import (
	"context"
	"log"

	"github.com/CodapeWild/dktrace-data-benchmark/agent"
)

func main() {
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
		var cancler context.CancelFunc
		switch v.Tracer {
		case dd:
			tr := task.createTree(&ddtracerwrapper{})
			agentAddress := agent.NewRandomPortWithLocalHost()
			cancler = agent.BuildDDAgentForWork(agentAddress, v.CollectorIP, v.CollectorPort, v.CollectorPath, tr.count(), v.SendThreads, v.SendTimesPerThread)
			tr.spawn(agentAddress)
		case jg:
		case otel:
		case pp:
		case sky:
		case zpk:
		default:
			log.Printf("unrecognized tracer %s\n", v.Tracer)
		}
		if cancler != nil {
			cancler()
		}
	}
}
