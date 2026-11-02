// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kubeadm

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.yaml.in/yaml/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// defaultConfigMapName is the ConfigMap kubeadm writes the cluster
	// configuration to.
	defaultConfigMapName = "kubeadm-config"
	// defaultKubeSystemNamespace holds both the kubeadm ConfigMap and the
	// Namespace whose UID identifies the cluster.
	defaultKubeSystemNamespace = "kube-system"
	// clusterConfigurationKey is the ConfigMap data key holding the kubeadm
	// ClusterConfiguration document, serialized as YAML.
	clusterConfigurationKey = "ClusterConfiguration"
)

// clusterConfiguration holds the fields read from kubeadm's
// ClusterConfiguration document.
type clusterConfiguration struct {
	ClusterName string `yaml:"clusterName"`
}

type config struct {
	kubeClient kubernetes.Interface
	filter     attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithKubeClient sets the Kubernetes client used to query the kubeadm
// ConfigMap and the kube-system namespace. If not set, an in-cluster client is
// created automatically during [ResourceDetector.Detect]. This option is
// primarily useful for testing or when running outside a cluster.
func WithKubeClient(client kubernetes.Interface) Option {
	return optionFunc(func(c *config) { c.kubeClient = client })
}

// WithAttributeFilter sets a filter that controls which detected attributes
// are included in the returned resource. Only attributes for which filter
// returns true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource attributes describing the
// kubeadm-provisioned Kubernetes cluster the process is running in.
type ResourceDetector struct {
	cfg                 config
	configMapName       string
	kubeSystemNamespace string
	createProvider      func(*rest.Config) (kubernetes.Interface, error)
	inClusterConfig     func() (*rest.Config, error)
}

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// NewResourceDetector returns a [resource.Detector] that detects the name and
// UID of the kubeadm-provisioned Kubernetes cluster the process is running in.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return &ResourceDetector{
		cfg:                 cfg,
		configMapName:       defaultConfigMapName,
		kubeSystemNamespace: defaultKubeSystemNamespace,
		createProvider: func(c *rest.Config) (kubernetes.Interface, error) {
			return kubernetes.NewForConfig(c)
		},
		inClusterConfig: rest.InClusterConfig,
	}
}

// Detect returns a [*resource.Resource] describing the kubeadm cluster.
//
// An empty resource and no error are returned when the process is not running
// inside a Kubernetes cluster, or when it is but the cluster was not
// provisioned with kubeadm: the absence of the kubeadm ConfigMap is how the
// latter is recognized. If the cluster is a kubeadm cluster but an attribute
// cannot be retrieved, a partial resource is returned together with
// [resource.ErrPartialResource].
func (rd *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	client := rd.cfg.kubeClient
	if client == nil {
		conf, err := rd.inClusterConfig()
		if err != nil {
			if errors.Is(err, rest.ErrNotInCluster) {
				return resource.Empty(), nil
			}
			return nil, fmt.Errorf("kubeadm detector: %w", err)
		}
		client, err = rd.createProvider(conf)
		if err != nil {
			return nil, fmt.Errorf("kubeadm detector: failed to create Kubernetes client: %w", err)
		}
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	cm, err := client.CoreV1().ConfigMaps(rd.kubeSystemNamespace).Get(ctx, rd.configMapName, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		// The cluster is reachable but has no kubeadm ConfigMap: it was not
		// provisioned with kubeadm. Detection does not apply here.
		return resource.Empty(), nil
	case err != nil:
		errs = append(errs, fmt.Errorf("configmap %s/%s: %w", rd.kubeSystemNamespace, rd.configMapName, err))
	default:
		name, nameErr := clusterName(rd.configMapName, cm.Data)
		if nameErr != nil {
			errs = append(errs, nameErr)
		} else {
			attrs = append(attrs, semconv.K8SClusterName(name))
		}
	}

	ns, err := client.CoreV1().Namespaces().Get(ctx, rd.kubeSystemNamespace, metav1.GetOptions{})
	switch {
	case err != nil:
		errs = append(errs, fmt.Errorf("namespace %s: %w", rd.kubeSystemNamespace, err))
	case ns.UID == "":
		errs = append(errs, fmt.Errorf("namespace %s: empty UID", rd.kubeSystemNamespace))
	default:
		attrs = append(attrs, semconv.K8SClusterUID(string(ns.UID)))
	}

	if rd.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if rd.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	if len(errs) > 0 {
		err := fmt.Errorf("%w: %w", resource.ErrPartialResource, errors.Join(errs...))
		if len(attrs) == 0 {
			return resource.Empty(), err
		}
		return resource.NewWithAttributes(semconv.SchemaURL, attrs...), err
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

// clusterName extracts clusterName from the kubeadm ClusterConfiguration
// document stored in the ConfigMap data.
func clusterName(cmName string, data map[string]string) (string, error) {
	raw, ok := data[clusterConfigurationKey]
	if !ok {
		return "", fmt.Errorf("%s key not found in configmap %s", clusterConfigurationKey, cmName)
	}

	var cc clusterConfiguration
	if err := yaml.Unmarshal([]byte(raw), &cc); err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", clusterConfigurationKey, err)
	}
	if cc.ClusterName == "" {
		return "", fmt.Errorf("%s does not set clusterName", clusterConfigurationKey)
	}
	return cc.ClusterName, nil
}
