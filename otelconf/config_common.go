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

// newErrMin creates a new error indicating that the field must be greater than
// or equal to the bound.
func newErrMin(field string, bound int) error {
	return &errBound{Field: field, Bound: bound, Op: ">="}
}

// newErrExclMin creates a new error indicating that the field must be greater
// than the bound.
func newErrExclMin(field string, bound int) error {
	return &errBound{Field: field, Bound: bound, Op: ">"}
}

// validateCardinalityLimits handles validation for CardinalityLimits.
func validateCardinalityLimits(plain *CardinalityLimits) error {
	if plain.Counter != nil && 0 >= *plain.Counter {
		return newErrExclMin("counter", 0)
	}
	if plain.Default != nil && 0 >= *plain.Default {
		return newErrExclMin("default", 0)
	}
	if plain.Gauge != nil && 0 >= *plain.Gauge {
		return newErrExclMin("gauge", 0)
	}
	if plain.Histogram != nil && 0 >= *plain.Histogram {
		return newErrExclMin("histogram", 0)
	}
	if plain.ObservableCounter != nil && 0 >= *plain.ObservableCounter {
		return newErrExclMin("observable_counter", 0)
	}
	if plain.ObservableGauge != nil && 0 >= *plain.ObservableGauge {
		return newErrExclMin("observable_gauge", 0)
	}
	if plain.ObservableUpDownCounter != nil && 0 >= *plain.ObservableUpDownCounter {
		return newErrExclMin("observable_up_down_counter", 0)
	}
	if plain.UpDownCounter != nil && 0 >= *plain.UpDownCounter {
		return newErrExclMin("up_down_counter", 0)
	}
	return nil
}

// validateSpanLimits handles validation for SpanLimits.
func validateSpanLimits(plain *SpanLimits) error {
	if plain.AttributeCountLimit != nil && 0 > *plain.AttributeCountLimit {
		return newErrMin("attribute_count_limit", 0)
	}
	if plain.AttributeValueLengthLimit != nil && 0 > *plain.AttributeValueLengthLimit {
		return newErrMin("attribute_value_length_limit", 0)
	}
	if plain.EventAttributeCountLimit != nil && 0 > *plain.EventAttributeCountLimit {
		return newErrMin("event_attribute_count_limit", 0)
	}
	if plain.EventCountLimit != nil && 0 > *plain.EventCountLimit {
		return newErrMin("event_count_limit", 0)
	}
	if plain.LinkAttributeCountLimit != nil && 0 > *plain.LinkAttributeCountLimit {
		return newErrMin("link_attribute_count_limit", 0)
	}
	if plain.LinkCountLimit != nil && 0 > *plain.LinkCountLimit {
		return newErrMin("link_count_limit", 0)
	}
	return nil
}
