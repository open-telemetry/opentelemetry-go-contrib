// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpc // import "go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const (
	metadataHost        = "api.metadata.cloud.ibm.com"
	tokenPath           = "/identity/v1/token"
	instancePath        = "/metadata/v1/instance"
	apiVersion          = "2026-01-30"
	metadataFlavorKey   = "Metadata-Flavor"
	metadataFlavorValue = "ibm"
	defaultProtocol     = "http"
	defaultTokenTTL     = 300
	tokenRefreshBuffer  = 30 * time.Second
	maxResponseSize     = 1 << 20

	// TODO: Use the semantic convention constant when one is added.
	cloudPlatformIBMCloudVPC = "ibm_cloud.vpc"
)

var _ resource.Detector = (*ResourceDetector)(nil)

// ResourceDetector queries the IBM Cloud VPC Instance Metadata Service and
// emits resource attributes for the current virtual server instance.
type ResourceDetector struct {
	endpoint string
	client   *http.Client
	filter   attribute.Filter
	err      error

	tokenMu     sync.Mutex
	token       string
	tokenExpiry time.Time
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

type config struct {
	protocol string
	filter   attribute.Filter
	err      error
}

// WithProtocol sets the metadata endpoint protocol. Accepted values are "http"
// and "https". The default is "http".
func WithProtocol(protocol string) Option {
	return optionFunc(func(c *config) {
		switch protocol {
		case "http", "https":
			c.protocol = protocol
		default:
			c.err = fmt.Errorf("invalid protocol %q: must be \"http\" or \"https\"", protocol)
		}
	})
}

// WithAttributeFilter sets a filter that controls which detected attributes
// are included in the returned resource. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on IBM Cloud VPC virtual server instances.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{protocol: defaultProtocol}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil

	return &ResourceDetector{
		endpoint: cfg.protocol + "://" + metadataHost,
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
		},
		filter: cfg.filter,
		err:    cfg.err,
	}
}

// Detect detects IBM Cloud VPC instance metadata. It returns an empty resource
// and no error when metadata cannot be retrieved.
func (d *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	if d.err != nil {
		return nil, d.err
	}

	meta, err := d.instanceMetadata(ctx)
	if err != nil {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderIBMCloud,
		semconv.CloudPlatformKey.String(cloudPlatformIBMCloudVPC),
	}
	if region := regionFromZone(meta.Zone.Name); region != "" {
		attrs = append(attrs, semconv.CloudRegion(region))
	}
	if meta.Zone.Name != "" {
		attrs = append(attrs, semconv.CloudAvailabilityZone(meta.Zone.Name))
	}
	if accountID := accountIDFromCRN(meta.CRN); accountID != "" {
		attrs = append(attrs, semconv.CloudAccountID(accountID))
	}
	if meta.CRN != "" {
		attrs = append(attrs, semconv.CloudResourceID(meta.CRN))
	}
	if meta.ID != "" {
		attrs = append(attrs, semconv.HostID(meta.ID))
	}
	if meta.Image.ID != "" {
		attrs = append(attrs, semconv.HostImageID(meta.Image.ID))
	}
	if meta.Image.Name != "" {
		attrs = append(attrs, semconv.HostImageName(meta.Image.Name))
	}
	if meta.Name != "" {
		attrs = append(attrs, semconv.HostName(meta.Name))
	}
	if meta.Profile.Name != "" {
		attrs = append(attrs, semconv.HostType(meta.Profile.Name))
	}

	if d.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

type instanceMetadata struct {
	ID      string `json:"id"`
	CRN     string `json:"crn"`
	Name    string `json:"name"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	Zone struct {
		Name string `json:"name"`
	} `json:"zone"`
	Image struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"image"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (d *ResourceDetector) instanceMetadata(ctx context.Context) (*instanceMetadata, error) {
	token, err := d.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance identity token: %w", err)
	}

	url := fmt.Sprintf("%s%s?version=%s", d.endpoint, instancePath, apiVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return nil, fmt.Errorf("instance metadata request returned %d: %s", resp.StatusCode, string(body))
	}

	var meta instanceMetadata
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode instance metadata: %w", err)
	}

	return &meta, nil
}

func (d *ResourceDetector) getToken(ctx context.Context) (string, error) {
	d.tokenMu.Lock()
	defer d.tokenMu.Unlock()

	if d.token != "" && time.Now().Before(d.tokenExpiry) {
		return d.token, nil
	}

	url := fmt.Sprintf("%s%s?version=%s", d.endpoint, tokenPath, apiVersion)
	body := fmt.Appendf(nil, `{"expires_in":%d}`, defaultTokenTTL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set(metadataFlavorKey, metadataFlavorValue)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get metadata token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&tr); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", errors.New("metadata token response missing access_token")
	}

	d.token = tr.AccessToken
	tokenTTL := time.Duration(tr.ExpiresIn) * time.Second
	if tokenTTL > tokenRefreshBuffer {
		tokenTTL -= tokenRefreshBuffer
	}
	if tokenTTL < 0 {
		tokenTTL = 0
	}
	d.tokenExpiry = time.Now().Add(tokenTTL)

	return d.token, nil
}

func regionFromZone(zone string) string {
	idx := strings.LastIndex(zone, "-")
	if idx > 0 {
		return zone[:idx]
	}
	return zone
}

func accountIDFromCRN(crn string) string {
	parts := strings.Split(crn, ":")
	if len(parts) >= 7 {
		return strings.TrimPrefix(parts[6], "a/")
	}
	return ""
}
