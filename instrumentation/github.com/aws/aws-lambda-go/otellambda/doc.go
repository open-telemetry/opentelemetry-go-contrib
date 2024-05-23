// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otellambda instruments the github.com/aws/aws-lambda-go package.
//
// Two wrappers are provided which can be used to instrument Lambda,
// one for each Lambda entrypoint. Their usages are shown below.
//
// lambda.Start(<user function>) entrypoint: lambda.Start(otellambda.InstrumentHandler(<user function>))
// lambda.StartHandler(<user Handler>) entrypoint: lambda.StartHandler(otellambda.WrapHandler(<user Handler>))
//
// Deprecated: otellambda has no Code Owner.
// After August 21, 2024, it may no longer be supported and may stop
// receiving new releases unless a new Code Owner is found. See
// [this issue] if you would like to become the Code Owner of this module.
//
// [this issue]: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5546
package otellambda // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
