// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	traceID     = trace.TraceID([16]byte{14, 54, 12})
	spanID      = trace.SpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 1})
	childSpanID = trace.SpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 2})
	headerFmt   = "\x00\x00\x0e6\f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00%s\x02%s"
)

func TestFields(t *testing.T) {
	b := Binary{}
	fields := b.Fields()
	if len(fields) != 1 {
		t.Fatalf("Got %d fields, expected 1", len(fields))
	}
	if fields[0] != "grpc-trace-bin" {
		t.Errorf("Got fields[0] == %s, expected grpc-trace-bin", fields[0])
	}
}

func TestInject(t *testing.T) {
	prop := Binary{}
	for _, tt := range []struct {
		desc       string
		scc        trace.SpanContextConfig
		wantHeader string
	}{
		{
			desc:       "empty",
			scc:        trace.SpanContextConfig{},
			wantHeader: "",
		},
		{
			desc: "valid spancontext, sampled",
			scc: trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			},
			wantHeader: fmt.Sprintf(headerFmt, "\x01", "\x01"),
		},
		{
			desc: "valid spancontext, not sampled",
			scc: trace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  spanID,
			},
			wantHeader: fmt.Sprintf(headerFmt, "\x01", "\x00"),
		},
		{
			desc: "valid spancontext, with unsupported bit set in traceflags",
			scc: trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: 0xff,
			},
			wantHeader: fmt.Sprintf(headerFmt, "\x01", "\x01"),
		},
		{
			desc:       "invalid spancontext",
			scc:        trace.SpanContextConfig{},
			wantHeader: "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			header := http.Header{}
			ctx := context.Background()
			if sc := trace.NewSpanContext(tt.scc); sc.IsValid() {
				ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
			}
			prop.Inject(ctx, propagation.HeaderCarrier(header))

			gotHeader := header.Get("grpc-trace-bin")
			if gotHeader != tt.wantHeader {
				t.Errorf("Got header = %q, want %q", gotHeader, tt.wantHeader)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	prop := Binary{}
	for _, tt := range []struct {
		desc    string
		header  string
		wantScc trace.SpanContextConfig
	}{
		{
			desc:    "empty",
			header:  "",
			wantScc: trace.SpanContextConfig{},
		},
		{
			desc:    "header not binary",
			header:  "5435j345io34t5904w3jt894j3t854w89tp95jgt9",
			wantScc: trace.SpanContextConfig{},
		},
		{
			desc:   "valid binary header",
			header: fmt.Sprintf(headerFmt, "\x02", "\x00"),
			wantScc: trace.SpanContextConfig{
				TraceID: traceID,
				SpanID:  childSpanID,
			},
		},
		{
			desc:   "valid binary and sampled",
			header: fmt.Sprintf(headerFmt, "\x02", "\x01"),
			wantScc: trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     childSpanID,
				TraceFlags: trace.FlagsSampled,
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			header := http.Header{
				http.CanonicalHeaderKey("grpc-trace-bin"): []string{tt.header},
			}

			ctx := context.Background()
			ctx = prop.Extract(ctx, propagation.HeaderCarrier(header))
			gotSc := trace.SpanContextFromContext(ctx)
			comparer := cmp.Comparer(func(a, b trace.SpanContext) bool {
				// Do not compare remote field, it is unset on empty
				// SpanContext.
				newA := a.WithRemote(b.IsRemote())
				return newA.Equal(b)
			})
			if diff := cmp.Diff(gotSc, trace.NewSpanContext(tt.wantScc), comparer); diff != "" {
				t.Errorf("%s: -got +want %s", tt.desc, diff)
			}
		})
	}
}
