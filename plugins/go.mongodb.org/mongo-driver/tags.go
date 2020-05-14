// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.
// Copyright 2020 The OpenTelemetry Authors

// Package ext contains a set of Datadog-specific constants. Most of them are used
// for setting span metadata.
package mongo

import "go.opentelemetry.io/otel/api/kv"

const (
	TargetHostKey   = kv.Key("out.host")
	TargetPortKey   = kv.Key("out.port")
	HTTPMethodKey   = kv.Key("http.method")
	HTTPCodeKey     = kv.Key("http.code")
	HTTPURLKey      = kv.Key("http.url")
	SpanTypeKey     = kv.Key("span.type")
	ServiceNameKey  = kv.Key("service.name")
	ResourceNameKey = kv.Key("resource.name")
	ErrorKey        = kv.Key("error")
	ErrorMsgKey     = kv.Key("error.msg")
)

// TargetHost sets the target host address.
func TargetHost(targetHost string) kv.KeyValue {
	return TargetHostKey.String(targetHost)
}

// TargetPort sets the target host port.
func TargetPort(targetPort string) kv.KeyValue {
	return TargetPortKey.String(targetPort)
}

// HTTPMethod specifies the HTTP method used in a span.
func HTTPMethod(httpMethod string) kv.KeyValue {
	return HTTPMethodKey.String(httpMethod)
}

// HTTPCode sets the HTTP status code as a attribute.
func HTTPCode(httpCode string) kv.KeyValue {
	return HTTPCodeKey.String(httpCode)
}

// HTTPURL sets the HTTP URL for a span.
func HTTPURL(httpURL string) kv.KeyValue {
	return HTTPURLKey.String(httpURL)
}

// SpanType defines the Span type (web, db, cache).
func SpanType(spanType string) kv.KeyValue {
	return SpanTypeKey.String(spanType)
}

// ServiceName defines the Service name for this Span.
func ServiceName(serviceName string) kv.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// ResourceName defines the Resource name for the Span.
func ResourceName(resourceName string) kv.KeyValue {
	return ResourceNameKey.String(resourceName)
}

// Error specifies whether an error occurred.
func Error(err bool) kv.KeyValue {
	return ErrorKey.Bool(err)
}

// ErrorMsg specifies the error message.
func ErrorMsg(errorMsg string) kv.KeyValue {
	return ErrorMsgKey.String(errorMsg)
}
