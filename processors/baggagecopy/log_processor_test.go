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

func TestLogProcessorOnEmit(t *testing.T) {
	tests := []struct {
		name    string
		baggage baggage.Baggage
		filter  Filter
		want    []api.KeyValue
	}{
		{
			name: "all baggage attributes",
			baggage: func() baggage.Baggage {
				b, _ := baggage.New()
				b = addEntryToBaggage(t, b, "baggage.test", "baggage value")
				return b
			}(),
			filter: AllowAllMembers,
			want:   []api.KeyValue{api.String("baggage.test", "baggage value")},
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
			want: []api.KeyValue{api.String("baggage.test", "baggage value")},
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
			want: []api.KeyValue{api.String("baggage.test", "baggage value")},
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
			want: []api.KeyValue{api.String("baggage.test", "baggage value")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := baggage.ContextWithBaggage(context.Background(), tt.baggage)

			wrapped := &processor{}
			lp := log.NewLoggerProvider(
				log.WithProcessor(NewLogProcessor(tt.filter)),
				log.WithProcessor(wrapped),
			)

			lp.Logger("test").Emit(ctx, api.Record{})

			require.Len(t, wrapped.records, 1)
			require.Equal(t, len(tt.want), wrapped.records[0].AttributesLen())

			var got []api.KeyValue
			wrapped.records[0].WalkAttributes(func(kv api.KeyValue) bool {
				got = append(got, kv)
				return true
			})

			require.Equal(t, tt.want, got)
		})
	}
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
