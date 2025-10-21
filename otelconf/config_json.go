// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON implements json.Unmarshaler.
func (j *TextMapPropagator) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Join(errors.New("unmarshaling error TextMapPropagator"))
	}
	type Plain TextMapPropagator
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return errors.Join(errors.New("unmarshaling error TextMapPropagator"))
	}
	unmarshalTextMapPropagatorTypes(raw, (*TextMapPropagator)(&plain))
	*j = TextMapPropagator(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Join(errors.New("unmarshaling error BatchLogRecordProcessor"))
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
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
		return errors.Join(errors.New("unmarshaling error BatchSpanProcessor"))
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CardinalityLimits) UnmarshalJSON(value []byte) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := json.Unmarshal(value, &plain); err != nil {
		return errors.Join(errors.New("unmarshaling error cardinality_limit"))
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
		return errors.Join(errors.New("unmarshaling error span_limit"), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}
