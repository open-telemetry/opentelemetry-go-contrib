// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"k8s.io/client-go/rest"
)

type MockDetectorUtils struct {
	mock.Mock
}

// Mock function for fileExists().
func (detectorUtils *MockDetectorUtils) fileExists(filename string) bool {
	args := detectorUtils.Called(filename)
	return args.Bool(0)
}

// Mock function for getConfigMap().
func (detectorUtils *MockDetectorUtils) getConfigMap(_ context.Context, namespace, name string) (map[string]string, error) {
	args := detectorUtils.Called(namespace, name)
	return args.Get(0).(map[string]string), args.Error(1)
}

// Mock function for getContainerID().
func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func TestResourceDetectorDetect(t *testing.T) {
	var nilMap map[string]string

	tests := []struct {
		name             string
		setupUtils       func(*MockDetectorUtils)
		detectorErr      error
		expectedResource *resource.Resource
		expectedErrMsg   string
		expectedErrIs    error
	}{
		{
			name: "success full attributes",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
				m.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
				m.On("getContainerID").Return("0123456789A", nil)
			},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderAWS,
				semconv.CloudPlatformAWSEKS,
				semconv.K8SClusterName("my-cluster"),
				semconv.ContainerID("0123456789A"),
			),
		},
		{
			name: "success empty optional attributes",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
				m.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{}, nil)
				m.On("getContainerID").Return("", nil)
			},
			expectedResource: resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.CloudProviderAWS,
				semconv.CloudPlatformAWSEKS,
			),
		},
		{
			name:             "not in kubernetes cluster error",
			detectorErr:      rest.ErrNotInCluster,
			expectedResource: resource.Empty(),
		},
		{
			name:          "generic initialization error",
			detectorErr:   errors.New("initialization failure"),
			expectedErrIs: errors.New("initialization failure"),
		},
		{
			name: "token missing not EKS",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(false)
			},
			expectedResource: resource.Empty(),
		},
		{
			name: "cert missing not EKS",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(false)
			},
			expectedResource: resource.Empty(),
		},
		{
			name: "auth configmap error",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(nilMap, errors.New("auth cm failure"))
			},
			expectedErrMsg: "isEks() error retrieving auth configmap",
		},
		{
			name: "cluster name lookup error",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
				m.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(nilMap, errors.New("cw lookup failure"))
			},
			expectedErrMsg: "getClusterName() error",
		},
		{
			name: "container ID lookup error",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(map[string]string{"not": "nil"}, nil)
				m.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "my-cluster"}, nil)
				m.On("getContainerID").Return("", errors.New("cgroup parse failure"))
			},
			expectedErrMsg: "cgroup parse failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectorUtils := new(MockDetectorUtils)
			if tt.setupUtils != nil {
				tt.setupUtils(detectorUtils)
			}

			detector := resourceDetector{utils: detectorUtils, err: tt.detectorErr}
			res, err := detector.Detect(t.Context())

			switch {
			case tt.expectedErrIs != nil:
				assert.EqualError(t, err, tt.expectedErrIs.Error())
				assert.Nil(t, res)
			case tt.expectedErrMsg != "":
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.expectedErrMsg)
				assert.Nil(t, res)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResource, res)
			}

			detectorUtils.AssertExpectations(t)
		})
	}
}

func TestNewResourceDetector(t *testing.T) {
	detector := NewResourceDetector()
	require.NotNil(t, detector)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestGetConfigMap(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		closeEarly     bool
		expectedData   map[string]string
		expectedErrMsg string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/namespaces/kube-system/configmaps/aws-auth", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"data":{"mapRoles":"test-role"}}`))
				assert.NoError(t, err)
			},
			expectedData: map[string]string{"mapRoles": "test-role"},
		},
		{
			name: "non-2xx response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedErrMsg: "unexpected status",
		},
		{
			name: "decode failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`invalid-json`))
				assert.NoError(t, err)
			},
			expectedErrMsg: "failed to decode ConfigMap",
		},
		{
			name:           "client transport failure",
			closeEarly:     true,
			expectedErrMsg: "failed to retrieve ConfigMap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			if tt.closeEarly {
				srv.Close()
			} else {
				t.Cleanup(srv.Close)
			}

			utils := &eksDetectorUtils{host: srv.URL, client: srv.Client()}
			data, err := utils.getConfigMap(t.Context(), authConfigmapNS, authConfigmapName)

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.expectedErrMsg)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedData, data)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o600))

	utils := eksDetectorUtils{}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing regular file",
			path:     filePath,
			expected: true,
		},
		{
			name:     "non-existent file",
			path:     filepath.Join(tempDir, "missing.txt"),
			expected: false,
		},
		{
			name:     "directory path",
			path:     tempDir,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, utils.fileExists(tt.path))
		})
	}
}
