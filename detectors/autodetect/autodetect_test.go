// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect

import (
	"context"
	"slices"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

// appDetector is an application-owned detector, used to check that an
// application registration coexists with the ones this package and the
// registry package make.
type appDetector struct{}

var _ resource.Detector = appDetector{}

func (appDetector) Detect(context.Context) (*resource.Resource, error) {
	return resource.NewWithAttributes("", attribute.String("deployment.zone", "eu-west-1a")), nil
}

func TestDetectorsWithDependenciesRegistered(t *testing.T) {
	ids := []ID{
		IDAkamai, IDAlibabaECS, IDAWSEC2, IDAWSECS, IDAWSEKS, IDAWSLambda, IDAWSElasticBeanstalk,
		IDAzureAKS, IDAzureContainerApps, IDAzureVM, IDConsul, IDDynatrace, IDDocker, IDGCP, IDHetzner,
		IDIBMCloudClassic, IDIBMCloudVPC, IDK8sAPI, IDOpenStackNova, IDScaleway, IDKubeadm, IDOpenShift, IDVultr,
	}

	registered := Registered()
	for _, id := range ids {
		if !slices.Contains(registered, id) {
			t.Errorf("detector %q is not registered", id)
			continue
		}
		if _, err := Detector(id); err != nil {
			t.Errorf("detector %q: got error %v, expected no error", id, err)
		}
	}
}

// TestApplicationRegistrationCoexists covers the ownership model across both
// packages: this package registers every shipped detector in its init, the
// registry package registers the detectors that need nothing beyond the SDK,
// and an application can still register and resolve an ID it owns.
func TestApplicationRegistrationCoexists(t *testing.T) {
	const id = ID("application.owned")
	Register(id, func() resource.Detector { return appDetector{} })

	registered := Registered()
	for _, want := range []ID{IDAkamai, IDAlibabaECS, IDHost, id} {
		if !slices.Contains(registered, want) {
			t.Errorf("detector %q is not registered", want)
		}
	}

	detector, err := Detector(id)
	if err != nil {
		t.Fatalf("got error: %v, expected no error", err)
	}

	res, err := detector.Detect(t.Context())
	if err != nil {
		t.Fatalf("got error: %v, expected no error", err)
	}

	want := attribute.String("deployment.zone", "eu-west-1a")
	if got := res.Attributes(); len(got) != 1 || got[0] != want {
		t.Errorf("got %v, expected %v", got, []attribute.KeyValue{want})
	}
}
