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
		version   string
		want      []attribute.KeyValue
	}{
		{
			name:      "no version",
			initAttrs: []attribute.KeyValue{},
			version:   "",
			want:      v1170,
		},
		{
			name:      "unsupported version",
			initAttrs: []attribute.KeyValue{},
			version:   "mongo/foo",
			want:      v1170,
		},
		{
			name:      "mongo",
			initAttrs: []attribute.KeyValue{},
			version:   "mongo",
			want:      v1260,
		},
		{
			name:      "mongo/dup",
			initAttrs: []attribute.KeyValue{},
			version:   "mongo/dup",
			want:      append(v1170, v1260...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(semconvOptIn, test.version)

			attrs := appendOpNameAttrs(test.initAttrs, opName)
			attrs = appendDBNamespace(attrs, dbNamespace)
			attrs = appendDBStatement(attrs, stmt)
			attrs = appendNetworkPort(attrs, port)
			attrs = appendNetworkHost(attrs, host)
			attrs = appendNetworkAddress(attrs, address)
			attrs = appendNetworkTransport(attrs)
			attrs = appendCollection(attrs, coll)

			assert.ElementsMatch(t, test.want, attrs)
		})
	}
}

func Benchmark_appendAttrs(b *testing.B) {
	ini := []attribute.KeyValue{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ini = appendOpNameAttrs(ini, "opName")
		ini = appendDBNamespace(ini, "dbNamespace")
		ini = appendDBStatement(ini, `{insert: "users"}`)
		ini = appendNetworkPort(ini, 1)
		ini = appendNetworkHost(ini, "host")
		ini = appendNetworkAddress(ini, "host:1")
		ini = appendNetworkTransport(ini)
		ini = appendCollection(ini, "coll")
	}
}
