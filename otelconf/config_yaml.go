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

	// Configure if the SDK is disabled or not.
	// If omitted or null, false is used.
	plain.Disabled = ptr(false)
	if v, ok := raw["disabled"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var disabled bool
		if err := yaml.Unmarshal(marshaled, &disabled); err != nil {
			return err
		}
		plain.Disabled = &disabled
	}

	// Configure the log level of the internal logger used by the SDK.
	// If omitted, info is used.
	plain.LogLevel = ptr("info")
	if v, ok := raw["log_level"]; ok && v != nil {
		marshaled, err := yaml.Marshal(v)
		if err != nil {
			return err
		}

		var logLevel string
		if err := yaml.Unmarshal(marshaled, &logLevel); err != nil {
			return err
		}
		plain.LogLevel = &logLevel
	}

	plain.FileFormat = fmt.Sprintf("%v", raw["file_format"])

	*j = OpenTelemetryConfiguration(plain)
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
		return newErrRequiredExporter(j)
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
func (j *NameStringValuePair) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; !ok {
		return errors.New("yaml: cannot unmarshal field name in NameStringValuePair required")
	}
	if _, ok := raw["value"]; !ok {
		return errors.New("yaml: cannot unmarshal field value in NameStringValuePair required")
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

func (j *PushMetricExporter) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain PushMetricExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if _, ok := raw["console"]; ok {
		plain.Console = ConsoleExporter{}
	}
	*j = PushMetricExporter(plain)
	return nil
}

func (j *SpanExporter) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain SpanExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if _, ok := raw["console"]; ok {
		plain.Console = ConsoleExporter{}
	}
	*j = SpanExporter(plain)
	return nil
}

func (j *LogRecordExporter) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain LogRecordExporter
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	// console can be nil, must check and set here
	if _, ok := raw["console"]; ok {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

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
	// always_on can be nil, must check and set here
	if _, ok := raw["always_on"]; ok {
		plain.AlwaysOn = AlwaysOnSampler{}
	}
	// always_off can be nil, must check and set here
	if _, ok := raw["always_off"]; ok {
		plain.AlwaysOff = AlwaysOffSampler{}
	}
	*j = Sampler(plain)
	return nil
}

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
	// opencensus can be nil, must check and set here
	if v, ok := raw["opencensus"]; ok && v == nil {
		delete(raw, "opencensus")
		plain.Opencensus = OpenCensusMetricProducer{}
	}
	if len(raw) > 0 {
		plain.AdditionalProperties = raw
	}

	*j = MetricProducer(plain)
	return nil
}

func (j *TextMapPropagator) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}
	type Plain TextMapPropagator
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return err
	}
	// b3 can be nil, must check and set here
	if v, ok := raw["b3"]; ok && v == nil {
		plain.B3 = B3Propagator{}
	}
	if v, ok := raw["b3multi"]; ok && v == nil {
		plain.B3Multi = B3MultiPropagator{}
	}
	if v, ok := raw["baggage"]; ok && v == nil {
		plain.Baggage = BaggagePropagator{}
	}
	if v, ok := raw["jaeger"]; ok && v == nil {
		plain.Jaeger = JaegerPropagator{}
	}
	if v, ok := raw["ottrace"]; ok && v == nil {
		plain.Ottrace = OpenTracingPropagator{}
	}
	if v, ok := raw["tracecontext"]; ok && v == nil {
		plain.Tracecontext = TraceContextPropagator{}
	}
	*j = TextMapPropagator(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	if !hasYAMLMapKey(node, "exporter") {
		return newErrRequiredExporter(j)
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
		return newErrRequiredExporter(j)
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
func (j *ExperimentalLanguageSpecificInstrumentation) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*j = raw
	return nil
}
