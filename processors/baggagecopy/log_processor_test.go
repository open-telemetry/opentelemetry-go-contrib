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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

var _ log.Processor = &processor{}

type processor struct {
	records []*log.Record
}

func (*processor) Enabled(context.Context, log.EnabledParameters) bool {
	return true
}

func (p *processor) OnEmit(_ context.Context, r *log.Record) error {
	p.records = append(p.records, r)
	return nil
}

func (*processor) Shutdown(context.Context) error { return nil }

func (*processor) ForceFlush(context.Context) error { return nil }

func NewTestProcessor() *processor {
	return &processor{}
}

func TestLogProcessorOnEmit(t *testing.T) {
	tests := []struct {
		name    string
		baggage baggage.Baggage
		filter  Filter
		want    []attribute.KeyValue
	}{
		{
			name: "all baggage attributes",
			baggage: func() baggage.Baggage {
				b, _ := baggage.New()
				b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
				return b
			}(),
			filter: AllowAllMembers,
			want:   []attribute.KeyValue{attribute.String("baggage.test", "baggage value")},
		},
		{
			name: "baggage attributes with prefix",
			baggage: func() baggage.Baggage {
				b, _ := baggage.New()
				b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
				return b
			}(),
			filter: func(m baggage.Member) bool {
				return strings.HasPrefix(m.Key(), "baggage.")
			},
			want: []attribute.KeyValue{attribute.String("baggage.test", "baggage value")},
		},
		{
			name: "baggage attributes with regex",
			baggage: func() baggage.Baggage {
				b, _ := baggage.New()
				b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
				return b
			}(),
			filter: func(m baggage.Member) bool {
				return regexp.MustCompile(`^baggage\..*`).MatchString(m.Key())
			},
			want: []attribute.KeyValue{attribute.String("baggage.test", "baggage value")},
		},
		{
			name: "only adds baggage entries that match predicate",
			baggage: func() baggage.Baggage {
				b, _ := baggage.New()
				b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
				b = addEntryToBaggage(t, b, "foo", "bar")
				return b
			}(),
			filter: func(m baggage.Member) bool {
				return m.Key() == "baggage.test"
			},
			want: []attribute.KeyValue{attribute.String("baggage.test", "baggage value")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := baggage.ContextWithBaggage(t.Context(), tt.baggage)

			wrapped := &processor{}
			lp := log.NewLoggerProvider(
				log.WithProcessor(NewLogProcessor(tt.filter)),
				log.WithProcessor(wrapped),
			)

			lp.Logger("test").Emit(ctx, api.Record{})

			require.Len(t, wrapped.records, 1)
			require.Equal(t, len(tt.want), wrapped.records[0].AttributesLen())

			var got []attribute.KeyValue
			wrapped.records[0].WalkAttributes(func(kv attribute.KeyValue) bool {
				got = append(got, kv)
				return true
			})

			require.Equal(t, tt.want, got)
		})
	}
}

func TestLogProcessorOverwritesRecordAttributesByDefault(t *testing.T) {
	want := []attribute.KeyValue{
		attribute.String("user.id", "bob"),
		attribute.String("baggage.test", "baggage value"),
	}
	got := emitRecordWithUserID(t, NewLogProcessor(AllowAllMembers))
	assert.ElementsMatch(t, want, got)
}

func TestLogProcessorKeepsRecordAttributes(t *testing.T) {
	want := []attribute.KeyValue{
		attribute.String("user.id", "alice"),
		attribute.String("baggage.test", "baggage value"),
	}
	got := emitRecordWithUserID(t, NewLogProcessor(AllowAllMembers, WithOverwriteExistingAttributes(false)))
	assert.ElementsMatch(t, want, got)
}

// emitRecordWithUserID emits a record that already has a user.id attribute, in
// a context carrying baggage with the same key, and returns the attributes the
// next processor sees.
func emitRecordWithUserID(t *testing.T, proc *LogProcessor) []attribute.KeyValue {
	b, _ := baggage.New()
	b = addEntryToBaggage(t, b, "user.id", "bob")
	b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
	ctx := baggage.ContextWithBaggage(t.Context(), b)

	wrapped := &processor{}
	lp := log.NewLoggerProvider(
		log.WithProcessor(proc),
		log.WithProcessor(wrapped),
	)

	var r api.Record
	r.AddAttributes(attribute.String("user.id", "alice"))
	lp.Logger("test").Emit(ctx, r)

	require.Len(t, wrapped.records, 1)
	var got []attribute.KeyValue
	wrapped.records[0].WalkAttributes(func(kv attribute.KeyValue) bool {
		got = append(got, kv)
		return true
	})
	return got
}

func TestZeroLogProcessorNoPanic(t *testing.T) {
	lp := new(LogProcessor)

	m, err := baggage.NewMember("key", "val")
	require.NoError(t, err)
	b, err := baggage.New(m)
	require.NoError(t, err)

	ctx := baggage.ContextWithBaggage(t.Context(), b)
	assert.NotPanics(t, func() {
		_ = lp.OnEmit(ctx, &log.Record{})
		_ = lp.Shutdown(ctx)
		_ = lp.ForceFlush(ctx)
	})
}
