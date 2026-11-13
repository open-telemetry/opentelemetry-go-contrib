// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

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
	testIssuer      = "https://oidc.eks.invalid/id/EXAMPLE"
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
	deadline      time.Time
	hasDeadline   bool
	returnContext bool
}

func (m *mockEKSClient) DescribeCluster(ctx context.Context, input *eks.DescribeClusterInput, _ ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	m.calls++
	m.name = aws.ToString(input.Name)
	m.deadline, m.hasDeadline = ctx.Deadline()
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

	before := time.Now()
	res, err := detector.Detect(t.Context())
	after := time.Now()

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
	require.True(t, client.hasDeadline)
	assert.WithinRange(t, client.deadline, before.Add(clusterARNTimeout), after.Add(clusterARNTimeout))
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

	t.Run("invalid host", func(t *testing.T) {
		_, err := (eksDetectorUtils{host: "https://[::1", client: &http.Client{}}).getOIDCIssuer(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to build OIDC discovery URL")
	})

	t.Run("body read fails", func(t *testing.T) {
		readErr := errors.New("connection reset")
		client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(iotest.ErrReader(readErr)),
				Request:    req,
			}, nil
		})}

		_, err := (eksDetectorUtils{host: "https://kubernetes.invalid", client: client}).getOIDCIssuer(t.Context())

		require.Error(t, err)
		assert.ErrorIs(t, err, readErr)
		assert.ErrorContains(t, err, "failed to read OIDC discovery document")
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
			return aws.Config{}, errors.New("configuration failed")
		}
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		_, err := detector.detect(t.Context(), testClusterName)

		require.Error(t, err)
		assert.ErrorIs(t, err, errAWSConfigUnavailable)
	})

	t.Run("missing Region", func(t *testing.T) {
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			return aws.Config{}, nil
		}
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		_, err := detector.detect(t.Context(), testClusterName)

		require.Error(t, err)
		assert.ErrorIs(t, err, errAWSConfigUnavailable)
		assert.ErrorIs(t, err, errAWSRegionUnavailable)
	})

	t.Run("waiter honors its own deadline", func(t *testing.T) {
		loadCalls := 0
		started := make(chan struct{})
		release := make(chan struct{})
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			loadCalls++
			close(started)
			<-release
			return aws.Config{Region: "us-west-2"}, nil
		}
		client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
		newEKSClient = func(aws.Config) eksClient { return client }
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		leaderDone := make(chan error, 1)
		go func() {
			_, err := detector.getClient(t.Context(), testIssuer)
			leaderDone <- err
		}()
		<-started

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := detector.getClient(ctx, testIssuer)
		elapsed := time.Since(start)
		close(release)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, elapsed, time.Second)
		require.NoError(t, <-leaderDone)
		assert.Equal(t, 1, loadCalls)
	})

	t.Run("failed load is retried", func(t *testing.T) {
		loadCalls := 0
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			loadCalls++
			if loadCalls == 1 {
				return aws.Config{}, errors.New("configuration failed")
			}
			return aws.Config{Region: "us-west-2"}, nil
		}
		client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
		newEKSClient = func(aws.Config) eksClient { return client }
		detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

		_, err := detector.detect(t.Context(), testClusterName)
		require.ErrorIs(t, err, errAWSConfigUnavailable)

		arn, err := detector.detect(t.Context(), testClusterName)
		require.NoError(t, err)
		assert.Equal(t, testClusterARN, arn)
		assert.Equal(t, 2, loadCalls)
	})

	t.Run("client is initialized once", func(t *testing.T) {
		loadCalls := 0
		newCalls := 0
		client := &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
		loadAWSConfig = func(context.Context, ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
			loadCalls++
			return aws.Config{Region: "us-west-2"}, nil
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

// issuerURL builds issuer URLs at runtime so the repository link checker does
// not try to fetch these example hosts.
func issuerURL(scheme, host, path string) string {
	return (&url.URL{Scheme: scheme, Host: host, Path: path}).String()
}

// awsTransport answers the STS and EKS requests made by clients built from
// the real AWS configuration loader, so the tests make no network requests.
type awsTransport struct {
	issuer string

	mu    sync.Mutex
	hosts []string
}

func (tr *awsTransport) Do(req *http.Request) (*http.Response, error) {
	tr.mu.Lock()
	tr.hosts = append(tr.hosts, req.URL.Host)
	tr.mu.Unlock()

	var contentType, body string
	switch {
	case strings.HasPrefix(req.URL.Host, "sts."):
		contentType = "text/xml"
		body = `<AssumeRoleWithWebIdentityResponse><AssumeRoleWithWebIdentityResult><Credentials>` +
			`<AccessKeyId>AKIDEXAMPLE</AccessKeyId><SecretAccessKey>SECRET</SecretAccessKey>` +
			`<SessionToken>TOKEN</SessionToken><Expiration>2100-01-01T00:00:00Z</Expiration>` +
			`</Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`
	case strings.HasPrefix(req.URL.Host, "eks."):
		contentType = "application/json"
		b, err := json.Marshal(map[string]any{"cluster": map[string]any{
			"arn":      testClusterARN,
			"identity": map[string]any{"oidc": map[string]any{"issuer": tr.issuer}},
		}})
		if err != nil {
			return nil, err
		}
		body = string(b)
	default:
		return nil, fmt.Errorf("unexpected request to %s", req.URL.Host)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func (tr *awsTransport) requestHosts() []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return slices.Clone(tr.hosts)
}

// setWebIdentityEnv configures web identity (IRSA) credentials, as EKS does
// for a pod with an IAM role, and clears every other credential and Region
// source.
func setWebIdentityEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("token"), 0o600))

	for k, v := range map[string]string{
		"AWS_CONFIG_FILE":                        filepath.Join(dir, "config"),
		"AWS_SHARED_CREDENTIALS_FILE":            filepath.Join(dir, "credentials"),
		"AWS_PROFILE":                            "",
		"AWS_DEFAULT_PROFILE":                    "",
		"AWS_REGION":                             "",
		"AWS_DEFAULT_REGION":                     "",
		"AWS_ACCESS_KEY_ID":                      "",
		"AWS_SECRET_ACCESS_KEY":                  "",
		"AWS_SESSION_TOKEN":                      "",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI":     "",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI": "",
		"AWS_ENDPOINT_URL":                       "",
		"AWS_ENDPOINT_URL_STS":                   "",
		"AWS_ENDPOINT_URL_EKS":                   "",
		"AWS_USE_FIPS_ENDPOINT":                  "",
		"AWS_USE_DUALSTACK_ENDPOINT":             "",
		"AWS_EC2_METADATA_DISABLED":              "true",
		"AWS_WEB_IDENTITY_TOKEN_FILE":            tokenFile,
		"AWS_ROLE_ARN":                           "arn:aws:iam::123456789012:role/example",
		"AWS_ROLE_SESSION_NAME":                  "example",
	} {
		t.Setenv(k, v)
	}
}

func TestClusterARNRegionWithWebIdentityCredentials(t *testing.T) {
	originalLoad := loadAWSConfig
	t.Cleanup(func() { loadAWSConfig = originalLoad })

	issuer := issuerURL("https", "oidc.eks.eu-west-1.amazonaws.com", "/id/EXAMPLE")
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{name: "Region from issuer", want: "eu-west-1"},
		{name: "configured Region takes precedence", region: "us-west-2", want: "us-west-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setWebIdentityEnv(t)
			t.Setenv("AWS_REGION", tt.region)
			transport := &awsTransport{issuer: issuer}
			loadAWSConfig = func(ctx context.Context, opts ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
				return awsconfig.LoadDefaultConfig(ctx, append(opts, awsconfig.WithHTTPClient(transport))...)
			}
			detector := &clusterARNDetector{utils: &arnTestUtils{issuer: issuer}}

			arn, err := detector.detect(t.Context(), testClusterName)

			require.NoError(t, err)
			assert.Equal(t, testClusterARN, arn)
			assert.Equal(t, []string{
				"sts." + tt.want + ".amazonaws.com",
				"eks." + tt.want + ".amazonaws.com",
			}, transport.requestHosts())
		})
	}
}

func TestRegionFromIssuer(t *testing.T) {
	const idPath = "/id/EXAMPLE"
	tests := []struct {
		name   string
		issuer string
		want   string
	}{
		{name: "commercial", issuer: issuerURL("https", "oidc.eks.us-west-2.amazonaws.com", idPath), want: "us-west-2"},
		{name: "GovCloud", issuer: issuerURL("https", "oidc.eks.us-gov-west-1.amazonaws.com", idPath), want: "us-gov-west-1"},
		{name: "China", issuer: issuerURL("https", "oidc.eks.cn-north-1.amazonaws.com.cn", idPath), want: "cn-north-1"},
		{name: "EU sovereign", issuer: issuerURL("https", "oidc.eks.eusc-de-east-1.amazonaws.eu", idPath), want: "eusc-de-east-1"},
		{name: "ISO", issuer: issuerURL("https", "oidc.eks.us-iso-east-1.c2s.ic.gov", idPath), want: "us-iso-east-1"},
		{name: "ISOB", issuer: issuerURL("https", "oidc.eks.us-isob-east-1.sc2s.sgov.gov", idPath), want: "us-isob-east-1"},
		{name: "empty", issuer: ""},
		{name: "test issuer", issuer: testIssuer},
		{name: "http scheme", issuer: issuerURL("http", "oidc.eks.us-west-2.amazonaws.com", idPath)},
		{name: "port", issuer: issuerURL("https", "oidc.eks.us-west-2.amazonaws.com:8443", idPath)},
		{name: "userinfo", issuer: (&url.URL{Scheme: "https", User: url.User("user"), Host: "oidc.eks.us-west-2.amazonaws.com", Path: idPath}).String()},
		{name: "no path", issuer: issuerURL("https", "oidc.eks.us-west-2.amazonaws.com", "")},
		{name: "empty id", issuer: issuerURL("https", "oidc.eks.us-west-2.amazonaws.com", "/id/")},
		{name: "unknown suffix", issuer: issuerURL("https", "oidc.eks.us-west-2.example.com", idPath)},
		{name: "suffix not at end", issuer: issuerURL("https", "oidc.eks.us-west-2.amazonaws.com.example.com", idPath)},
		{name: "upper case Region", issuer: issuerURL("https", "oidc.eks.US-WEST-2.amazonaws.com", idPath)},
		{name: "Region without number", issuer: issuerURL("https", "oidc.eks.us-west.amazonaws.com", idPath)},
		{name: "no oidc prefix", issuer: issuerURL("https", "eks.us-west-2.amazonaws.com", idPath)},
		{name: "API server", issuer: issuerURL("https", "kubernetes.default.svc", "")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, regionFromIssuer(tt.issuer))
		})
	}
}

func TestClusterARNClientInitializationCancellation(t *testing.T) {
	originalLoad := loadAWSConfig
	originalNew := newEKSClient
	t.Cleanup(func() {
		loadAWSConfig = originalLoad
		newEKSClient = originalNew
	})

	loadReturned := make(chan struct{})
	loadAWSConfig = func(ctx context.Context, _ ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		defer close(loadReturned)
		<-ctx.Done()
		return aws.Config{}, ctx.Err()
	}
	newEKSClient = func(aws.Config) eksClient {
		return &mockEKSClient{output: describedCluster(testClusterARN, testIssuer)}
	}
	detector := &clusterARNDetector{utils: &arnTestUtils{issuer: testIssuer}}

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err := detector.detect(ctx, testClusterName)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NotErrorIs(t, err, errAWSConfigUnavailable)
	select {
	case <-loadReturned:
	default:
		t.Fatal("AWS configuration load still running after detect returned")
	}
}

func TestNewResourceDetectorWithOptionsWiresClusterARN(t *testing.T) {
	utils := &eksDetectorUtils{host: "https://kubernetes.invalid", client: &http.Client{}}
	detector, ok := newResourceDetector(utils, nil, WithClusterARN()).(*resourceDetector)
	require.True(t, ok)
	require.NotNil(t, detector.clusterARN)
	assert.Same(t, utils, detector.clusterARN.utils)
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

func TestDefaultLeavesClusterARNDisabled(t *testing.T) {
	utils := &eksDetectorUtils{host: "https://kubernetes.invalid", client: &http.Client{}}
	detector, ok := newResourceDetector(utils, nil).(*resourceDetector)
	require.True(t, ok)
	assert.Nil(t, detector.clusterARN)
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

func TestGetOIDCIssuerRequestDeadline(t *testing.T) {
	t.Run("no caller deadline is bounded by oidcDiscoveryTimeout", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		before := time.Now()
		_, err := utils.getOIDCIssuer(context.WithoutCancel(t.Context()))
		after := time.Now()
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.WithinRange(t, rt.deadline, before.Add(oidcDiscoveryTimeout), after.Add(oidcDiscoveryTimeout))
	})

	t.Run("shorter caller deadline is preserved", func(t *testing.T) {
		rt := &deadlineRoundTripper{}
		utils := &eksDetectorUtils{host: "http://k8s.invalid", client: &http.Client{Transport: rt}}

		callerDeadline := time.Now().Add(time.Second)
		ctx, cancel := context.WithDeadline(t.Context(), callerDeadline)
		defer cancel()

		_, err := utils.getOIDCIssuer(ctx)
		require.NoError(t, err)

		require.True(t, rt.ok, "request context must carry a deadline")
		assert.Equal(t, callerDeadline, rt.deadline)
	})
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
