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

package consistent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	for _, tc := range []struct {
		prob   float64
		expect string
	}{
		{0.75, "ConsistentProbabilityBased{0.75}"},
		{0.05, "ConsistentProbabilityBased{0.05}"},
		{0.003, "ConsistentProbabilityBased{0.003}"},
		{0.99999999, "ConsistentProbabilityBased{0.99999999}"},
		{0.00000001, "ConsistentProbabilityBased{1e-08}"},
		{1, "ConsistentProbabilityBased{1}"},
		{0, "ConsistentProbabilityBased{0}"},
	} {
		s := ConsistentProbabilityBased(tc.prob)
		require.Equal(t, tc.expect, s.Description())
	}
}
