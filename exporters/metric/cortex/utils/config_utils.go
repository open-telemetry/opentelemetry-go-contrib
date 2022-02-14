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

// Package utils provides utilities for the Cortex exporter.
//
// Deprecated: This package is no longer supported.
package utils

import (
	"net/http"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"go.opentelemetry.io/contrib/exporters/metric/cortex" // nolint:staticcheck // allow import of deprecated pkg.
)

// Option sets an option for a Config struct.
type Option interface {
	Apply(*cortex.Config)
}

// WithFilepath adds a path where Viper will search for the YAML file in.
func WithFilepath(filepath string) Option {
	return filepathOption(filepath)
}

type filepathOption string

func (o filepathOption) Apply(config *cortex.Config) {
	viper.AddConfigPath(string(o))
}

// WithFilesystem tells Viper which file system to search for the YAML file in. By
// default, Viper will search the OS file system, but users can pass in an in-memory
// filesystem for testing.
func WithFilesystem(fs afero.Fs) Option {
	return fsOption{fs}
}

type fsOption struct {
	fs afero.Fs
}

func (o fsOption) Apply(config *cortex.Config) {
	viper.SetFs(o.fs)
}

// WithClient adds a custom http.Client to the Config struct.
func WithClient(client *http.Client) Option {
	return clientOption{client}
}

type clientOption struct {
	client *http.Client
}

func (o clientOption) Apply(config *cortex.Config) {
	config.Client = (*http.Client)(o.client)
}

// NewConfig creates a Config struct with a YAML file and applies Option functions to the
// Config struct.
func NewConfig(filename string, opts ...Option) (*cortex.Config, error) {
	var config cortex.Config

	// Use OS file system and look for YAML file in local directory by default.
	viper.SetFs(afero.NewOsFs())
	viper.SetConfigName(filename)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Apply Options afterwards to change the file system, add a custom Client, or add a
	// filepath.
	for _, opt := range opts {
		opt.Apply(&config)
	}

	// Read YAML file into struct and then check its properties.
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
}
