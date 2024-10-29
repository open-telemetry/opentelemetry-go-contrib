// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package baggagecopy

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/baggage"
	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

var _ log.Processor = &processor{}

type processor struct {
	records []*log.Record
}

func (p *processor) OnEmit(ctx context.Context, r *log.Record) error {
	p.records = append(p.records, r)
	return nil
}

func (p *processor) Shutdown(ctx context.Context) error { return nil }

func (p *processor) ForceFlush(ctx context.Context) error { return nil }

func NewTestProcessor() *processor {
	return &processor{}
}

func TestLogProcessorAppendsAllBaggageAttributes(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(context.Background(), b)

	wrapped := &processor{}
	lp := log.NewLoggerProvider(
		log.WithProcessor(NewLogProcessor(AllowAllMembers)),
		log.WithProcessor(wrapped),
	)

	lp.Logger("test").Emit(ctx, api.Record{})

	require.Len(t, wrapped.records, 1)
	require.Equal(t, 1, wrapped.records[0].AttributesLen())

	want := []api.KeyValue{api.String("baggage.test", "baggage value")}
	var got []api.KeyValue
	wrapped.records[0].WalkAttributes(func(kv api.KeyValue) bool {
		got = append(got, kv)
		return true
	})

	require.Equal(t, want, got)
}

func TestLogProcessorAppendsBaggageAttributesWithHasPrefixPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(context.Background(), b)

	baggageKeyPredicate := func(m baggage.Member) bool {
		return strings.HasPrefix(m.Key(), "baggage.")
	}

	wrapped := &processor{}
	lp := log.NewLoggerProvider(
		log.WithProcessor(NewLogProcessor(baggageKeyPredicate)),
		log.WithProcessor(wrapped),
	)

	lp.Logger("test").Emit(ctx, api.Record{})

	require.Len(t, wrapped.records, 1)
	require.Equal(t, 1, wrapped.records[0].AttributesLen())

	want := []api.KeyValue{api.String("baggage.test", "baggage value")}
	var got []api.KeyValue
	wrapped.records[0].WalkAttributes(func(kv api.KeyValue) bool {
		got = append(got, kv)
		return true
	})

	require.Equal(t, want, got)
}

func TestLogProcessorAppendsBaggageAttributesWithRegexPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(context.Background(), b)

	expr := regexp.MustCompile(`^baggage\..*`)
	baggageKeyPredicate := func(m baggage.Member) bool {
		return expr.MatchString(m.Key())
	}

	wrapped := &processor{}
	lp := log.NewLoggerProvider(
		log.WithProcessor(NewLogProcessor(baggageKeyPredicate)),
		log.WithProcessor(wrapped),
	)

	lp.Logger("test").Emit(ctx, api.Record{})

	require.Len(t, wrapped.records, 1)
	require.Equal(t, 1, wrapped.records[0].AttributesLen())

	want := []api.KeyValue{api.String("baggage.test", "baggage value")}
	var got []api.KeyValue
	wrapped.records[0].WalkAttributes(func(kv api.KeyValue) bool {
		got = append(got, kv)
		return true
	})

	require.Equal(t, want, got)
}

func TestLogProcessorOnlyAddsBaggageEntriesThatMatchPredicate(t *testing.T) {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	b = addEntryToBaggage(t, b, "foo", "bar")
	ctx := baggage.ContextWithBaggage(context.Background(), b)

	baggageKeyPredicate := func(m baggage.Member) bool {
		return m.Key() == "baggage.test"
	}

	wrapped := &processor{}
	lp := log.NewLoggerProvider(
		log.WithProcessor(NewLogProcessor(baggageKeyPredicate)),
		log.WithProcessor(wrapped),
	)

	lp.Logger("test").Emit(ctx, api.Record{})

	require.Len(t, wrapped.records, 1)
	require.Equal(t, 1, wrapped.records[0].AttributesLen())

	want := []api.KeyValue{api.String("baggage.test", "baggage value")}
	var got []api.KeyValue
	wrapped.records[0].WalkAttributes(func(kv api.KeyValue) bool {
		got = append(got, kv)
		return true
	})

	require.Equal(t, want, got)
}

func TestZeroLogProcessorNoPanic(t *testing.T) {
	lp := new(LogProcessor)

	m, err := baggage.NewMember("key", "val")
	require.NoError(t, err)
	b, err := baggage.New(m)
	require.NoError(t, err)

	ctx := baggage.ContextWithBaggage(context.Background(), b)
	assert.NotPanics(t, func() {
		_ = lp.OnEmit(ctx, &log.Record{})
		_ = lp.Shutdown(ctx)
		_ = lp.ForceFlush(ctx)
	})
}
