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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

const otelTracesExportersEnvKey = "OTEL_EXPORTERS"

// errUnknownExpoter is returned when an unknown exporter name is used in
// the OTEL_EXPORTERS environment variable.
var errUnknownExpoter = errors.New("unknown exporter")

func NewTraceExporters(exporters ...trace.SpanExporter) []trace.SpanExporter {
	// prefer exporters configured via environment variables over exporters
	// passed in via exporters paramter
	envExporters, err := parseEnv()
	if err != nil {
		otel.Handle(err)
	}
	if len(envExporters) > 0 {
		return envExporters
	}

	return exporters
}

// parseEnv returns an array of SpanExporter's defined by the OTEL_EXPORTERS
// environment variable.
// A nil slice is returned if no exporters are defined for the environment variable.
func parseEnv() ([]trace.SpanExporter, error) {
	expTypes, defined := os.LookupEnv(otelTracesExportersEnvKey)
	if !defined {
		return nil, nil
	}

	const sep = ","

	var (
		exporters []trace.SpanExporter
		unknown   []string
		errors    []error
	)

	for _, expType := range strings.Split(expTypes, sep) {
		switch expType {
		case "otlp":
			// TODO: switch between otlp exporter protocol (grpc, http)
			exp, err := otlptracegrpc.New(context.Background())
			if err != nil {
				errors = append(errors, err)
			} else {
				exporters = append(exporters, exp)
			}
			break
		default:
			exp, ok := envRegistry.load(expType)
			if !ok {
				unknown = append(unknown, expType)
				continue
			}
			exporters = append(exporters, exp)
		}
	}

	var err error
	if len(unknown) > 0 {
		joined := strings.Join(unknown, sep)
		err = fmt.Errorf("%w: %s", errUnknownExpoter, joined)
	}

	// TODO: combine start errors with unknown exporter error
	return exporters, err
}
