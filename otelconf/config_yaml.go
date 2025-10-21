// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return errors.Join(errors.New("unmarshaling error BatchLogRecordProcessor"))
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(errors.New("unmarshaling error BatchLogRecordProcessor"))
	}
	if err := validateBatchLogRecordProcessor((*BatchLogRecordProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalYAML(node *yaml.Node) error {
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return errors.Join(errors.New("unmarshaling error BatchSpanProcessor"))
	}
	if _, ok := raw["exporter"]; raw != nil && !ok {
		return errors.New("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(errors.New("unmarshaling error BatchSpanProcessor"))
	}
	if err := validateBatchSpanProcessor((*BatchSpanProcessor)(&plain)); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *CardinalityLimits) UnmarshalYAML(node *yaml.Node) error {
	type Plain CardinalityLimits
	var plain Plain
	if err := node.Decode(&plain); err != nil {
		return errors.Join(errors.New("unmarshaling error cardinality_limit"), err)
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
		return errors.Join(errors.New("unmarshaling error span_limit"), err)
	}
	if err := validateSpanLimits((*SpanLimits)(&plain)); err != nil {
		return err
	}
	*j = SpanLimits(plain)
	return nil
}
