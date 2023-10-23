// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	_ "go.opentelemetry.io/otel"
	__atel_runtime "runtime"
	__atel_otel "go.opentelemetry.io/otel"
	__atel_context "context"
	_ "context"
)

type element struct {
}

type driver struct {
	e element
}

type i interface {
	anotherfoo(p int) int
}

type impl struct {
}

func (i impl) anotherfoo(p int) int {
	__atel_tracing_ctx := __atel_runtime.InstrgenGetTls().(__atel_context.Context)
	defer __atel_runtime.InstrgenSetTls(__atel_tracing_ctx)
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("anotherfoo").Start(__atel_tracing_ctx, "anotherfoo")
	__atel_runtime.InstrgenSetTls(__atel_child_tracing_ctx)
	defer __atel_span.End()

	return 5
}

func anotherfoo(p int) int {
	__atel_tracing_ctx := __atel_runtime.InstrgenGetTls().(__atel_context.Context)
	defer __atel_runtime.InstrgenSetTls(__atel_tracing_ctx)
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("anotherfoo").Start(__atel_tracing_ctx, "anotherfoo")
	__atel_runtime.InstrgenSetTls(__atel_child_tracing_ctx)
	defer __atel_span.End()

	return 1
}

func (d driver) process(a int) {
	__atel_tracing_ctx := __atel_runtime.InstrgenGetTls().(__atel_context.Context)
	defer __atel_runtime.InstrgenSetTls(__atel_tracing_ctx)
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("process").Start(__atel_tracing_ctx, "process")
	__atel_runtime.InstrgenSetTls(__atel_child_tracing_ctx)
	defer __atel_span.End()

}

func (e element) get(a int) {
	__atel_tracing_ctx := __atel_runtime.InstrgenGetTls().(__atel_context.Context)
	defer __atel_runtime.InstrgenSetTls(__atel_tracing_ctx)
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("get").Start(__atel_tracing_ctx, "get")
	__atel_runtime.InstrgenSetTls(__atel_child_tracing_ctx)
	defer __atel_span.End()

}

func methods() {
	__atel_tracing_ctx := __atel_runtime.InstrgenGetTls().(__atel_context.Context)
	defer __atel_runtime.InstrgenSetTls(__atel_tracing_ctx)
	__atel_child_tracing_ctx, __atel_span := __atel_otel.Tracer("methods").Start(__atel_tracing_ctx, "methods")
	__atel_runtime.InstrgenSetTls(__atel_child_tracing_ctx)
	defer __atel_span.End()

	d := driver{}
	d.process(10)
	d.e.get(5)
	var in i
	in = impl{}
	in.anotherfoo(10)
	anotherfoo(5)
}
