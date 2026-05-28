// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8snode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newFakeNode(name, uid string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(uid),
		},
	}
}

const (
	testNodeName = "my-node"
	testNodeUID  = "4b15c589-1a33-42cc-927a-b78ba9947095"
)

func TestDetect(t *testing.T) {
	client := k8sfake.NewSimpleClientset(newFakeNode(testNodeName, testNodeUID))
	t.Setenv("K8S_NODE_NAME", testNodeName)

	res, err := NewResourceDetector(WithKubeClient(client)).Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.K8SNodeName(testNodeName),
		semconv.K8SNodeUID(testNodeUID),
	)
	assert.Equal(t, expected, res)
}
