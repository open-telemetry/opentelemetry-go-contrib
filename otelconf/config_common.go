// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/baggage"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	yaml "go.yaml.in/yaml/v3"
)

const (
	compressionGzip = "gzip"
	compressionNone = "none"
)

type configOptions struct {
	ctx                   context.Context
	opentelemetryConfig   OpenTelemetryConfiguration
	loggerProviderOptions []sdklog.LoggerProviderOption
}

type shutdownFunc func(context.Context) error

func noopShutdown(context.Context) error {
	return nil
}

type errBound struct {
	Field string
	Bound int
	Op    string
}

func (e *errBound) Error() string {
	return fmt.Sprintf("field %s: must be %s %d", e.Field, e.Op, e.Bound)
}

func (e *errBound) Is(target error) bool {
	t, ok := target.(*errBound)
	if !ok {
		return false
	}
	return e.Field == t.Field && e.Bound == t.Bound && e.Op == t.Op
}

type errRequiredExporter struct {
	Object any
}

func (e *errRequiredExporter) Error() string {
	return fmt.Sprintf("field exporter in %s: required", reflect.TypeOf(e.Object))
}

func (e *errRequiredExporter) Is(target error) bool {
	t, ok := target.(*errRequiredExporter)
	if !ok {
		return false
	}
	return reflect.TypeOf(e.Object) == reflect.TypeOf(t.Object)
}

type errUnmarshal struct {
	Object any
}

func (e *errUnmarshal) Error() string {
	return fmt.Sprintf("unmarshal error in %T", e.Object)
}

func (e *errUnmarshal) Is(target error) bool {
	t, ok := target.(*errUnmarshal)
	if !ok {
		return false
	}
	return reflect.TypeOf(e.Object) == reflect.TypeOf(t.Object)
}

// newErrGreaterOrEqualZero creates a new error indicating that the field must be greater than
// or equal to zero.
func newErrGreaterOrEqualZero(field string) error {
	return &errBound{Field: field, Bound: 0, Op: ">="}
}

// newErrGreaterThanZero creates a new error indicating that the field must be greater
// than zero.
func newErrGreaterThanZero(field string) error {
	return &errBound{Field: field, Bound: 0, Op: ">"}
}

// newErrRequiredExporter creates a new error indicating that the exporter field is required.
func newErrRequiredExporter(object any) error {
	return &errRequiredExporter{Object: object}
}

// newErrUnmarshal creates a new error indicating that an error occurred during unmarshaling.
func newErrUnmarshal(object any) error {
	return &errUnmarshal{Object: object}
}

type errInvalid struct {
	Identifier string
}

func (e *errInvalid) Error() string {
	return "invalid config: " + e.Identifier
}

func (e *errInvalid) Is(target error) bool {
	t, ok := target.(*errInvalid)
	if !ok {
		return false
	}
	return reflect.TypeOf(e.Identifier) == reflect.TypeOf(t.Identifier)
}

// newErrInvalid creates a new error indicating that an error occurred due to misconfiguration.
func newErrInvalid(id string) error {
	return &errInvalid{Identifier: id}
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

// validatePeriodicMetricReader handles validation for PeriodicMetricReader.
func validatePeriodicMetricReader(plain *PeriodicMetricReader) error {
	if plain.Timeout != nil && 0 > *plain.Timeout {
		return newErrGreaterOrEqualZero("timeout")
	}
	if plain.Interval != nil && 0 > *plain.Interval {
		return newErrGreaterOrEqualZero("interval")
	}
	return nil
}

// validateBatchLogRecordProcessor handles validation for BatchLogRecordProcessor.
func validateBatchLogRecordProcessor(plain *BatchLogRecordProcessor) error {
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return newErrGreaterOrEqualZero("export_timeout")
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return newErrGreaterThanZero("max_export_batch_size")
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return newErrGreaterThanZero("max_queue_size")
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return newErrGreaterOrEqualZero("schedule_delay")
	}
	return nil
}

// validateBatchSpanProcessor handles validation for BatchSpanProcessor.
func validateBatchSpanProcessor(plain *BatchSpanProcessor) error {
	if plain.ExportTimeout != nil && 0 > *plain.ExportTimeout {
		return newErrGreaterOrEqualZero("export_timeout")
	}
	if plain.MaxExportBatchSize != nil && 0 >= *plain.MaxExportBatchSize {
		return newErrGreaterThanZero("max_export_batch_size")
	}
	if plain.MaxQueueSize != nil && 0 >= *plain.MaxQueueSize {
		return newErrGreaterThanZero("max_queue_size")
	}
	if plain.ScheduleDelay != nil && 0 > *plain.ScheduleDelay {
		return newErrGreaterOrEqualZero("schedule_delay")
	}
	return nil
}

// validateCardinalityLimits handles validation for CardinalityLimits.
func validateCardinalityLimits(plain *CardinalityLimits) error {
	if plain.Counter != nil && 0 >= *plain.Counter {
		return newErrGreaterThanZero("counter")
	}
	if plain.Default != nil && 0 >= *plain.Default {
		return newErrGreaterThanZero("default")
	}
	if plain.Gauge != nil && 0 >= *plain.Gauge {
		return newErrGreaterThanZero("gauge")
	}
	if plain.Histogram != nil && 0 >= *plain.Histogram {
		return newErrGreaterThanZero("histogram")
	}
	if plain.ObservableCounter != nil && 0 >= *plain.ObservableCounter {
		return newErrGreaterThanZero("observable_counter")
	}
	if plain.ObservableGauge != nil && 0 >= *plain.ObservableGauge {
		return newErrGreaterThanZero("observable_gauge")
	}
	if plain.ObservableUpDownCounter != nil && 0 >= *plain.ObservableUpDownCounter {
		return newErrGreaterThanZero("observable_up_down_counter")
	}
	if plain.UpDownCounter != nil && 0 >= *plain.UpDownCounter {
		return newErrGreaterThanZero("up_down_counter")
	}
	return nil
}

// validateSpanLimits handles validation for SpanLimits.
func validateSpanLimits(plain *SpanLimits) error {
	if plain.AttributeCountLimit != nil && 0 > *plain.AttributeCountLimit {
		return newErrGreaterOrEqualZero("attribute_count_limit")
	}
	if plain.AttributeValueLengthLimit != nil && 0 > *plain.AttributeValueLengthLimit {
		return newErrGreaterOrEqualZero("attribute_value_length_limit")
	}
	if plain.EventAttributeCountLimit != nil && 0 > *plain.EventAttributeCountLimit {
		return newErrGreaterOrEqualZero("event_attribute_count_limit")
	}
	if plain.EventCountLimit != nil && 0 > *plain.EventCountLimit {
		return newErrGreaterOrEqualZero("event_count_limit")
	}
	if plain.LinkAttributeCountLimit != nil && 0 > *plain.LinkAttributeCountLimit {
		return newErrGreaterOrEqualZero("link_attribute_count_limit")
	}
	if plain.LinkCountLimit != nil && 0 > *plain.LinkCountLimit {
		return newErrGreaterOrEqualZero("link_count_limit")
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

// createHeadersConfig combines the two header config fields. Headers take precedence over headersList.
func createHeadersConfig(headers []NameStringValuePair, headersList *string) (map[string]string, error) {
	result := make(map[string]string)
	if headersList != nil {
		// Parsing follows https://github.com/open-telemetry/opentelemetry-configuration/blob/568e5080816d40d75792eb754fc96bde09654159/schema/type_descriptions.yaml#L584.
		headerslist, err := baggage.Parse(*headersList)
		if err != nil {
			return nil, errors.Join(newErrInvalid("invalid headers_list"), err)
		}
		for _, kv := range headerslist.Members() {
			result[kv.Key()] = kv.Value()
		}
	}
	// Headers take precedence over HeadersList, so this has to be after HeadersList is processed.
	for _, kv := range headers {
		if kv.Value != nil {
			result[kv.Name] = *kv.Value
		}
	}
	return result, nil
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
