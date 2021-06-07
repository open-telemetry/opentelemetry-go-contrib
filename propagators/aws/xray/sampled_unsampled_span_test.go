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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("sample-app")

func startAndEndSampledSpan() {
	var span trace.Span
	_, span = tracer.Start(
		context.Background(),
		"Example Trace",
	)

	defer span.End()
}

func startAndEndNestedSampledSpan() {
	var span trace.Span
	ctx, span := tracer.Start(context.Background(), "Parent operation...")
	defer span.End()

	_, span = tracer.Start(ctx, "Sub operation...")
	defer span.End()
}

func getCurrentSampledSpan() trace.Span {
	var span trace.Span
	ctx, span := tracer.Start(
		context.Background(),
		"Example Trace",
	)
	defer span.End()

	return trace.SpanFromContext(ctx)
}

func addAttributesToSampledSpan() {
	var span trace.Span
	_, span = tracer.Start(
		context.Background(),
		"Example Trace",
	)
	defer span.End()

	span.SetAttributes(attribute.Key("example attribute 1").String("value 1"))
	span.SetAttributes(attribute.Key("example attribute 2").String("value 2"))
}

func startAndEndUnSampledSpan() {
	var span trace.Span
	_, span = tracer.Start(
		context.Background(),
		"Example Trace",
	)

	defer span.End()
}

func startAndEndNestedUnSampledSpan() {
	var span trace.Span
	ctx, span := tracer.Start(context.Background(), "Parent operation...")
	defer span.End()

	_, span = tracer.Start(ctx, "Sub operation...")
	defer span.End()
}

func getCurrentUnSampledSpan() trace.Span {
	var span trace.Span
	ctx, span := tracer.Start(
		context.Background(),
		"Example Trace",
	)
	defer span.End()

	return trace.SpanFromContext(ctx)
}

func addAttributesToUnSampledSpan() {
	var span trace.Span
	_, span = tracer.Start(
		context.Background(),
		"Example Trace",
	)
	defer span.End()

	span.SetAttributes(attribute.Key("example attribute 1").String("value 1"))
	span.SetAttributes(attribute.Key("example attribute 2").String("value 2"))
}

func initialization() {
	idg := NewIDGenerator()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithIDGenerator(idg),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(Propagator{})
}

func BenchmarkStartAndEndSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		startAndEndSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkStartAndEndNestedSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		startAndEndNestedSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkGetCurrentSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		getCurrentSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkAddAttributesToSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		addAttributesToSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkStartAndEndUnSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		startAndEndUnSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkStartAndEndNestedUnSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		startAndEndNestedUnSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkGetCurrentUnSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		getCurrentUnSampledSpan()
	}
	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}

func BenchmarkAddAttributesToUnSampledSpan(b *testing.B) {
	currentTraceProvider := otel.GetTracerProvider()
	currentPropagator := otel.GetTextMapPropagator()

	initialization()
	for i := 0; i < b.N; i++ {
		addAttributesToUnSampledSpan()
	}

	otel.SetTracerProvider(currentTraceProvider)
	otel.SetTextMapPropagator(currentPropagator)
}
