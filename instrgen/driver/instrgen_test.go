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

//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alib "go.opentelemetry.io/contrib/instrgen/lib"
	"go.opentelemetry.io/contrib/instrgen/rewriters"
)

var testcases = map[string]string{
	"testdata/basic":     "testdata/expected/basic",
	"testdata/interface": "testdata/expected/interface",
}

var failures []string

func TestCommand(t *testing.T) {
	executor := &NullExecutor{}
	err := executeCommand("--unknown", "./testdata/basic", "testdata/basic", "yes", "main.main", executor)
	assert.Error(t, err)
}

func TestInstrumentation(t *testing.T) {
	cwd, _ := os.Getwd()
	var args []string
	for k := range testcases {
		filePaths := make(map[string]int)

		files := alib.SearchFiles(k, ".go")
		for index, file := range files {
			filePaths[file] = index
		}
		pruner := rewriters.OtelPruner{
			FilePattern: k, Replace: true}
		analyzePackage(pruner, "main", filePaths, nil, "", args)

		rewriter := rewriters.BasicRewriter{
			FilePattern: k, Replace: "yes", Pkg: "main", Fun: "main"}
		analyzePackage(rewriter, "main", filePaths, nil, "", args)
	}
	fmt.Println(cwd)

	for k, v := range testcases {
		files := alib.SearchFiles(cwd+"/"+k, ".go")
		expectedFiles := alib.SearchFiles(cwd+"/"+v, ".go")
		numOfFiles := len(expectedFiles)
		fmt.Println("Go Files:", len(files))
		fmt.Println("Expected Go Files:", len(expectedFiles))
		assert.True(t, len(files) > 0)
		numOfComparisons := 0
		for _, file := range files {
			fmt.Println(filepath.Base(file))
			for _, expectedFile := range expectedFiles {
				fmt.Println(filepath.Base(expectedFile))
				if filepath.Base(file) == filepath.Base(expectedFile) {
					fmt.Println(file, " : ", expectedFile)
					f1, err1 := os.ReadFile(file)
					require.NoError(t, err1)
					f2, err2 := os.ReadFile(expectedFile)
					require.NoError(t, err2)
					if !assert.True(t, bytes.Equal(f1, f2), file) {
						failures = append(failures, file)
					}
					numOfComparisons = numOfComparisons + 1
				}
			}
		}
		if numOfFiles != numOfComparisons {
			fmt.Println("numberOfComparisons:", numOfComparisons)
			panic("not all files were compared")
		}
	}
}

type NullExecutor struct {
}

func (executor *NullExecutor) Execute(_ string, _ []string) {
}

func (executor *NullExecutor) Run() error {
	return nil
}

func TestToolExecMain(t *testing.T) {
	for k := range testcases {
		var args []string
		files := alib.SearchFiles(k, ".go")
		args = append(args, []string{"-o", "/tmp/go-build", "-p", "main", "-pack", "-asmhdr", "go_asm.h"}...)
		args = append(args, files...)
		instrgenCfg := InstrgenCmd{FilePattern: k, Cmd: "prune", Replace: "yes",
			EntryPoint: EntryPoint{Pkg: "main", FunName: "main"}}
		rewriterS := makeRewriters(instrgenCfg)
		analyze(args, rewriterS)
		instrgenCfg.Cmd = "inject"
		rewriterS = makeRewriters(instrgenCfg)
		analyze(args, rewriterS)
	}
	for k := range testcases {
		var args []string
		files := alib.SearchFiles(k, ".go")
		args = append(args, []string{"-pack", "-asmhdr", "go_asm.h"}...)
		args = append(args, files...)
		instrgenCfg := InstrgenCmd{FilePattern: k, Cmd: "prune", Replace: "no",
			EntryPoint: EntryPoint{Pkg: "main", FunName: "main"}}
		rewriterS := makeRewriters(instrgenCfg)
		analyze(args, rewriterS)
		instrgenCfg.Cmd = "inject"
		rewriterS = makeRewriters(instrgenCfg)
		analyze(args, rewriterS)
	}
	for k := range testcases {
		instrgenCfg := InstrgenCmd{FilePattern: k, Cmd: "prune", Replace: "yes",
			EntryPoint: EntryPoint{Pkg: "main", FunName: "main"}}
		rewriterS := makeRewriters(instrgenCfg)
		var args []string
		executor := &NullExecutor{}
		err := toolExecMain(args, rewriterS, executor)
		assert.Error(t, err)
	}
}

func TestGetCommandName(t *testing.T) {
	cmd := GetCommandName([]string{"/usr/local/go/compile"})
	assert.True(t, cmd == "compile")
	cmd = GetCommandName([]string{"/usr/local/go/compile.exe"})
	assert.True(t, cmd == "compile")
	cmd = GetCommandName([]string{})
	assert.True(t, cmd == "")
}

func TestExecutePass(t *testing.T) {
	executor := &ToolExecutor{}
	require.NoError(t, executePass([]string{"go", "version"}, executor))
}

func TestDriverMain(t *testing.T) {
	executor := &NullExecutor{}
	{
		err := os.Remove("instrgen_cmd.json")
		_ = err
		var args []string
		args = append(args, "compile")
		err = driverMain(args, executor)
		require.Error(t, err)
	}
	for k := range testcases {
		var args []string
		files := alib.SearchFiles(k, ".go")
		args = append(args, []string{"-o", "/tmp/go-build", "-p", "main", "-pack", "-asmhdr", "go_asm.h"}...)
		args = append(args, files...)
		instrgenCfg := InstrgenCmd{FilePattern: k, Cmd: "prune", Replace: "yes",
			EntryPoint: EntryPoint{Pkg: "main", FunName: "main"}}
		err := driverMain(args, executor)
		assert.NoError(t, err)
		instrgenCfg.Cmd = "inject"
		err = driverMain(args, executor)
		assert.NoError(t, err)
	}
	{
		var args []string
		args = append(args, "compile")
		instrgenCfg := InstrgenCmd{FilePattern: "/testdata/basic", Cmd: "inject", Replace: "yes",
			EntryPoint: EntryPoint{Pkg: "main", FunName: "main"}}
		file, _ := json.MarshalIndent(instrgenCfg, "", " ")
		err := os.WriteFile("instrgen_cmd.json", file, 0644)
		require.NoError(t, err)
		err = driverMain(args, executor)
		require.NoError(t, err)
	}
	for k := range testcases {
		var args []string
		args = append(args, []string{"--inject", k, "yes", "main.main"}...)
		err := driverMain(args, executor)
		assert.NoError(t, err)
	}
	{
		var args []string
		args = append(args, "--inject")
		err := driverMain(args, executor)
		assert.Error(t, err)
	}
	{
		var args []string
		err := driverMain(args, executor)
		assert.NoError(t, err)
	}
}
