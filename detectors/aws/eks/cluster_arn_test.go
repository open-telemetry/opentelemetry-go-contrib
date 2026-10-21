// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	testClusterName = "example-cluster"
	testClusterARN  = "arn:aws:eks:us-west-2:123456789012:cluster/example-cluster"
	testIssuer      = "https://oidc.eks.us-west-2.amazonaws.com/id/EXAMPLE"
)

type arnTestUtils struct {
	clusterName string
	issuer      string
	issuerErr   error
	issuerCalls int
	fileMissing bool
}

func (u *arnTestUtils) fileExists(string) bool {
	return !u.fileMissing
}

func (u *arnTestUtils) getConfigMap(_ context.Context, namespace, name string) (map[string]string, error) {
	if namespace == authConfigmapNS && name == authConfigmapName {
		return map[string]string{"mapRoles": "present"}, nil
	}
	return map[string]string{"cluster.name": u.clusterName}, nil
}

func (*arnTestUtils) getContainerID() (string, error) {
	return "container-id", nil
}

func (u *arnTestUtils) getOIDCIssuer(context.Context) (string, error) {
	u.issuerCalls++
	return u.issuer, u.issuerErr
}

type mockEKSClient struct {
	output        *eks.DescribeClusterOutput
	err           error
	calls         int
	name          string
	hasDeadline   bool
	returnContext bool
}

func (m *mockEKSClient) DescribeCluster(ctx context.Context, input *eks.DescribeClusterInput, _ ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	m.calls++
	m.name = aws.ToString(input.Name)
	_, m.hasDeadline = ctx.Deadline()
	if m.returnContext {
		return nil, ctx.Err()
	}
	return m.output, m.err
}

func describedCluster(arn, issuer string) *eks.DescribeClusterOutput {
	return &eks.DescribeClusterOutput{Cluster: &ekstypes.Cluster{
		Arn: aws.String(arn),
		Identity: &ekstypes.Identity{Oidc: &ekstypes.OIDC{
			Issuer: aws.String(issuer),
		}},
	}}
}

func TestDetectWithClusterARN(t *testing.T) {
	utils := &arnTestUtils{clusterName: testClusterName, issuer: testIssuer}
	client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer+"/")}
	detector := &resourceDetector{
		utils: utils,
		clusterARN: &clusterARNDetector{
			utils:  utils,
			client: client,
		},
	}

	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.Equal(t, resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName(testClusterName),
		semconv.ContainerID("container-id"),
		semconv.AWSEKSClusterARN(testClusterARN),
	), res)
	assert.Equal(t, 1, utils.issuerCalls)
	assert.Equal(t, 1, client.calls)
	assert.Equal(t, testClusterName, client.name)
	assert.True(t, client.hasDeadline)
}

func TestDetectWithClusterARNMismatchReturnsPartialResource(t *testing.T) {
	utils := &arnTestUtils{clusterName: testClusterName, issuer: testIssuer}
	client := &mockEKSClient{output: describedCluster(testClusterARN, "https://example.invalid/other")}
	detector := &resourceDetector{
		utils: utils,
		clusterARN: &clusterARNDetector{
			utils:  utils,
			client: client,
		},
	}

	res, err := detector.Detect(t.Context())

	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorIs(t, err, errClusterMismatch)
	assert.Equal(t, resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName(testClusterName),
		semconv.ContainerID("container-id"),
	), res)
	assert.Equal(t, 1, utils.issuerCalls)
	assert.Equal(t, 1, client.calls)
}

func TestDetectWithoutClusterARNMakesNoAdditionalRequests(t *testing.T) {
	utils := &arnTestUtils{
		clusterName: testClusterName,
		issuerErr:   errors.New("must not be called"),
	}
	detector := &resourceDetector{utils: utils}

	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.Equal(t, resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
		semconv.K8SClusterName(testClusterName),
		semconv.ContainerID("container-id"),
	), res)
	assert.Zero(t, utils.issuerCalls)
}

func TestDetectWithClusterARNRequiresClusterName(t *testing.T) {
	utils := &arnTestUtils{issuer: testIssuer}
	client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
	detector := &resourceDetector{
		utils: utils,
		clusterARN: &clusterARNDetector{
			utils:  utils,
			client: client,
		},
	}

	res, err := detector.Detect(t.Context())

	require.Error(t, err)
	assert.ErrorIs(t, err, resource.ErrPartialResource)
	assert.ErrorIs(t, err, errClusterNameUnavailable)
	assert.NotNil(t, res)
	assert.Zero(t, utils.issuerCalls)
	assert.Zero(t, client.calls)
}

func TestWithClusterARN(t *testing.T) {
	var cfg config
	WithClusterARN().apply(&cfg)
	assert.True(t, cfg.clusterARN)
}

func TestGetOIDCIssuer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"issuer":"` + testIssuer + `"}`))
			assert.NoError(t, err)
		}))
		t.Cleanup(srv.Close)

		issuer, err := (eksDetectorUtils{host: srv.URL, client: srv.Client()}).getOIDCIssuer(t.Context())

		require.NoError(t, err)
		assert.Equal(t, testIssuer, issuer)
	})

	t.Run("non-success response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)

		_, err := (eksDetectorUtils{host: srv.URL, client: srv.Client()}).getOIDCIssuer(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "unexpected status 403 Forbidden")
	})
}

func TestClusterARNFailures(t *testing.T) {
	tests := []struct {
		name    string
		utils   *arnTestUtils
		output  *eks.DescribeClusterOutput
		apiErr  error
		wantErr error
	}{
		{
			name:    "live issuer request fails",
			utils:   &arnTestUtils{clusterName: testClusterName, issuerErr: errors.New("forbidden")},
			wantErr: errLiveIssuerUnavailable,
		},
		{
			name:    "live issuer is empty",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: "  //"},
			wantErr: errLiveIssuerUnavailable,
		},
		{
			name:    "describe cluster fails",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			apiErr:  errors.New("access denied"),
			wantErr: errDescribeClusterFailed,
		},
		{
			name:    "nil response",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			wantErr: errClusterResponseMissing,
		},
		{
			name:    "response has no cluster",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			output:  &eks.DescribeClusterOutput{},
			wantErr: errClusterResponseMissing,
		},
		{
			name:    "cluster has no identity",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			output:  &eks.DescribeClusterOutput{Cluster: &ekstypes.Cluster{}},
			wantErr: errClusterIssuerUnavailable,
		},
		{
			name:  "cluster has no OIDC identity",
			utils: &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			output: &eks.DescribeClusterOutput{Cluster: &ekstypes.Cluster{
				Identity: &ekstypes.Identity{},
			}},
			wantErr: errClusterIssuerUnavailable,
		},
		{
			name:    "cluster issuer is empty",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			output:  describedCluster(testClusterARN, " / "),
			wantErr: errClusterIssuerUnavailable,
		},
		{
			name:    "cluster ARN is empty",
			utils:   &arnTestUtils{clusterName: testClusterName, issuer: testIssuer},
			output:  describedCluster("", testIssuer),
			wantErr: errClusterARNUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockEKSClient{output: tt.output, err: tt.apiErr}
			detector := &clusterARNDetector{utils: tt.utils, client: client}

			arn, err := detector.detect(t.Context(), testClusterName)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, arn)
		})
	}
}

func TestClusterARNClientInitialization(t *testing.T) {
	originalLoad := loadAWSConfig
	originalNew := newEKSClient
	t.Cleanup(func() {
		loadAWSConfig = originalLoad
		newEKSClient = originalNew
	})

	t.Run("configuration failure", func(t *testing.T) {
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, errors.New("no region")
		}
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		_, err := detector.detect(t.Context(), testClusterName)

		require.Error(t, err)
		assert.ErrorIs(t, err, errAWSConfigUnavailable)
	})

	t.Run("client is initialized once", func(t *testing.T) {
		loadCalls := 0
		newCalls := 0
		client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			loadCalls++
			return aws.Config{}, nil
		}
		newEKSClient = func(aws.Config) eksClient {
			newCalls++
			return client
		}
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		for range 2 {
			arn, err := detector.detect(t.Context(), testClusterName)
			require.NoError(t, err)
			assert.Equal(t, testClusterARN, arn)
		}
		assert.Equal(t, 1, loadCalls)
		assert.Equal(t, 1, newCalls)
		assert.Equal(t, 2, client.calls)
	})
}

func TestNewResourceDetectorWithOptionsWiresClusterARN(t *testing.T) {
	utils := &eksDetectorUtils{host: "https://kubernetes.invalid", client: &http.Client{}}
	detector, ok := newResourceDetector(utils, nil, WithClusterARN()).(*resourceDetector)
	require.True(t, ok)
	assert.NotNil(t, detector.clusterARN)
}

func TestNewResourceDetectorDoesNotWireClusterARNAfterInitializationFailure(t *testing.T) {
	initializationErr := errors.New("kubernetes initialization failed")
	detector, ok := newResourceDetector(nil, initializationErr, WithClusterARN()).(*resourceDetector)
	require.True(t, ok)
	assert.ErrorIs(t, detector.err, initializationErr)
	assert.Nil(t, detector.clusterARN)
}

func TestGetOIDCIssuerRejectsMalformedDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"issuer":`))
		assert.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	_, err := (eksDetectorUtils{host: srv.URL, client: srv.Client()}).getOIDCIssuer(t.Context())

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode OIDC discovery document")
}

func TestDefaultConstructorsLeaveClusterARNDisabled(t *testing.T) {
	for _, detector := range []resource.Detector{
		NewResourceDetector(),
		NewResourceDetectorWithOptions(),
	} {
		internal, ok := detector.(*resourceDetector)
		require.True(t, ok)
		assert.Nil(t, internal.clusterARN)
	}
}

func TestClusterARNOptionMakesNoRequestsOutsideEKS(t *testing.T) {
	utils := &arnTestUtils{fileMissing: true, issuer: testIssuer}
	client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
	detector := &resourceDetector{
		utils: utils,
		clusterARN: &clusterARNDetector{
			utils:  utils,
			client: client,
		},
	}

	res, err := detector.Detect(t.Context())

	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
	assert.Zero(t, utils.issuerCalls)
	assert.Zero(t, client.calls)
}

func TestClusterARNPropagatesDescribeClusterCancellation(t *testing.T) {
	utils := &arnTestUtils{issuer: testIssuer}
	client := &mockEKSClient{returnContext: true}
	detector := &clusterARNDetector{utils: utils, client: client}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := detector.detect(ctx, testClusterName)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, err, errDescribeClusterFailed)
}

func TestGetOIDCIssuerHonorsCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := (eksDetectorUtils{
		host:   "https://kubernetes.invalid",
		client: &http.Client{},
	}).getOIDCIssuer(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetOIDCIssuerRejectsOversizedDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(strings.Repeat(" ", maxOIDCDiscoveryBodySize+1)))
		assert.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	_, err := (eksDetectorUtils{host: srv.URL, client: srv.Client()}).getOIDCIssuer(t.Context())

	require.Error(t, err)
	assert.ErrorContains(t, err, "OIDC discovery document exceeds")
}
