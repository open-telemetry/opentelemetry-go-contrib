// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"errors"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v3"
)

// hasYAMLMapKey reports whether the provided mapping node contains the given
// key. It assumes the node is a mapping node and performs a linear scan of its
// key nodes.
func hasYAMLMapKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return true
		}
	}
	return false
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PushMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain PushMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = ConsoleExporter{}
	}
	*j = PushMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any

	if err := node.Decode(&raw); err != nil {
		return err
	}

	type Plain OpenTelemetryConfiguration
	var plain Plain

	if v, ok := raw["logger_provider"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		var lp LoggerProviderJson
		if err := yaml.Unmarshal(marshaled, &lp); err != nil {
			return err
		}
		plain.LoggerProvider = &lp
	}

	if v, ok := raw["meter_provider"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var mp MeterProviderJson
		if err := yaml.Unmarshal(marshaled, &mp); err != nil {
			return err
		}
		plain.MeterProvider = &mp
	}

	if v, ok := raw["tracer_provider"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var tp TracerProviderJson
		if err := yaml.Unmarshal(marshaled, &tp); err != nil {
			return err
		}
		plain.TracerProvider = &tp
	}

	if v, ok := raw["propagator"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var p PropagatorJson
		if err := yaml.Unmarshal(marshaled, &p); err != nil {
			return err
		}
		plain.Propagator = &p
	}

	if v, ok := raw["resource"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var r ResourceJson
		if err := yaml.Unmarshal(marshaled, &r); err != nil {
			return err
		}
		plain.Resource = &r
	}

	if v, ok := raw["instrumentation/development"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var i InstrumentationJson
		if err := yaml.Unmarshal(marshaled, &i); err != nil {
			return err
		}
		plain.InstrumentationDevelopment = &i
	}

	if v, ok := raw["attribute_limits"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var a AttributeLimits
		if err := yaml.Unmarshal(marshaled, &a); err != nil {
			return err
		}
		plain.AttributeLimits = &a
	}

	plainConfig := (*OpenTelemetryConfiguration)(&plain)
	if err := setConfigDefaults(raw, plainConfig, yamlCodec{}); err != nil {
		return err
	}

	plain.FileFormat = fmt.Sprintf("%v", raw["file_format"])
	*j = *plainConfig
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeType) UnmarshalYAML(node *yaml.Node) error {
	var v struct {
		Value any
	}
	if err := node.Decode(&v.Value); err != nil {
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

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SpanExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain SpanExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = ConsoleExporter{}
	}
	*j = SpanExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *LogRecordExporter) UnmarshalYAML(node *yaml.Node) error {
	type Plain LogRecordExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// console can be nil, must check and set here
	if hasYAMLMapKey(node, "console") && plain.Console == nil {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *TextMapPropagator) UnmarshalYAML(node *yaml.Node) error {
	type Plain TextMapPropagator
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	// b3 can be nil, must check and set here
	if hasYAMLMapKey(node, "b3") && plain.B3 == nil {
		plain.B3 = B3Propagator{}
	}
	// b3multi can be nil, must check and set here
	if hasYAMLMapKey(node, "b3multi") && plain.B3Multi == nil {
		plain.B3Multi = B3MultiPropagator{}
	}
	// baggage can be nil, must check and set here
	if hasYAMLMapKey(node, "baggage") && plain.Baggage == nil {
		plain.Baggage = BaggagePropagator{}
	}
	// jaeger can be nil, must check and set here
	if hasYAMLMapKey(node, "jaeger") && plain.Jaeger == nil {
		plain.Jaeger = JaegerPropagator{}
	}
	// ottrace can be nil, must check and set here
	if hasYAMLMapKey(node, "ottrace") && plain.Ottrace == nil {
		plain.Ottrace = OpenTracingPropagator{}
	}
	// tracecontext can be nil, must check and set here
	if hasYAMLMapKey(node, "tracecontext") && plain.Tracecontext == nil {
		plain.Tracecontext = TraceContextPropagator{}
	}
	*j = TextMapPropagator(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *NameStringValuePair) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
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

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *Sampler) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain Sampler
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	unmarshalSamplerTypes(raw, (*Sampler)(&plain))
	*j = Sampler(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *MetricProducer) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain MetricProducer
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	unmarshalMetricProducer(raw, (*MetricProducer)(&plain))
	*j = MetricProducer(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validatePeriodicMetricReader((*PeriodicMetricReader)(&plain)); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *CardinalityLimits) UnmarshalYAML(node *yaml.Node) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateCardinalityLimits((*CardinalityLimits)(&plain)); err != nil {
		return err
	}
	*j = CardinalityLimits(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SpanLimits) UnmarshalYAML(node *yaml.Node) error {
	type Plain SpanLimits
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPHttpMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPHttpMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPGrpcMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPGrpcMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPHttpExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPHttpExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPHttpExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *OTLPGrpcExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain OTLPGrpcExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = OTLPGrpcExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeType) UnmarshalYAML(node *yaml.Node) error {
	var v struct {
		Value any
	}
	if err := node.Decode(&v.Value); err != nil {
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

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeNameValue) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "name") {
		return newErrRequired(j, "name")
	}
	if !hasYAMLMapKey(node, "value") {
		return newErrRequired(j, "value")
	}
	type Plain AttributeNameValue
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	// yaml unmarshaller defaults to unmarshalling to int
	if plain.Type != nil && plain.Type.Value == "double" {
		val, ok := plain.Value.(int)
		if ok {
			plain.Value = float64(val)
		}
	}

	if plain.Type != nil && plain.Type.Value == "double_array" {
		m, ok := plain.Value.([]any)
		if ok {
			var vals []any
			for _, v := range m {
				val, ok := v.(int)
				if ok {
					vals = append(vals, float64(val))
				} else {
					vals = append(vals, v)
				}
			}
			plain.Value = vals
		}
	}

	*j = AttributeNameValue(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ZipkinSpanExporter) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "endpoint") {
		return newErrRequired(j, "endpoint")
	}
	type Plain ZipkinSpanExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	*j = ZipkinSpanExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *NameStringValuePair) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "name") {
		return newErrRequired(j, "name")
	}
	if !hasYAMLMapKey(node, "value") {
		return newErrRequired(j, "value")
	}

	type Plain NameStringValuePair
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = NameStringValuePair(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *InstrumentType) UnmarshalYAML(node *yaml.Node) error {
	type Plain InstrumentType
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedInstrumentType(InstrumentType(plain)); err != nil {
		return err
	}

	*j = InstrumentType(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExperimentalPeerInstrumentationServiceMappingElem) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "peer") {
		return newErrRequired(j, "peer")
	}
	if !hasYAMLMapKey(node, "service") {
		return newErrRequired(j, "service")
	}

	type Plain ExperimentalPeerInstrumentationServiceMappingElem
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}

	*j = ExperimentalPeerInstrumentationServiceMappingElem(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExporterDefaultHistogramAggregation) UnmarshalYAML(node *yaml.Node) error {
	type Plain ExporterDefaultHistogramAggregation
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	if err := supportedHistogramAggregation(ExporterDefaultHistogramAggregation(plain)); err != nil {
		return err
	}

	*j = ExporterDefaultHistogramAggregation(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *PullMetricReader) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequired(j, "exporter")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(newErrUnmarshal(j), err)
	}
	*j = PullMetricReader(plain)
	return nil
}

func (j *ExperimentalLanguageSpecificInstrumentation) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*j = raw
	return nil
}
