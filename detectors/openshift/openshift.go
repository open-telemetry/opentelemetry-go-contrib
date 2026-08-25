// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	// infrastructurePath is the path of the cluster-scoped Infrastructure
	// config object on the OpenShift API server.
	infrastructurePath = "/apis/config.openshift.io/v1/infrastructures/cluster/status"

	// defaultTokenPath is where Kubernetes projects the service account token
	// of the pod the process runs in.
	defaultTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec G101 -- file path, not a credential.
	// defaultCAPath is where Kubernetes projects the certificate authority
	// that signed the API server certificate.
	defaultCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	hostEnvVar = "KUBERNETES_SERVICE_HOST"
	portEnvVar = "KUBERNETES_SERVICE_PORT"
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

// infrastructureResponse is the response of the OpenShift Infrastructure
// status endpoint. Only the fields the detector reads are declared.
type infrastructureResponse struct {
	Status infrastructureStatus `json:"status"`
}

type infrastructureStatus struct {
	// InfrastructureName uniquely identifies a cluster with a human friendly
	// name.
	InfrastructureName string         `json:"infrastructureName"`
	PlatformStatus     platformStatus `json:"platformStatus"`
}

// platformStatus holds status information specific to the underlying
// infrastructure provider.
type platformStatus struct {
	Type      string            `json:"type"`
	AWS       awsPlatform       `json:"aws"`
	Azure     azurePlatform     `json:"azure"`
	GCP       gcpPlatform       `json:"gcp"`
	IBMCloud  ibmCloudPlatform  `json:"ibmcloud"`
	OpenStack openStackPlatform `json:"openstack"`
}

type awsPlatform struct {
	// Region holds the default AWS region for new AWS resources created by the
	// cluster.
	Region string `json:"region"`
}

type azurePlatform struct {
	// CloudName is the name of the Azure cloud environment.
	CloudName string `json:"cloudName"`
}

type gcpPlatform struct {
	// Region holds the region for new GCP resources created for the cluster.
	Region string `json:"region"`
}

type ibmCloudPlatform struct {
	// Location is where the cluster has been deployed.
	Location string `json:"location"`
}

type openStackPlatform struct {
	// CloudName is the name of the desired OpenStack cloud in the client
	// configuration file (clouds.yaml).
	CloudName string `json:"cloudName"`
}

type config struct {
	address   string
	token     string
	tlsConfig *tls.Config
	filter    attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAddress sets the base URL of the OpenShift API server, for example
// "https://api.example.com:6443". By default it is derived from the
// KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT environment variables
// that Kubernetes sets in every pod. A trailing slash is ignored.
func WithAddress(address string) Option {
	return optionFunc(func(c *config) { c.address = address })
}

// WithToken sets the bearer token used to authenticate against the OpenShift
// API server. By default the service account token projected into the pod is
// used.
func WithToken(token string) Option {
	return optionFunc(func(c *config) { c.token = token })
}

// WithTLSConfig sets the TLS configuration used to reach the OpenShift API
// server. By default the certificate authority projected into the pod is used
// as the sole root of trust.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return optionFunc(func(c *config) { c.tlsConfig = tlsConfig })
}

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information of OpenShift 4 clusters.
type ResourceDetector struct {
	cfg config

	// tokenPath and caPath are the locations of the projected service account
	// credentials. They are fields so tests can point them elsewhere.
	tokenPath string
	caPath    string
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes of the OpenShift 4 cluster the process is running in.
//
// The detector reads the cluster-scoped Infrastructure config object and
// therefore requires the following RBAC:
//
//   - apiGroups: ["config.openshift.io"]
//     resources: ["infrastructures"]
//     resourceNames: ["cluster"]
//     verbs: ["get"]
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{
		cfg:       cfg,
		tokenPath: defaultTokenPath,
		caPath:    defaultCAPath,
	}
}

// address returns the base URL of the OpenShift API server. The returned
// boolean is false when the process does not appear to run inside a cluster.
func (d *ResourceDetector) address() (string, bool) {
	if d.cfg.address != "" {
		// A trailing slash would double the separator: the request paths
		// appended to this value already start with one.
		return strings.TrimRight(d.cfg.address, "/"), true
	}

	host, port := os.Getenv(hostEnvVar), os.Getenv(portEnvVar)
	if host == "" || port == "" {
		return "", false
	}
	// Kubernetes sets the host to a bare IPv6 address on IPv6 clusters.
	// JoinHostPort adds the brackets a URL authority requires.
	return "https://" + net.JoinHostPort(host, port), true
}

// token returns the bearer token to authenticate with. The returned boolean is
// false when the process does not appear to run inside a cluster.
func (d *ResourceDetector) token() (string, bool) {
	if d.cfg.token != "" {
		return d.cfg.token, true
	}

	token, err := os.ReadFile(d.tokenPath)
	if err != nil {
		return "", false
	}
	// Trailing whitespace would make the Authorization header value invalid.
	return strings.TrimSpace(string(token)), true
}

// client returns the HTTP client used to reach the API server at address.
func (d *ResourceDetector) client(address string) (*http.Client, error) {
	tlsConfig := d.cfg.tlsConfig
	// The certificate authority is only needed to verify a TLS connection. An
	// address that is not served over TLS is never verified, so do not require
	// the projected certificate authority to be present for one.
	if tlsConfig == nil && strings.HasPrefix(address, "https://") {
		ca, err := os.ReadFile(d.caPath)
		if err != nil {
			return nil, fmt.Errorf("read certificate authority %s: %w", d.caPath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("no certificate found in %s", d.caPath)
		}
		tlsConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}

	// Use a transport with Proxy explicitly disabled. The API server is reached
	// over a cluster-internal address that must never be contacted through an
	// HTTP(S) proxy: doing so would hand the service account bearer token to
	// the proxy in environments where users set HTTP_PROXY/HTTPS_PROXY for
	// outbound traffic.
	return &http.Client{
		Transport: &http.Transport{Proxy: nil, TLSClientConfig: tlsConfig},
	}, nil
}

// infrastructure requests the Infrastructure status from the OpenShift API
// server. The returned boolean reports whether the process appears to run on
// OpenShift: it is false when the API server cannot be reached or when it does
// not serve the OpenShift config API.
func (d *ResourceDetector) infrastructure(ctx context.Context) (*infrastructureResponse, bool, error) {
	address, ok := d.address()
	if !ok {
		return nil, false, nil
	}
	token, ok := d.token()
	if !ok {
		return nil, false, nil
	}

	client, err := d.client(address)
	if err != nil {
		return nil, true, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address+infrastructurePath, http.NoBody)
	if err != nil {
		return nil, true, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			// The caller gave up. Do not report this as "not running on
			// OpenShift".
			return nil, true, err
		}
		// The API server is unreachable: not running on OpenShift.
		return nil, false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// A plain Kubernetes API server answers 404 for the OpenShift config
		// API group. Any other client error means the request was rejected,
		// which is not evidence of an OpenShift cluster either.
		return nil, false, fmt.Errorf("infrastructure request returned status %d", resp.StatusCode)
	default:
		return nil, true, fmt.Errorf("infrastructure request returned status %d", resp.StatusCode)
	}

	var infra infrastructureResponse
	if err := json.NewDecoder(resp.Body).Decode(&infra); err != nil {
		return nil, true, fmt.Errorf("decode infrastructure response: %w", err)
	}
	return &infra, true, nil
}

// Detect detects resource attributes of the OpenShift 4 cluster the process is
// running in. It returns an empty resource and no error when not running on
// OpenShift, and an error when the OpenShift API is reachable but does not
// return usable metadata. If the process runs on OpenShift but some attributes
// cannot be retrieved, a partial resource is returned together with
// [resource.ErrPartialResource].
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	infra, onOpenShift, err := d.infrastructure(ctx)
	if err != nil {
		if !onOpenShift {
			return resource.Empty(), nil
		}
		return nil, err
	}
	if infra == nil {
		return resource.Empty(), nil
	}

	var (
		attrs []attribute.KeyValue
		errs  []error
	)

	if name := infra.Status.InfrastructureName; name != "" {
		attrs = append(attrs, semconv.K8SClusterName(name))
	} else {
		errs = append(errs, errors.New("k8s.cluster.name: infrastructureName not present in infrastructure status"))
	}

	// The OpenShift API reports regions in the casing of the underlying
	// provider. Normalize to lower case so cloud.region matches the value the
	// collector's OpenShift detector reports for the same cluster.
	region := func(value, field string) {
		if value == "" {
			errs = append(errs, fmt.Errorf("cloud.region: %s not present in infrastructure status", field))
			return
		}
		attrs = append(attrs, semconv.CloudRegion(strings.ToLower(value)))
	}

	platform := infra.Status.PlatformStatus
	switch strings.ToLower(platform.Type) {
	case "aws":
		attrs = append(attrs, semconv.CloudProviderAWS, semconv.CloudPlatformAWSOpenShift)
		region(platform.AWS.Region, "aws.region")
	case "azure":
		attrs = append(attrs, semconv.CloudProviderAzure, semconv.CloudPlatformAzureOpenShift)
		region(platform.Azure.CloudName, "azure.cloudName")
	case "gcp":
		attrs = append(attrs, semconv.CloudProviderGCP, semconv.CloudPlatformGCPOpenShift)
		region(platform.GCP.Region, "gcp.region")
	case "ibmcloud":
		attrs = append(attrs, semconv.CloudProviderIBMCloud, semconv.CloudPlatformIBMCloudOpenShift)
		region(platform.IBMCloud.Location, "ibmcloud.location")
	case "openstack":
		// Semantic conventions define no cloud.provider value for OpenStack and
		// no cloud.platform value for OpenShift on OpenStack, so only the
		// region is reported.
		region(platform.OpenStack.CloudName, "openstack.cloudName")
	}
	// Any other platform type (baremetal, vsphere, ovirt, none, external, ...)
	// is a cluster that does not run on a cloud provider. Reporting no cloud
	// attributes for it is correct, not partial.

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
