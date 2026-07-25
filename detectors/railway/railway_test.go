// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package railway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

func setFullEnv(t *testing.T) {
	t.Helper()
	t.Setenv(projectIDEnvVar, "project-123")
	t.Setenv(projectNameEnvVar, "my-project")
	t.Setenv(environmentIDEnvVar, "env-456")
	t.Setenv(environmentNameEnvVar, "production")
	t.Setenv(serviceIDEnvVar, "service-789")
	t.Setenv(serviceNameEnvVar, "my-service")
	t.Setenv(replicaIDEnvVar, "replica-abc")
	t.Setenv(replicaRegionEnvVar, "us-west1")
	t.Setenv(deploymentIDEnvVar, "deployment-def")
	t.Setenv(gitCommitSHAEnvVar, "e6134959463efd8966b20e75b913cafe3f5ec")
	t.Setenv(gitBranchEnvVar, "main")
	t.Setenv(gitRepoNameEnvVar, "my-repo")
	t.Setenv(gitRepoOwnerEnvVar, "my-org")
}

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	assert.NotNil(t, d)
}

func TestDetect_OK(t *testing.T) {
	setFullEnv(t)

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderKey.String("railway"),
		ProjectIDKey.String("project-123"),
		semconv.ServiceNamespace("my-project"),
		EnvironmentIDKey.String("env-456"),
		semconv.DeploymentEnvironmentNameKey.String("production"),
		ServiceIDKey.String("service-789"),
		semconv.ServiceName("my-service"),
		semconv.ServiceInstanceID("replica-abc"),
		semconv.CloudRegion("us-west1"),
		semconv.DeploymentID("deployment-def"),
		semconv.VCSRefHeadRevision("e6134959463efd8966b20e75b913cafe3f5ec"),
		semconv.VCSRefHeadName("main"),
		semconv.VCSRepositoryName("my-repo"),
		semconv.VCSOwnerName("my-org"),
		semconv.VCSRefHeadTypeBranch,
		semconv.VCSProviderNameGithub,
	)
	assert.Equal(t, expected, res)
}

func TestDetect_NotOnRailway(t *testing.T) {
	t.Setenv(projectIDEnvVar, "")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_PartialEnv(t *testing.T) {
	// Only identity variables are set; no git variables, simulating a
	// deployment from a Docker image rather than a GitHub repository.
	t.Setenv(projectIDEnvVar, "project-123")
	t.Setenv(serviceNameEnvVar, "my-service")
	t.Setenv(replicaRegionEnvVar, "us-west1")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	presentAttrs := []attribute.KeyValue{
		semconv.CloudProviderKey.String("railway"),
		ProjectIDKey.String("project-123"),
		semconv.ServiceName("my-service"),
		semconv.CloudRegion("us-west1"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}

	absentKeys := []attribute.Key{
		semconv.VCSRefHeadRevisionKey,
		semconv.VCSRefHeadNameKey,
		semconv.VCSRepositoryNameKey,
		semconv.VCSOwnerNameKey,
		semconv.VCSRefHeadTypeKey,
		semconv.VCSProviderNameKey,
		semconv.ServiceInstanceIDKey,
		semconv.DeploymentIDKey,
	}
	for _, k := range absentKeys {
		_, ok := res.Set().Value(k)
		assert.False(t, ok, "expected attribute %s to be absent", k)
	}
}

func TestDetect_GitBranchOnly(t *testing.T) {
	t.Setenv(projectIDEnvVar, "project-123")
	t.Setenv(gitBranchEnvVar, "main")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.VCSRefHeadTypeKey)
	assert.True(t, ok, "expected vcs.ref.head.type to be present")
	assert.Equal(t, semconv.VCSRefHeadTypeBranch.Value, val)
}

func TestDetect_GitRepoOwnerOnly(t *testing.T) {
	t.Setenv(projectIDEnvVar, "project-123")
	t.Setenv(gitRepoOwnerEnvVar, "my-org")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	val, ok := res.Set().Value(semconv.VCSProviderNameKey)
	assert.True(t, ok, "expected vcs.provider.name to be present")
	assert.Equal(t, semconv.VCSProviderNameGithub.Value, val)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	setFullEnv(t)

	filter := attribute.NewDenyKeysFilter(semconv.ServiceNameKey)
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.ServiceNameKey)
	assert.False(t, ok, "expected service.name to be absent")

	val, ok := res.Set().Value(semconv.CloudRegionKey)
	assert.True(t, ok, "expected cloud.region to be present")
	assert.Equal(t, attribute.StringValue("us-west1"), val)
}
