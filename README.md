# trace-data-demo

## how to

start a demo and send testing data to collector

with application startup parameters

> start demo with config file

```shell
./tdd -config ./config.json
```

> start demo with startup parameters

```shell
./tdd -tracer [ddtrace | jeager | otel | pinpoint | skywalking | zipkin]
      -task_config ./tasks/def.json
      -send_threads 10
      -send_times_per_thread 100
      -collector_proto http
      -collector_ip 127.0.0.1
      -collector_port 9529
      -collector_path /v0.4/traces
```

with environment variables

```shell
export TDD_TRACER=[ddtrace | jeager | otel | pinpoint | skywalking | zipkin]
export TDD_TASK_CONFIG=./tasks/def.json
export TDD_THREADS=10
export TDD_SEND_TIMES_PER_THREAD=100
export TDD_COLLECTOR_PROTO=http
export TDD_COLLECTOR_IP=127.0.0.1
export TDD_COLLECTOR_PORT=9529
export TDD_COLLECTOR_PATH=/v0.4/traces
```