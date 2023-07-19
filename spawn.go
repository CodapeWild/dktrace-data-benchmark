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
