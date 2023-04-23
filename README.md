# trace-data-demo

## how to

start a demo to send testing data

with application startup parameters

```shell
./tdd -test ddtrace,jeager,otel,pinpoint,skywalking,zipkin \
      -task ./tasks/def.json \
      -thread 100 -repeat 10 \
      -proto http -host 127.0.0.1 -port 9529 -path /v0.3/trace
```

with environment variables

```shell
export TDD_TEST=ddtrace,jeager,otel,pinpoint,skywalking,zipkin
export TDD_TASK=./tasks/def.json
export TDD_THREAD=100
export TDD_REPEAT=10
export TDD_PROTO=http
export TDD_HOST=127.0.0.1
export TDD_PORT=9529
export TDD_PATH=/v0.3/trace
./tdd
```