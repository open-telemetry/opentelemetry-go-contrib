// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect

import (
	"slices"
	"testing"
)

func TestDetectorsWithDependenciesRegistered(t *testing.T) {
	ids := []ID{
		IDAkamai, IDAWSEC2, IDAWSECS, IDAWSEKS, IDAWSLambda, IDAWSElasticBeanstalk,
		IDAzureAKS, IDAzureContainerApps, IDAzureVM, IDDocker, IDGCP, IDHetzner,
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
