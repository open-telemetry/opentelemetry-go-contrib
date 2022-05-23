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
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func setupForGKETest(c *GKE, mc metadataClient, getenv func(string) string) {
	c.mc = mc
	c.getenv = getenv
}

var _ metadataClient = (*client)(nil)

func TestGKEDetectorNotOnGKE(t *testing.T) {
	ctx := context.Background()

	if res, err := NewGKE().Detect(ctx); res != nil || err != nil {
		t.Errorf("Expect NewGKE().Detect(ctx) to return (nil, nil), got (%v, %v)", res, err)
	}

}

func TestGKEDetectorOnGKE(t *testing.T) {
	ctx := context.Background()

	metadata := map[string]string{
		"instance/attributes/cluster-name": "test-cluster",
	}

	envvars := map[string]string{
		"KUBERNETES_SERVICE_HOST": "kubernetes-service-host",
		"NAMESPACE":               "project-namespace",
		"HOSTNAME":                "instance-hostname",
		"CONTAINER_NAME":          "container-name",
	}

	want, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("cloud.provider", "gcp"),
			attribute.String("cloud.platform", "gcp_kubernetes_engine"),
			attribute.String("k8s.namespace.name", "project-namespace"),
			attribute.String("k8s.pod.name", "instance-hostname"),
			attribute.String("container.name", "container-name"),
			attribute.String("k8s.cluster.name", "test-cluster"),
		),
	)
	if err != nil {
		t.Fatalf("failed to create a resource: %v", err)
	}
	c := NewGKE()
	setupForGKETest(c, &client{m: metadata}, getenv(envvars))
	if res, err := c.Detect(ctx); err != nil {
		t.Fatalf("got unexpected failure: %v", err)
	} else if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
	}
}
