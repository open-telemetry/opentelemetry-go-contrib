// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kubeadm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

const testClusterName = "my-cluster"

func newFakeConfigMap(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConfigMapName,
			Namespace: defaultKubeSystemNamespace,
		},
		Data: data,
	}
}

func newKubeadmConfigMap() *corev1.ConfigMap {
	return newFakeConfigMap(map[string]string{
		clusterConfigurationKey: "apiVersion: kubeadm.k8s.io/v1beta4\n" +
			"kind: ClusterConfiguration\n" +
			"clusterName: " + testClusterName + "\n",
	})
}

func newFakeNamespace(uid types.UID) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultKubeSystemNamespace,
			UID:  uid,
		},
	}
}

// denyGet makes every get of the named resource fail with a Forbidden error,
// simulating missing RBAC.
func denyGet(client *k8sfake.Clientset, resourceName string) {
	client.PrependReactor("get", resourceName, func(action k8stesting.Action) (bool, runtime.Object, error) {
		gr := schema.GroupResource{Resource: resourceName}
		return true, nil, apierrors.NewForbidden(gr, action.(k8stesting.GetAction).GetName(), errors.New("no RBAC"))
	})
}

func TestDetect(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newKubeadmConfigMap(),
		newFakeNamespace(clusterUID),
	)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName(testClusterName),
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectWithFilter(t *testing.T) {
	client := k8sfake.NewClientset(
		newKubeadmConfigMap(),
		newFakeNamespace(uuid.NewUUID()),
	)

	filter := attribute.NewAllowKeysFilter(semconv.K8SClusterNameKey)
	res, err := NewResourceDetector(WithKubeClient(client), WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName(testClusterName),
	)
	assert.Equal(t, expected, res)
}

func TestDetectNotKubeadmCluster(t *testing.T) {
	// A reachable cluster without the kubeadm ConfigMap was not provisioned
	// with kubeadm. Detection does not apply and must stay silent.
	client := k8sfake.NewClientset(newFakeNamespace(uuid.NewUUID()))

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectConfigMapForbidden(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newKubeadmConfigMap(),
		newFakeNamespace(clusterUID),
	)
	denyGet(client, "configmaps")

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectMissingClusterConfiguration(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newFakeConfigMap(map[string]string{"ClusterStatus": "{}"}),
		newFakeNamespace(clusterUID),
	)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectMalformedClusterConfiguration(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newFakeConfigMap(map[string]string{clusterConfigurationKey: "\tnot: [valid yaml"}),
		newFakeNamespace(clusterUID),
	)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectEmptyClusterName(t *testing.T) {
	clusterUID := uuid.NewUUID()

	client := k8sfake.NewClientset(
		newFakeConfigMap(map[string]string{clusterConfigurationKey: "kind: ClusterConfiguration\n"}),
		newFakeNamespace(clusterUID),
	)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDetectNamespaceError(t *testing.T) {
	// The kubeadm ConfigMap exists but the namespace itself cannot be read,
	// e.g. because RBAC only grants access to configmaps.
	client := k8sfake.NewClientset(newKubeadmConfigMap())

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName(testClusterName),
	)
	assert.Equal(t, expected, res)
}

func TestDetectEmptyNamespaceUID(t *testing.T) {
	client := k8sfake.NewClientset(
		newKubeadmConfigMap(),
		newFakeNamespace(""),
	)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName(testClusterName),
	)
	assert.Equal(t, expected, res)
}

func TestDetectAllAttributesUnavailable(t *testing.T) {
	client := k8sfake.NewClientset(newKubeadmConfigMap())
	denyGet(client, "configmaps")

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.ErrorIs(t, err, resource.ErrPartialResource)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectNotInCluster(t *testing.T) {
	// Empty host/port makes rest.InClusterConfig return ErrNotInCluster.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectInClusterConfigError(t *testing.T) {
	rd := NewResourceDetector()
	rd.inClusterConfig = func() (*rest.Config, error) {
		return nil, errors.New("injected failure")
	}

	_, err := rd.Detect(t.Context())
	require.Error(t, err)
	require.NotErrorIs(t, err, resource.ErrPartialResource)
}

func TestDetectCreateProviderError(t *testing.T) {
	rd := NewResourceDetector()
	rd.inClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}
	rd.createProvider = func(*rest.Config) (kubernetes.Interface, error) {
		return nil, errors.New("injected failure")
	}

	_, err := rd.Detect(t.Context())
	require.Error(t, err)
	require.NotErrorIs(t, err, resource.ErrPartialResource)
}

func TestDetectInClusterSuccess(t *testing.T) {
	clusterUID := uuid.NewUUID()

	rd := NewResourceDetector()
	rd.inClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}
	rd.createProvider = func(*rest.Config) (kubernetes.Interface, error) {
		return k8sfake.NewClientset(
			newKubeadmConfigMap(),
			newFakeNamespace(clusterUID),
		), nil
	}

	res, err := rd.Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SClusterName(testClusterName),
		semconv.K8SClusterUID(string(clusterUID)),
	)
	assert.Equal(t, expected, res)
}

func TestDefaultCreateProvider(t *testing.T) {
	rd := NewResourceDetector()
	client, err := rd.createProvider(&rest.Config{Host: "https://example.com"})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestDetectFetchesEachObjectOnce(t *testing.T) {
	client := k8sfake.NewClientset(
		newKubeadmConfigMap(),
		newFakeNamespace(uuid.NewUUID()),
	)

	gets := make(map[string]int)
	client.PrependReactor("get", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gets[action.GetResource().Resource]++
		return false, nil, nil
	})

	_, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, map[string]int{"configmaps": 1, "namespaces": 1}, gets)
}
