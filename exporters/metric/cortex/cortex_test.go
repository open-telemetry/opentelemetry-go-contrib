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

package cortex

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
)

// ValidConfig is a Config struct that should cause no errors.
var validConfig = Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: 30 * time.Second,
	Name:          "Valid Config Example",
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken:     "",
	BearerTokenFile: "",
	TLSConfig: map[string]string{
		"ca_file":              "cafile",
		"cert_file":            "certfile",
		"key_file":             "keyfile",
		"server_name":          "server",
		"insecure_skip_verify": "1",
	},
	ProxyURL:     "",
	PushInterval: 10 * time.Second,
	Headers: map[string]string{
		"x-prometheus-remote-write-version": "0.1.0",
		"tenant-id":                         "123",
	},
	Client: http.DefaultClient,
}

func TestExportKindFor(t *testing.T) {
	exporter := Exporter{}
	got := exporter.ExportKindFor(nil, aggregation.Kind(0))
	want := metric.CumulativeExporter

	if got != want {
		t.Errorf("ExportKindFor() =  %q, want %q", got, want)
	}
}

// TestNewRawExporter tests whether NewRawExporter successfully creates an Exporter with
// the same Config struct as the one passed in.
func TestNewRawExporter(t *testing.T) {
	exporter, err := NewRawExporter(validConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter with error %v", err)
	}

	if !cmp.Equal(validConfig, exporter.config) {
		t.Fatalf("Got configuration %v, wanted %v", exporter.config, validConfig)
	}
}
