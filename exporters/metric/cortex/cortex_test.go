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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/sdk/export/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
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

var testResource = resource.New(kv.String("R", "V"))
var mockTime int64 = time.Time{}.Unix()

func TestExportKindFor(t *testing.T) {
	exporter := Exporter{}
	got := exporter.ExportKindFor(nil, aggregation.Kind(0))
	want := metric.CumulativeExporter

	if got != want {
		t.Errorf("ExportKindFor() =  %q, want %q", got, want)
	}
}

func TestConvertToTimeSeries(t *testing.T) {
	// Setup exporter with default quantiles and histogram buckets
	exporter := Exporter{
		config: Config{
			Quantiles: []float64{0.5, 0.9, .99},
		},
	}

	// Test conversions based on aggregation type
	tests := []struct {
		name       string
		input      export.CheckpointSet
		want       []*prompb.TimeSeries
		wantLength int
	}{
		{
			name:       "validCheckpointSet",
			input:      getValidCheckpointSet(t),
			want:       wantValidCheckpointSet,
			wantLength: 1,
		},
		{
			name:       "convertFromSum",
			input:      getSumCheckpoint(t, 321),
			want:       wantSumCheckpointSet,
			wantLength: 1,
		},
		{
			name:       "convertFromLastValue",
			input:      getLastValueCheckpoint(t, 123),
			want:       wantLastValueCheckpointSet,
			wantLength: 1,
		},
		{
			name:       "convertFromMinMaxSumCount",
			input:      getMMSCCheckpoint(t, 123.456, 876.543),
			want:       wantMMSCCheckpointSet,
			wantLength: 4,
		},
		{
			name:       "convertFromDistribution",
			input:      getDistributionCheckpoint(t),
			want:       wantDistributionCheckpointSet,
			wantLength: 7,
		},
		{
			name:       "convertFromHistogram",
			input:      getHistogramCheckpoint(t),
			want:       wantHistogramCheckpointSet,
			wantLength: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exporter.ConvertToTimeSeries(tt.input)
			want := tt.want

			assert.Nil(t, err, "ConvertToTimeSeries error")
			assert.Len(t, got, tt.wantLength, "Incorrect number of timeseries")
			cmp.Equal(got, want)
		})
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

// verifyExporterRequest checks a HTTP request from the export pipeline. It checks whether
// the request contains a correctly formatted remote_write body and the required headers.
func verifyExporterRequest(req *http.Request) error {
	// Check for required headers.
	if req.Header.Get("X-Prometheus-Remote-Write-Version") != "0.1.0" ||
		req.Header.Get("Content-Encoding") != "snappy" ||
		req.Header.Get("Content-Type") != "application/x-protobuf" {
		return fmt.Errorf("Request does not contain the three required headers")
	}

	// Check body format and headers.
	compressed, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("Failed to read request body")
	}
	uncompressed, err := snappy.Decode(nil, compressed)
	if err != nil {
		return fmt.Errorf("Failed to uncompress request body")
	}
	wr := &prompb.WriteRequest{}
	err = proto.Unmarshal(uncompressed, wr)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal message into WriteRequest struct")
	}

	return nil
}

// TestSendRequest checks if the Exporter can successfully send a http request with a
// correctly formatted body and the correct headers. A test server returns different
// status codes to test if the Exporter responds to a send failure correctly.
func TestSendRequest(t *testing.T) {
	tests := []struct {
		testName         string
		config           *Config
		expectedError    error
		isStatusNotFound bool
	}{
		{
			testName:         "Successful Export",
			config:           &validConfig,
			expectedError:    nil,
			isStatusNotFound: false,
		},
		{
			testName:         "Export Failure",
			config:           &Config{},
			expectedError:    fmt.Errorf("%v", "404 Not Found"),
			isStatusNotFound: true,
		},
	}

	// Set up a test server to receive the request. The server responds with a 400 Bad
	// Request status code if any headers are missing or if the body is not of the correct
	// format. Additionally, the server can respond with status code 404 Not Found to
	// simulate send failures.
	handler := func(rw http.ResponseWriter, req *http.Request) {
		err := verifyExporterRequest(req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return a status code 400 if header isStatusNotFound is "true", 200 otherwise.
		if req.Header.Get("isStatusNotFound") == "true" {
			rw.WriteHeader(http.StatusNotFound)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Set up an Exporter that uses the test server's endpoint and attaches the
			// test's isStatusNotFound header.
			test.config.Endpoint = server.URL
			test.config.Headers = map[string]string{
				"isStatusNotFound": strconv.FormatBool(test.isStatusNotFound),
			}
			exporter := Exporter{*test.config}

			// Create an empty Snappy-compressed message.
			msg, err := exporter.buildMessage([]*prompb.TimeSeries{})
			require.Nil(t, err)

			// Create a http POST request with the compressed message.
			req, err := exporter.buildRequest(msg)
			require.Nil(t, err)

			// Send the request to the test server and verify the error.
			err = exporter.sendRequest(req)
			if err != nil {
				errorString := err.Error()
				require.Equal(t, errorString, test.expectedError.Error())
			} else {
				require.Nil(t, test.expectedError)
			}
		})
	}
}
