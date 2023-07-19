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

func (ddwrap *ddtracerwrapper) Start(agentAddress, service string) {
	ddtracer.Start(ddtracer.WithAgentAddr(agentAddress), ddtracer.WithService(service), ddtracer.WithDebugMode(true), ddtracer.WithLogStartup(true))
}

func (ddwrap *ddtracerwrapper) StartSpan(ctx context.Context) (Span, context.Context) {
	n := ctx.Value(ctxNodeInfoKey{}).(*node)
	operation := "unknow-operation"
	if n != nil {
		operation = n.action
	}

	span, ctx := ddtracer.StartSpanFromContext(ctx, operation)

	return &ddspanwrapper{Span: span}, ctx
}

func (ddwrap *ddtracerwrapper) Stop() {
	ddtracer.Stop()
}

type ddspanwrapper struct {
	ddtracer.Span
}

func (ddwrap *ddspanwrapper) SetTag(key string, value interface{}) {
	ddwrap.Span.SetTag(key, value)
}

func (ddwrap *ddspanwrapper) EndSpan() {
	ddwrap.Span.Finish()
}
