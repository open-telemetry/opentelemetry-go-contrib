// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build omit_detector_aws_lambda

package detectors

import "go.opentelemetry.io/otel/sdk/resource"

func awsLambdaDetector() resource.Detector {
	return nil
}
