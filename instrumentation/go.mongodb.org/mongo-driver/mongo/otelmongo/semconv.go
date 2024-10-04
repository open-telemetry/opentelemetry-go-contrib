// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"go.opentelemetry.io/otel/attribute"

	semconv1170 "go.opentelemetry.io/otel/semconv/v1.17.0"
	semconv1260 "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	// "OTEL_SEMCONV_STABILITY_OPT_IN" can be used to opt into semconv/v1.26.0. See
	// doc.go for more information.
	semconvOptIn     = "OTEL_SEMCONV_STABILITY_OPT_IN"
	semconvOptIn1260 = "mongo/v1.26.0"
	semconvOptIn1170 = "mongo/v1.17.0"
	semconvOptInDup  = "mongo/dup" // v1.17.0 and all other supported semconv
)

type semconvRegistry struct {
	versions []string
	dup      bool

	opName           map[string]func(string) attribute.KeyValue
	dbNamespace      map[string]func(string) attribute.KeyValue
	dbStatement      map[string]func(string) attribute.KeyValue
	networkPort      map[string]func(int) attribute.KeyValue
	networkHost      map[string]func(string) attribute.KeyValue
	networkAddress   map[string]func(string) attribute.KeyValue
	networkTransport map[string]func(string) attribute.KeyValue
	collection       map[string]func(string) attribute.KeyValue
}

func newSemconvRegistry(versions ...string) *semconvRegistry {
	reg := &semconvRegistry{
		opName:           make(map[string]func(string) attribute.KeyValue),
		dbNamespace:      make(map[string]func(string) attribute.KeyValue),
		dbStatement:      make(map[string]func(string) attribute.KeyValue),
		networkPort:      make(map[string]func(int) attribute.KeyValue),
		networkHost:      make(map[string]func(string) attribute.KeyValue),
		networkAddress:   make(map[string]func(string) attribute.KeyValue),
		networkTransport: make(map[string]func(string) attribute.KeyValue),
		collection:       make(map[string]func(string) attribute.KeyValue),
	}

	// Don't include unknown versions
	for _, version := range versions {
		if version == semconvOptInDup {
			reg.dup = true
		}

		if version == semconvOptIn1170 || version == semconvOptIn1260 {
			reg.versions = append(reg.versions, version)
		}
	}

	// If we didn't pick up any versions and we are not duplicating, then use
	// the default v1.17.0.
	if len(reg.versions) == 0 && !reg.dup {
		reg.versions = append(reg.versions, semconvOptIn1170)
	}

	// v1.17.0
	reg.opName[semconvOptIn1170] = semconv1170.DBOperation
	reg.dbNamespace[semconvOptIn1170] = semconv1170.DBName
	reg.dbStatement[semconvOptIn1170] = semconv1170.DBStatement
	reg.networkPort[semconvOptIn1170] = semconv1170.NetPeerPort
	reg.networkHost[semconvOptIn1170] = semconv1170.NetPeerName
	reg.collection[semconvOptIn1170] = semconv1170.DBMongoDBCollection
	reg.networkTransport[semconvOptIn1170] = func(string) attribute.KeyValue { return semconv1170.NetTransportTCP }

	// v1.26.0
	reg.opName[semconvOptIn1260] = semconv1260.DBOperationName
	reg.dbNamespace[semconvOptIn1260] = semconv1260.DBNamespace
	reg.dbStatement[semconvOptIn1260] = semconv1260.DBQueryText
	reg.networkPort[semconvOptIn1260] = semconv1260.NetworkPeerPort
	reg.networkAddress[semconvOptIn1260] = semconv1260.NetworkPeerAddress
	reg.collection[semconvOptIn1260] = semconv1260.DBCollectionName
	reg.networkTransport[semconvOptIn1260] = func(string) attribute.KeyValue { return semconv1260.NetworkTransportTCP }

	return reg
}

func appendAttrs[T string | int](
	attrs []attribute.KeyValue,
	reg *semconvRegistry,
	semconvMap map[string]func(T) attribute.KeyValue,
	val T,
) []attribute.KeyValue {
	if reg.dup {
		for _, fn := range semconvMap {
			attrs = append(attrs, fn(val))
		}

		return attrs
	}

	for _, version := range reg.versions {
		fn, ok := semconvMap[version]
		if ok {
			attrs = append(attrs, fn(val))
		}
	}

	return attrs
}

func appendOpNameAttrs(attrs []attribute.KeyValue, reg *semconvRegistry, op string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.opName, op)
}

func appendDBNamespace(attrs []attribute.KeyValue, reg *semconvRegistry, ns string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.dbNamespace, ns)
}

func appendDBStatement(attrs []attribute.KeyValue, reg *semconvRegistry, stmt string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.dbStatement, stmt)
}

func appendNetworkPort(attrs []attribute.KeyValue, reg *semconvRegistry, p int) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.networkPort, p)
}

func appendNetworkHost(attrs []attribute.KeyValue, reg *semconvRegistry, h string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.networkHost, h)
}

func appendNetworkAddress(attrs []attribute.KeyValue, reg *semconvRegistry, addr string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.networkAddress, addr)
}

func appendNetworkTransport(attrs []attribute.KeyValue, reg *semconvRegistry) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.networkTransport, "")
}

func appendCollection(attrs []attribute.KeyValue, reg *semconvRegistry, coll string) []attribute.KeyValue {
	return appendAttrs(attrs, reg, reg.collection, coll)
}
