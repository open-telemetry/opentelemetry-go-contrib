// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"

	"go.opentelemetry.io/otel/attribute"
	semconv1210 "go.opentelemetry.io/otel/semconv/v1.21.0"
	semconv1260 "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestNewEventMonitor(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "Default Version",
			version: "",
			want:    "",
		},
		{
			name:    "Version 1260",
			version: semconvOptIn1260,
			want:    "database",
		},
		{
			name:    "Duplicate Version",
			version: semconvOptInDup,
			want:    "database/dup",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(semconvOptIn, test.version)

			monitor := NewEventMonitor()
			assert.Equal(t, test.want, monitor.version, "Expected version does not match")
		})
	}
}

func TestPeerInfo(t *testing.T) {
	// Test cases for peerInfo
	tests := []struct {
		name         string
		connectionID string
		wantHostname string
		wantPort     int
	}{
		{"No Port", "localhost", "localhost", 27017},
		{"With Port", "localhost:12345", "localhost", 12345},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &event.CommandStartedEvent{ConnectionID: tt.connectionID}
			hostname, port := peerInfo(evt)
			assert.Equal(t, tt.wantHostname, hostname, "Hostname does not match")
			assert.Equal(t, tt.wantPort, port, "Port does not match")
		})
	}
}

func TestCommandStartedTraceAttrs(t *testing.T) {
	const (
		opName      = "opName"
		dbNamespace = "dbNamespace"
		port        = 1
		host        = "host"
		address     = "host:1"
		stmt        = `{"insert":"users"}`
		coll        = "coll"
	)

	v1210 := []attribute.KeyValue{
		semconv1210.DBSystemMongoDB,
		{Key: "db.operation", Value: attribute.StringValue(opName)},
		{Key: "db.name", Value: attribute.StringValue(dbNamespace)},
		{Key: "db.statement", Value: attribute.StringValue(stmt)},
		{Key: "net.peer.port", Value: attribute.IntValue(port)},
		{Key: "net.peer.name", Value: attribute.StringValue(host)},
		{Key: "net.transport", Value: attribute.StringValue("ip_tcp")},
		{Key: "db.mongodb.collection", Value: attribute.StringValue("coll")},
	}

	v1260 := []attribute.KeyValue{
		semconv1260.DBSystemMongoDB,
		{Key: "db.operation.name", Value: attribute.StringValue(opName)},
		{Key: "db.namespace", Value: attribute.StringValue(dbNamespace)},
		{Key: "db.query.text", Value: attribute.StringValue(stmt)},
		{Key: "network.peer.port", Value: attribute.IntValue(port)},
		{Key: "network.peer.address", Value: attribute.StringValue(address)},
		{Key: "network.transport", Value: attribute.StringValue("tcp")},
		{Key: "db.collection.name", Value: attribute.StringValue("coll")},
	}

	tests := []struct {
		name      string
		initAttrs []attribute.KeyValue
		version   string
		want      []attribute.KeyValue
	}{
		{
			name:      "no version",
			initAttrs: []attribute.KeyValue{},
			version:   "",
			want:      v1210,
		},
		{
			name:      "unsupported version",
			initAttrs: []attribute.KeyValue{},
			version:   "database/foo",
			want:      v1210,
		},
		{
			name:      "database",
			initAttrs: []attribute.KeyValue{},
			version:   "database",
			want:      v1260,
		},
		{
			name:      "database/dup",
			initAttrs: []attribute.KeyValue{},
			version:   "database/dup",
			want:      append(v1210, v1260...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(semconvOptIn, test.version)

			stmtBytes, err := bson.Marshal(bson.D{{Key: "insert", Value: "users"}})
			assert.NoError(t, err)

			monitor := NewEventMonitor()
			attrs := monitor.CommandStartedTraceAttrs(&event.CommandStartedEvent{
				DatabaseName: dbNamespace,
				CommandName:  opName,
				Command:      bson.Raw(stmtBytes),
				ConnectionID: net.JoinHostPort(host, strconv.FormatInt(int64(port), 10)),
			}, WithCollectionName(coll))

			assert.ElementsMatch(t, test.want, attrs)
		})
	}
}
