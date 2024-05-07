// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellambda instruments the github.com/aws/aws-lambda-go package.
//
// Two wrappers are provided which can be used to instrument Lambda,
// one for each Lambda entrypoint. Their usages are shown below.
//
// lambda.Start(<user function>) entrypoint: lambda.Start(otellambda.InstrumentHandler(<user function>))
// lambda.StartHandler(<user Handler>) entrypoint: lambda.StartHandler(otellambda.WrapHandler(<user Handler>))
package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
