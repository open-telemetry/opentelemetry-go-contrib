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
	return nil
}

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
		return newErrRequiredExporter(j)
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
		return newErrRequiredExporter(j)
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
		return newErrRequiredExporter(j)
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
