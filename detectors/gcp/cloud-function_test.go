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

package gcp

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var (
	errTest = errors.New("testError")
)

const (
	projectIDValue = "some-projectID"
	regionValue    = "some-region"
	functionName   = "sample-function"
)

type metaDataClientImpl struct {
	projectID  func() (string, error)
	get        func(string) (string, error)
	instanceID func() (string, error)
}

func (mock *metaDataClientImpl) ProjectID() (string, error) {
	if mock.projectID != nil {
		return mock.projectID()
	}
	return "", nil
}

func (mock *metaDataClientImpl) Get(key string) (string, error) {
	if mock.get != nil {
		return mock.get(key)
	}
	return "", nil
}

func (mock *metaDataClientImpl) InstanceID() (string, error) {
	if mock.instanceID != nil {
		return mock.instanceID()
	}
	return "", nil
}

type want struct {
	res *resource.Resource
	err error
}

func TestCloudFunctionDetect(t *testing.T) {
	oldValue, ok := os.LookupEnv(gcpFunctionNameKey)
	if !ok {
		err := os.Setenv(gcpFunctionNameKey, functionName)
		if err != nil {
			t.Error("unable to set environment variable ", err)
		}
	}
	defer func() {
		if !ok {
			_ = os.Unsetenv(gcpFunctionNameKey)
		} else {
			_ = os.Setenv(gcpFunctionNameKey, oldValue)
		}
	}()
	tests := []struct {
		name     string
		cr       *CloudRun
		expected want
	}{
		{
			name: "error in reading ProjectID",
			cr: &CloudRun{
				mc: &metaDataClientImpl{
					projectID: func() (string, error) {
						return "", errTest
					},
				},
			},
			expected: want{
				res: nil,
				err: errTest,
			},
		},
		{
			name: "error in reading region",
			cr: &CloudRun{
				mc: &metaDataClientImpl{
					get: func(key string) (string, error) {
						return "", errTest
					},
				},
			},
			expected: want{
				res: nil,
				err: errTest,
			},
		},
		{
			name: "success",
			cr: &CloudRun{
				mc: &metaDataClientImpl{
					projectID: func() (string, error) {
						return projectIDValue, nil
					},
					get: func(key string) (string, error) {
						return regionValue, nil
					},
				},
			},
			expected: want{
				res: resource.NewSchemaless([]attribute.KeyValue{
					semconv.CloudProviderGCP,
					semconv.CloudPlatformGCPCloudFunctions,
					semconv.FaaSNameKey.String(functionName),
					semconv.CloudAccountIDKey.String(projectIDValue),
					semconv.CloudRegionKey.String(regionValue),
				}...),
				err: nil,
			},
		},
	}

	for _, test := range tests {
		detector := cloudFunction{
			cloudRun: test.cr,
		}
		res, err := detector.Detect(context.Background())
		if err != test.expected.err {
			t.Fatalf("got unexpected failure: %v", err)
		} else if diff := cmp.Diff(test.expected.res, res); diff != "" {
			t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
		}
	}
}

func TestNotOnCloudFunction(t *testing.T) {
	oldValue, ok := os.LookupEnv(gcpFunctionNameKey)
	if ok {
		_ = os.Unsetenv(gcpFunctionNameKey)
	}
	defer func() {
		if ok {
			_ = os.Setenv(gcpFunctionNameKey, oldValue)
		}
	}()
	detector := NewCloudFunction()
	res, err := detector.Detect(context.Background())
	if err != nil {
		t.Errorf("expected cloud function detector to return error as nil, but returned %v", err)
	} else if res != nil {
		t.Errorf("expected cloud function detector to return resource as nil, but returned %v", res)
	}
}
