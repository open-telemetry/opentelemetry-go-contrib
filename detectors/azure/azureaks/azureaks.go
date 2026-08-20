// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azureaks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const defaultEndpoint = "http://169.254.169.254/metadata/instance/compute?api-version=2021-12-13&format=json"

// kubernetesServiceHostEnvVar is set by the kubelet in every pod. Its presence
// is what distinguishes an AKS workload from a plain Azure VM.
const kubernetesServiceHostEnvVar = "KUBERNETES_SERVICE_HOST"

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// aksMetadata is the subset of the Azure Instance Metadata Service compute
// document this detector uses.
type aksMetadata struct {
	ResourceGroupName string `json:"resourceGroupName"`
}

type config struct {
	filter attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information of Azure Kubernetes Service workloads.
type ResourceDetector struct {
	endpoint string
	cfg      config
	client   *http.Client
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Azure Kubernetes Service.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	// Use a transport with Proxy explicitly disabled. The metadata endpoint is
	// a link-local address (169.254.169.254) that must never be reached via an
	// HTTP(S) proxy: doing so could leak instance metadata or break detection
	// in environments where users set HTTP_PROXY/HTTPS_PROXY for outbound
	// traffic.
	transport := &http.Transport{Proxy: nil}
	return &ResourceDetector{
		endpoint: defaultEndpoint,
		cfg:      cfg,
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// fetchMetadata queries the Azure Instance Metadata Service. The returned
// boolean reports whether the process appears to be running on Azure: it is
// false when the metadata service cannot be reached or when something other
// than the Azure Instance Metadata Service answered the request.
func (d *ResourceDetector) fetchMetadata(ctx context.Context) (*aksMetadata, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}
	// Required by the Azure Instance Metadata Service to prevent unintended
	// redirects and server-side request forgery.
	req.Header.Add("Metadata", "true")

	resp, err := d.client.Do(req)
	if err != nil {
		// The metadata service is unreachable: not running on Azure.
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A client error means the link-local address was answered by
		// something that is not the Azure Instance Metadata Service. Any other
		// status is a failure of the metadata service itself.
		onAzure := resp.StatusCode < 400 || resp.StatusCode > 499
		return nil, onAzure, fmt.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	var meta aksMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, true, err
	}
	return &meta, true, nil
}

// Detect detects resource attributes of the Azure Kubernetes Service cluster the
// process is running on. It returns an empty resource and no error when not
// running on Azure Kubernetes Service, and an error when the metadata service is
// reachable but does not return usable metadata. If the process is running on
// Azure Kubernetes Service but some attributes cannot be retrieved, a partial
// resource is returned together with [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	// Skip the metadata request entirely when not running on Kubernetes: an
	// Azure VM that is not a cluster node is not AKS.
	if os.Getenv(kubernetesServiceHostEnvVar) == "" {
		return resource.Empty(), nil
	}

	meta, onAzure, err := d.fetchMetadata(ctx)
	if err != nil {
		if !onAzure {
			return resource.Empty(), nil
		}

		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureAKS,
	}

	var errs []error

	if meta.ResourceGroupName == "" {
		errs = append(errs, errors.New("cluster name: resourceGroupName not present in metadata"))
	} else {
		attrs = append(attrs, semconv.K8SClusterName(parseClusterName(meta.ResourceGroupName)))
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	if len(errs) > 0 {
		return res, fmt.Errorf("%w: %v", resource.ErrPartialResource, errs)
	}
	return res, nil
}

// parseClusterName parses the cluster name from the infrastructure resource
// group name. The Azure Instance Metadata Service returns the resource group
// name in one of two formats:
//
//  1. Generated group: MC_<resource group>_<cluster name>_<location>
//     - Example:
//     - Resource group: my-resource-group
//     - Cluster name:   my-cluster
//     - Location:       eastus
//     - Generated name: MC_my-resource-group_my-cluster_eastus
//
//  2. Custom group: custom-infra-resource-group-name
//
// When using the generated infrastructure resource group, the resource group
// includes the cluster name. If the cluster's resource group or cluster name
// contains underscores, parsing falls back on the unparsed infrastructure
// resource group name.
//
// When using a custom infrastructure resource group, the resource group name
// does not contain the cluster name. The custom infrastructure resource group
// name is returned instead.
//
// It is safe to use the infrastructure resource group name as a unique
// identifier because Azure will not allow the user to create multiple AKS
// clusters with the same infrastructure resource group name.
func parseClusterName(resourceGroup string) string {
	splitAll := strings.Split(resourceGroup, "_")

	if len(splitAll) == 4 && strings.EqualFold(splitAll[0], "mc") {
		return splitAll[len(splitAll)-2]
	}

	return resourceGroup
}
