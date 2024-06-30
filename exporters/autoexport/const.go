// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

const (
	none    = "none"
	otlp    = "otlp"
	console = "console"

	httpProtobuf = "http/protobuf"
	grpc         = "grpc"

	otelExporterOTLPProtoEnvKey = "OTEL_EXPORTER_OTLP_PROTOCOL"
)
