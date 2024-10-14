// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
