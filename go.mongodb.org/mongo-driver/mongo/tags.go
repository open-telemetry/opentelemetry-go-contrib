// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Package ext contains a set of Datadog-specific constants. Most of them are used
// for setting span metadata.
package mongo

import "go.opentelemetry.io/otel/api/core"

const (
	TargetHostKey   = core.Key("out.host")
	TargetPortKey   = core.Key("out.port")
	HTTPMethodKey   = core.Key("http.method")
	HTTPCodeKey     = core.Key("http.code")
	HTTPURLKey      = core.Key("http.url")
	SpanTypeKey     = core.Key("span.type")
	ServiceNameKey  = core.Key("service.name")
	ResourceNameKey = core.Key("resource.name")
	ErrorKey        = core.Key("error")
	ErrorMsgKey     = core.Key("error.msg")
)

// TargetHost sets the target host address.
func TargetHost(targetHost string) core.KeyValue {
	return TargetHostKey.String(targetHost)
}

// TargetPort sets the target host port.
func TargetPort(targetPort string) core.KeyValue {
	return TargetPortKey.String(targetPort)
}

// HTTPMethod specifies the HTTP method used in a span.
func HTTPMethod(httpMethod string) core.KeyValue {
	return HTTPMethodKey.String(httpMethod)
}

// HTTPCode sets the HTTP status code as a attribute.
func HTTPCode(httpCode string) core.KeyValue {
	return HTTPCodeKey.String(httpCode)
}

// HTTPURL sets the HTTP URL for a span.
func HTTPURL(httpURL string) core.KeyValue {
	return HTTPURLKey.String(httpURL)
}

// SpanType defines the Span type (web, db, cache).
func SpanType(spanType string) core.KeyValue {
	return SpanTypeKey.String(spanType)
}

// ServiceName defines the Service name for this Span.
func ServiceName(serviceName string) core.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// ResourceName defines the Resource name for the Span.
func ResourceName(resourceName string) core.KeyValue {
	return ResourceNameKey.String(resourceName)
}

// Error specifies whether an error occurred.
func Error(err bool) core.KeyValue {
	return ErrorKey.Bool(err)
}

// ErrorMsg specifies the error message.
func ErrorMsg(errorMsg string) core.KeyValue {
	return ErrorMsgKey.String(errorMsg)
}

