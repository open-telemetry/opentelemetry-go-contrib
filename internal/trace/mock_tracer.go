// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"sync/atomic"

	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"go.opentelemetry.io/contrib/internal/trace/parent"
)

type Provider struct {
	tracersLock sync.Mutex
	tracers     map[string]*Tracer
}

var _ oteltrace.Provider = &Provider{}

func (p *Provider) Tracer(name string, _ ...oteltrace.TracerOption) oteltrace.Tracer {
	p.tracersLock.Lock()
	defer p.tracersLock.Unlock()
	if p.tracers == nil {
		p.tracers = make(map[string]*Tracer)
	}
	if tracer, ok := p.tracers[name]; ok {
		return tracer
	}
	tracer := NewTracer(name)
	p.tracers[name] = tracer
	return tracer
}

// Tracer is a simple tracer used for testing purpose only.
// SpanID is atomically increased every time a new span is created.
type Tracer struct {
	// StartSpanID is used to initialize span ID. It is incremented
	// by one every time a new span is created.
	//
	// StartSpanID has to be aligned for 64-bit atomic operations.
	StartSpanID uint64

	// Name of the tracer, received from the provider.
	Name string

	// Sampled specifies if the new span should be sampled or not.
	Sampled bool

	// OnSpanStarted is called every time a new span is started.
	OnSpanStarted func(span *Span)

	endedSpansLock sync.Mutex
	endedSpans     []*Span
}

var _ oteltrace.Tracer = (*Tracer)(nil)

func NewTracer(name string) *Tracer {
	return &Tracer{
		Name: name,
	}
}

func (mt *Tracer) EndedSpans() []*Span {
	var endedSpans []*Span

	mt.endedSpansLock.Lock()
	endedSpans, mt.endedSpans = mt.endedSpans, nil
	mt.endedSpansLock.Unlock()

	return endedSpans
}

func (mt *Tracer) addEndedSpan(span *Span) {
	mt.endedSpansLock.Lock()
	mt.endedSpans = append(mt.endedSpans, span)
	mt.endedSpansLock.Unlock()
}

// WithSpan does nothing except creating a new span and executing the
// body.
func (mt *Tracer) WithSpan(ctx context.Context, name string, body func(context.Context) error, opts ...oteltrace.StartOption) error {
	ctx, span := mt.Start(ctx, name, opts...)
	defer span.End()

	return body(ctx)
}

// Start starts a new Span and puts it into the context.
//
// The function generates a new random TraceID if either there is no
// parent SpanContext in context or the WithNewRoot option is passed
// to the function. Otherwise the function will take the TraceID from
// parent SpanContext.
//
// Currently no other StartOption has any effect here.
func (mt *Tracer) Start(ctx context.Context, name string, o ...oteltrace.StartOption) (context.Context, oteltrace.Span) {
	var opts oteltrace.StartConfig
	for _, op := range o {
		op(&opts)
	}
	var span *Span
	var sc oteltrace.SpanContext

	parentSpanContext, _, links := parent.GetSpanContextAndLinks(ctx, opts.NewRoot)
	parentSpanID := parentSpanContext.SpanID

	if !parentSpanContext.IsValid() {
		sc = oteltrace.SpanContext{}
		_, _ = rand.Read(sc.TraceID[:])
		if mt.Sampled {
			sc.TraceFlags = oteltrace.FlagsSampled
		}
	} else {
		sc = parentSpanContext
	}

	binary.BigEndian.PutUint64(sc.SpanID[:], atomic.AddUint64(&mt.StartSpanID, 1))
	span = &Span{
		sc:           sc,
		tracer:       mt,
		Name:         name,
		Attributes:   nil,
		ParentSpanID: parentSpanID,
		Links:        make(map[oteltrace.SpanContext][]label.KeyValue),
	}
	if len(opts.Attributes) > 0 {
		span.SetAttributes(opts.Attributes...)
	}
	span.Kind = opts.SpanKind
	if mt.OnSpanStarted != nil {
		mt.OnSpanStarted(span)
	}

	for _, link := range links {
		span.Links[link.SpanContext] = link.Attributes
	}
	for _, link := range opts.Links {
		span.Links[link.SpanContext] = link.Attributes
	}

	return oteltrace.ContextWithSpan(ctx, span), span
}

// NewProviderAndTracer return mock provider and tracer.
func NewProviderAndTracer(tracerName string) (*Provider, *Tracer) {
	var provider Provider
	tracer := provider.Tracer(tracerName)

	return &provider, tracer.(*Tracer)
}
