// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !otelcontrib_no_aws_lambda

package autodetect

import (
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
)

func init() {
	registry[IDAWSLambda] = lambda.NewResourceDetector
}
