// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

func TestUnmarshalPushMetricExporterInvalidData(t *testing.T) {
	cl := PushMetricExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("PushMetricExporter"))

	cl = PushMetricExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("ConsoleExporter"))

	cl = PushMetricExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal("PushMetricExporter"))
}

func TestUnmarshalLogRecordExporterInvalidData(t *testing.T) {
	cl := LogRecordExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("LogRecordExporter"))

	cl = LogRecordExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("ConsoleExporter"))

	cl = LogRecordExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal("LogRecordExporter"))
}

func TestUnmarshalSpanExporterInvalidData(t *testing.T) {
	cl := SpanExporter{}
	err := cl.UnmarshalJSON([]byte(`{:2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("SpanExporter"))

	cl = SpanExporter{}
	err = cl.UnmarshalJSON([]byte(`{"console":2000}`))
	assert.ErrorIs(t, err, newErrUnmarshal("ConsoleExporter"))

	cl = SpanExporter{}
	err = yaml.Unmarshal([]byte("console: !!str str"), &cl)
	assert.ErrorIs(t, err, newErrUnmarshal("SpanExporter"))
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
			wantErrT:   newErrRequiredExporter("BatchLogRecordProcessor"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   errUnmarshalingBatchLogRecordProcessor,
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
			wantErrT:   newErrRequiredExporter("BatchSpanProcessor"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\nexport_timeout: !!str str"),
			wantErrT:   errUnmarshalingBatchSpanProcessor,
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
			wantErrT:   newErrRequiredExporter("PeriodicMetricReader"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("exporter:\n  console: {}\ntimeout: !!str str"),
			wantErrT:   errUnmarshalingPeriodicMetricReader,
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
			wantErrT:   errUnmarshalingCardinalityLimits,
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
			wantErrT:   errUnmarshalingSpanLimits,
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
