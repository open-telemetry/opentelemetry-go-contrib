// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package k8sapi // import "go.opentelemetry.io/contrib/detectors/k8sapi"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const defaultNodeEnvVar = "K8S_NODE_NAME"

type config struct {
	nodeEnvVar string
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

// WithNodeEnvVar sets the environment variable name from which the Kubernetes
// node name is read. Defaults to "K8S_NODE_NAME".
func WithNodeEnvVar(name string) Option {
	return optionFunc(func(c *config) { c.nodeEnvVar = name })
}

// WithKubeClient sets the Kubernetes client used to query the node and the kube-system namespace. If not
// set, an in-cluster client is created automatically during
// [ResourceDetector.Detect]. This option is primarily useful for testing or
// when running outside a cluster.
func WithKubeClient(client kubernetes.Interface) Option {
	return optionFunc(func(c *config) { c.kubeClient = client })
}

// WithAttributeFilter sets a filter that controls which detected attributes
// are included in the returned resource. Only attributes for which filter
// returns true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource attributes from the Kubernetes node the
// process is running on.
type ResourceDetector struct {
	cfg             config
	createProvider  func(*rest.Config) (kubernetes.Interface, error)
	inClusterConfig func() (*rest.Config, error)
}

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// Detect returns a [*resource.Resource] describing the Kubernetes node and
// cluster. If the node-name environment variable is not set, node attributes
// are omitted but k8s.cluster.uid may still be detected. If the process is
// not running inside a cluster, an empty resource and no error are returned.
func (rd *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	client := rd.cfg.kubeClient
	if client == nil {
		conf, err := rd.inClusterConfig()
		if err != nil {
			if errors.Is(err, rest.ErrNotInCluster) {
				slog.Warn("k8sapi detector: not running in a Kubernetes cluster", "err", err)
				return resource.Empty(), nil
			}
			return nil, fmt.Errorf("k8sapi detector: %w", err)
		}
		var clientErr error
		client, clientErr = rd.createProvider(conf)
		if clientErr != nil {
			return nil, fmt.Errorf("k8sapi detector: failed to create Kubernetes client: %w", clientErr)
		}
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	if nodeName := os.Getenv(rd.cfg.nodeEnvVar); nodeName != "" {
		node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("node %q: %w", nodeName, err))
		} else {
			attrs = append(attrs, semconv.K8SNodeName(node.Name), semconv.K8SNodeUID(string(node.UID)))
		}
	}

	ns, err := client.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		errs = append(errs, fmt.Errorf("kube-system namespace: %w", err))
	} else if uid := string(ns.UID); uid != "" {
		attrs = append(attrs, semconv.K8SClusterUID(uid))
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

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on the Kubernetes node the process is running on.
//
// The node name is read from the K8S_NODE_NAME environment variable by
// default. Use [WithNodeEnvVar] to customize the variable name. The variable
// is typically populated via the Kubernetes downward API:
//
//	env:
//	  - name: K8S_NODE_NAME
//	    valueFrom:
//	      fieldRef:
//	        fieldPath: spec.nodeName
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{nodeEnvVar: defaultNodeEnvVar}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return &ResourceDetector{
		cfg: cfg,
		createProvider: func(c *rest.Config) (kubernetes.Interface, error) {
			return kubernetes.NewForConfig(c)
		},
		inClusterConfig: rest.InClusterConfig,
	}
}
