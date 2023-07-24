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
		<-finish
		log.Println("### finished")
	}
}

func benchDDTraceCollector(tconf *tracerConfig, task task) (canceler context.CancelFunc, finish chan struct{}, err error) {
	tr := task.createTree(&ddtracerwrapper{})
	agentAddress := agent.NewRandomPortWithLocalHost()
	canceler, finish, err = agent.BuildDDAgentForWork(agentAddress, tconf.CollectorIP, tconf.CollectorPort, tconf.CollectorPath, tr.count(), tconf.SendThreads, tconf.SendTimesPerThread)
	tr.spawn(context.TODO(), agentAddress)

	return
}
