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

package gcp

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func setupForGCETest(c *GCE, mc metadataClient, ongce func() bool) {
	c.mc = mc
	c.onGCE = ongce
}

var _ metadataClient = (*client)(nil)

func TestGCEDetectorNotOnGCE(t *testing.T) {
	ctx := context.Background()
	c := NewGCE()
	setupForGCETest(c, nil, notOnGCE)

	if res, err := c.Detect(ctx); res != nil || err != nil {
		t.Errorf("Expect c.Detect(ctx) to return (nil, nil), got (%v, %v)", res, err)
	}
}

func TestGCEDetectorExpectSuccess(t *testing.T) {
	ctx := context.Background()

	metadata := map[string]string{
		"project/project-id":    "foo",
		"instance/id":           "bar",
		"instance/region":       "/projects/123/regions/utopia",
		"instance/machine-type": "n1-standard-1",
		"instance/name":         "test-instance",
		"instance/zone":         "us-central1-a",
	}
	hostname, _ := os.Hostname()
	want, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("cloud.account.id", "foo"),
			attribute.String("cloud.provider", "gcp"),
			attribute.String("cloud.platform", "gcp_compute_engine"),
			attribute.String("cloud.availability_zone", "us-central1-a"),
			attribute.String("host.id", "bar"),
			attribute.String("host.name", hostname),
			attribute.String("host.type", "n1-standard-1"),
		),
	)
	if err != nil {
		t.Fatalf("failed to create a resource: %v", err)
	}
	c := NewGCE()
	setupForGCETest(c, &client{m: metadata}, onGCE)

	if res, err := c.Detect(ctx); err != nil {
		t.Fatalf("got unexpected failure: %v", err)
	} else if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
	}
}

func TestGCEDetectorExpectFail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		metadata map[string]string
		envvars  map[string]string
	}{
		{
			name: "Missing ProjectID",
			metadata: map[string]string{
				"instance/id":           "bar",
				"instance/region":       "/projects/123/regions/utopia",
				"instance/machine-type": "n1-standard-1",
				"instance/name":         "test-instance",
				"instance/zone":         "us-central1-a",
			},
		},
		{
			name: "Missing InstanceID",
			metadata: map[string]string{
				"project/project-id":    "foo",
				"instance/region":       "/projects/123/regions/utopia",
				"instance/machine-type": "n1-standard-1",
				"instance/name":         "test-instance",
				"instance/zone":         "us-central1-a",
			},
		},
		{
			name: "Missing Zone",
			metadata: map[string]string{
				"project/project-id":    "foo",
				"instance/id":           "bar",
				"instance/region":       "/projects/123/regions/utopia",
				"instance/machine-type": "n1-standard-1",
				"instance/name":         "test-instance",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewGCE()
			setupForGCETest(c, &client{m: test.metadata}, onGCE)

			if res, err := c.Detect(ctx); err == nil {
				t.Errorf("Expect c.Detect(ctx) to return error, got nil (resource: %v)", res)
			} else {
				t.Logf("err: %v", err)
			}
		})
	}
}
