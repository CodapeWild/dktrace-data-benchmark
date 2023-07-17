package main

import (
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
		switch v.Tracer {
		case dd:
			tr := task.createTree(&ddtracerwrapper{})
			agentAddress := agent.NewRandomPortWithLocalHost()
			_ = agent.BuildDDAgentForWork(agentAddress, v.CollectorIP, v.CollectorPort, v.CollectorPath, tr.count(), v.SendThreads, v.SendTimesPerThread)
			tr.spawn(agentAddress)
		case jg:
		case otel:
		case pp:
		case sky:
		case zpk:
		default:
			log.Printf("unrecognized tracer %s\n", v.Tracer)
		}
	}

	select {}
}
