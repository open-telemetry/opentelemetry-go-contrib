// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// MarshalJSON implements json.Marshaler.
func (j *AttributeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeType) UnmarshalJSON(b []byte) error {
	var v struct {
		Value any
	}
	if err := json.Unmarshal(b, &v.Value); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesAttributeType {
		if reflect.DeepEqual(v.Value, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesAttributeType, v.Value)
	}
	*j = AttributeType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return fmt.Errorf("field %s: must be >= %v", "export_timeout", 0)
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return fmt.Errorf("field %s: must be > %v", "max_export_batch_size", 0)
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return fmt.Errorf("field %s: must be > %v", "max_queue_size", 0)
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return fmt.Errorf("field %s: must be >= %v", "schedule_delay", 0)
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return fmt.Errorf("field %s: must be >= %v", "export_timeout", 0)
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return fmt.Errorf("field %s: must be > %v", "max_export_batch_size", 0)
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return fmt.Errorf("field %s: must be > %v", "max_queue_size", 0)
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return fmt.Errorf("field %s: must be >= %v", "schedule_delay", 0)
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalPeerInstrumentationServiceMappingElem) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["peer"]; raw != nil && !ok {
		return errors.New("field peer in ExperimentalPeerInstrumentationServiceMappingElem: required")
	}
	if _, ok := raw["service"]; raw != nil && !ok {
		return errors.New("field service in ExperimentalPeerInstrumentationServiceMappingElem: required")
	}
	type Plain ExperimentalPeerInstrumentationServiceMappingElem
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ExperimentalPeerInstrumentationServiceMappingElem(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *NameStringValuePair) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	name, err := validateStringField(raw, "name")
	if err != nil {
		return err
	}

	value, err := validateStringField(raw, "value")
	if err != nil {
		return err
	}

	*j = NameStringValuePair{
		Name:  name,
		Value: &value,
	}
	return nil
}

var enumValuesOTLPMetricDefaultHistogramAggregation = []any{
	"explicit_bucket_histogram",
	"base2_exponential_bucket_histogram",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExporterDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesOTLPMetricDefaultHistogramAggregation {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesOTLPMetricDefaultHistogramAggregation, v)
	}
	*j = ExporterDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPHttpMetricExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLPMetric: required")
	}
	type Plain OTLPHttpMetricExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = OTLPHttpMetricExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPGrpcMetricExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLPMetric: required")
	}
	type Plain OTLPGrpcMetricExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = OTLPGrpcMetricExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPHttpExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLP: required")
	}
	type Plain OTLPHttpExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = OTLPHttpExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPGrpcExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in OTLP: required")
	}
	type Plain OTLPGrpcExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = OTLPGrpcExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["file_format"]; raw != nil && !ok {
		return errors.New("field file_format in OpenTelemetryConfiguration: required")
	}
	type Plain OpenTelemetryConfiguration
	var plain Plain

	if v, ok := raw["logger_provider"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var lp LoggerProviderJson
		if err := json.Unmarshal(marshaled, &lp); err != nil {
			return err
		}
		plain.LoggerProvider = &lp
	}

	if v, ok := raw["meter_provider"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var mp MeterProviderJson
		if err := json.Unmarshal(marshaled, &mp); err != nil {
			return err
		}
		plain.MeterProvider = &mp
	}

	if v, ok := raw["tracer_provider"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var tp TracerProviderJson
		if err := json.Unmarshal(marshaled, &tp); err != nil {
			return err
		}
		plain.TracerProvider = &tp
	}

	if v, ok := raw["propagator"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var p PropagatorJson
		if err := json.Unmarshal(marshaled, &p); err != nil {
			return err
		}
		plain.Propagator = &p
	}

	if v, ok := raw["resource"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var r ResourceJson
		if err := json.Unmarshal(marshaled, &r); err != nil {
			return err
		}
		plain.Resource = &r
	}

	if v, ok := raw["instrumentation/development"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var i InstrumentationJson
		if err := json.Unmarshal(marshaled, &i); err != nil {
			return err
		}
		plain.InstrumentationDevelopment = &i
	}

	if v, ok := raw["attribute_limits"]; ok && v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return err
		}

		var a AttributeLimits
		if err := json.Unmarshal(marshaled, &a); err != nil {
			return err
		}
		plain.AttributeLimits = &a
	}

	plainConfig := (*OpenTelemetryConfiguration)(&plain)
	if err := setConfigDefaults(raw, plainConfig, jsonCodec{}); err != nil {
		return err
	}

	plain.FileFormat = fmt.Sprintf("%v", raw["file_format"])

	*j = OpenTelemetryConfiguration(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in PeriodicMetricReader: required")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Interval != nil && 0 > *plain.Interval {
		return fmt.Errorf("field %s: must be >= %v", "interval", 0)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in PullMetricReader: required")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PullMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in SimpleLogRecordProcessor: required")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in SimpleSpanProcessor: required")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

var enumValuesViewSelectorInstrumentType = []any{
	"counter",
	"gauge",
	"histogram",
	"observable_counter",
	"observable_gauge",
	"observable_up_down_counter",
	"up_down_counter",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesViewSelectorInstrumentType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesViewSelectorInstrumentType, v)
	}
	*j = InstrumentType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ZipkinSpanExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["endpoint"]; raw != nil && !ok {
		return errors.New("field endpoint in ZipkinSpanExporter: required")
	}
	type Plain ZipkinSpanExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return fmt.Errorf("field %s: must be >= %v", "timeout", 0)
	}
	*j = ZipkinSpanExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeNameValue) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return errors.New("field name in AttributeNameValue: required")
	}
	if _, ok := raw["value"]; raw != nil && !ok {
		return errors.New("field value in AttributeNameValue: required")
	}
	type Plain AttributeNameValue
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Type != nil && plain.Type.Value == "int" {
		val, ok := plain.Value.(float64)
		if ok {
			plain.Value = int(val)
		}
	}
	if plain.Type != nil && plain.Type.Value == "int_array" {
		m, ok := plain.Value.([]any)
		if ok {
			var vals []any
			for _, v := range m {
				val, ok := v.(float64)
				if ok {
					vals = append(vals, int(val))
				} else {
					vals = append(vals, val)
				}
			}
			plain.Value = vals
		}
	}

	*j = AttributeNameValue(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PushMetricExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain PushMetricExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = PushMetricExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain SpanExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = SpanExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LogRecordExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain LogRecordExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Sampler) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain Sampler
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	unmarshalSamplerTypes(raw, (*Sampler)(&plain))
	*j = Sampler(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *MetricProducer) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain MetricProducer
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	unmarshalMetricProducer(raw, (*MetricProducer)(&plain))
	*j = MetricProducer(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TextMapPropagator) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain TextMapPropagator
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	unmarshalTextMapPropagatorTypes(raw, (*TextMapPropagator)(&plain))
	*j = TextMapPropagator(plain)
	return nil
}
