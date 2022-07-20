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

package otelgocql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gocql/gocql/otelgocql/internal"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func TestHostOrIP(t *testing.T) {
	hostAndPort := "127.0.0.1:9042"
	attribute := internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerIPKey, attribute.Key)
	assert.Equal(t, "127.0.0.1", attribute.Value.AsString())

	hostAndPort = "exampleHost:9042"
	attribute = internal.HostOrIP(hostAndPort)
	assert.Equal(t, semconv.NetPeerNameKey, attribute.Key)
	assert.Equal(t, "exampleHost", attribute.Value.AsString())

	hostAndPort = "invalid-host-and-port-string"
	attribute = internal.HostOrIP(hostAndPort)
	require.Empty(t, attribute.Value.AsString())
}
