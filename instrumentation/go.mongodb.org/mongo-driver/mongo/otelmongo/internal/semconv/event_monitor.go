// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package semconv provides semantic convention types and functionality.
package semconv // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo/internal/semconv"

import (
	"net"
	"os"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/otel/attribute"
	semconv1210 "go.opentelemetry.io/otel/semconv/v1.21.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// Constants for environment variable keys and versions.
const (
	semconvOptIn    = "OTEL_SEMCONV_STABILITY_OPT_IN"
	semconvOptInDup = "database/dup"
)

// EventMonitor is responsible for monitoring events with a specified semantic
// version.
type EventMonitor struct {
	version string
}

// NewEventMonitor creates an EventMonitor with the version set based on the
// OTEL_SEMCONV_STABILITY_OPT_IN environment variable.
func NewEventMonitor() EventMonitor {
	return EventMonitor{
		version: strings.ToLower(os.Getenv(semconvOptIn)),
	}
}

// AttributeOptions represents options for tracing attributes.
type AttributeOptions struct {
	collectionName           string
	commandAttributeDisabled bool
}

// AttributeOption is a function type that modifies AttributeOptions.
type AttributeOption func(*AttributeOptions)

// WithCollectionName is a functional option to set the collection name in
// AttributeOptions.
func WithCollectionName(collName string) AttributeOption {
	return func(opts *AttributeOptions) {
		opts.collectionName = collName
	}
}

// WithCommandAttributeDisabled is a functional option to enable or disable
// command attributes.
func WithCommandAttributeDisabled(disabled bool) AttributeOption {
	return func(opts *AttributeOptions) {
		opts.commandAttributeDisabled = disabled
	}
}

// hasOptIn returns true if the comma-separated version string contains the
// exact optIn value.
func hasOptIn(version, optIn string) bool {
	for _, v := range strings.Split(version, ",") {
		if strings.TrimSpace(v) == optIn {
			return true
		}
	}
	return false
}

// CommandStartedTraceAttrs generates trace attributes for a CommandStartedEvent
// based on the EventMonitor version.
func (m EventMonitor) CommandStartedTraceAttrs(
	evt *event.CommandStartedEvent,
	opts ...AttributeOption,
) []attribute.KeyValue {
	// Dup implies both v1.26.0 and v1.21.0
	if hasOptIn(m.version, semconvOptInDup) {
		return append(
			commandStartedTraceAttrs(evt, opts...),
			commandStartedTraceAttrsV1210(evt, opts...)...,
		)
	}

	return commandStartedTraceAttrs(evt, opts...)
}

// peerInfo extracts the hostname and port from a CommandStartedEvent.
func peerInfo(evt *event.CommandStartedEvent) (hostname string, port int) {
	hostname = evt.ConnectionID
	port = 27017 // Default MongoDB port

	host, portStr, err := net.SplitHostPort(hostname)
	if err != nil {
		// If there's an error (likely because there's no port), assume default port
		// and use ConnectionID as hostname
		return hostname, port
	}

	if parsedPort, err := strconv.Atoi(portStr); err == nil {
		port = parsedPort
	}

	return host, port
}

// sanitizeCommand converts a BSON command to a sanitized JSON string.
// TODO: Sanitize values where possible.
// TODO: Limit maximum size.
func sanitizeCommand(command bson.Raw) string {
	b, _ := bson.MarshalExtJSON(command, false, false)

	return string(b)
}

// commandStartedTraceAttrs generates trace attributes for the latest semantic
// version.
func commandStartedTraceAttrs(evt *event.CommandStartedEvent, setters ...AttributeOption) []attribute.KeyValue {
	opts := &AttributeOptions{}
	for _, set := range setters {
		set(opts)
	}

	attrs := []attribute.KeyValue{semconv.DBSystemNameMongoDB}

	attrs = append(
		attrs,
		semconv.DBOperationName(evt.CommandName),
		semconv.DBNamespace(evt.DatabaseName),
		semconv.NetworkTransportTCP,
	)

	hostname, port := peerInfo(evt)
	attrs = append(
		attrs,
		semconv.NetworkPeerPort(port),
		semconv.NetworkPeerAddress(net.JoinHostPort(hostname, strconv.Itoa(port))),
	)

	if !opts.commandAttributeDisabled {
		attrs = append(attrs, semconv.DBQueryText(sanitizeCommand(evt.Command)))
	}

	if opts.collectionName != "" {
		attrs = append(attrs, semconv.DBCollectionName(opts.collectionName))
	}

	return attrs
}

// commandStartedTraceAttrsV1210 generates trace attributes for semantic version
// 1.21.0.
func commandStartedTraceAttrsV1210(evt *event.CommandStartedEvent, setters ...AttributeOption) []attribute.KeyValue {
	opts := &AttributeOptions{}
	for _, set := range setters {
		set(opts)
	}

	attrs := []attribute.KeyValue{semconv1210.DBSystemMongoDB}

	attrs = append(
		attrs,
		semconv1210.DBOperation(evt.CommandName),
		semconv1210.DBName(evt.DatabaseName),
		semconv1210.NetTransportTCP,
	)

	hostname, port := peerInfo(evt)
	attrs = append(
		attrs,
		semconv1210.NetPeerPort(port),
		semconv1210.NetPeerName(hostname),
	)

	if !opts.commandAttributeDisabled {
		attrs = append(attrs, semconv1210.DBStatement(sanitizeCommand(evt.Command)))
	}

	if opts.collectionName != "" {
		attrs = append(attrs, semconv1210.DBMongoDBCollection(opts.collectionName))
	}

	return attrs
}
