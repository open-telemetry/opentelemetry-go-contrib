// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func viewOptionsFromConfig(views []View) []sdkmetric.Option {
	opts := []sdkmetric.Option{}
	for _, view := range views {
		if view.Selector == nil || view.Stream == nil {
			continue
		}
		opts = append(opts, sdkmetric.WithView(
			sdkmetric.NewView(
				sdkmetric.Instrument{
					Name: view.Selector.instrumentNameStr(),
					Kind: instrumentTypeToKind(view.Selector.InstrumentType),
					Unit: view.Selector.unitStr(),
					Scope: instrumentation.Scope{
						Name:      view.Selector.meterNameStr(),
						Version:   view.Selector.meterVersionStr(),
						SchemaURL: view.Selector.meterSchemaURLStr(),
					},
				},
				sdkmetric.Stream{
					Name:            view.Stream.nameStr(),
					Description:     view.Stream.descriptionStr(),
					Aggregation:     viewStreamAggregationToAggregation(view.Stream.Aggregation),
					AttributeFilter: attributeKeysToAttributeFilter(view.Stream.AttributeKeys),
				},
			),
		))
	}
	return opts
}

var invalidInstrumentKind = sdkmetric.InstrumentKind(0)

func instrumentTypeToKind(instrument *ViewSelectorInstrumentType) sdkmetric.InstrumentKind {
	if instrument == nil {
		return invalidInstrumentKind
	}
	switch *instrument {
	case ViewSelectorInstrumentTypeCounter:
		return sdkmetric.InstrumentKindCounter
	case ViewSelectorInstrumentTypeHistogram:
		return sdkmetric.InstrumentKindHistogram
	case ViewSelectorInstrumentTypeObservableCounter:
		return sdkmetric.InstrumentKindObservableCounter
	case ViewSelectorInstrumentTypeObservableGauge:
		return sdkmetric.InstrumentKindObservableGauge
	case ViewSelectorInstrumentTypeObservableUpDownCounter:
		return sdkmetric.InstrumentKindObservableUpDownCounter
	case ViewSelectorInstrumentTypeUpDownCounter:
		return sdkmetric.InstrumentKindUpDownCounter
	}
	return invalidInstrumentKind
}

func attributeKeysToAttributeFilter(keys []string) attribute.Filter {
	kvs := make([]attribute.KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = attribute.Bool(key, true)
	}
	filter := attribute.NewSet(kvs...)
	return func(kv attribute.KeyValue) bool {
		return !filter.HasValue(kv.Key)
	}
}

func viewStreamAggregationToAggregation(agg *ViewStreamAggregation) sdkmetric.Aggregation {
	if agg == nil {
		return sdkmetric.AggregationDefault{}
	}
	if agg.Sum != nil {
		return sdkmetric.AggregationSum{}
	}
	if agg.Drop != nil {
		return sdkmetric.AggregationDrop{}
	}
	if agg.LastValue != nil {
		return sdkmetric.AggregationLastValue{}
	}
	if agg.ExplicitBucketHistogram != nil {
		return sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: agg.ExplicitBucketHistogram.Boundaries,
			NoMinMax:   !agg.ExplicitBucketHistogram.recordMinMaxBool(),
		}
	}
	return sdkmetric.AggregationDefault{}
}
