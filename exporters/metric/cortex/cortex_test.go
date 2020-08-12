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
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/global"
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

// TestNewExportPipeline tests whether a push Controller was successfully created with an
// Exporter from NewRawExporter. Errors in this function will be from calls to push
// controller package and NewRawExport. Both have their own tests.
func TestNewExportPipeline(t *testing.T) {
	_, err := NewExportPipeline(validConfig)
	if err != nil {
		t.Fatalf("Failed to create export pipeline with error %v", err)
	}
}

// TestInstallNewPipeline checks whether InstallNewPipeline successfully returns a push
// Controller and whether that controller's Provider is registered globally.
func TestInstallNewPipeline(t *testing.T) {
	pusher, err := InstallNewPipeline(validConfig)
	if err != nil {
		t.Fatalf("Failed to create install pipeline with error %v", err)
	}
	if global.MeterProvider() != pusher.Provider() {
		t.Fatalf("Failed to register push Controller provider globally")
	}
}

// TestAddHeaders tests whether the correct headers are correctly added to a http request.
func TestAddHeaders(t *testing.T) {
	testConfig := Config{
		Headers: map[string]string{
			"TestHeaderOne": "TestFieldTwo",
			"TestHeaderTwo": "TestFieldTwo",
		},
	}
	exporter := Exporter{testConfig}

	// Create http request to add headers to.
	req, err := http.NewRequest("POST", "test.com", nil)
	require.Nil(t, err)
	exporter.addHeaders(req)

	// Check that all the headers are there.
	for name, field := range testConfig.Headers {
		require.Equal(t, req.Header.Get(name), field)
	}
	require.Equal(t, req.Header.Get("Content-Encoding"), "snappy")
	require.Equal(t, req.Header.Get("Content-Type"), "application/x-protobuf")
	require.Equal(t, req.Header.Get("X-Prometheus-Remote-Write-Version"), "0.1.0")
}

// TestBuildMessage tests whether BuildMessage successfully returns a Snappy-compressed
// protobuf message.
func TestBuildMessage(t *testing.T) {
	exporter := Exporter{validConfig}
	timeseries := []*prompb.TimeSeries{}

	// buildMessage returns the error that proto.Marshal() returns. Since the proto
	// package has its own tests, buildMessage should work as expected as long as there
	// are no errors.
	_, err := exporter.buildMessage(timeseries)
	require.Nil(t, err)
}

// TestBuildRequest tests whether a http request is a POST request, has the correct body,
// and has the correct headers.
func TestBuildRequest(t *testing.T) {
	// Make fake exporter and message for testing.
	var testMessage = []byte(`Test Message`)
	exporter := Exporter{validConfig}

	// Create the http request.
	req, err := exporter.buildRequest(testMessage)
	require.Nil(t, err)

	// Verify the http method, url, and body.
	require.Equal(t, req.Method, http.MethodPost)
	require.Equal(t, req.URL.String(), validConfig.Endpoint)

	reqMessage, err := ioutil.ReadAll(req.Body)
	require.Nil(t, err)
	require.Equal(t, reqMessage, testMessage)

	// Verify headers.
	for name, field := range exporter.config.Headers {
		require.Equal(t, req.Header.Get(name), field)
	}
	require.Equal(t, req.Header.Get("Content-Encoding"), "snappy")
	require.Equal(t, req.Header.Get("Content-Type"), "application/x-protobuf")
	require.Equal(t, req.Header.Get("X-Prometheus-Remote-Write-Version"), "0.1.0")
}
