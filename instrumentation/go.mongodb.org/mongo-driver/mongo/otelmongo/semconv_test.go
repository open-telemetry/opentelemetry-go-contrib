// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func Test_appendOpNameAttrs(t *testing.T) {
	const (
		opName      = "opName"
		dbNamespace = "dbNamespace"
		port        = 1
		host        = "host"
		address     = "host:1"
		stmt        = `{insert: "users"}`
		coll        = "coll"
	)

	v1170 := []attribute.KeyValue{
		{Key: "db.operation", Value: attribute.StringValue(opName)},
		{Key: "db.name", Value: attribute.StringValue(dbNamespace)},
		{Key: "db.statement", Value: attribute.StringValue(stmt)},
		{Key: "net.peer.port", Value: attribute.IntValue(port)},
		{Key: "net.peer.name", Value: attribute.StringValue(host)},
		{Key: "net.transport", Value: attribute.StringValue("ip_tcp")},
		{Key: "db.mongodb.collection", Value: attribute.StringValue("coll")},
	}

	v1260 := []attribute.KeyValue{
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
		versions  []string
		want      []attribute.KeyValue
	}{
		{
			name:      "no version",
			initAttrs: []attribute.KeyValue{},
			versions:  []string{},
			want:      v1170,
		},
		{
			name:      "unsupported version",
			initAttrs: []attribute.KeyValue{},
			versions:  []string{"mongo/meep"},
			want:      v1170,
		},
		{
			name:      "mongo/v1.26.0",
			initAttrs: []attribute.KeyValue{},
			versions:  []string{"mongo/v1.26.0"},
			want:      v1260,
		},
		{
			name:      "mongo/dup",
			initAttrs: []attribute.KeyValue{},
			versions:  []string{"mongo/dup"},
			want:      append(v1170, v1260...),
		},
		{
			name:      "mongo/dup and mongo/v1.26.0",
			initAttrs: []attribute.KeyValue{},
			versions:  []string{"mongo/dup", "mongo/v1.26.0"},
			want:      append(v1170, v1260...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reg := newSemconvRegistry(test.versions...)

			attrs := appendOpNameAttrs(test.initAttrs, reg, opName)
			attrs = appendDBNamespace(attrs, reg, dbNamespace)
			attrs = appendDBStatement(attrs, reg, stmt)
			attrs = appendNetworkPort(attrs, reg, port)
			attrs = appendNetworkHost(attrs, reg, host)
			attrs = appendNetworkAddress(attrs, reg, address)
			attrs = appendNetworkTransport(attrs, reg)
			attrs = appendCollection(attrs, reg, coll)

			assert.ElementsMatch(t, test.want, attrs)
		})
	}
}

func Benchmark_appendAttrs(b *testing.B) {
	reg := newSemconvRegistry("mongo/dup")
	ini := []attribute.KeyValue{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ini = appendOpNameAttrs(ini, reg, "opName")
		ini = appendDBNamespace(ini, reg, "dbNamespace")
		ini = appendDBStatement(ini, reg, `{insert: "users"}`)
		ini = appendNetworkPort(ini, reg, 1)
		ini = appendNetworkHost(ini, reg, "host")
		ini = appendNetworkAddress(ini, reg, "host:1")
		ini = appendNetworkTransport(ini, reg)
		ini = appendCollection(ini, reg, "coll")
	}
}
