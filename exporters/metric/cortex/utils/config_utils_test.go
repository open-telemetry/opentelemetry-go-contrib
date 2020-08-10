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

package utils_test

import (
	"path/filepath"

	"github.com/spf13/afero"
)

// initYAML creates a YAML file at a given filepath in a in-memory file system.
func initYAML(yamlBytes []byte, path string) (afero.Fs, error) {
	// Create an in-memory file system.
	fs := afero.NewMemMapFs()

	// Retrieve the directory path.
	dirPath := filepath.Dir(path)

	// Create the directory and the file in the directory.
	if err := fs.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}
	if err := afero.WriteFile(fs, path, yamlBytes, 0644); err != nil {
		return nil, err
	}

	return fs, nil
}
