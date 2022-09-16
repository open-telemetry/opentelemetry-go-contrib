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

package autoexport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func TestNoExportersAreConfiguredIfEnvNotSetAndNoExportersProvided(t *testing.T) {
	exporter := NewTraceExporter(nil)
	assert.Nil(t, exporter)
}

func TestProvidedExportersAreUsedWhenEnvVarIsNotSet(t *testing.T) {
	exp := otlptracegrpc.NewUnstarted()
	exporter := NewTraceExporter(
		exp,
	)
	assert.Equal(t, exp, exporter)
}

func TestExportersConfiguredInEnvVarAreReturned(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTERS", "otlp")
	exporter := NewTraceExporter(nil)
	assert.NotNil(t, 1, exporter)
}
