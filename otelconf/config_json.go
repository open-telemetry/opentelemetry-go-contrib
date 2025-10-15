// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// UnmarshalJSON implements json.Unmarshaler.
func (j *ConsoleExporter) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = ConsoleExporter{}
	} else {
		*j = ConsoleExporter(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *B3Propagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = B3Propagator{}
	} else {
		*j = B3Propagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *B3MultiPropagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = B3MultiPropagator{}
	} else {
		*j = B3MultiPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BaggagePropagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = BaggagePropagator{}
	} else {
		*j = BaggagePropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JaegerPropagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = JaegerPropagator{}
	} else {
		*j = JaegerPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTracingPropagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = OpenTracingPropagator{}
	} else {
		*j = OpenTracingPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TraceContextPropagator) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// If key is present (even if empty object), ensure non-nil value.
	if p == nil {
		*j = TraceContextPropagator{}
	} else {
		*j = TraceContextPropagator(p)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PushMetricExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain PushMetricExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = PushMetricExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain SpanExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = SpanExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LogRecordExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain LogRecordExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = LogRecordExporter(sh.Plain)
}

// MarshalJSON implements json.Marshaler.
func (j *AttributeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Value)
}

var enumValuesAttributeType = []any{
	nil,
	"string",
	"bool",
	"int",
	"double",
	"string_array",
	"bool_array",
	"int_array",
	"double_array",
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
func (j *TextMapPropagator) UnmarshalJSON(b []byte) error {
	type Plain TextMapPropagator
	type shadow struct {
		Plain
		B3           json.RawMessage `json:"b3"`
		B3Multi      json.RawMessage `json:"b3multi"`
		Baggage      json.RawMessage `json:"baggage"`
		Jaeger       json.RawMessage `json:"jaeger"`
		Ottrace      json.RawMessage `json:"ottrace"`
		Tracecontext json.RawMessage `json:"tracecontext"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	if sh.B3 != nil {
		var p B3Propagator
		if err := json.Unmarshal(sh.B3, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.B3 = p
	}

	if sh.B3Multi != nil {
		var p B3MultiPropagator
		if err := json.Unmarshal(sh.B3Multi, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.B3Multi = p
	}

	if sh.Baggage != nil {
		var p BaggagePropagator
		if err := json.Unmarshal(sh.Baggage, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Baggage = p
	}

	if sh.Jaeger != nil {
		var p JaegerPropagator
		if err := json.Unmarshal(sh.Jaeger, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Jaeger = p
	}

	if sh.Ottrace != nil {
		var p OpenTracingPropagator
		if err := json.Unmarshal(sh.Ottrace, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Ottrace = p
	}

	if sh.Tracecontext != nil {
		var p TraceContextPropagator
		if err := json.Unmarshal(sh.Tracecontext, &p); err != nil {
			return errors.Join(newErrUnmarshal(j), err)
		}
		sh.Plain.Tracecontext = p
	}

	*j = TextMapPropagator(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	type Plain BatchLogRecordProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&sh.Plain))
	if err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	type Plain BatchSpanProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	err := validateBatchSpanProcessor((*BatchSpanProcessor)(&sh.Plain))
	if err != nil {
		return err
	}
	*j = BatchSpanProcessor(sh.Plain)
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
	if _, ok := raw["name"]; !ok {
		return errors.New("json: cannot unmarshal field name in NameStringValuePair required")
	}
	if _, ok := raw["value"]; !ok {
		return errors.New("json: cannot unmarshal field value in NameStringValuePair required")
	}
	var name, value string
	var ok bool
	if name, ok = raw["name"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field name in NameStringValuePair must be string")
	}
	if value, ok = raw["value"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field value in NameStringValuePair must be string")
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
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OpenTelemetryConfiguration(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	type Plain PeriodicMetricReader
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	err := validatePeriodicMetricReader((*PeriodicMetricReader)(&sh.Plain))
	if err != nil {
		return err
	}
	*j = PeriodicMetricReader(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CardinalityLimits) UnmarshalJSON(value []byte) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateCardinalityLimits((*CardinalityLimits)(&plain)); err != nil {
		return err
	}
	*j = CardinalityLimits(plain)
	return nil
}
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
func (j *SpanLimits) UnmarshalJSON(value []byte) error {
	type Plain SpanLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPHttpMetricExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPHttpMetricExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpMetricExporter(sh.Plain)
	return nil
}

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
func (j *OTLPGrpcMetricExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPGrpcMetricExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcMetricExporter(sh.Plain)
	return nil
}
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
func (j *OTLPHttpExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPHttpExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpExporter(sh.Plain)
	return nil
}

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
func (j *OTLPGrpcExporter) UnmarshalJSON(b []byte) error {
	type Plain OTLPGrpcExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}
	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeType) UnmarshalJSON(b []byte) error {
	var v struct {
		Value any
	}
	if err := json.Unmarshal(b, &v.Value); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	var ok bool
	for _, expected := range enumValuesAttributeType {
		if reflect.DeepEqual(v.Value, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return newErrInvalid(fmt.Sprintf("unexpected value type %#v, expected one of %#v)", v.Value, enumValuesAttributeType))
	}
	*j = AttributeType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeNameValue) UnmarshalJSON(b []byte) error {
	type Plain AttributeNameValue
	type shadow struct {
		Plain
		Name  json.RawMessage `json:"name"`
		Value json.RawMessage `json:"value"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Name == nil {
		return newErrRequired(j, "name")
	}
	if err := json.Unmarshal(sh.Name, &sh.Plain.Name); err != nil {
		return err
	}
	if sh.Value == nil {
		return newErrRequired(j, "value")
	}
	if err := json.Unmarshal(sh.Value, &sh.Plain.Value); err != nil {
		return err
	}

	// json unmarshaller defaults to unmarshalling to float for int values
	if sh.Type != nil && sh.Type.Value == "int" {
		val, ok := sh.Plain.Value.(float64)
		if ok {
			sh.Plain.Value = int(val)
		}
	}

	if sh.Type != nil && sh.Type.Value == "int_array" {
		m, ok := sh.Plain.Value.([]any)
		if ok {
			var vals []any
			for _, v := range m {
				val, ok := v.(float64)
				if ok {
					vals = append(vals, int(val))
				} else {
					vals = append(vals, v)
				}
			}
			sh.Plain.Value = vals
		}
	}

	*j = AttributeNameValue(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	type Plain SimpleLogRecordProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	type Plain SimpleSpanProcessor
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ZipkinSpanExporter) UnmarshalJSON(b []byte) error {
	type Plain ZipkinSpanExporter
	type shadow struct {
		Plain
		Endpoint json.RawMessage `json:"endpoint"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Endpoint == nil {
		return newErrRequired(j, "endpoint")
	}

	if err := json.Unmarshal(sh.Endpoint, &sh.Plain.Endpoint); err != nil {
		return err
	}
	if sh.Timeout != nil && 0 > *sh.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = ZipkinSpanExporter(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *NameStringValuePair) UnmarshalJSON(b []byte) error {
	type Plain NameStringValuePair
	type shadow struct {
		Plain
		Name  json.RawMessage `json:"name"`
		Value json.RawMessage `json:"value"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Name == nil {
		return newErrRequired(j, "name")
	}
	if err := json.Unmarshal(sh.Name, &sh.Plain.Name); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Value == nil {
		return newErrRequired(j, "value")
	}
	if err := json.Unmarshal(sh.Value, &sh.Plain.Value); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = NameStringValuePair(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedInstrumentType(InstrumentType(v)); err != nil {
		return err
	}
	*j = InstrumentType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExperimentalPeerInstrumentationServiceMappingElem) UnmarshalJSON(b []byte) error {
	type Plain ExperimentalPeerInstrumentationServiceMappingElem
	type shadow struct {
		Plain
		Peer    json.RawMessage `json:"peer"`
		Service json.RawMessage `json:"service"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Peer == nil {
		return newErrRequired(j, "peer")
	}
	if err := json.Unmarshal(sh.Peer, &sh.Plain.Peer); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Service == nil {
		return newErrRequired(j, "service")
	}
	if err := json.Unmarshal(sh.Service, &sh.Plain.Service); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = ExperimentalPeerInstrumentationServiceMappingElem(sh.Plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ExporterDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedHistogramAggregation(ExporterDefaultHistogramAggregation(v)); err != nil {
		return err
	}
	*j = ExporterDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	type Plain PullMetricReader
	type shadow struct {
		Plain
		Exporter json.RawMessage `json:"exporter"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if sh.Exporter == nil {
		return newErrRequired(j, "exporter")
	}
	// Hydrate the exporter into the underlying field.
	if err := json.Unmarshal(sh.Exporter, &sh.Plain.Exporter); err != nil {
		return err
	}
	*j = PullMetricReader(sh.Plain)
	return nil
}
