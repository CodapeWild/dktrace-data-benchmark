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

	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var (
	_ Tracer = (*DDTracerWrapper)(nil)
	_ Span   = (*DDSpanWrapper)(nil)
)

type DDTracerWrapper struct{}

func (ddt *DDTracerWrapper) Start(agentAddress, service string) {
	ddtracer.Start(ddtracer.WithAgentAddr(agentAddress), ddtracer.WithService(service), ddtracer.WithDebugMode(true), ddtracer.WithLogStartup(true))
}

func (ddt *DDTracerWrapper) StartSpan(ctx context.Context) (Span, context.Context) {
	n := ctx.Value(ctxNodeInfoKey{}).(*node)
	operation := "unknow-operation"
	if n != nil {
		operation = n.action
	}

	span, ctx := ddtracer.StartSpanFromContext(ctx, operation)

	return &DDSpanWrapper{span}, ctx
}

func (ddt *DDTracerWrapper) Stop() {
	ddtracer.Stop()
}

type DDSpanWrapper struct {
	ddtracer.Span
}

func (dds *DDSpanWrapper) SetTag(key string, value interface{}) {
	dds.Span.SetTag(key, value)
}

func (dds *DDSpanWrapper) EndSpan() {
	dds.Span.Finish()
}
