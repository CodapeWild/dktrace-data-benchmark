package main

import (
	"context"

	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var globalTracer Tracer

type Span interface {
	SetTag(key string, value interface{})
	EndSpan()
}

type Tracer interface {
	StartSpan(ctx context.Context) (Span, context.Context)
}

type ddtracerwrapper struct {
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

type ddspanwrapper struct {
	ddtracer.Span
}

func (dd *ddspanwrapper) SetTag(key string, value interface{}) {
	dd.Span.SetTag(key, value)
}

func (dd *ddspanwrapper) EndSpan() {
	dd.Span.Finish()
}
