package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	dd   string = "ddtrace"
	jg   string = "jaeger"
	otel string = "open-telemetry"
	pp   string = "pinpoint"
	sky  string = "skywalking"
	zpk  string = "zipkin"
)

type tracerConfigOption func(tconfig *tracerConfig)

func tracerWithName(name string) tracerConfigOption {
	return func(tconfig *tracerConfig) {
		tconfig.Tracer = name
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

type benchConfigOption func(dconfig *benchConfig)

func benchWithLog(enable bool) benchConfigOption {
	return func(dconfig *benchConfig) {
		dconfig.DisableLog = enable
	}
}

func benchWithTracers(tconfigs ...*tracerConfig) benchConfigOption {
	return func(dconfig *benchConfig) {
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

type benchConfig struct {
	DisableLog bool            `json:"disable_log"`
	Tracers    []*tracerConfig `json:"tracers"`
}

func (bconfig *benchConfig) With(opts ...benchConfigOption) *benchConfig {
	for _, opt := range opts {
		opt(bconfig)
	}

	return bconfig
}

func (bconfig *benchConfig) Print() {
	log.Println("benchmark config:")
	log.Println("##################")
	if bconfig.DisableLog {
		log.Println("log: disabled")
	} else {
		log.Println("log: enabled")
	}
	for _, tconf := range bconfig.Tracers {
		log.Println("------")
		log.Printf("tracer: %s", tconf.Tracer)
		log.Printf("task config: %s", tconf.TaskConfig)
		log.Printf("send threads: %d", tconf.SendThreads)
		log.Printf("send times per thread: %d", tconf.SendTimesPerThread)
		log.Printf("collector proto: %s", tconf.CollectorProto)
		log.Printf("collector ip: %s", tconf.CollectorIP)
		log.Printf("collector port: %d", tconf.CollectorPort)
		log.Printf("collector path: %s", tconf.CollectorPath)
	}
}

func NewBenchmarkConfig(opts ...benchConfigOption) *benchConfig {
	dconfig := &benchConfig{}
	for _, opt := range opts {
		opt(dconfig)
	}

	return dconfig
}

var (
	tracers = map[string]bool{
		dd:   true,
		jg:   true,
		otel: true,
		pp:   true,
		sky:  true,
		zpk:  true,
	}
	envs = []string{"DKTRACE_CONFIG", "DKTRACE_DISABLE_LOG",
		"DKTRACE_TRACER", "DKTRACE_TASK_CONFIG", "DKTRACE_THREADS", "DKTRACE_SEND_TIMES_PER_THREAD",
		"DKTRACE_COLLECTOR_PROTO", "DKTRACE_COLLECTOR_IP", "DKTRACE_COLLECTOR_PORT", "DKTRACE_COLLECTOR_PATH"}
	benchConf *benchConfig
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
			tracer = v
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

func loadConfigFile(path string) (*benchConfig, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var benchConf benchConfig
	if err = json.Unmarshal(bts, &benchConf); err != nil {
		return nil, err
	}

	return &benchConf, nil
}

func buildBenchmarkConfig() (*benchConfig, error) {
	if configFilePath != "" {
		return loadConfigFile(configFilePath)
	} else {
		return NewBenchmarkConfig(benchWithLog(disableLog),
			benchWithTracers(NewTracerConfig(tracerWithName(tracer), tracerWithTask(taskConfig),
				tracerWithAmplifier(sendThreads, sendTimesPerThread),
				tracerWithCollector(collectorProto, collectorIP, collectorPort, collectorPath)))), nil
	}
}

func init() {
	loadEnvVariables()
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
}
