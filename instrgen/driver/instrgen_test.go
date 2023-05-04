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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alib "go.opentelemetry.io/contrib/instrgen/lib"
)

var testcases = map[string]string{
	"./testdata/basic":     "./testdata/expected/basic",
	"./testdata/selector":  "./testdata/expected/selector",
	"./testdata/interface": "./testdata/expected/interface",
}

var failures []string

func inject(t *testing.T, root string, packagePattern string) {
	err := executeCommand("--inject-dump-ir", root, packagePattern)
	require.NoError(t, err)
}

func TestCommands(t *testing.T) {
	err := executeCommand("--dumpcfg", "./testdata/dummy", "./...")
	require.NoError(t, err)
	err = executeCommand("--rootfunctions", "./testdata/dummy", "./...")
	require.NoError(t, err)
	err = executeCommand("--prune", "./testdata/dummy", "./...")
	require.NoError(t, err)
	err = executeCommand("--inject", "./testdata/dummy", "./...")
	require.NoError(t, err)
	err = usage()
	require.NoError(t, err)
}

func TestCallGraph(t *testing.T) {
	cg := makeCallGraph("./testdata/dummy", "./...")
	dumpCallGraph(cg)
	assert.Equal(t, len(cg), 0, "callgraph should contain 0 elems")
	rf := makeRootFunctions("./testdata/dummy", "./...")
	dumpRootFunctions(rf)
	assert.Equal(t, len(rf), 0, "rootfunctions set should be empty")
}

func TestArgs(t *testing.T) {
	err := checkArgs(nil)
	require.Error(t, err)
	args := []string{"driver", "--inject", "", "./..."}
	err = checkArgs(args)
	require.NoError(t, err)
}

func TestUnknownCommand(t *testing.T) {
	err := executeCommand("unknown", "a", "b")
	require.Error(t, err)
}

func TestInstrumentation(t *testing.T) {
	for k, v := range testcases {
		inject(t, k, "./...")
		files := alib.SearchFiles(k, ".go_pass_tracing")
		expectedFiles := alib.SearchFiles(v, ".go")
		numOfFiles := len(expectedFiles)
		fmt.Println("Go Files:", len(files))
		fmt.Println("Expected Go Files:", len(expectedFiles))
		numOfComparisons := 0
		for _, file := range files {
			fmt.Println(filepath.Base(file))
			for _, expectedFile := range expectedFiles {
				fmt.Println(filepath.Base(expectedFile))
				if filepath.Base(file) == filepath.Base(expectedFile+"_pass_tracing") {
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
		_, err := Prune(k, "./...", false)
		if err != nil {
			fmt.Println("Prune failed")
		}
	}
	for _, f := range failures {
		fmt.Println("FAILURE : ", f)
	}
}
