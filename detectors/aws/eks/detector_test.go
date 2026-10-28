// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"k8s.io/client-go/rest"
)

// errIMDSClient is an imdsClient that is always unreachable. It keeps tests
// from querying the real IMDS endpoint.
type errIMDSClient struct{}

func (errIMDSClient) GetInstanceIdentityDocument(context.Context, *imds.GetInstanceIdentityDocumentInput, ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	return nil, errors.New("imds unreachable")
}

func (errIMDSClient) GetMetadata(context.Context, *imds.GetMetadataInput, ...func(*imds.Options)) (*imds.GetMetadataOutput, error) {
	return nil, errors.New("imds unreachable")
}

// newTestDetector returns a ResourceDetector that does not reach any real
// IMDS, EC2, or Kubernetes endpoint unless overridden by opts.
func newTestDetector(t *testing.T, utils detectorUtils, opts ...Option) *ResourceDetector {
	t.Helper()
	t.Setenv(defaultNodeEnvVar, "")

	cfg := config{nodeEnvVar: defaultNodeEnvVar, imdsClient: errIMDSClient{}}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg, utils: utils}
}

type mockIMDSClient struct {
	mock.Mock
}

func (m *mockIMDSClient) GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, _ ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*imds.GetInstanceIdentityDocumentOutput), args.Error(1)
}

func (m *mockIMDSClient) GetMetadata(ctx context.Context, params *imds.GetMetadataInput, _ ...func(*imds.Options)) (*imds.GetMetadataOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*imds.GetMetadataOutput), args.Error(1)
}

type mockEC2Client struct {
	mock.Mock
}

func (m *mockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

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

// Mock function for getServerVersion().
func (detectorUtils *MockDetectorUtils) getServerVersion(context.Context) (string, error) {
	for _, call := range detectorUtils.ExpectedCalls {
		if call.Method == "getServerVersion" {
			args := detectorUtils.Called()
			return args.String(0), args.Error(1)
		}
	}
	return "", errors.New("server version unavailable")
}

// Mock function for getNodeProviderID().
func (detectorUtils *MockDetectorUtils) getNodeProviderID(_ context.Context, nodeName string) (string, error) {
	args := detectorUtils.Called(nodeName)
	return args.String(0), args.Error(1)
}

func TestResourceDetectorDetect(t *testing.T) {
	var nilMap map[string]string
	errInitialization := errors.New("initialization failure")

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
			detectorErr:   errInitialization,
			expectedErrIs: errInitialization,
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
			name: "auth configmap error not EKS",
			setupUtils: func(m *MockDetectorUtils) {
				m.On("fileExists", k8sTokenPath).Return(true)
				m.On("fileExists", k8sCertPath).Return(true)
				m.On("getConfigMap", authConfigmapNS, authConfigmapName).Return(nilMap, errors.New("auth cm failure"))
			},
			expectedResource: resource.Empty(),
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

			detector := newTestDetector(t, detectorUtils)
			detector.err = tt.detectorErr
			res, err := detector.Detect(t.Context())

			switch {
			case tt.expectedErrIs != nil:
				assert.ErrorIs(t, err, tt.expectedErrIs)
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

// deadlineRoundTripper records the deadline of the request context and
// returns a static ConfigMap response without touching the network.
type deadlineRoundTripper struct {
	deadline time.Time
	ok       bool
}

func (rt *deadlineRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.deadline, rt.ok = req.Context().Deadline()
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		Request:    req,
	}, nil
}

func TestGetConfigMapRequestDeadline(t *testing.T) {
	t.Run("no caller deadline is bounded by configMapTimeout", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		before := time.Now()
		_, err := utils.getConfigMap(context.WithoutCancel(t.Context()), authConfigmapNS, authConfigmapName)
		after := time.Now()
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.WithinRange(t, rt.deadline, before.Add(configMapTimeout), after.Add(configMapTimeout))
	})

	t.Run("shorter caller deadline is preserved", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		callerDeadline := time.Now().Add(time.Second)
		ctx, cancel := context.WithDeadline(t.Context(), callerDeadline)
		t.Cleanup(cancel)

		_, err := utils.getConfigMap(ctx, authConfigmapNS, authConfigmapName)
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.Equal(t, callerDeadline, rt.deadline)
	})
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

func TestDetectWithIRSA(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "irsa-cluster"}, nil)
	utils.On("getContainerID").Return("irsa-container-123456789012345678901234567890123456789012345678901234", nil)

	detector := newTestDetector(t, utils)
	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName("irsa-cluster"),
		semconv.ContainerID("irsa-container-123456789012345678901234567890123456789012345678901234"),
	}, res.Attributes())
	utils.AssertExpectations(t)
}

func TestDetectWithPodIdentity(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsContainerAuthTokenFileEnvVar, "/var/run/secrets/eks-pod-identity/token")

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{}, nil)
	utils.On("getContainerID").Return("", nil)

	detector := newTestDetector(t, utils)
	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
	}, res.Attributes())
	utils.AssertExpectations(t)
}

func TestDetectWithServerVersion(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")

	utils := new(MockDetectorUtils)
	utils.On("getServerVersion").Return("v1.30.2-eks-123456", nil)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "ver-cluster"}, nil)
	utils.On("getContainerID").Return("", nil)

	detector := newTestDetector(t, utils)
	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.ElementsMatch(t, []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName("ver-cluster"),
	}, res.Attributes())
	utils.AssertExpectations(t)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "filtered-cluster"}, nil)
	utils.On("getContainerID").Return("cid", nil)

	detector := newTestDetector(t, utils, WithAttributeFilter(func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.K8SClusterNameKey
	}))
	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []attribute.KeyValue{
		semconv.K8SClusterName("filtered-cluster"),
	}, res.Attributes())
	utils.AssertExpectations(t)
}

func TestWithNodeEnvVar(t *testing.T) {
	detector := NewResourceDetector(WithNodeEnvVar("CUSTOM_NODE_NAME"))
	require.NotNil(t, detector)
	assert.Equal(t, "CUSTOM_NODE_NAME", detector.cfg.nodeEnvVar)
}

func TestComposition_MergeWithDefault(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "merge-cluster"}, nil)
	utils.On("getContainerID").Return("", nil)

	detected, err := newTestDetector(t, utils).Detect(t.Context())
	require.NoError(t, err)

	merged, err := resource.Merge(resource.Default(), detected)
	require.NoError(t, err)
	assert.NotErrorIs(t, err, resource.ErrSchemaURLConflict)
	assert.Equal(t, resource.Default().SchemaURL(), merged.SchemaURL())
}

func TestComposition_WithCoreDetectors(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{"cluster.name": "core-cluster"}, nil)
	utils.On("getContainerID").Return("", nil)

	res, err := resource.New(t.Context(),
		resource.WithDetectors(newTestDetector(t, utils)),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcessPID(),
		resource.WithTelemetrySDK(),
	)
	require.NoError(t, err)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
}

func TestDetectFromIMDS_AllAttributes(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	imdsMock := new(mockIMDSClient)
	imdsMock.On("GetInstanceIdentityDocument", mock.Anything, mock.Anything).Return(
		&imds.GetInstanceIdentityDocumentOutput{
			InstanceIdentityDocument: imds.InstanceIdentityDocument{
				InstanceID:       "i-0123456789abcdef0",
				AvailabilityZone: "us-west-2a",
				Region:           "us-west-2",
				AccountID:        "123456789012",
				ImageID:          "ami-0123456789abcdef0",
				InstanceType:     "m5.large",
			},
		}, nil,
	)
	imdsMock.On("GetMetadata", mock.Anything, &imds.GetMetadataInput{Path: "hostname"}).Return(
		&imds.GetMetadataOutput{
			Content: io.NopCloser(strings.NewReader("ip-10-0-0-1.ec2.internal")),
		}, nil,
	)

	ec2Mock := new(mockEC2Client)
	tagKey := clusterNameAwsEksTag
	tagVal := "imds-cluster"
	ec2Mock.On("DescribeInstances", mock.Anything, &ec2.DescribeInstancesInput{
		InstanceIds: []string{"i-0123456789abcdef0"},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						Tags: []types.Tag{
							{Key: &tagKey, Value: &tagVal},
						},
					},
				},
			},
		},
	}, nil)

	utils := new(MockDetectorUtils)
	utils.On("getContainerID").Return("cid-12345", nil)

	detector := newTestDetector(t, utils, withIMDSClient(imdsMock), withEC2Client(ec2Mock))

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expectedAttrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.CloudAccountID("123456789012"),
		semconv.CloudAvailabilityZone("us-west-2a"),
		semconv.CloudRegion("us-west-2"),
		semconv.HostID("i-0123456789abcdef0"),
		semconv.HostImageID("ami-0123456789abcdef0"),
		semconv.HostType("m5.large"),
		semconv.HostNameKey.String("ip-10-0-0-1.ec2.internal"),
		semconv.K8SClusterName("imds-cluster"),
		semconv.ContainerID("cid-12345"),
	}

	assert.ElementsMatch(t, expectedAttrs, res.Attributes())
	imdsMock.AssertExpectations(t)
	ec2Mock.AssertExpectations(t)
}

func TestDetectFromK8sNode_IMDSDisabled(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsContainerAuthTokenFileEnvVar, "/var/run/secrets/eks-pod-identity/token")
	t.Setenv("TEST_NODE_NAME", "ip-10-0-0-2.ec2.internal")

	imdsMock := new(mockIMDSClient)
	imdsMock.On("GetInstanceIdentityDocument", mock.Anything, mock.Anything).Return(
		(*imds.GetInstanceIdentityDocumentOutput)(nil), errors.New("imds unreachable"),
	)

	ec2Mock := new(mockEC2Client)
	ownerID := "987654321098"
	imgID := "ami-0987654321"
	dnsName := "ip-10-0-0-2.ec2.internal"
	tagKey := clusterNameEksTag
	tagVal := "k8s-node-cluster"
	ec2Mock.On("DescribeInstances", mock.Anything, &ec2.DescribeInstancesInput{
		InstanceIds: []string{"i-0fedcba9876543210"},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				OwnerId: &ownerID,
				Instances: []types.Instance{
					{
						ImageId:        &imgID,
						InstanceType:   types.InstanceTypeC5Xlarge,
						PrivateDnsName: &dnsName,
						Tags: []types.Tag{
							{Key: &tagKey, Value: &tagVal},
						},
					},
				},
			},
		},
	}, nil)

	utils := new(MockDetectorUtils)
	utils.On("getNodeProviderID", "ip-10-0-0-2.ec2.internal").Return("aws:///us-east-1b/i-0fedcba9876543210", nil)
	utils.On("getContainerID").Return("", nil)

	detector := newTestDetector(t, utils,
		withIMDSClient(imdsMock),
		withEC2Client(ec2Mock),
		WithNodeEnvVar("TEST_NODE_NAME"),
	)

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	expectedAttrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.CloudAccountID("987654321098"),
		semconv.CloudAvailabilityZone("us-east-1b"),
		semconv.CloudRegion("us-east-1"),
		semconv.HostID("i-0fedcba9876543210"),
		semconv.HostImageID("ami-0987654321"),
		semconv.HostType("c5.xlarge"),
		semconv.HostNameKey.String("ip-10-0-0-2.ec2.internal"),
		semconv.K8SClusterName("k8s-node-cluster"),
	}

	assert.ElementsMatch(t, expectedAttrs, res.Attributes())
	imdsMock.AssertExpectations(t)
	ec2Mock.AssertExpectations(t)
}

func TestDetectClusterName_ConfigMapFallback(t *testing.T) {
	t.Setenv(kubernetesServiceHostEnvVar, "10.0.0.1")
	t.Setenv(awsWebIdentityTokenFileEnvVar, "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	imdsMock := new(mockIMDSClient)
	imdsMock.On("GetInstanceIdentityDocument", mock.Anything, mock.Anything).Return(
		&imds.GetInstanceIdentityDocumentOutput{
			InstanceIdentityDocument: imds.InstanceIdentityDocument{
				InstanceID: "i-0123456789abcdef0",
				Region:     "us-west-2",
			},
		}, nil,
	)
	imdsMock.On("GetMetadata", mock.Anything, mock.Anything).Return((*imds.GetMetadataOutput)(nil), errors.New("metadata unavailable"))

	// EC2 DescribeInstances returns no cluster tag
	ec2Mock := new(mockEC2Client)
	ec2Mock.On("DescribeInstances", mock.Anything, mock.Anything).Return(&ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{},
	}, nil)

	utils := new(MockDetectorUtils)
	utils.On("getConfigMap", cwConfigmapNS, cwConfigmapName).Return(map[string]string{
		"cluster.name": "configmap-fallback-cluster",
	}, nil)
	utils.On("getContainerID").Return("", nil)

	detector := newTestDetector(t, utils, withIMDSClient(imdsMock), withEC2Client(ec2Mock))

	res, err := detector.Detect(t.Context())
	require.NoError(t, err)

	assert.Contains(t, res.Attributes(), semconv.K8SClusterName("configmap-fallback-cluster"))
	imdsMock.AssertExpectations(t)
	ec2Mock.AssertExpectations(t)
	utils.AssertExpectations(t)
}

func TestGetK8sAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.EscapedPath() {
		case "/version":
			_, _ = w.Write([]byte(`{"gitVersion":"v1.30.2-eks-123456"}`))
		case "/api/v1/nodes/ip-10-0-0-1.ec2.internal":
			_, _ = w.Write([]byte(`{"spec":{"providerID":"aws:///us-west-2a/i-049ca2df511bec762"}}`))
		case "/api/v1/nodes/bad%2Fnode":
			_, _ = w.Write([]byte(`invalid-json`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	utils := &eksDetectorUtils{host: srv.URL, client: srv.Client()}

	v, err := utils.getServerVersion(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "v1.30.2-eks-123456", v)

	id, err := utils.getNodeProviderID(t.Context(), "ip-10-0-0-1.ec2.internal")
	require.NoError(t, err)
	assert.Equal(t, "aws:///us-west-2a/i-049ca2df511bec762", id)

	_, err = utils.getNodeProviderID(t.Context(), "missing-node")
	assert.ErrorContains(t, err, "unexpected status")

	// Node names are path-escaped so they cannot alter the request path.
	_, err = utils.getNodeProviderID(t.Context(), "bad/node")
	assert.ErrorContains(t, err, "failed to decode response")

	// Dot segments are not escaped by url.PathEscape and would otherwise
	// resolve to a different API path.
	for _, name := range []string{"", ".", ".."} {
		_, err = utils.getNodeProviderID(t.Context(), name)
		assert.ErrorContains(t, err, "invalid node name", "node name %q", name)
	}
}

func TestGetK8sAPIRequestDeadline(t *testing.T) {
	rt := &deadlineRoundTripper{}
	utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

	before := time.Now()
	_, err := utils.getServerVersion(context.WithoutCancel(t.Context()))
	after := time.Now()
	require.NoError(t, err)

	require.True(t, rt.ok, "request context must carry a deadline")
	assert.WithinRange(t, rt.deadline, before.Add(k8sAPITimeout), after.Add(k8sAPITimeout))
}

func TestWithAWSConfig(t *testing.T) {
	cfg := aws.Config{Region: "us-east-2"}
	detector := NewResourceDetector(WithAWSConfig(cfg))
	require.NotNil(t, detector)
	require.NotNil(t, detector.cfg.awsConfig)
	assert.Equal(t, "us-east-2", detector.cfg.awsConfig.Region)
}
