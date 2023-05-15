package main

import "flag"

var (
	tracer = flag.String("tracer", "ddtrace", "Tracer SDKs splited by comma if multiple tracers present that used to send trace data")
)
