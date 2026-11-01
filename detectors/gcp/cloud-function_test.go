// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type cloudFunctionTestCtxKey struct{}

var errTest = errors.New("testError")

const (
	projectIDValue = "some-projectID"
	regionValue    = "some-region"
	functionName   = "sample-function"
)

type metaDataClientImpl struct {
	projectID  func() (string, error)
	get        func(string) (string, error)
	instanceID func() (string, error)
	contexts   []context.Context
}

func (mock *metaDataClientImpl) ProjectIDWithContext(ctx context.Context) (string, error) {
	mock.contexts = append(mock.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if mock.projectID != nil {
		return mock.projectID()
	}
	return "", nil
}

func (mock *metaDataClientImpl) GetWithContext(ctx context.Context, key string) (string, error) {
	mock.contexts = append(mock.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if mock.get != nil {
		return mock.get(key)
	}
	return "", nil
}

func (mock *metaDataClientImpl) InstanceIDWithContext(ctx context.Context) (string, error) {
	mock.contexts = append(mock.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return "", err
	}
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
	t.Setenv(gcpFunctionNameKey, functionName)

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
					get: func(string) (string, error) {
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
					get: func(string) (string, error) {
						return regionValue, nil
					},
				},
			},
			expected: want{
				res: resource.NewSchemaless([]attribute.KeyValue{
					semconv.CloudProviderGCP,
					semconv.CloudPlatformGCPCloudFunctions,
					semconv.FaaSName(functionName),
					semconv.CloudAccountID(projectIDValue),
					semconv.CloudRegion(regionValue),
				}...),
				err: nil,
			},
		},
	}

	for _, test := range tests {
		detector := cloudFunction{
			cloudRun: test.cr,
		}
		res, err := detector.Detect(t.Context())
		if !errors.Is(err, test.expected.err) {
			t.Fatalf("got unexpected failure: %v", err)
		} else if diff := cmp.Diff(test.expected.res, res); diff != "" {
			t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
		}
	}
}

func TestNotOnCloudFunction(t *testing.T) {
	detector := NewCloudFunction()
	res, err := detector.Detect(t.Context())
	if err != nil {
		t.Errorf("expected cloud function detector to return error as nil, but returned %v", err)
	} else if res != nil {
		t.Errorf("expected cloud function detector to return resource as nil, but returned %v", res)
	}
}

func TestCloudFunctionDetectContextPropagation(t *testing.T) {
	t.Setenv(gcpFunctionNameKey, functionName)

	t.Run("propagates context value to ProjectID and cloudRegion", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), cloudFunctionTestCtxKey{}, "cf-ctx")
		mc := &metaDataClientImpl{
			projectID: func() (string, error) { return projectIDValue, nil },
			get:       func(string) (string, error) { return regionValue, nil },
		}
		detector := cloudFunction{cloudRun: &CloudRun{mc: mc}}

		if _, err := detector.Detect(ctx); err != nil {
			t.Fatalf("got unexpected failure: %v", err)
		}
		if len(mc.contexts) != 2 {
			t.Fatalf("expected 2 metadata calls, got %d", len(mc.contexts))
		}
		for i, reqCtx := range mc.contexts {
			if got := reqCtx.Value(cloudFunctionTestCtxKey{}); got != "cf-ctx" {
				t.Errorf("metadata call %d context value = %v, want %q", i, got, "cf-ctx")
			}
		}
	})

	t.Run("ProjectID and cloudRegion respect canceled and deadline exceeded contexts", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(t.Context())
		cancel()
		expiredCtx, cancelDeadline := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancelDeadline()

		for _, tc := range []struct {
			name    string
			ctx     context.Context
			wantErr error
		}{
			{name: "canceled", ctx: canceledCtx, wantErr: context.Canceled},
			{name: "deadline exceeded", ctx: expiredCtx, wantErr: context.DeadlineExceeded},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mc := &metaDataClientImpl{
					projectID: func() (string, error) { return projectIDValue, nil },
					get:       func(string) (string, error) { return regionValue, nil },
				}
				detector := cloudFunction{cloudRun: &CloudRun{mc: mc}}
				if _, err := detector.Detect(tc.ctx); !errors.Is(err, tc.wantErr) {
					t.Errorf("Detect(ctx) error = %v, want %v", err, tc.wantErr)
				}

				ctxAfterProjectID, cancelAfterProjectID := context.WithCancel(t.Context())
				mcRegion := &metaDataClientImpl{
					projectID: func() (string, error) {
						cancelAfterProjectID()
						return projectIDValue, nil
					},
					get: func(string) (string, error) { return regionValue, nil },
				}
				detectorRegion := cloudFunction{cloudRun: &CloudRun{mc: mcRegion}}
				if _, err := detectorRegion.Detect(ctxAfterProjectID); !errors.Is(err, context.Canceled) {
					t.Errorf("Detect(ctx) after ProjectID cancel error = %v, want %v", err, context.Canceled)
				}
			})
		}
	})
}
