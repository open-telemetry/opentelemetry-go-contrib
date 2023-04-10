// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
