// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package isolate

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	logapi "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

const testAttrCount = 10

var testCtx = context.WithValue(context.Background(), "k", "v") //nolint // Simplify for testing.

func TestLogProcessorOnEmit(t *testing.T) {
	wrapped := &processor{ReturnErr: assert.AnError}

	p := NewLogProcessor(wrapped)

	var r log.Record
	for i := 0; i < testAttrCount; i++ {
		r.AddAttributes(logapi.Int(strconv.Itoa(i), i))
	}

	assert.ErrorIs(t, p.OnEmit(testCtx, r), assert.AnError)

	// Assert passthrough of the arguments.
	if assert.Len(t, wrapped.OnEmitCalls, 1) {
		assert.Equal(t, testCtx, wrapped.OnEmitCalls[0].Ctx)
		assert.Equal(t, r, wrapped.OnEmitCalls[0].Record)
	}

	// Assert that the record is not being affected by subsequent modifications.
	r.AddAttributes(logapi.String("foo", "bar"))
	assert.Equal(t, testAttrCount, wrapped.OnEmitCalls[0].Record.AttributesLen(), "should be isolated from subsequent modifications")
}

func TestLogProcessorEnabled(t *testing.T) {
	wrapped := &processor{}

	p := NewLogProcessor(wrapped)

	var r log.Record
	for i := 0; i < testAttrCount; i++ {
		r.AddAttributes(logapi.Int(strconv.Itoa(i), i))
	}

	assert.True(t, p.Enabled(testCtx, r))

	// Assert passthrough of the arguments.
	if assert.Len(t, wrapped.EnabledCalls, 1) {
		assert.Equal(t, testCtx, wrapped.EnabledCalls[0].Ctx)
		assert.Equal(t, r, wrapped.EnabledCalls[0].Record)
	}

	// Assert that the record is not being affected by subsequent modifications.
	r.AddAttributes(logapi.String("foo", "bar"))
	assert.Equal(t, testAttrCount, wrapped.EnabledCalls[0].Record.AttributesLen(), "should be isolated from subsequent modifications")
}

type args struct {
	Ctx    context.Context
	Record log.Record
}

type processor struct {
	ReturnErr error

	OnEmitCalls     []args
	EnabledCalls    []args
	ForceFlushCalls []context.Context
	ShutdownCalls   []context.Context
}

func (p *processor) OnEmit(ctx context.Context, r log.Record) error {
	p.OnEmitCalls = append(p.OnEmitCalls, args{ctx, r})
	return p.ReturnErr
}

func (p *processor) Enabled(ctx context.Context, r log.Record) bool {
	p.EnabledCalls = append(p.EnabledCalls, args{ctx, r})
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

func BenchmarkLogProcessor(b *testing.B) {
	var ok bool
	var err error

	var r log.Record
	r.SetBody(logapi.StringValue("message"))

	var rWithShared log.Record
	for i := 0; i < testAttrCount; i++ {
		rWithShared.AddAttributes(logapi.Int(strconv.Itoa(i), i))
	}

	testCases := []struct {
		desc string
		r    log.Record
	}{
		{
			desc: "Record without shared data",
			r:    r,
		},
		{
			desc: "Record with shared data",
			r:    rWithShared,
		},
	}

	p := NewLogProcessor(noopProcessor{})

	for _, tc := range testCases {
		b.Run(tc.desc, func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				ok = p.Enabled(testCtx, tc.r)
				err = p.OnEmit(testCtx, tc.r)
			}
		})
	}

	_, _ = ok, err
}
