// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !omit_detector_aws_lambda

package detectors

import (
	"go.opentelemetry.io/otel/sdk/resource"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
)

func awsLambdaDetector() resource.Detector {
	return lambdadetector.NewResourceDetector()
}
