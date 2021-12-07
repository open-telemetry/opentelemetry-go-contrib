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
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

// initYAML creates a YAML file at a given filepath in a in-memory file system.
func initYAML(yamlBytes []byte, path string) (afero.Fs, error) {
	// Create an in-memory file system.
	fs := afero.NewMemMapFs()

	// https://github.com/spf13/viper/blob/v1.8.1/viper.go#L480
	// absPathify uses filepath.Clean, so here you also need to use filepath.Clean
	if filepath.IsAbs(path) {
		path = filepath.Clean(path)
	} else {
		p, err := filepath.Abs(path)
		if err == nil {
			path = filepath.Clean(p)
		}
	}

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

// TestNewConfig tests whether NewConfig returns a correct Config struct. It checks
// whether the YAML file was read correctly and whether validation of the struct
// succeeded.
func TestNewConfig(t *testing.T) {
	tests := []struct {
		testName       string
		yamlByteString []byte
		fileName       string
		directoryPath  string
		expectedConfig *cortex.Config
		expectedError  error
	}{
		{
			testName:       "Valid Config file",
			yamlByteString: validYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: &validConfig,
			expectedError:  nil,
		},
		{
			testName:       "No Timeout",
			yamlByteString: noTimeoutYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: &validConfig,
			expectedError:  nil,
		},
		{
			testName:       "No Endpoint URL",
			yamlByteString: noEndpointYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: &validConfig,
			expectedError:  nil,
		},
		{
			testName:       "Two passwords",
			yamlByteString: twoPasswordsYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: nil,
			expectedError:  cortex.ErrTwoPasswords,
		},
		{
			testName:       "Two Bearer Tokens",
			yamlByteString: twoBearerTokensYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: nil,
			expectedError:  cortex.ErrTwoBearerTokens,
		},
		{
			testName:       "Custom Quantiles",
			yamlByteString: quantilesYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: &customQuantilesConfig,
			expectedError:  nil,
		},
		{
			testName:       "Custom Histogram Boundaries",
			yamlByteString: bucketBoundariesYAML,
			fileName:       "config.yml",
			directoryPath:  "/test",
			expectedConfig: &customBucketBoundariesConfig,
			expectedError:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Create YAML file.
			fullPath := test.directoryPath + "/" + test.fileName
			fs, err := initYAML(test.yamlByteString, fullPath)
			require.NoError(t, err)

			// Create new Config struct from the specified YAML file with an in-memory
			// filesystem.
			config, err := utils.NewConfig(
				test.fileName,
				utils.WithFilepath(test.directoryPath),
				utils.WithFilesystem(fs),
			)

			// Verify error and struct contents.
			require.Equal(t, err, test.expectedError)
			require.Equal(t, config, test.expectedConfig)
		})
	}
}

// TestWithFilepath tests whether NewConfig can find a YAML file that is not in the
// current directory.
func TestWithFilepath(t *testing.T) {
	tests := []struct {
		testName       string
		yamlByteString []byte
		fileName       string
		directoryPath  string
		addPath        bool
	}{
		{
			testName:       "Filepath provided, successful construction of Config",
			yamlByteString: validYAML,
			fileName:       "config.yml",
			directoryPath:  "/success",
			addPath:        true,
		},
		{
			testName:       "Filepath not provided, unsuccessful construction of Config",
			yamlByteString: validYAML,
			fileName:       "config.yml",
			directoryPath:  "/fail",
			addPath:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Create YAML file.
			fullPath := test.directoryPath + "/" + test.fileName
			fs, err := initYAML(test.yamlByteString, fullPath)
			require.NoError(t, err)

			// Create new Config struct from the specified YAML file with an in-memory
			// filesystem. If a path is added, Viper should be able to find the file and
			// there should be no error. Otherwise, an error should occur as Viper cannot
			// find the file.
			if test.addPath {
				_, err := utils.NewConfig(
					test.fileName,
					utils.WithFilepath(test.directoryPath),
					utils.WithFilesystem(fs),
				)
				require.NoError(t, err)
			} else {
				_, err := utils.NewConfig(test.fileName, utils.WithFilesystem(fs))
				require.Error(t, err)
			}
		})
	}
}

// TestWithClient tests whether NewConfig successfully adds a HTTP client to the Config
// struct.
func TestWithClient(t *testing.T) {
	// Create a YAML file.
	fs, err := initYAML(validYAML, "/test/config.yml")
	require.NoError(t, err)

	// Create a new Config struct with a custom HTTP client.
	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	config, _ := utils.NewConfig(
		"config.yml",
		utils.WithClient(customClient),
		utils.WithFilepath("/test"),
		utils.WithFilesystem(fs),
	)

	// Verify that the clients are the same.
	require.Equal(t, customClient, config.Client)
}
