// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package detectors contains functionality to return resource
// detectors that can be disabled via go build tags to avoid taking
// on additional dependencies at compile time.
package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"
)

type Detector struct {
	AWSEC2  bool
	AWSECS  bool
	AWSEKS  bool
	AzureVM bool
	GCP     bool
}

func Options(d Detector) []resource.Option {
	opts := []resource.Option{}
	if d.AWSEC2 {
		opts = append(opts, resource.WithDetectors(ec2Detector()))
	}
	if d.AWSECS {
		opts = append(opts, resource.WithDetectors(ecsDetector()))
	}
	if d.AWSEKS {
		opts = append(opts, resource.WithDetectors(eksDetector()))
	}
	if d.AzureVM {
		opts = append(opts, resource.WithDetectors(azurevmDetector()))
	}
	if d.GCP {
		opts = append(opts, resource.WithDetectors(gcpDetector()))
	}
	return opts
}
