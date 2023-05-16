package main

import "flag"

// start up parameters
// -config ./config.json
// -tracer ddtrace | jeager | otel | pinpoint | skywalking | zipkin
// -task_config ./tasks/def.json
// -send_threads 10
// -send_times_per_thread 100
// -collector_proto http
// -collector_ip 127.0.0.1
// -collector_port 9529
// -collector_path /v0.3/trace
var (
	config             = flag.String("config", "./config.json", "A JSON file is used to configurate how to start demo.")
	tracer             = flag.String("tracer", "ddtrace", "Tracer SDK that is used to generate trace data, only one value is accepted. Currently ddtrace or jeager or otel or pinpoint or skywalking or zipkin is accecpted.")
	taskConfig         = flag.String("task_config", "./tasks/user-login.json", "A JSON file contains the task that describes how the service trace would look like.")
	sendThreads        = flag.Int("send_threads", 10, "Define the number of threads need to start sending trace data.")
	sendTimesPerThread = flag.Int("send_times_per_thread ", 100, "Define the number of times that data should be repeatedly sent in each thread.")
	collectorProto     = flag.String("collector_proto", "http", "The transport protocol accepted by trace collector.")
	collectorIP        = flag.String("collector_ip", "127.0.0.1", "The IP address on which the trace collector is listening.")
	collectorPort      = flag.Int("colloctor_port", 9529, "The trace collector uses this port number to receive trace data.")
	collectorPath      = flag.String("collector_path", "/v0.4/trace", "The trace collector uses this URL path string to receive trace data.")
)

func init() {
	flag.Parse()
}
