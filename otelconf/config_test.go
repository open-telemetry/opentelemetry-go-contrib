// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestUnmarshalPushMetricExporterInvalidData(t *testing.T) {
	cl := PushMetricExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&PushMetricExporter{}))

	cl = PushMetricExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = PushMetricExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&PushMetricExporter{}))
}

func TestUnmarshalLogRecordExporterInvalidData(t *testing.T) {
	cl := LogRecordExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&LogRecordExporter{}))

	cl = LogRecordExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = LogRecordExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&LogRecordExporter{}))
}

func TestUnmarshalSpanExporterInvalidData(t *testing.T) {
	cl := SpanExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&SpanExporter{}))

	cl = SpanExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal(&ConsoleExporter{}))

	cl = SpanExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal(&SpanExporter{}))
}

func TestUnmarshalTextMapPropagator(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		yamlConfig            []byte
		jsonConfig            []byte
		wantErrT              error
		wantTextMapPropagator TextMapPropagator
	}{
		{
			name:                  "valid with b3 propagator",
			jsonConfig:            []byte(`{"b3":{}}`),
			yamlConfig:            []byte("b3: {}\n"),
			wantTextMapPropagator: TextMapPropagator{B3: B3Propagator{}},
		},
		{
			name:       "valid with all propagators",
			jsonConfig: []byte(`{"b3":{},"b3multi":{},"baggage":{},"jaeger":{},"ottrace":{},"tracecontext":{}}`),
			yamlConfig: []byte("b3: {}\nb3multi: {}\nbaggage: {}\njaeger: {}\nottrace: {}\ntracecontext: {}\n"),
			wantTextMapPropagator: TextMapPropagator{
				B3:           B3Propagator{},
				B3Multi:      B3MultiPropagator{},
				Baggage:      BaggagePropagator{},
				Jaeger:       JaegerPropagator{},
				Ottrace:      OpenTracingPropagator{},
				Tracecontext: TraceContextPropagator{},
			},
		},
		{
			name:       "valid with all propagators nil",
			jsonConfig: []byte(`{"b3":null,"b3multi":null,"baggage":null,"jaeger":null,"ottrace":null,"tracecontext":null}`),
			yamlConfig: []byte("b3:\nb3multi:\nbaggage:\njaeger:\nottrace:\ntracecontext:\n"),
			wantTextMapPropagator: TextMapPropagator{
				B3:           B3Propagator{},
				B3Multi:      B3MultiPropagator{},
				Baggage:      BaggagePropagator{},
				Jaeger:       JaegerPropagator{},
				Ottrace:      OpenTracingPropagator{},
				Tracecontext: TraceContextPropagator{},
			},
		},
		{
			name:       "invalid b3 data",
			jsonConfig: []byte(`{"b3":2000}`),
			yamlConfig: []byte("b3: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid b3multi data",
			jsonConfig: []byte(`{"b3multi":2000}`),
			yamlConfig: []byte("b3multi: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid baggage data",
			jsonConfig: []byte(`{"baggage":2000}`),
			yamlConfig: []byte("baggage: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid jaeger data",
			jsonConfig: []byte(`{"jaeger":2000}`),
			yamlConfig: []byte("jaeger: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid ottrace data",
			jsonConfig: []byte(`{"ottrace":2000}`),
			yamlConfig: []byte("ottrace: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
		{
			name:       "invalid tracecontext data",
			jsonConfig: []byte(`{"tracecontext":2000}`),
			yamlConfig: []byte("tracecontext: !!str str"),
			wantErrT:   newErrUnmarshal(&TextMapPropagator{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := TextMapPropagator{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantTextMapPropagator, cl)

			cl = TextMapPropagator{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantTextMapPropagator, cl)
		})
	}
}

func TestUnmarshalSimpleLogRecordProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter LogRecordExporter
	}{
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&SimpleLogRecordProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: []"),
			wantErrT:   newErrUnmarshal(&SimpleLogRecordProcessor{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SimpleLogRecordProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = SimpleLogRecordProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalSimpleSpanProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter SpanExporter
	}{
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&SimpleSpanProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: []"),
			wantErrT:   newErrUnmarshal(&SimpleSpanProcessor{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SimpleSpanProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = SimpleSpanProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalBatchLogRecordProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter LogRecordExporter
	}{
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":5000,"max_export_batch_size":512,"max_queue_size":2048,"schedule_delay":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 5000\nmax_export_batch_size: 512\nmax_queue_size: 2048\nschedule_delay: 1000"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero export_timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 0"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero schedule_delay",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"schedule_delay":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nschedule_delay: 0"),
			wantExporter: LogRecordExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&BatchLogRecordProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   newErrUnmarshal(&BatchLogRecordProcessor{}),
		},
		{
			name:       "invalid export_timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"export_timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name:       "invalid max_export_batch_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_export_batch_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_queue_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid max_queue_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid schedule_delay negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"schedule_delay":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nschedule_delay: -1"),
			wantErrT:   newErrGreaterOrEqualZero("schedule_delay"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := BatchLogRecordProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = BatchLogRecordProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalBatchSpanProcessor(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter SpanExporter
	}{
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":5000,"max_export_batch_size":512,"max_queue_size":2048,"schedule_delay":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 5000\nmax_export_batch_size: 512\nmax_queue_size: 2048\nschedule_delay: 1000"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero export_timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"export_timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nexport_timeout: 0"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero schedule_delay",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"schedule_delay":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\nschedule_delay: 0"),
			wantExporter: SpanExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&BatchSpanProcessor{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   newErrUnmarshal(&BatchSpanProcessor{}),
		},
		{
			name:       "invalid export_timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"export_timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("export_timeout"),
		},
		{
			name:       "invalid max_export_batch_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_export_batch_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_export_batch_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_export_batch_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_export_batch_size"),
		},
		{
			name:       "invalid max_queue_size zero",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":0}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: 0"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid max_queue_size negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"max_queue_size":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nmax_queue_size: -1"),
			wantErrT:   newErrGreaterThanZero("max_queue_size"),
		},
		{
			name:       "invalid schedule_delay negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"schedule_delay":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\nschedule_delay: -1"),
			wantErrT:   newErrGreaterOrEqualZero("schedule_delay"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := BatchSpanProcessor{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)

			cl = BatchSpanProcessor{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl.Exporter)
		})
	}
}

func TestUnmarshalPeriodicMetricReader(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter PushMetricExporter
	}{
		{
			name:         "valid with null console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":null}}`),
			yamlConfig:   []byte("exporter:\n  console:\n"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with console exporter",
			jsonConfig:   []byte(`{"exporter":{"console":{}}}`),
			yamlConfig:   []byte("exporter:\n  console: {}"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with all fields positive",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"timeout":5000,"interval":1000}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ntimeout: 5000\ninterval: 1000"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"timeout":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ntimeout: 0"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:         "valid with zero interval",
			jsonConfig:   []byte(`{"exporter":{"console":{}},"interval":0}`),
			yamlConfig:   []byte("exporter:\n  console: {}\ninterval: 0"),
			wantExporter: PushMetricExporter{Console: ConsoleExporter{}},
		},
		{
			name:       "missing required exporter field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&PeriodicMetricReader{}, "exporter"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&PeriodicMetricReader{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"timeout":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
		{
			name:       "invalid interval negative",
			jsonConfig: []byte(`{"exporter":{"console":{}},"interval":-1}`),
			yamlConfig: []byte("exporter:\n  console: {}\ninterval: -1"),
			wantErrT:   newErrGreaterOrEqualZero("interval"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pmr := PeriodicMetricReader{}
			err := pmr.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, pmr.Exporter)

			pmr = PeriodicMetricReader{}
			err = yaml.Unmarshal(tt.yamlConfig, &pmr)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, pmr.Exporter)
		})
	}
}

func TestUnmarshalCardinalityLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErrT   error
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"counter":100,"default":200,"gauge":300,"histogram":400,"observable_counter":500,"observable_gauge":600,"observable_up_down_counter":700,"up_down_counter":800}`),
			yamlConfig: []byte("counter: 100\ndefault: 200\ngauge: 300\nhistogram: 400\nobservable_counter: 500\nobservable_gauge: 600\nobservable_up_down_counter: 700\nup_down_counter: 800"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"default":2000}`),
			yamlConfig: []byte("default: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("counter: !!str 2000"),
			wantErrT:   newErrUnmarshal(&CardinalityLimits{}),
		},
		{
			name:       "invalid counter zero",
			jsonConfig: []byte(`{"counter":0}`),
			yamlConfig: []byte("counter: 0"),
			wantErrT:   newErrGreaterThanZero("counter"),
		},
		{
			name:       "invalid counter negative",
			jsonConfig: []byte(`{"counter":-1}`),
			yamlConfig: []byte("counter: -1"),
			wantErrT:   newErrGreaterThanZero("counter"),
		},
		{
			name:       "invalid default zero",
			jsonConfig: []byte(`{"default":0}`),
			yamlConfig: []byte("default: 0"),
			wantErrT:   newErrGreaterThanZero("default"),
		},
		{
			name:       "invalid default negative",
			jsonConfig: []byte(`{"default":-1}`),
			yamlConfig: []byte("default: -1"),
			wantErrT:   newErrGreaterThanZero("default"),
		},
		{
			name:       "invalid gauge zero",
			jsonConfig: []byte(`{"gauge":0}`),
			yamlConfig: []byte("gauge: 0"),
			wantErrT:   newErrGreaterThanZero("gauge"),
		},
		{
			name:       "invalid gauge negative",
			jsonConfig: []byte(`{"gauge":-1}`),
			yamlConfig: []byte("gauge: -1"),
			wantErrT:   newErrGreaterThanZero("gauge"),
		},
		{
			name:       "invalid histogram zero",
			jsonConfig: []byte(`{"histogram":0}`),
			yamlConfig: []byte("histogram: 0"),
			wantErrT:   newErrGreaterThanZero("histogram"),
		},
		{
			name:       "invalid histogram negative",
			jsonConfig: []byte(`{"histogram":-1}`),
			yamlConfig: []byte("histogram: -1"),
			wantErrT:   newErrGreaterThanZero("histogram"),
		},
		{
			name:       "invalid observable_counter zero",
			jsonConfig: []byte(`{"observable_counter":0}`),
			yamlConfig: []byte("observable_counter: 0"),
			wantErrT:   newErrGreaterThanZero("observable_counter"),
		},
		{
			name:       "invalid observable_counter negative",
			jsonConfig: []byte(`{"observable_counter":-1}`),
			yamlConfig: []byte("observable_counter: -1"),
			wantErrT:   newErrGreaterThanZero("observable_counter"),
		},
		{
			name:       "invalid observable_gauge zero",
			jsonConfig: []byte(`{"observable_gauge":0}`),
			yamlConfig: []byte("observable_gauge: 0"),
			wantErrT:   newErrGreaterThanZero("observable_gauge"),
		},
		{
			name:       "invalid observable_gauge negative",
			jsonConfig: []byte(`{"observable_gauge":-1}`),
			yamlConfig: []byte("observable_gauge: -1"),
			wantErrT:   newErrGreaterThanZero("observable_gauge"),
		},
		{
			name:       "invalid observable_up_down_counter zero",
			jsonConfig: []byte(`{"observable_up_down_counter":0}`),
			yamlConfig: []byte("observable_up_down_counter: 0"),
			wantErrT:   newErrGreaterThanZero("observable_up_down_counter"),
		},
		{
			name:       "invalid observable_up_down_counter negative",
			jsonConfig: []byte(`{"observable_up_down_counter":-1}`),
			yamlConfig: []byte("observable_up_down_counter: -1"),
			wantErrT:   newErrGreaterThanZero("observable_up_down_counter"),
		},
		{
			name:       "invalid up_down_counter zero",
			jsonConfig: []byte(`{"up_down_counter":0}`),
			yamlConfig: []byte("up_down_counter: 0"),
			wantErrT:   newErrGreaterThanZero("up_down_counter"),
		},
		{
			name:       "invalid up_down_counter negative",
			jsonConfig: []byte(`{"up_down_counter":-1}`),
			yamlConfig: []byte("up_down_counter: -1"),
			wantErrT:   newErrGreaterThanZero("up_down_counter"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := CardinalityLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)

			cl = CardinalityLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
		})
	}
}

func TestCreateHeadersConfig(t *testing.T) {
	tests := []struct {
		name        string
		headers     []NameStringValuePair
		headersList *string
		wantHeaders map[string]string
		wantErr     error
	}{
		{
			name:        "no headers",
			headers:     []NameStringValuePair{},
			headersList: nil,
			wantHeaders: map[string]string{},
		},
		{
			name:        "headerslist only",
			headers:     []NameStringValuePair{},
			headersList: ptr("a=b,c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "headers only",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
				{
					Name:  "c",
					Value: ptr("d"),
				},
			},
			headersList: nil,
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "both headers and headerslist",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
			},
			headersList: ptr("c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name: "headers supersedes headerslist",
			headers: []NameStringValuePair{
				{
					Name:  "a",
					Value: ptr("b"),
				},
				{
					Name:  "c",
					Value: ptr("override"),
				},
			},
			headersList: ptr("c=d"),
			wantHeaders: map[string]string{
				"a": "b",
				"c": "override",
			},
		},
		{
			name:        "invalid headerslist",
			headersList: ptr("==="),
			wantErr:     newErrInvalid("invalid headers_list"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headersMap, err := createHeadersConfig(tt.headers, tt.headersList)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantHeaders, headersMap)
		})
	}
}

func TestUnmarshalSpanLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErrT   error
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"attribute_count_limit":100,"attribute_value_length_limit":200,"event_attribute_count_limit":300,"event_count_limit":400,"link_attribute_count_limit":500,"link_count_limit":600}`),
			yamlConfig: []byte("attribute_count_limit: 100\nattribute_value_length_limit: 200\nevent_attribute_count_limit: 300\nevent_count_limit: 400\nlink_attribute_count_limit: 500\nlink_count_limit: 600"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"attribute_value_length_limit":2000}`),
			yamlConfig: []byte("attribute_value_length_limit: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("attribute_count_limit: !!str 2000"),
			wantErrT:   newErrUnmarshal(&SpanLimits{}),
		},
		{
			name:       "invalid attribute_count_limit negative",
			jsonConfig: []byte(`{"attribute_count_limit":-1}`),
			yamlConfig: []byte("attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("attribute_count_limit"),
		},
		{
			name:       "invalid attribute_value_length_limit negative",
			jsonConfig: []byte(`{"attribute_value_length_limit":-1}`),
			yamlConfig: []byte("attribute_value_length_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("attribute_value_length_limit"),
		},
		{
			name:       "invalid event_attribute_count_limit negative",
			jsonConfig: []byte(`{"event_attribute_count_limit":-1}`),
			yamlConfig: []byte("event_attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("event_attribute_count_limit"),
		},
		{
			name:       "invalid event_count_limit negative",
			jsonConfig: []byte(`{"event_count_limit":-1}`),
			yamlConfig: []byte("event_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("event_count_limit"),
		},
		{
			name:       "invalid link_attribute_count_limit negative",
			jsonConfig: []byte(`{"link_attribute_count_limit":-1}`),
			yamlConfig: []byte("link_attribute_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("link_attribute_count_limit"),
		},
		{
			name:       "invalid link_count_limit negative",
			jsonConfig: []byte(`{"link_count_limit":-1}`),
			yamlConfig: []byte("link_count_limit: -1"),
			wantErrT:   newErrGreaterOrEqualZero("link_count_limit"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SpanLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)

			cl = SpanLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
		})
	}
}

func TestUnmarshalOTLPHttpExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPHttpExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPHttpExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPHttpExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPHttpExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPHttpExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPHttpExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPHttpExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPGrpcExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPGrpcExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPGrpcExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPGrpcExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPGrpcExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPGrpcExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPGrpcExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPGrpcExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPHttpMetricExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPHttpMetricExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPHttpMetricExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPHttpMetricExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPHttpMetricExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPHttpMetricExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPHttpMetricExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPHttpMetricExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalOTLPGrpcMetricExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter OTLPGrpcMetricExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318"}`),
			yamlConfig:   []byte("endpoint: localhost:4318\n"),
			wantExporter: OTLPGrpcMetricExporter{Endpoint: ptr("localhost:4318")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&OTLPGrpcMetricExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:4318", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:4318\ntimeout: 0"),
			wantExporter: OTLPGrpcMetricExporter{Endpoint: ptr("localhost:4318"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&OTLPGrpcMetricExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:4318", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:4318\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := OTLPGrpcMetricExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = OTLPGrpcMetricExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalZipkinSpanExporter(t *testing.T) {
	for _, tt := range []struct {
		name         string
		yamlConfig   []byte
		jsonConfig   []byte
		wantErrT     error
		wantExporter ZipkinSpanExporter
	}{
		{
			name:         "valid with exporter",
			jsonConfig:   []byte(`{"endpoint":"localhost:9000"}`),
			yamlConfig:   []byte("endpoint: localhost:9000\n"),
			wantExporter: ZipkinSpanExporter{Endpoint: ptr("localhost:9000")},
		},
		{
			name:       "missing required endpoint field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&ZipkinSpanExporter{}, "endpoint"),
		},
		{
			name:         "valid with zero timeout",
			jsonConfig:   []byte(`{"endpoint":"localhost:9000", "timeout":0}`),
			yamlConfig:   []byte("endpoint: localhost:9000\ntimeout: 0"),
			wantExporter: ZipkinSpanExporter{Endpoint: ptr("localhost:9000"), Timeout: ptr(0)},
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("endpoint: localhost:9000\ntimeout: !!str str"),
			wantErrT:   newErrUnmarshal(&ZipkinSpanExporter{}),
		},
		{
			name:       "invalid timeout negative",
			jsonConfig: []byte(`{"endpoint":"localhost:9000", "timeout":-1}`),
			yamlConfig: []byte("endpoint: localhost:9000\ntimeout: -1"),
			wantErrT:   newErrGreaterOrEqualZero("timeout"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := ZipkinSpanExporter{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)

			cl = ZipkinSpanExporter{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantExporter, cl)
		})
	}
}

func TestUnmarshalAttributeNameValueType(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		yamlConfig             []byte
		jsonConfig             []byte
		wantErrT               error
		wantAttributeNameValue AttributeNameValue
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&AttributeNameValue{}),
		},
		{
			name:       "missing required name field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&AttributeNameValue{}, "name"),
		},
		{
			name:       "missing required value field",
			jsonConfig: []byte(`{"name":"test"}`),
			yamlConfig: []byte("name: test"),
			wantErrT:   newErrRequired(&AttributeNameValue{}, "value"),
		},
		{
			name:       "valid string value",
			jsonConfig: []byte(`{"name":"test", "value": "test-val", "type": "string"}`),
			yamlConfig: []byte("name: test\nvalue: test-val\ntype: string\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: "test-val",
				Type:  &AttributeType{Value: "string"},
			},
		},
		{
			name:       "valid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{"test-val", "test-val-2"},
				Type:  &AttributeType{Value: "string_array"},
			},
		},
		{
			name:       "valid bool value",
			jsonConfig: []byte(`{"name":"test", "value": true, "type": "bool"}`),
			yamlConfig: []byte("name: test\nvalue: true\ntype: bool\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: true,
				Type:  &AttributeType{Value: "bool"},
			},
		},
		{
			name:       "valid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{"test-val", "test-val-2"},
				Type:  &AttributeType{Value: "string_array"},
			},
		},
		{
			name:       "valid int value",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "int"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: int\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: int(1),
				Type:  &AttributeType{Value: "int"},
			},
		},
		{
			name:       "valid int_array value",
			jsonConfig: []byte(`{"name":"test", "value": [1, 2], "type": "int_array"}`),
			yamlConfig: []byte("name: test\nvalue: [1, 2]\ntype: int_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{1, 2},
				Type:  &AttributeType{Value: "int_array"},
			},
		},
		{
			name:       "valid double value",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "double"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: double\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: float64(1),
				Type:  &AttributeType{Value: "double"},
			},
		},
		{
			name:       "valid double_array value",
			jsonConfig: []byte(`{"name":"test", "value": [1, 2], "type": "double_array"}`),
			yamlConfig: []byte("name: test\nvalue: [1.0, 2.0]\ntype: double_array\n"),
			wantAttributeNameValue: AttributeNameValue{
				Name:  "test",
				Value: []any{float64(1), float64(2)},
				Type:  &AttributeType{Value: "double_array"},
			},
		},
		{
			name:       "invalid type",
			jsonConfig: []byte(`{"name":"test", "value": 1, "type": "float"}`),
			yamlConfig: []byte("name: test\nvalue: 1\ntype: float\n"),
			wantErrT:   newErrInvalid("unexpected value type"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := AttributeNameValue{}
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantAttributeNameValue, val)

			val = AttributeNameValue{}
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantAttributeNameValue, val)
		})
	}
}

func TestUnmarshalNameStringValuePairType(t *testing.T) {
	for _, tt := range []struct {
		name                    string
		yamlConfig              []byte
		jsonConfig              []byte
		wantErrT                error
		wantNameStringValuePair NameStringValuePair
	}{
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("name: []\nvalue: true\ntype: bool\n"),
			wantErrT:   newErrUnmarshal(&NameStringValuePair{}),
		},
		{
			name:       "missing required name field",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
			wantErrT:   newErrRequired(&NameStringValuePair{}, "name"),
		},
		{
			name:       "missing required value field",
			jsonConfig: []byte(`{"name":"test"}`),
			yamlConfig: []byte("name: test"),
			wantErrT:   newErrRequired(&NameStringValuePair{}, "value"),
		},
		{
			name:       "valid string value",
			jsonConfig: []byte(`{"name":"test", "value": "test-val", "type": "string"}`),
			yamlConfig: []byte("name: test\nvalue: test-val\ntype: string\n"),
			wantNameStringValuePair: NameStringValuePair{
				Name:  "test",
				Value: ptr("test-val"),
			},
		},
		{
			name:       "invalid string_array value",
			jsonConfig: []byte(`{"name":"test", "value": ["test-val", "test-val-2"], "type": "string_array"}`),
			yamlConfig: []byte("name: test\nvalue: [test-val, test-val-2]\ntype: string_array\n"),
			wantErrT:   newErrUnmarshal(&NameStringValuePair{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			val := NameStringValuePair{}
			err := val.UnmarshalJSON(tt.jsonConfig)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantNameStringValuePair, val)

			val = NameStringValuePair{}
			err = yaml.Unmarshal(tt.yamlConfig, &val)
			assert.ErrorIs(t, err, tt.wantErrT)
			assert.Equal(t, tt.wantNameStringValuePair, val)
		})
	}
}
