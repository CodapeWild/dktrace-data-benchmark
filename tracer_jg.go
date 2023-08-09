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
	"context"
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport"
)

var (
	_ Tracer = (*JgTracerWrapper)(nil)
	_ Span   = (*JgSpanWrapper)(nil)
)

type JgSpanCtxKey struct{}

type JgTracerWrapper struct {
	tracer opentracing.Tracer
	closer io.Closer
}

func (jgt *JgTracerWrapper) Start(agentAddress, service string) {
	trans := transport.NewHTTPTransport(fmt.Sprintf("http://%s/apis/traces", agentAddress))
	reporter := jaeger.NewRemoteReporter(trans)
	var tracer opentracing.Tracer
	tracer, jgt.closer = jaeger.NewTracer(service, jaeger.NewConstSampler(true), reporter)
	opentracing.SetGlobalTracer(tracer)
	jgt.tracer = tracer
}

func (jgt *JgTracerWrapper) StartSpan(ctx context.Context) (Span, context.Context) {
	n := ctx.Value(ctxNodeInfoKey{}).(*node)
	operation := "unknow-operation"
	if n != nil {
		operation = n.action
	}

	opctx, _ := ctx.Value(JgSpanCtxKey{}).(opentracing.SpanContext)
	span := jgt.tracer.StartSpan(operation, opentracing.ChildOf(opctx))
	ctx = context.WithValue(context.Background(), JgSpanCtxKey{}, span.Context())

	return &JgSpanWrapper{span}, ctx
}

func (jgt *JgTracerWrapper) Stop() {
	jgt.closer.Close()
}

type JgSpanWrapper struct {
	opentracing.Span
}

func (jgs *JgSpanWrapper) SetTag(key string, value interface{}) {
	jgs.Span.SetTag(key, value)
}

func (jgs *JgSpanWrapper) EndSpan() {
	jgs.Span.Finish()
}
