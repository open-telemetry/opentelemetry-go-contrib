// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func intToPtr(input int) *int {
	return &input
}

func strToPtr(input string) *string {
	return &input
}

func instrumentTypeToPtr(input ViewSelectorInstrumentType) *ViewSelectorInstrumentType {
	return &input
}

func TestViewOptionsFromConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		views    []View
		expected int
	}{
		{
			name:  "empty views",
			views: []View{},
		},
		{
			name: "nil selector",
			views: []View{
				{},
			},
		},
		{
			name: "all instruments",
			views: []View{
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("counter_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeCounter),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
						Aggregation: &ViewStreamAggregation{Sum: ViewStreamAggregationSum{}},
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("histogram_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeHistogram),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
						Aggregation: &ViewStreamAggregation{ExplicitBucketHistogram: &ViewStreamAggregationExplicitBucketHistogram{}},
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_counter_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeObservableCounter),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_gauge_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeObservableGauge),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
						Aggregation: &ViewStreamAggregation{LastValue: ViewStreamAggregationLastValue{}},
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("observable_updown_counter_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeObservableUpDownCounter),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("updown_counter_instrument"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentTypeUpDownCounter),
						MeterName:      strToPtr("meter-1"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("nil type"),
						InstrumentType: nil,
						MeterName:      strToPtr("no-meter"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
					},
				},
				{
					Selector: &ViewSelector{
						InstrumentName: strToPtr("invalid type"),
						InstrumentType: instrumentTypeToPtr(ViewSelectorInstrumentType("invalid-type")),
						MeterName:      strToPtr("no-meter"),
						MeterVersion:   strToPtr("0.1.0"),
						MeterSchemaUrl: strToPtr("http://schema123"),
					},
					Stream: &ViewStream{
						Name:        strToPtr("new-stream"),
						Description: strToPtr("new-description"),
						Aggregation: &ViewStreamAggregation{Drop: ViewStreamAggregationDrop{}},
					},
				},
			},
			expected: 8,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, len(viewOptionsFromConfig(tc.views)), tc.expected)
		})
	}
}
