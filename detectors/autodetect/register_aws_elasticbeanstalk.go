// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_aws_elasticbeanstalk

package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"
)

func init() {
	registry[IDAWSElasticBeanstalk] = func() resource.Detector { return elasticbeanstalk.NewResourceDetector() }
}
