// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spaneventbridge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/log/logtest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type recordedEvent struct {
	name string
	cfg  trace.EventConfig
}

type spanStub struct {
	embedded.Span

	sc        trace.SpanContext
	recording bool
	events    []recordedEvent
}

func (*spanStub) End(...trace.SpanEndOption) {}

func (s *spanStub) AddEvent(name string, options ...trace.EventOption) {
	s.events = append(s.events, recordedEvent{name: name, cfg: trace.NewEventConfig(options...)})
}

func (*spanStub) AddLink(trace.Link) {}

func (s *spanStub) IsRecording() bool { return s.recording }

func (*spanStub) RecordError(error, ...trace.EventOption) {}

func (s *spanStub) SpanContext() trace.SpanContext { return s.sc }

func (*spanStub) SetStatus(codes.Code, string) {}

func (*spanStub) SetName(string) {}

func (*spanStub) SetAttributes(...attribute.KeyValue) {}

func (*spanStub) TracerProvider() trace.TracerProvider { return nil }

func TestLogProcessorEnabled(t *testing.T) {
	p := NewLogProcessor()

	recording := &spanStub{recording: true}
	assert.True(t, p.Enabled(trace.ContextWithSpan(t.Context(), recording), sdkEnabledParams()))

	notRecording := &spanStub{recording: false}
	assert.False(t, p.Enabled(trace.ContextWithSpan(t.Context(), notRecording), sdkEnabledParams()))
}

func TestLogProcessorOnEmit(t *testing.T) {
	ts := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	observed := ts.Add(10 * time.Millisecond)
	sc := newSpanContext(2)
	span := &spanStub{sc: sc, recording: true}

	record := logtest.RecordFactory{
		EventName:         "cache.miss",
		Timestamp:         ts,
		ObservedTimestamp: observed,
		Severity:          log.SeverityInfo2,
		Body: log.MapValue(
			log.String("item", "widget"),
			log.Int("retry", 2),
		),
		Attributes: []log.KeyValue{
			log.String("message", "cache lookup missed"),
			log.String("cache", "users"),
			log.Slice("tags", log.StringValue("hot"), log.StringValue("read")),
			log.Map("detail", log.String("state", "miss")),
		},
		TraceID:           sc.TraceID(),
		SpanID:            sc.SpanID(),
		TraceFlags:        sc.TraceFlags(),
		DroppedAttributes: 2,
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	require.Len(t, span.events, 1)

	evt := span.events[0]
	assert.Equal(t, "cache.miss", evt.name)
	assert.Equal(t, ts, evt.cfg.Timestamp())
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("message", "cache lookup missed"),
		attribute.String("cache", "users"),
		attribute.StringSlice("tags", []string{"hot", "read"}),
		attribute.String("detail", `{"state":"miss"}`),
	}, evt.cfg.Attributes())
}

func TestLogProcessorOnEmitUsesObservedTimestampWhenTimestampUnset(t *testing.T) {
	observed := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	sc := newSpanContext(2)
	span := &spanStub{sc: sc, recording: true}

	record := logtest.RecordFactory{
		EventName:         "cache.miss",
		ObservedTimestamp: observed,
		TraceID:           sc.TraceID(),
		SpanID:            sc.SpanID(),
		TraceFlags:        sc.TraceFlags(),
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	require.Len(t, span.events, 1)

	assert.Equal(t, observed, span.events[0].cfg.Timestamp())
}

func TestLogProcessorOnEmitPrefersTimestampOverObservedTimestamp(t *testing.T) {
	ts := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	observed := ts.Add(10 * time.Millisecond)
	sc := newSpanContext(2)
	span := &spanStub{sc: sc, recording: true}

	record := logtest.RecordFactory{
		EventName:         "cache.miss",
		Timestamp:         ts,
		ObservedTimestamp: observed,
		TraceID:           sc.TraceID(),
		SpanID:            sc.SpanID(),
		TraceFlags:        sc.TraceFlags(),
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	require.Len(t, span.events, 1)

	assert.Equal(t, ts, span.events[0].cfg.Timestamp())
}

func TestLogProcessorSkipsNonEventRecords(t *testing.T) {
	sc := newSpanContext(2)
	span := &spanStub{sc: sc, recording: true}
	record := logtest.RecordFactory{
		TraceID: sc.TraceID(),
		SpanID:  sc.SpanID(),
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	assert.Empty(t, span.events)
}

func TestLogProcessorSkipsMismatchedSpan(t *testing.T) {
	spanCtx := newSpanContext(2)
	recordCtx := newSpanContext(3)
	span := &spanStub{sc: spanCtx, recording: true}
	record := logtest.RecordFactory{
		EventName: "cache.miss",
		TraceID:   recordCtx.TraceID(),
		SpanID:    recordCtx.SpanID(),
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	assert.Empty(t, span.events)
}

func TestLogProcessorSkipsNonRecordingSpan(t *testing.T) {
	sc := newSpanContext(2)
	span := &spanStub{sc: sc, recording: false}
	record := logtest.RecordFactory{
		EventName: "cache.miss",
		TraceID:   sc.TraceID(),
		SpanID:    sc.SpanID(),
	}.NewRecord()

	err := NewLogProcessor().OnEmit(trace.ContextWithSpan(t.Context(), span), &record)
	require.NoError(t, err)
	assert.Empty(t, span.events)
}

func TestZeroLogProcessorNoPanic(t *testing.T) {
	var p LogProcessor
	assert.NotPanics(t, func() {
		assert.False(t, p.Enabled(t.Context(), sdkEnabledParams()))
		assert.NoError(t, p.OnEmit(t.Context(), nil))
		assert.NoError(t, p.ForceFlush(t.Context()))
		assert.NoError(t, p.Shutdown(t.Context()))
	})
}

func sdkEnabledParams() sdklog.EnabledParameters {
	return sdklog.EnabledParameters{EventName: "cache.miss"}
}

func newSpanContext(spanID byte) trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID([16]byte{1}),
		SpanID:     trace.SpanID([8]byte{spanID}),
		TraceFlags: trace.FlagsSampled,
	})
}
