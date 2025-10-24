// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"errors"

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
