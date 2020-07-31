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

package push

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/transform"
)

const TestAddress string = "localhost:50420"

var TestFingerprint = []byte{'f', 'o', 'o'}

func TestReadConfig(t *testing.T) {
	// Mock config service returns config with a suggested wait time of 5 minutes.
	config := GetDefaultConfig(60, TestFingerprint)
	config.SuggestedWaitTimeSec = 300
	stopFunc := RunMockConfigService(t, TestAddress, config)

	reader := NewServiceReader(
		TestAddress,
		transform.Resource(MockResource("servicereadertest")),
	)

	response, err := reader.readConfig()
	assert.NoError(t, err)

	stopFunc()

	require.Equal(t, config.Fingerprint, response.Fingerprint)
}

func TestSuggestedWaitTime(t *testing.T) {
	clock := clock.NewMock()

	// ServiceReader with suggestedWaitTimeSec of 5 minutes.
	reader := ServiceReader{
		clock:                clock,
		lastTimestamp:        clock.Now(),
		suggestedWaitTimeSec: 5 * 60,
	}

	require.Equal(t, 5*time.Minute, reader.suggestedWaitTime())

	clock.Add(5 * time.Minute)

	require.Equal(t, time.Duration(0), reader.suggestedWaitTime())

	clock.Add(5 * time.Minute)

	require.Equal(t, time.Duration(0), reader.suggestedWaitTime())
}
