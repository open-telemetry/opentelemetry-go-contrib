// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.
// Copyright 2020 The OpenTelemetry Authors

// Package mongo provides functions to trace the mongodb/mongo-go-driver package (https://github.com/mongodb/mongo-go-driver).
// It support v0.2.0 of github.com/mongodb/mongo-go-driver
//
// `NewMonitor` will return an event.CommandMonitor which is used to trace requests.
package mongo

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
)

type spanKey struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	sync.Mutex
	spans map[spanKey]trace.Span
	cfg   config
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	hostname, port := peerInfo(evt)
	b, _ := bson.MarshalExtJSON(evt.Command, false, false)
	attrs := []kv.KeyValue{
		ServiceName(m.cfg.serviceName),
		ResourceName("mongo." + evt.CommandName),
		DBInstance(evt.DatabaseName),
		DBStatement(string(b)),
		DBType("mongo"),
		PeerHostname(hostname),
		PeerPort(port),
	}
	opts := []trace.StartOption{
		trace.WithAttributes(attrs...),
	}
	_, span := m.cfg.tracer.Start(ctx, "mongodb.query", opts...)
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	m.spans[key] = span
	m.Unlock()
}

func (m *monitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	m.Finished(&evt.CommandFinishedEvent, nil)
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	m.Finished(&evt.CommandFinishedEvent, fmt.Errorf("%s", evt.Failure))
}

func (m *monitor) Finished(evt *event.CommandFinishedEvent, err error) {
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	m.Lock()
	span, ok := m.spans[key]
	if ok {
		delete(m.spans, key)
	}
	m.Unlock()
	if !ok {
		return
	}
	span.SetAttributes(Error(err != nil))

	if err != nil {
		span.SetAttributes(ErrorMsg(err.Error()))
	}

	span.End()
}

// NewMonitor creates a new mongodb event CommandMonitor.
func NewMonitor(opts ...Option) *event.CommandMonitor {
	cfg := newConfig(opts...)
	m := &monitor{
		spans: make(map[spanKey]trace.Span),
		cfg:   cfg,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

func peerInfo(evt *event.CommandStartedEvent) (hostname, port string) {
	hostname = evt.ConnectionID
	port = "27017"
	if idx := strings.IndexByte(hostname, '['); idx >= 0 {
		hostname = hostname[:idx]
	}
	if idx := strings.IndexByte(hostname, ':'); idx >= 0 {
		port = hostname[idx+1:]
		hostname = hostname[:idx]
	}
	return hostname, port
}
