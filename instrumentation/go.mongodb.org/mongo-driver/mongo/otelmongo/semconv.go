// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"os"

	"go.opentelemetry.io/otel/attribute"
	semconv1210 "go.opentelemetry.io/otel/semconv/v1.21.0"
	semconv1260 "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	semconvOptIn     = "OTEL_SEMCONV_STABILITY_OPT_IN"
	semconvOptIn1260 = "mongo"
	semconvOptInDup  = "mongo/dup"
)

func appendAttrs[T string | int](
	attrs []attribute.KeyValue,
	semconvMap1210 func(T) attribute.KeyValue,
	semconvMap1260 func(T) attribute.KeyValue,
	val T,
) []attribute.KeyValue {
	switch os.Getenv(semconvOptIn) {
	case semconvOptIn1260:
		if semconvMap1260 != nil {
			attrs = append(attrs, semconvMap1260(val))
		}
	case semconvOptInDup:
		if semconvMap1210 != nil {
			attrs = append(attrs, semconvMap1210(val))
		}

		if semconvMap1260 != nil {
			attrs = append(attrs, semconvMap1260(val))
		}
	default:
		if semconvMap1210 != nil {
			attrs = append(attrs, semconvMap1210(val))
		}
	}

	return attrs
}

func appendOpNameAttrs(attrs []attribute.KeyValue, op string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.DBOperation, semconv1260.DBOperationName, op)
}

func appendDBNamespace(attrs []attribute.KeyValue, ns string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.DBName, semconv1260.DBNamespace, ns)
}

func appendDBStatement(attrs []attribute.KeyValue, stmt string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.DBStatement, semconv1260.DBQueryText, stmt)
}

func appendNetworkPort(attrs []attribute.KeyValue, p int) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.NetPeerPort, semconv1260.NetworkPeerPort, p)
}

func appendNetworkHost(attrs []attribute.KeyValue, h string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.NetPeerName, nil, h)
}

func appendNetworkAddress(attrs []attribute.KeyValue, addr string) []attribute.KeyValue {
	return appendAttrs(attrs, nil, semconv1260.NetworkPeerAddress, addr)
}

func appendNetworkTransport(attrs []attribute.KeyValue) []attribute.KeyValue {
	switch os.Getenv(semconvOptIn) {
	case semconvOptIn1260:
		attrs = append(attrs, semconv1260.NetworkTransportTCP)
	case semconvOptInDup:
		attrs = append(attrs, semconv1260.NetworkTransportTCP)
		fallthrough
	default:
		attrs = append(attrs, semconv1210.NetTransportTCP)
	}

	return attrs
}

func appendCollection(attrs []attribute.KeyValue, coll string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1210.DBMongoDBCollection, semconv1260.DBCollectionName, coll)
}
