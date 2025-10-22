// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import (
	"errors"
	"fmt"
)

var (
	errUnmarshalingCardinalityLimits = errors.New("unmarshaling error cardinality_limit")
	errUnmarshalingSpanLimits        = errors.New("unmarshaling error span_limit")
)

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
