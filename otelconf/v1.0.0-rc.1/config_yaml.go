// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v1.0.0-rc.1"

import (
	"fmt"
	"reflect"

	yaml "go.yaml.in/yaml/v3"
)

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
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = PushMetricExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
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
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = SpanExporter(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
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
	if checkConsoleExporter(raw) {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
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
	unmarshalTextMapPropagatorTypes(raw, (*TextMapPropagator)(&plain))
	*j = TextMapPropagator(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *ExperimentalLanguageSpecificInstrumentation) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*j = raw
	return nil
}
