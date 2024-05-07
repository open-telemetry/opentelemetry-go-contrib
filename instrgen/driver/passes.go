// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrgen/lib"
)

const (
	otelPrunerPassSuffix          = "_pass_pruner"
	contextPassFileSuffix         = "_pass_ctx"
	instrumentationPassFileSuffix = "_pass_tracing"
)

// ExecutePassesDumpIr.
func ExecutePassesDumpIr(analysis *lib.PackageAnalysis) error {
	fmt.Println("Instrumentation")
	_, err := analysis.Execute(&lib.InstrumentationPass{}, "")
	if err != nil {
		return err
	}

	fmt.Println("ContextPropagation")
	_, err = analysis.Execute(&lib.ContextPropagationPass{}, instrumentationPassFileSuffix)
	return err
}

// ExecutePasses.
func ExecutePasses(analysis *lib.PackageAnalysis) error {
	fmt.Println("Instrumentation")
	_, err := analysis.Execute(&lib.InstrumentationPass{}, instrumentationPassFileSuffix)
	if err != nil {
		return err
	}
	fmt.Println("ContextPropagation")
	_, err = analysis.Execute(&lib.ContextPropagationPass{}, contextPassFileSuffix)
	return err
}
