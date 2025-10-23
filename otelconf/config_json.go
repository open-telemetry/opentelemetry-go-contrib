// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"encoding/json"
	"errors"
)

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
	if _, ok := raw["console"]; ok {
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
	if _, ok := raw["console"]; ok {
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
	if _, ok := raw["console"]; ok {
		plain.Console = ConsoleExporter{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Join(errUnmarshalingBatchLogRecordProcessor, err)
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return newErrRequiredExporter("BatchLogRecordProcessor")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return errors.Join(errUnmarshalingBatchLogRecordProcessor, err)
	}
	if err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Join(errUnmarshalingBatchSpanProcessor, err)
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return newErrRequiredExporter("BatchSpanProcessor")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return errors.Join(errUnmarshalingBatchSpanProcessor, err)
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Join(errUnmarshalingPeriodicMetricReader, err)
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return newErrRequiredExporter("PeriodicMetricReader")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return errors.Join(errUnmarshalingPeriodicMetricReader, err)
	}
	if err := validatePeriodicMetricReader((*PeriodicMetricReader)(&plain)); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CardinalityLimits) UnmarshalJSON(value []byte) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(errUnmarshalingCardinalityLimits, err)
	}
	if err := validateCardinalityLimits((*CardinalityLimits)(&plain)); err != nil {
		return err
	}
	*j = CardinalityLimits(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanLimits) UnmarshalJSON(value []byte) error {
	type Plain SpanLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(errUnmarshalingSpanLimits, err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}
