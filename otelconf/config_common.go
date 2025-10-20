// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf"

import "fmt"

// validateCardinalityLimits handles validation for CardinalityLimits.
func validateCardinalityLimits(plain *CardinalityLimits) error {
	if plain.Counter != nil && 0 >= *plain.Counter {
		return fmt.Errorf("field %s: must be > %v", "counter", 0)
	}
	if plain.Default != nil && 0 >= *plain.Default {
		return fmt.Errorf("field %s: must be > %v", "default", 0)
	}
	if plain.Gauge != nil && 0 >= *plain.Gauge {
		return fmt.Errorf("field %s: must be > %v", "gauge", 0)
	}
	if plain.Histogram != nil && 0 >= *plain.Histogram {
		return fmt.Errorf("field %s: must be > %v", "histogram", 0)
	}
	if plain.ObservableCounter != nil && 0 >= *plain.ObservableCounter {
		return fmt.Errorf("field %s: must be > %v", "observable_counter", 0)
	}
	if plain.ObservableGauge != nil && 0 >= *plain.ObservableGauge {
		return fmt.Errorf("field %s: must be > %v", "observable_gauge", 0)
	}
	if plain.ObservableUpDownCounter != nil && 0 >= *plain.ObservableUpDownCounter {
		return fmt.Errorf("field %s: must be > %v", "observable_up_down_counter", 0)
	}
	if plain.UpDownCounter != nil && 0 >= *plain.UpDownCounter {
		return fmt.Errorf("field %s: must be > %v", "up_down_counter", 0)
	}
	return nil
}

// validateSpanLimits handles validation for SpanLimits.
func validateSpanLimits(plain *SpanLimits) error {
	if plain.AttributeCountLimit != nil && 0 > *plain.AttributeCountLimit {
		return fmt.Errorf("field %s: must be >= %v", "attribute_count_limit", 0)
	}
	if plain.AttributeValueLengthLimit != nil && 0 > *plain.AttributeValueLengthLimit {
		return fmt.Errorf("field %s: must be >= %v", "attribute_value_length_limit", 0)
	}
	if plain.EventAttributeCountLimit != nil && 0 > *plain.EventAttributeCountLimit {
		return fmt.Errorf("field %s: must be >= %v", "event_attribute_count_limit", 0)
	}
	if plain.EventCountLimit != nil && 0 > *plain.EventCountLimit {
		return fmt.Errorf("field %s: must be >= %v", "event_count_limit", 0)
	}
	if plain.LinkAttributeCountLimit != nil && 0 > *plain.LinkAttributeCountLimit {
		return fmt.Errorf("field %s: must be >= %v", "link_attribute_count_limit", 0)
	}
	if plain.LinkCountLimit != nil && 0 > *plain.LinkCountLimit {
		return fmt.Errorf("field %s: must be >= %v", "link_count_limit", 0)
	}
	return nil
}
