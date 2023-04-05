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
	"go/parser"
	"log"

	"go.opentelemetry.io/contrib/instrgen/lib"
	"golang.org/x/tools/go/loader"
)

const (
	otelPrunerPassSuffix          = "_pass_pruner"
	contextPassFileSuffix         = "_pass_ctx"
	instrumentationPassFileSuffix = "_pass_tracing"
)

func CheckSema(analysis *lib.PackageAnalysis) error {
	conf := loader.Config{ParserMode: parser.ParseComments}
	conf.Import(analysis.ProjectPath)
	_, err := conf.Load()
	// TODO only log error for now
	if err != nil {
		log.Println(err)
	}
	return nil
}

// ExecutePassesDumpIr.
func ExecutePassesDumpIr(analysis *lib.PackageAnalysis) error {
	fmt.Println("Instrumentation")
	_, err := analysis.Execute(&lib.InstrumentationPass{}, "")
	if err != nil {
		return err
	}
	fmt.Println("ContextPropagation")
	_, err = analysis.Execute(&lib.ContextPropagationPass{}, instrumentationPassFileSuffix)
	if err != nil {
		return err
	}
	return CheckSema(analysis)
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
	if err != nil {
		return err
	}
	return CheckSema(analysis)
}
