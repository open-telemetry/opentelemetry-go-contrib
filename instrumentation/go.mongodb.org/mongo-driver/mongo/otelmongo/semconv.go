package otelmongo // import "go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

import (
	"os"

	"go.opentelemetry.io/otel/attribute"
	semconv1170 "go.opentelemetry.io/otel/semconv/v1.17.0"
	semconv1260 "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	semconvOptIn     = "OTEL_SEMCONV_STABILITY_OPT_IN"
	semconvOptIn1260 = "mongo"
	semconvOptInDup  = "mongo/dup"
)

func appendAttrs[T string | int](
	attrs []attribute.KeyValue,
	semconvMap1170 func(T) attribute.KeyValue,
	semconvMap1260 func(T) attribute.KeyValue,
	val T,
) []attribute.KeyValue {
	switch os.Getenv(semconvOptIn) {
	case semconvOptIn1260:
		if semconvMap1260 != nil {
			attrs = append(attrs, semconvMap1260(val))
		}
	case semconvOptInDup:
		if semconvMap1170 != nil {
			attrs = append(attrs, semconvMap1170(val))
		}

		if semconvMap1260 != nil {
			attrs = append(attrs, semconvMap1260(val))
		}
	default:
		if semconvMap1170 != nil {
			attrs = append(attrs, semconvMap1170(val))
		}
	}

	return attrs
}

func appendOpNameAttrs(attrs []attribute.KeyValue, op string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.DBOperation, semconv1260.DBOperationName, op)
}

func appendDBNamespace(attrs []attribute.KeyValue, ns string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.DBName, semconv1260.DBNamespace, ns)
}

func appendDBStatement(attrs []attribute.KeyValue, stmt string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.DBStatement, semconv1260.DBQueryText, stmt)
}

func appendNetworkPort(attrs []attribute.KeyValue, p int) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.NetPeerPort, semconv1260.NetworkPeerPort, p)
}

func appendNetworkHost(attrs []attribute.KeyValue, h string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.NetPeerName, nil, h)
}

func appendNetworkAddress(attrs []attribute.KeyValue, addr string) []attribute.KeyValue {
	return appendAttrs(attrs, nil, semconv1260.NetworkPeerAddress, addr)
}

func appendNetworkTransport(attrs []attribute.KeyValue) []attribute.KeyValue {
	optIn := os.Getenv(semconvOptIn)
	useSemconv1260 := optIn == semconvOptIn1260
	useSemconvDup := optIn == semconvOptInDup

	if useSemconv1260 || useSemconvDup {
		attrs = append(attrs, semconv1260.NetworkTransportTCP)
	}

	if !useSemconv1260 || useSemconvDup {
		attrs = append(attrs, semconv1170.NetTransportTCP)
	}

	return attrs
}

func appendCollection(attrs []attribute.KeyValue, coll string) []attribute.KeyValue {
	return appendAttrs(attrs, semconv1170.DBMongoDBCollection, semconv1260.DBCollectionName, coll)
}
