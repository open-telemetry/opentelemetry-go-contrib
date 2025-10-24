// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON implements json.Unmarshaler.
func (j *ConsoleExporter) UnmarshalJSON(b []byte) error {
	type plain ConsoleExporter
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return errors.Join(newErrUnmarshal("ConsoleExporter"), err)
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
func (j *PushMetricExporter) UnmarshalJSON(b []byte) error {
	// Use a shadow struct with a RawMessage field to detect key presence.
	type Plain PushMetricExporter
	type shadow struct {
		Plain
		Console json.RawMessage `json:"console"`
	}
	var sh shadow
	if err := json.Unmarshal(b, &sh); err != nil {
		return errors.Join(newErrUnmarshal("PushMetricExporter"), err)
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
		return errors.Join(newErrUnmarshal("SpanExporter"), err)
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
		return errors.Join(newErrUnmarshal("LogRecordExporter"), err)
	}

	if sh.Console != nil {
		var c ConsoleExporter
		if err := json.Unmarshal(sh.Console, &c); err != nil {
			return err
		}
		sh.Plain.Console = c
	}
	*j = LogRecordExporter(sh.Plain)
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
