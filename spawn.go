package main

import (
	"context"
	"time"
)

type ctxTracerKey struct{}

type ctxNodeInfoKey struct{}

func (n *node) spawn(ctx context.Context, tracer Tracer) {
	start := time.Now().UnixNano()

	var span Span
	span, ctx = tracer.StartSpan(context.WithValue(ctx, ctxNodeInfoKey{}, n))
	defer func() {
		if time.Now().UnixNano()-start < int64(30*time.Millisecond) {
			time.Sleep(30 * time.Millisecond)
		}
		span.EndSpan()
	}()

	span.SetTag("id", n.id)
	span.SetTag("service", n.service)
	span.SetTag("name", n.name)
	span.SetTag("action", n.action)
	span.SetTag("status", n.status)
	span.SetTag("message", n.message)

	for _, c := range n.children {
		c.spawn(ctx, tracer)
	}
}
