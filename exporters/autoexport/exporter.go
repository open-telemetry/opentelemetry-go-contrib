// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	otelTracesExportersEnvKey = "OTEL_TRACES_EXPORTER"
)

// NewTraceExporter returns a configured SpanExporter defined using the environment
// variable OTEL_TRACES_EXPORTER or the passed in exporter. The exporter defined
// in OTEL_TRACES_EXPORTER is preferred over the exporter passed in.
func NewTraceExporter(exporter trace.SpanExporter) trace.SpanExporter {
	// prefer exporter configured via environment variables over exporter
	// passed in via exporter parameter
	envExporter, err := parseEnv()
	if err != nil {
		otel.Handle(err)
	}
	if envExporter != nil {
		return envExporter
	}

	return exporter
}

// parseEnv returns a configured SpanExporter defined by the OTEL_TRACES_EXPORTER
// environment variable.
// nil is returned if no exporter is defined for the environment variable.
func parseEnv() (trace.SpanExporter, error) {
	expType, defined := os.LookupEnv(otelTracesExportersEnvKey)
	if !defined {
		return nil, nil
	}

	return SpanExporter(expType)
}
