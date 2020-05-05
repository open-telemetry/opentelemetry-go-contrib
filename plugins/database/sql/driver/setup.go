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

package driver

import (
	"context"

	otelcore "go.opentelemetry.io/otel/api/core"
	otelglobal "go.opentelemetry.io/otel/api/global"
	otelkey "go.opentelemetry.io/otel/api/key"
	oteltrace "go.opentelemetry.io/otel/api/trace"
)

const (
	tracerName = "go.opentelemetry.io/contrib/plugins/database/sql/driver"
)

type tracingSetup struct {
	tracer oteltrace.Tracer
	attrs  []otelcore.KeyValue
}

func (s *tracingSetup) Start(ctx context.Context, name, statement string) (context.Context, oteltrace.Span) {
	attrs := s.attrs
	if statement != "" {
		attrs = dupAttrsWithExtraCap(s.attrs, 1)
		attrs = append(attrs, otelkey.String("db.statement", statement))
	}
	return s.tracer.Start(
		ctx,
		name,
		oteltrace.WithAttributes(attrs...),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)
}

func (s *tracingSetup) StartNoStmt(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	return s.Start(ctx, name, "")
}

func (s *tracingSetup) StartNoCtx(name, statement string) (context.Context, oteltrace.Span) {
	return s.Start(context.Background(), name, statement)
}

func (s *tracingSetup) StartNoCtxNoStmt(name string) (context.Context, oteltrace.Span) {
	return s.StartNoCtx(name, "")
}

func setupFromConfig(cfg *Config) *tracingSetup {
	tracer := cfg.Tracer
	if tracer == nil {
		tracer = otelglobal.Tracer(tracerName)
	}
	return &tracingSetup{
		tracer: tracer,
		attrs: []otelcore.KeyValue{
			otelkey.String("db.type", "sql"),
		},
	}
}

func (s *tracingSetup) setupWithExtraAttrs(attrs ...otelcore.KeyValue) *tracingSetup {
	if len(attrs) == 0 {
		return s
	}
	dup := dupAttrsWithExtraCap(s.attrs, len(attrs))
	dup = append(dup, attrs...)
	return &tracingSetup{
		tracer: s.tracer,
		attrs:  dup,
	}
}

func dupAttrsWithExtraCap(attrs []otelcore.KeyValue, extraCap int) []otelcore.KeyValue {
	if extraCap < 0 {
		extraCap = 0
	}
	dup := make([]otelcore.KeyValue, len(attrs), len(attrs)+extraCap)
	copy(dup, attrs)
	return dup
}
