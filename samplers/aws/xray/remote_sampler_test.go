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

package xray

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/contrib/samplers/aws/xray/internal"
	"go.opentelemetry.io/contrib/samplers/aws/xray/internal/util"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// assert that when manifest is not expired sampling happens with 1 req/sec.
func TestShouldSample(t *testing.T) {
	clock := &util.MockClock{
		NowTime: 100,
	}

	r1 := internal.Rule{}

	rules := []internal.Rule{r1}

	m := &internal.Manifest{
		Rules: rules,
		Clock: clock,
	}

	rs := &remoteSampler{
		manifest: m,
	}

	sd := rs.ShouldSample(sdktrace.SamplingParameters{})
	assert.Equal(t, sd.Decision, sdktrace.RecordAndSample)
}

// assert remote sampling description.
func TestRemoteSamplerDescription(t *testing.T) {
	rs := &remoteSampler{}

	s := rs.Description()
	assert.Equal(t, s, "AwsXrayRemoteSampler{remote sampling with AWS X-Ray}")
}
