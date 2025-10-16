// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"encoding/json"
	"fmt"

	yaml "go.yaml.in/yaml/v3"
)

func ptr[T any](v T) *T {
	return &v
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

// MarshalUnmarshaler combines marshal and unmarshal operations.
type MarshalUnmarshaler interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// jsonCodec implements MarshalUnmarshaler for JSON.
type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// yamlCodec implements MarshalUnmarshaler for YAML.
type yamlCodec struct{}

func (yamlCodec) Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func (yamlCodec) Unmarshal(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

// setConfigDefaults sets default values for disabled and log_level.
func setConfigDefaults(raw map[string]any, plain *OpenTelemetryConfiguration, codec MarshalUnmarshaler) error {
	// Configure if the SDK is disabled or not.
	// If omitted or null, false is used.
	plain.Disabled = ptr(false)
	if v, ok := raw["disabled"]; ok && v != nil {
		marshaled, err := codec.Marshal(v)
		if err != nil {
			return err
		}
		var disabled bool
		if err := codec.Unmarshal(marshaled, &disabled); err != nil {
			return err
		}
		plain.Disabled = &disabled
	}

	// Configure the log level of the internal logger used by the SDK.
	// If omitted, info is used.
	plain.LogLevel = ptr("info")
	if v, ok := raw["log_level"]; ok && v != nil {
		marshaled, err := codec.Marshal(v)
		if err != nil {
			return err
		}
		var logLevel string
		if err := codec.Unmarshal(marshaled, &logLevel); err != nil {
			return err
		}
		plain.LogLevel = &logLevel
	}

	return nil
}

// validateStringField validates a string field is present and correct type.
func validateStringField(raw map[string]any, fieldName string) (string, error) {
	v, ok := raw[fieldName]
	if !ok {
		return "", fmt.Errorf("cannot unmarshal field %s in NameStringValuePair required", fieldName)
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("cannot unmarshal field %s in NameStringValuePair must be string", fieldName)
	}
	return str, nil
}

// unmarshalExporterWithConsole handles the console exporter unmarshaling pattern.
// Returns true if console field was present in raw.
func checkConsoleExporter(raw map[string]any) bool {
	_, ok := raw["console"]
	return ok
}

// unmarshalSamplerTypes handles always_on and always_off sampler unmarshaling.
func unmarshalSamplerTypes(raw map[string]any, plain *Sampler) {
	// always_on can be nil, must check and set here
	if _, ok := raw["always_on"]; ok {
		plain.AlwaysOn = AlwaysOnSampler{}
	}
	// always_off can be nil, must check and set here
	if _, ok := raw["always_off"]; ok {
		plain.AlwaysOff = AlwaysOffSampler{}
	}
}

// unmarshalTextMapPropagatorTypes handles all propagator type unmarshaling.
func unmarshalTextMapPropagatorTypes(raw map[string]any, plain *TextMapPropagator) {
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
}

// unmarshalMetricProducer handles opencensus metric producer unmarshaling.
func unmarshalMetricProducer(raw map[string]any, plain *MetricProducer) {
	// opencensus can be nil, must check and set here
	if v, ok := raw["opencensus"]; ok && v == nil {
		delete(raw, "opencensus")
		plain.Opencensus = OpenCensusMetricProducer{}
	}
	if len(raw) > 0 {
		plain.AdditionalProperties = raw
	}
}
