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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	notOnGCE = func() bool { return false }
	onGCE    = func() bool { return true }
)

func getenv(m map[string]string) func(string) string {
	return func(s string) string {
		if m == nil {
			return ""
		}
		return m[s]
	}
}

type client struct {
	m map[string]string
}

func setupForTest(c *CloudRun, mc metadataClient, ongce func() bool, getenv func(string) string) {
	c.mc = mc
	c.onGCE = ongce
	c.getenv = getenv
}

func (c *client) Get(s string) (string, error) {
	got, ok := c.m[s]
	if !ok {
		return "", fmt.Errorf("%q do not exist", s)
	} else if got == "" {
		return "", fmt.Errorf("%q is empty", s)
	}
	return got, nil
}

func (c *client) InstanceID() (string, error) {
	return c.Get("instance/id")
}

func (c *client) ProjectID() (string, error) {
	return c.Get("project/project-id")
}

var _ metadataClient = (*client)(nil)

func TestCloudRunDetectorNotOnGCE(t *testing.T) {
	ctx := context.Background()
	c := NewCloudRun()
	setupForTest(c, nil, notOnGCE, getenv(nil))

	if res, err := c.Detect(ctx); res != nil || err != nil {
		t.Errorf("Expect c.Detect(ctx) to return (nil, nil), got (%v, %v)", res, err)
	}
}

func TestCloudRunDetectorExpectSuccess(t *testing.T) {
	ctx := context.Background()

	metadata := map[string]string{
		"project/project-id": "foo",
		"instance/id":        "bar",
		"instance/region":    "/projects/123/regions/utopia",
	}
	envvars := map[string]string{
		"K_SERVICE": "x-service",
	}
	want, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("cloud.account.id", "foo"),
			attribute.String("cloud.provider", "gcp"),
			attribute.String("cloud.region", "utopia"),
			attribute.String("service.instance.id", "bar"),
			attribute.String("service.name", "x-service"),
			attribute.String("service.namespace", "cloud-run-managed"),
		),
	)
	if err != nil {
		t.Fatalf("failed to create a resource: %v", err)
	}
	c := NewCloudRun()
	setupForTest(c, &client{m: metadata}, onGCE, getenv(envvars))

	if res, err := c.Detect(ctx); err != nil {
		t.Fatalf("got unexpected failure: %v", err)
	} else if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
	}
}

func TestCloudRunDetectorExpectFail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		metadata map[string]string
		envvars  map[string]string
	}{
		{
			name: "Missing ProjectID",
			metadata: map[string]string{
				"instance/id":     "bar",
				"instance/region": "utopia",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing InstanceID",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/region":    "utopia",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing Region",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/id":        "bar",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing K_SERVICE envvar",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/id":        "bar",
				"instance/region":    "utopia",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewCloudRun()
			setupForTest(c, &client{m: test.metadata}, onGCE, getenv(test.envvars))

			if res, err := c.Detect(ctx); err == nil {
				t.Errorf("Expect c.Detect(ctx) to return error, got nil (resource: %v)", res)
			} else {
				t.Logf("err: %v", err)
			}
		})
	}
}
