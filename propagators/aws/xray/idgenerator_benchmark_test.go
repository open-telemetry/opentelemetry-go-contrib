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

package xray

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	idg := NewIDGenerator()

	tracer = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithIDGenerator(idg),
	).Tracer("sample-app")
}

func BenchmarkStartAndEndSampledSpan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(context.Background(), "Example Trace")
		span.End()
	}
}

func BenchmarkStartAndEndNestedSampledSpan(b *testing.B) {
	ctx, parent := tracer.Start(context.Background(), "Parent operation...")
	defer parent.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "Sub operation...")
		span.End()
	}
}
