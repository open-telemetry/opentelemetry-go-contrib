// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/compute/metadata"
)

func NewTestDetector(metadataTransport *FakeMetadataTransport, os *FakeOSProvider) *Detector {
	return &Detector{metadata: metadata.NewClient(&http.Client{
		Transport: metadataTransport,
	}), os: os}
}

const (
	fakeRegion              = "us-central1"
	fakeZone                = fakeRegion + "-a"
	fakeProjectID           = "my-project"
	fakeProjectNumber       = "1234567890"
	fakeInstanceID          = "1234567891"
	fakeInstanceName        = "my-instance"
	fakeInstanceHostname    = fakeInstanceName + "." + fakeZone + ".c." + fakeProjectID + ".internal"
	fakeInstanceMachineType = "n1-standard1"
)

type FakeMetadataTransport struct {
	T        *testing.T
	Err      error
	Metadata map[string]string
}

const computeMetadataPrefix = "/computeMetadata/v1/"

func (f *FakeMetadataTransport) getResponse(path string) (string, bool) {
	if !strings.HasPrefix(path, computeMetadataPrefix) {
		return "", false
	}
	path = strings.TrimPrefix(path, computeMetadataPrefix)
	content, ok := f.Metadata[path]
	return content, ok
}

func (f *FakeMetadataTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.T != nil {
		f.T.Helper()
		f.T.Logf("metadata request: %v", req)
	}
	if req.URL.Host != "169.254.169.254" && req.URL.Host != "metadata.google.internal" {
		return nil, fmt.Errorf("unknown host %q", req.URL.Host)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	pr, pw := io.Pipe()
	res := &http.Response{
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		Header:     make(http.Header),
		Close:      true,
		Body:       pr,
	}
	content, ok := f.getResponse(req.URL.Path)
	if ok {
		res.StatusCode = 200
	} else {
		res.StatusCode = 404
	}
	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte(content))
	}()
	res.Status = fmt.Sprintf("%d %s", res.StatusCode, http.StatusText(res.StatusCode))
	return res, nil
}

// newFakeMetadataTransport constructs a FakeMetadataTransport that responds with the basic GCE metadata fields.
// To add more metadata, pass key-value pairs as arguments.
func newFakeMetadataTransport(t *testing.T, keyValues ...string) *FakeMetadataTransport {
	out := map[string]string{
		"project/project-id":         fakeProjectID,
		"project/numeric-project-id": fakeProjectNumber,
		"instance/id":                fakeInstanceID,
		"instance/name":              fakeInstanceName,
		"instance/hostname":          fakeInstanceHostname,
		"instance/machine-type":      fakeInstanceMachineType,
		"instance/zone":              fmt.Sprintf("projects/%s/zones/%s", fakeProjectNumber, fakeZone),
	}
	for i := 0; i < len(keyValues); i += 2 {
		out[keyValues[i]] = keyValues[i+1]
	}
	return &FakeMetadataTransport{
		T:        t,
		Metadata: out,
	}
}

type FakeOSProvider struct {
	Vars map[string]string
}

func (f *FakeOSProvider) LookupEnv(env string) (string, bool) {
	v, ok := f.Vars[env]
	return v, ok
}
