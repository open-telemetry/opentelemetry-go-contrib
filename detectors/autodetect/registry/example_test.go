// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"context"
	"fmt"
	"log"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/autodetect/registry"
)

// deployZoneDetector reports an attribute that no detector shipped with the
// registry package provides, so the application owns its ID.
type deployZoneDetector struct{}

var _ resource.Detector = deployZoneDetector{}

func (deployZoneDetector) Detect(context.Context) (*resource.Resource, error) {
	return resource.NewWithAttributes(
		"",
		attribute.String("deployment.zone", "eu-west-1a"),
	), nil
}

func ExampleRegister() {
	const idDeployZone = registry.ID("deployment.zone")

	// The application registers only the ID it owns. The detectors shipped
	// with the registry package are registered already, and registering one of
	// their IDs panics instead of replacing it.
	registry.Register(idDeployZone, func() resource.Detector { return deployZoneDetector{} })

	// Detector IDs usually come from configuration. The shipped ones and the
	// application ones resolve through the same registry.
	for _, id := range []registry.ID{registry.IDHost, idDeployZone} {
		fmt.Printf("%s registered: %t\n", id, slices.Contains(registry.Registered(), id))
	}

	detector, err := registry.Detector(idDeployZone)
	if err != nil {
		log.Fatal(err)
	}

	res, err := detector.Detect(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res.Attributes() {
		fmt.Printf("%s=%s\n", kv.Key, kv.Value.AsString())
	}
	// Output:
	// host registered: true
	// deployment.zone registered: true
	// deployment.zone=eu-west-1a
}
