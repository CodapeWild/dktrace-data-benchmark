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
	"encoding/json"
	"log"
	"os"
	"strings"
)

type tracerConfigOption func(tkconf *taskConfig)

func tracerWithName(name string) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.Name = name
	}
}

func tracerWithTracer(tracer string) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.Tracer = tracer
	}
}

func tracerWithVersion(version string) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.Version = version
	}
}

func tracerWithRoute(path string) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.RouteConfig = path
	}
}

func tracerWithAmplifier(threads, repeat int) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.SendThreads = threads
		tkconf.SendTimesPerThread = repeat
	}
}

func tracerWithCollector(proto string, ip string, port int, path string) tracerConfigOption {
	return func(tkconf *taskConfig) {
		tkconf.CollectorProto = proto
		tkconf.CollectorIP = ip
		tkconf.CollectorPort = port
		tkconf.CollectorPath = path
	}
}

type taskConfig struct {
	Name               string `json:"name"`
	Tracer             string `json:"tracer"`
	Version            string `json:"version"`
	RouteConfig        string `json:"route_config"`
	SendThreads        int    `json:"send_threads"`
	SendTimesPerThread int    `json:"send_times_per_thread"`
	CollectorProto     string `json:"collector_proto"`
	CollectorIP        string `json:"collector_ip"`
	CollectorPort      int    `json:"collector_port"`
	CollectorPath      string `json:"collector_path"`
}

func (tkconf *taskConfig) With(opts ...tracerConfigOption) *taskConfig {
	for _, opt := range opts {
		opt(tkconf)
	}

	return tkconf
}

func (tkconf *taskConfig) Print() {
	log.Println("------")
	log.Printf("Name: %s", tkconf.Name)
	log.Printf("Tracer: %s", tkconf.Tracer)
	log.Printf("Version: %s", tkconf.Version)
	log.Printf("Route: %s", tkconf.RouteConfig)
	log.Printf("Threads: %d Repeated: %d", tkconf.SendThreads, tkconf.SendTimesPerThread)
	log.Printf("Collector: <%s://%s:%d%s>", tkconf.CollectorProto, tkconf.CollectorIP, tkconf.CollectorPort, tkconf.CollectorPath)
}

func NewTaskConfig(opts ...tracerConfigOption) *taskConfig {
	tkconf := &taskConfig{}
	for _, opt := range opts {
		opt(tkconf)
	}

	return tkconf
}

type benchConfigOption func(bconf *benchConfig)

func benchWithLog(enable bool) benchConfigOption {
	return func(bconf *benchConfig) {
		bconf.DisableLog = enable
	}
}

func benchWithTasks(tasks ...*taskConfig) benchConfigOption {
	return func(bconf *benchConfig) {
		for _, new := range tasks {
			found := false
			for _, origin := range bconf.Tasks {
				if origin.Name == new.Name {
					origin = new
					found = true
					break
				}
			}
			if !found {
				bconf.Tasks = append(bconf.Tasks, new)
			}
		}
	}
}

type benchConfig struct {
	DisableLog bool          `json:"disable_log"`
	Tasks      []*taskConfig `json:"tasks"`
}

func (bconf *benchConfig) With(opts ...benchConfigOption) *benchConfig {
	for _, opt := range opts {
		opt(bconf)
	}

	return bconf
}

func (bconf *benchConfig) Print() {
	log.Println("trace benchmark config:")
	log.Println("### ### ###")
	if bconf.DisableLog {
		log.Println("log: disabled")
	} else {
		log.Println("log: enabled")
	}
	for _, tkconf := range bconf.Tasks {
		tkconf.Print()
	}
}

func newBenchmarkConfig(opts ...benchConfigOption) *benchConfig {
	dconfig := &benchConfig{}
	for _, opt := range opts {
		opt(dconfig)
	}

	return dconfig
}

const (
	dd   string = "ddtrace"
	jg   string = "jaeger"
	otel string = "open-telemetry"
	pp   string = "pinpoint"
	sky  string = "skywalking"
	zpk  string = "zipkin"
)

var (
	tracers = map[string]bool{
		dd:   true,
		jg:   true,
		otel: true,
		pp:   true,
		sky:  true,
		zpk:  true,
	}
	envs       = []string{"DKTRACE_CONFIG", "DKTRACE_DISABLE_LOG", "DKTRACE_TASKS"}
	gBenchConf *benchConfig
	gTasks     []*taskConfig
)

// default configurations
var (
	defBenchConf  = "./config.json"
	defDisableLog = false
	defTask       = &taskConfig{
		Name:               "default",
		Tracer:             "v0.4",
		Version:            "ddtrace",
		RouteConfig:        "./routes/user-login.json",
		SendThreads:        3,
		SendTimesPerThread: 10,
		CollectorProto:     "http",
		CollectorIP:        "127.0.0.1",
		CollectorPort:      9529,
		CollectorPath:      "/v0.4/traces",
	}
)

func loadEnvVariables() {
	for _, key := range envs {
		v, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		switch key {
		case "DKTRACE_CONFIG":
			defBenchConf = v
		case "DKTRACE_DISABLE_LOG":
			if b := strings.ToLower(v); b == "true" {
				defDisableLog = true
			}
		case "DKTRACE_TASKS":
			var tasks = &[]*taskConfig{}
			if err := json.Unmarshal([]byte(v), tasks); err != nil {
				log.Println(err.Error())
			} else {
				for _, task := range *tasks {
					gTasks = append(gTasks, task)
				}
			}
		}
	}
}

func loadBenchConfigFile(path string) (*benchConfig, error) {
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

func mergeTasks(dst *[]*taskConfig, src []*taskConfig) {
	for _, s := range src {
		found := false
		for _, d := range *dst {
			if d.Name == s.Name {
				found = true
				d = s
				break
			}
		}
		if !found {
			*dst = append(*dst, s)
		}
	}
}

func dumpBenchConfigFile(path string, benchConf *benchConfig) error {
	bts, err := json.Marshal(benchConf)
	if err != nil {
		return err
	}

	return os.WriteFile(defBenchConf, bts, 0644)
}

// exec config procedure
func init() {
	loadEnvVariables()
	var err error
	gBenchConf, err = loadBenchConfigFile(defBenchConf)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if len(gTasks) != 0 {
		mergeTasks(&gBenchConf.Tasks, gTasks)
	}
}
