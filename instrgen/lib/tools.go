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

package lib // import "go.opentelemetry.io/contrib/instrgen/lib"

import (
	"os"
	"path/filepath"
)

// SearchFiles.
func SearchFiles(root string, ext string) []string {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

func isPath(
	callGraph map[FuncDescriptor][]FuncDescriptor,
	current FuncDescriptor,
	goal FuncDescriptor,
	visited map[FuncDescriptor]bool,
) bool {
	if current == goal {
		return true
	}

	value, ok := callGraph[current]
	if ok {
		for _, child := range value {
			exists := visited[child]
			if exists {
				continue
			}
			visited[child] = true
			if isPath(callGraph, child, goal, visited) {
				return true
			}
		}
	}
	return false
}

// Contains.
func Contains(a []FuncDescriptor, x FuncDescriptor) bool {
	for _, n := range a {
		if x.TypeHash() == n.TypeHash() {
			return true
		}
	}
	return false
}
