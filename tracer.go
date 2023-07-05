package main

import (
	"context"

	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Span interface {
	SetTag(key string, value interface{})
	EndSpan()
}

type Tracer interface {
	Start(agentAddress, service string)
	StartSpan(ctx context.Context) (Span, context.Context)
	Stop()
}

type ddtracerwrapper struct{}

func (dd *ddtracerwrapper) Start(agentAddress, service string) {
	ddtracer.Start(ddtracer.WithAgentAddr(agentAddress), ddtracer.WithService(service), ddtracer.WithDebugMode(true), ddtracer.WithLogStartup(true))
}

func (dd *ddtracerwrapper) StartSpan(ctx context.Context) (Span, context.Context) {
	n := ctx.Value(ctxNodeInfoKey{}).(*node)
	operation := "unknow-operation"
	if n != nil {
		operation = n.action
	}

	span, ctx := ddtracer.StartSpanFromContext(ctx, operation)

	return &ddspanwrapper{Span: span}, ctx
}

func (dd *ddtracerwrapper) Stop() {
	ddtracer.Stop()
}

type ddspanwrapper struct {
	ddtracer.Span
}

func (dd *ddspanwrapper) SetTag(key string, value interface{}) {
	dd.Span.SetTag(key, value)
}

func (dd *ddspanwrapper) EndSpan() {
	dd.Span.Finish()
}
