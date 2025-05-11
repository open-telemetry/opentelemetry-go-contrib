// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main // import "go.opentelemetry.io/contrib/tools/verifygoversion"

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var expectedGoVersion string

func main() {
	expectedGoVersion = os.Getenv("MINIMUM_GO_VERSION")
	if expectedGoVersion == "" {
		log.Fatal("MINIMUM_GO_VERSION environment variable is not set")
	}

	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Current working directory:", root)

	if err := verifyGoVersion(root); err != nil {
		os.Exit(1)
	}
}

func verifyGoVersion(root string) error {
	var modFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Base(path) == "go.mod" {
			modFiles = append(modFiles, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking the path %q: %v", root, err)
	}
	if len(modFiles) == 0 {
		return fmt.Errorf("there is no go.mod file in the current directory")
	}

	for _, file := range modFiles {
		bytes, err := os.ReadFile(file)
		if err != nil {
			err = errors.Join(err, fmt.Errorf("error reading %s: %v", file, err))
		}

		content := string(bytes)
		if strings.Contains(content, "toolchain") {
			err = errors.Join(err, fmt.Errorf("toolchain is not supported in %s", file))
		}

		contents := strings.Split(content, "\n")
		goVersionFound := false

		for _, line := range contents {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				goVersionFound = true
				if line != expectedGoVersion {
					err = errors.Join(err, fmt.Errorf("expected %s in %s, but found %s", expectedGoVersion, file, line))
				}
				break
			}
		}

		if !goVersionFound {
			err = errors.Join(err, fmt.Errorf("expected %s in %s, but not found", expectedGoVersion, file))
		}

		if err != nil {
			log.Println("Verification failed:", err)
		} else {
			log.Println("Verification succeeded:", file)
		}
	}

	return err
}
