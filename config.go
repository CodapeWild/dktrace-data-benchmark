package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
)

type tracerName string

const (
	dd   tracerName = "ddtrace"
	jg   tracerName = "jaeger"
	otel tracerName = "open-telemetry"
	pp   tracerName = "pinpoint"
	sky  tracerName = "skywalking"
	zp   tracerName = "zipkin"
)

type tracerConfigOption func(tconfig *tracerConfig)

func tracerWithName(name tracerName) tracerConfigOption {
	return func(tconfig *tracerConfig) {
		tconfig.Tracer = string(name)
	}
}

func tracerWithTask(path string) tracerConfigOption {
	return func(tconfig *tracerConfig) {
		tconfig.TaskConfig = path
	}
}

func tracerWithAmplifier(threads, repeat int) tracerConfigOption {
	return func(tconfig *tracerConfig) {
		tconfig.SendThreads = threads
		tconfig.SendTimesPerThread = repeat
	}
}

func tracerWithCollector(proto string, ip string, port int, path string) tracerConfigOption {
	return func(tconfig *tracerConfig) {
		tconfig.CollectorProto = proto
		tconfig.CollectorIP = ip
		tconfig.CollectorPort = port
		tconfig.CollectorPath = path
	}
}

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

func (tconfig *tracerConfig) With(opts ...tracerConfigOption) *tracerConfig {
	for _, opt := range opts {
		opt(tconfig)
	}

	return tconfig
}

func NewTracerConfig(opts ...tracerConfigOption) *tracerConfig {
	tconfig := &tracerConfig{}
	for _, opt := range opts {
		opt(tconfig)
	}

	return tconfig
}

type demoConfigOption func(dconfig *demoConfig)

func demoWithLog(enable bool) demoConfigOption {
	return func(dconfig *demoConfig) {
		dconfig.DisableLog = enable
	}
}

func demoWithTracers(tconfigs ...*tracerConfig) demoConfigOption {
	return func(dconfig *demoConfig) {
		for i := range tconfigs {
			found := false
			for j := 0; j < len(dconfig.Tracers); j++ {
				if dconfig.Tracers[j].Tracer == tconfigs[i].Tracer {
					dconfig.Tracers[j] = tconfigs[i]
					found = true
					break
				}
			}
			if !found {
				dconfig.Tracers = append(dconfig.Tracers, tconfigs[i])
			}
		}
	}
}

type demoConfig struct {
	DisableLog bool            `json:"disable_log"`
	Tracers    []*tracerConfig `json:"tracers"`
}

func (dconfig *demoConfig) With(opts ...demoConfigOption) *demoConfig {
	for _, opt := range opts {
		opt(dconfig)
	}

	return dconfig
}

func NewDemoConfig(opts ...demoConfigOption) *demoConfig {
	dconfig := &demoConfig{}
	for _, opt := range opts {
		opt(dconfig)
	}

	return dconfig
}

var (
	tracers = map[tracerName]bool{
		dd:   true,
		jg:   true,
		otel: true,
		pp:   true,
		sky:  true,
		zp:   true,
	}
	envs = []string{"DKTRACE_CONFIG", "DKTRACE_DISABLE_LOG",
		"DKTRACE_TRACER", "DKTRACE_TASK_CONFIG", "DKTRACE_THREADS", "DKTRACE_SEND_TIMES_PER_THREAD",
		"DKTRACE_COLLECTOR_PROTO", "DKTRACE_COLLECTOR_IP", "DKTRACE_COLLECTOR_PORT", "DKTRACE_COLLECTOR_PATH"}
	demoConf *demoConfig
)

// default configurations
var (
	configFilePath     = "./config.json"
	disableLog         = false
	tracer             = dd
	taskConfig         = "./tasks/user-login.json"
	sendThreads        = 10
	sendTimesPerThread = 100
	collectorProto     = "http"
	collectorIP        = "127.0.0.1"
	collectorPort      = 9529
	collectorPath      = "/v0.4/traces"
)

func loadEnvVariables() {
	for _, key := range envs {
		v, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		switch key {
		case "DKTRACE_CONFIG":
			configFilePath = v
		case "DKTRACE_DISABLE_LOG":
			if b := strings.ToLower(v); b == "true" {
				disableLog = true
			}
		case "DKTRACE_TRACER":
			tracer = tracerName(v)
		case "DKTRACE_TASK_CONFIG":
			taskConfig = v
		case "DKTRACE_THREADS":
			if threads, err := strconv.Atoi(v); err == nil || threads > 0 {
				sendThreads = threads
			}
		case "DKTRACE_SEND_TIMES_PER_THREAD":
			if times, err := strconv.Atoi(v); err == nil || times > 0 {
				sendTimesPerThread = times
			}
		case "DKTRACE_COLLECTOR_PROTO":
			collectorProto = v
		case "DKTRACE_COLLECTOR_IP":
			collectorIP = v
		case "DKTRACE_COLLECTOR_PORT":
			if port, err := strconv.Atoi(v); err == nil || port > 0 {
				collectorPort = port
			}
		case "DKTRACE_COLLECTOR_PATH":
			collectorPath = v
		}
	}
}

func loadConfigFile(path string) (*demoConfig, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var demoConf demoConfig
	if err = json.Unmarshal(bts, &demoConf); err != nil {
		return nil, err
	}

	return &demoConf, nil
}

func buildDemoConfig() (*demoConfig, error) {
	if configFilePath != "" {
		return loadConfigFile(configFilePath)
	} else {
		return NewDemoConfig(demoWithLog(disableLog),
			demoWithTracers(NewTracerConfig(tracerWithName(tracer), tracerWithTask(taskConfig),
				tracerWithAmplifier(sendThreads, sendTimesPerThread),
				tracerWithCollector(collectorProto, collectorIP, collectorPort, collectorPath)))), nil
	}
}

func init() {
	loadEnvVariables()
	Execute()

	var err error
	demoConf, err = buildDemoConfig()
	if err != nil {
		log.Fatalln(err.Error())
	}
	log.Printf("final merged demo config is: %#v", *demoConf)

	if demoConf.DisableLog {
		log.SetOutput(nil)
	} else {
		log.SetFlags(log.Lshortfile | log.LstdFlags)
		log.SetOutput(os.Stdout)
	}
}
