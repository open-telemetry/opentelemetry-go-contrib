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

package autoexport_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func TestNoExportersAreConfiguredIfEnvNotSetAndNoExportersProvided(t *testing.T) {
	exporters := autoexport.NewTraceExporters()
	assert.Equal(t, 0, len(exporters))
}

func TestProvidedExportersAreUsedWhenEnvVarIsNotSet(t *testing.T) {
	exporters := autoexport.NewTraceExporters(
		otlptracegrpc.NewUnstarted(),
	)
	assert.Equal(t, 1, len(exporters))
}

func TestExportersConfiguredInEnvVarAreReturned(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTERS", "otlp")
	exporters := autoexport.NewTraceExporters()
	assert.Equal(t, 1, len(exporters))
}
