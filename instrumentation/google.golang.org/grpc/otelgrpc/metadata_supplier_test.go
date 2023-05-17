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

package otelgrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestMetadataSupplier(t *testing.T) {
	md := metadata.New(map[string]string{
		"k1": "v1",
	})
	ms := &metadataSupplier{&md}

	v1 := ms.Get("k1")
	assert.Equal(t, v1, "v1")

	ms.Set("k2", "v2")

	v1 = ms.Get("k1")
	v2 := ms.Get("k2")
	assert.Equal(t, v1, "v1")
	assert.Equal(t, v2, "v2")
}
