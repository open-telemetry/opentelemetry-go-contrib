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

// This module provides easy access to configuring a trace exporter
// that can be used when configuring an OpenTelemetry Go SDK trace export
// pipeline.
//
// [NewSpanExporter] looks for the `OTEL_TRACES_EXPORTER` environment
// variable and if set, attempts to load the exporter from it's registry of
// exporters. The registry is always loaded with an OTLP exporter with the key
// `otlp` and additional exporters can be registered using
// [RegisterSpanExporter].
// Exporter registration uses a factory method pattern to not unneccarily build
// exporters and use resources until they are requested.
//
// If the environment variable is not set, the fallback exporter is returned.
// The fallback exporter defaults to an
// [OTLP exporter](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlptrace)
// and can be overridden using the [RegisterSpanExporter](https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport#WithFallbackSpanExporter)
// option.

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"
