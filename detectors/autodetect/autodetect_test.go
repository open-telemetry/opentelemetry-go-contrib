// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect

import (
	"slices"
	"testing"
)

func TestDetectorsWithDependenciesRegistered(t *testing.T) {
	ids := []ID{
		IDAWSEC2, IDAWSECS, IDAWSEKS, IDAWSLambda, IDAWSElasticBeanstalk,
		IDAzureContainerApps, IDAzureVM, IDDocker, IDGCP, IDHetzner,
		IDIBMCloudVPC, IDK8sAPI, IDKubeadm, IDOpenShift, IDVultr,
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
