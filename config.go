package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

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
	config             = flag.String("config", "", "A JSON file is used to configurate how to start demo.")
	tracer             = flag.String("tracer", "", "Tracer SDK that is used to generate trace data, only one value is accepted. Currently ddtrace or jeager or otel or pinpoint or skywalking or zipkin is accecpted.")
	taskConfig         = flag.String("task_config", "", "A JSON file contains the task that describes how the service trace would look like.")
	sendThreads        = flag.Int("send_threads", 0, "Define the number of threads need to start sending trace data.")
	sendTimesPerThread = flag.Int("send_times_per_thread ", 0, "Define the number of times that data should be repeatedly sent in each thread.")
	collectorProto     = flag.String("collector_proto", "", "The transport protocol accepted by trace collector.")
	collectorIP        = flag.String("collector_ip", "", "The IP address on which the trace collector is listening.")
	collectorPort      = flag.Int("colloctor_port", 0, "The trace collector uses this port number to receive trace data.")
	collectorPath      = flag.String("collector_path", "", "The trace collector uses this URL path string to receive trace data.")
)

var (
	envKeys       = []string{"TDD_TRACER", "TDD_TASK_CONFIG", "TDD_THREADS", "TDD_SEND_TIMES_PER_THREAD", "TDD_COLLECTOR_PROTO", "TDD_COLLECTOR_IP", "TDD_COLLECTOR_PORT", "TDD_COLLECTOR_PATH"}
	tracerConfigs []*tracerConfig
)

type tracerConfig struct {
	Tracer             string `json:"tracer"`
	TaskConfig         string `json:"task_config"`
	SendThreads        int    `json:"send_threads"`
	SendTimesPerThread int    `json:"send_times_per_thread"`
	CollectorProto     string `json:"collector_proto"`
	CollectorIP        string `json:"collector_ip"`
	CollectorPort      int    `json:"collector_port"`
	CollectorPath      string `json:"collector_path"`
}

func loadConfigFile(path string) ([]*tracerConfig, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	configs := &[]*tracerConfig{}
	if err = json.Unmarshal(bts, configs); err != nil {
		return nil, err
	}

	return *configs, nil
}

func loadStartupParameters() (*tracerConfig, error) {
	var (
		config = &tracerConfig{}
		ok     = true
	)
	if *tracer != "" {
		config.Tracer = *tracer
	} else {
		ok = false
	}
	if *taskConfig != "" {
		config.TaskConfig = *taskConfig
	} else {
		ok = false
	}
	if *sendThreads > 0 {
		config.SendThreads = *sendThreads
	} else {
		ok = false
	}
	if *sendTimesPerThread > 0 {
		config.SendTimesPerThread = *sendTimesPerThread
	} else {
		ok = false
	}
	if *collectorProto != "" {
		config.CollectorProto = *collectorProto
	} else {
		ok = false
	}
	if *collectorIP != "" {
		config.CollectorIP = *collectorIP
	} else {
		ok = false
	}
	if *collectorPort > 0 {
		config.CollectorPort = *collectorPort
	} else {
		ok = false
	}
	config.CollectorPath = *collectorPath

	var err error
	if !ok {
		err = fmt.Errorf("load startup parameters failed: %v", os.Args)
	}

	return config, err
}

func loadEnvVariables() (*tracerConfig, error) {
	var (
		config = &tracerConfig{}
		ok     = true
	)
	for _, key := range envKeys {
		var v string
		v, ok = os.LookupEnv(key)
		if key == "TDD_COLLECTOR_PATH" {
			config.CollectorPath = v
			ok = true
		} else if ok && v != "" {
			switch key {
			case "TDD_TRACER":
				config.Tracer = v
			case "TDD_TASK_CONFIG":
				config.TaskConfig = v
			case "TDD_THREADS":
				if threads, err := strconv.Atoi(v); err != nil || threads <= 0 {
					ok = false
				} else {
					config.SendThreads = threads
				}
			case "TDD_SEND_TIMES_PER_THREAD":
				if times, err := strconv.Atoi(v); err != nil || times <= 0 {
					ok = false
				} else {
					config.SendTimesPerThread = times
				}
			case "TDD_COLLECTOR_PROTO":
				config.CollectorProto = v
			case "TDD_COLLECTOR_IP":
				config.CollectorIP = v
			case "TDD_COLLECTOR_PORT":
				if port, err := strconv.Atoi(v); err != nil || port <= 0 {
					ok = false
				} else {
					config.CollectorPort = port
				}
			}
		}
		if !ok {
			return nil, fmt.Errorf("load environment variables failed: %v", os.Environ())
		}
	}

	return config, nil
}

func loadDefaultStartupParameters() *tracerConfig {
	return &tracerConfig{
		Tracer:             "ddtrace",
		TaskConfig:         "./tasks/def.json",
		SendThreads:        10,
		SendTimesPerThread: 100,
		CollectorProto:     "http",
		CollectorIP:        "127.0.0.1",
		CollectorPort:      9529,
		CollectorPath:      "/v0.4/traces",
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	if len(os.Args) > 1 {
		flag.Parse()
	}

	var err error
	if *config != "" {
		if tracerConfigs, err = loadConfigFile(*config); err == nil {
			return
		}
	}
	var config *tracerConfig
	if config, err = loadStartupParameters(); err == nil {
		tracerConfigs = []*tracerConfig{config}

		return
	}
	if config, err = loadEnvVariables(); err == nil {
		tracerConfigs = []*tracerConfig{config}

		return
	}
	if tracerConfigs, err = loadConfigFile("./config.json"); err == nil {
		return
	}
	if err != nil {
		log.Fatalln(err.Error())
	}
}
