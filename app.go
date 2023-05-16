package main

import "log"

func main() {
	log.Println(*config)
	log.Println(*tracer)
	log.Println(*taskConfig)
	log.Println(*sendThreads)
	log.Println(*sendTimesPerThread)
	log.Println(*collectorProto)
	log.Println(*collectorIP)
	log.Println(*collectorPort)
	log.Println(*collectorPath)
}
