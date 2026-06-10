// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8sapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newFakeNode(uid types.UID) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			UID:  uid,
		},
	}
}

func newFakeNamespace(uid types.UID) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  uid,
		},
	}
}

const testNodeName = "my-node"

func TestDetect(t *testing.T) {
	nodeUID := uuid.NewUUID()
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newFakeNode(nodeUID),
		newFakeNamespace(clusterUID),
	)
	t.Setenv("K8S_NODE_NAME", testNodeName)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SNodeName(testNodeName),
		semconv.K8SNodeUID(string(nodeUID)),
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectWithFilter(t *testing.T) {
	nodeUID := uuid.NewUUID()
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newFakeNode(nodeUID),
		newFakeNamespace(clusterUID),
	)
	t.Setenv("K8S_NODE_NAME", testNodeName)

	filter := attribute.NewDenyKeysFilter(semconv.K8SNodeNameKey, semconv.K8SNodeUIDKey)
	res, err := NewResourceDetector(WithKubeClient(client), WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectClusterUIDError(t *testing.T) {
	nodeUID := uuid.NewUUID()

	client := k8sfake.NewClientset(newFakeNode(nodeUID))
	t.Setenv("K8S_NODE_NAME", testNodeName)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SNodeName(testNodeName),
		semconv.K8SNodeUID(string(nodeUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectNodeError(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(newFakeNamespace(clusterUID))
	t.Setenv("K8S_NODE_NAME", testNodeName)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectNodeOnlyNoClusterRBAC(t *testing.T) {
	nodeUID := uuid.NewUUID()

	// no kube-system namespace — simulates missing RBAC for namespace GET
	client := k8sfake.NewClientset(newFakeNode(nodeUID))
	t.Setenv("K8S_NODE_NAME", testNodeName)

	filter := attribute.NewAllowKeysFilter(semconv.K8SNodeNameKey, semconv.K8SNodeUIDKey)
	res, err := NewResourceDetector(WithKubeClient(client), WithAttributeFilter(filter)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SNodeName(testNodeName),
		semconv.K8SNodeUID(string(nodeUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectClusterUIDOnlyNoNodeEnv(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(newFakeNamespace(clusterUID))
	// K8S_NODE_NAME intentionally not set — cluster UID is independently detectable

	filter := attribute.NewAllowKeysFilter(semconv.K8SClusterUIDKey)
	res, err := NewResourceDetector(WithKubeClient(client), WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectBothError(t *testing.T) {
	client := k8sfake.NewClientset()
	t.Setenv("K8S_NODE_NAME", testNodeName)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectInClusterConfigError(t *testing.T) {
	// Set the env vars that make InClusterConfig proceed past the ErrNotInCluster
	// guard, so it fails on the missing token file instead.
	t.Setenv("KUBERNETES_SERVICE_HOST", "fake-host")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")
	t.Setenv("K8S_NODE_NAME", testNodeName)

	_, err := NewResourceDetector().Detect(t.Context())
	require.ErrorContains(t, err, "k8sapi detector:")
}
