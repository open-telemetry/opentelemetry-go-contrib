// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

var severities = []api.Severity{
	api.SeverityTrace, api.SeverityTrace1, api.SeverityTrace2, api.SeverityTrace3, api.SeverityTrace4,
	api.SeverityDebug, api.SeverityDebug1, api.SeverityDebug2, api.SeverityDebug3, api.SeverityDebug4,
	api.SeverityInfo, api.SeverityInfo1, api.SeverityInfo2, api.SeverityInfo3, api.SeverityInfo4,
	api.SeverityWarn, api.SeverityWarn1, api.SeverityWarn2, api.SeverityWarn3, api.SeverityWarn4,
	api.SeverityError, api.SeverityError1, api.SeverityError2, api.SeverityError3, api.SeverityError4,
	api.SeverityFatal, api.SeverityFatal1, api.SeverityFatal2, api.SeverityFatal3, api.SeverityFatal4,
}

type args struct {
	Ctx    context.Context
	Record *log.Record
}

type processor struct {
	ReturnErr error

	OnEmitCalls     []args
	EnabledCalls    []args
	ForceFlushCalls []context.Context
	ShutdownCalls   []context.Context
}

func (p *processor) OnEmit(ctx context.Context, r *log.Record) error {
	p.OnEmitCalls = append(p.OnEmitCalls, args{ctx, r})
	return p.ReturnErr
}

func (p *processor) Enabled(ctx context.Context, r log.Record) bool {
	p.EnabledCalls = append(p.EnabledCalls, args{ctx, &r})
	return true
}

func (p *processor) Shutdown(ctx context.Context) error {
	p.ShutdownCalls = append(p.ShutdownCalls, ctx)
	return p.ReturnErr
}

func (p *processor) ForceFlush(ctx context.Context) error {
	p.ForceFlushCalls = append(p.ForceFlushCalls, ctx)
	return p.ReturnErr
}

func (p *processor) Reset() {
	p.OnEmitCalls = p.OnEmitCalls[:0]
	p.EnabledCalls = p.EnabledCalls[:0]
	p.ShutdownCalls = p.ShutdownCalls[:0]
	p.ForceFlushCalls = p.ForceFlushCalls[:0]
}

func TestLogProcessorOnEmit(t *testing.T) {
	t.Run("Passthrough", func(t *testing.T) {
		wrapped := &processor{ReturnErr: assert.AnError}

		p := NewLogProcessor(wrapped, api.SeverityTrace1)
		ctx := context.Background()
		r := &log.Record{}
		for _, sev := range severities {
			r.SetSeverity(sev)
			assert.ErrorIs(t, p.OnEmit(ctx, r), assert.AnError, sev.String())

			if assert.Lenf(t, wrapped.OnEmitCalls, 1, "Record with severity %s not passed-through", sev) {
				assert.Equal(t, ctx, wrapped.OnEmitCalls[0].Ctx, sev.String())
				assert.Equal(t, r, wrapped.OnEmitCalls[0].Record, sev.String())
			}
			wrapped.Reset()
		}
	})

	t.Run("Dropped", func(t *testing.T) {
		wrapped := &processor{ReturnErr: assert.AnError}

		p := NewLogProcessor(wrapped, api.SeverityFatal4+1)
		ctx := context.Background()
		r := &log.Record{}
		for _, sev := range severities {
			r.SetSeverity(sev)
			assert.NoError(t, p.OnEmit(ctx, r), assert.AnError, sev.String())

			if !assert.Lenf(t, wrapped.OnEmitCalls, 0, "Record with severity %s passed-through", sev) {
				wrapped.Reset()
			}
		}
	})
}

func TestLogProcessorEnabled(t *testing.T) {
	t.Run("Passthrough", func(t *testing.T) {
		wrapped := &processor{}

		p := NewLogProcessor(wrapped, api.SeverityTrace1)
		ctx := context.Background()
		r := &log.Record{}
		for _, sev := range severities {
			r.SetSeverity(sev)
			assert.True(t, p.Enabled(ctx, *r), sev.String())

			if assert.Lenf(t, wrapped.EnabledCalls, 1, "Record with severity %s not passed-through", sev) {
				assert.Equal(t, ctx, wrapped.EnabledCalls[0].Ctx, sev.String())
				assert.Equal(t, r, wrapped.EnabledCalls[0].Record, sev.String())
			}
			wrapped.Reset()
		}
	})

	t.Run("NotEnabled", func(t *testing.T) {
		wrapped := &processor{}

		p := NewLogProcessor(wrapped, api.SeverityFatal4+1)
		ctx := context.Background()
		r := &log.Record{}
		for _, sev := range severities {
			r.SetSeverity(sev)
			assert.False(t, p.Enabled(ctx, *r), sev.String())

			if !assert.Lenf(t, wrapped.EnabledCalls, 0, "Record with severity %s passed-through", sev) {
				wrapped.Reset()
			}
		}
	})
}

func TestLogProcessorForceFlushPassthrough(t *testing.T) {
	wrapped := &processor{ReturnErr: assert.AnError}

	p := NewLogProcessor(wrapped, api.SeverityTrace1)
	ctx := context.Background()
	assert.ErrorIs(t, p.ForceFlush(ctx), assert.AnError)
	assert.Len(t, wrapped.ForceFlushCalls, 1, "ForceFlush not passed-through")
}

func TestLogProcessorShutdownPassthrough(t *testing.T) {
	wrapped := &processor{ReturnErr: assert.AnError}

	p := NewLogProcessor(wrapped, api.SeverityTrace1)
	ctx := context.Background()
	assert.ErrorIs(t, p.Shutdown(ctx), assert.AnError)
	assert.Len(t, wrapped.ShutdownCalls, 1, "Shutdown not passed-through")
}

func TestLogProcessorNilDownstream(t *testing.T) {
	p := NewLogProcessor(nil, api.SeverityTrace1)
	ctx := context.Background()
	r := new(log.Record)
	r.SetSeverity(api.SeverityTrace1)
	assert.NotPanics(t, func() {
		assert.NoError(t, p.OnEmit(ctx, r))
		assert.False(t, p.Enabled(ctx, *r))
		assert.NoError(t, p.ForceFlush(ctx))
		assert.NoError(t, p.Shutdown(ctx))
	})
}

func BenchmarkLogProcessor(b *testing.B) {
	r := new(log.Record)
	r.SetSeverity(api.SeverityTrace)
	ctx := context.Background()

	type combo interface {
		log.Processor
		filterProcessor
	}

	run := func(p combo) func(b *testing.B) {
		return func(b *testing.B) {
			var err error
			var enabled bool
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				enabled = p.Enabled(ctx, *r)
				err = p.OnEmit(ctx, r)
			}

			_, _ = err, enabled
		}
	}

	b.Run("Base", run(defaultProcessor))
	b.Run("Enabled", run(NewLogProcessor(nil, api.SeverityTrace)))
	b.Run("Disabled", run(NewLogProcessor(nil, api.SeverityDebug)))
}
