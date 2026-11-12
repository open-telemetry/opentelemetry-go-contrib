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
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

const (
	clusterARNTimeout        = 10 * time.Second
	oidcDiscoveryTimeout     = 10 * time.Second
	maxOIDCDiscoveryBodySize = 64 * 1024
)

var (
	errClusterNameUnavailable   = errors.New("EKS cluster name is unavailable")
	errLiveIssuerUnavailable    = errors.New("live EKS OIDC issuer is unavailable")
	errAWSConfigUnavailable     = errors.New("AWS configuration is unavailable")
	errAWSRegionUnavailable     = errors.New("AWS Region is unavailable")
	errDescribeClusterFailed    = errors.New("EKS DescribeCluster failed")
	errClusterResponseMissing   = errors.New("EKS DescribeCluster response has no cluster")
	errClusterIssuerUnavailable = errors.New("EKS DescribeCluster response has no OIDC issuer")
	errClusterARNUnavailable    = errors.New("EKS DescribeCluster response has no cluster ARN")
	errClusterMismatch          = errors.New("EKS cluster identity does not match the live cluster")
)

// issuerRegion matches the Region label of an EKS OIDC issuer host.
var issuerRegion = regexp.MustCompile(`^[a-z]{2,4}(-[a-z]+)+-\d+$`)

// issuerDNSSuffixes are the DNS suffixes of the AWS partitions that EKS OIDC
// issuer hosts (oidc.eks.<region>.<suffix>) are published under.
var issuerDNSSuffixes = []string{
	"amazonaws.com",
	"amazonaws.com.cn",
	"amazonaws.eu",
	"c2s.ic.gov",
	"sc2s.sgov.gov",
	"cloud.adc-e.uk",
	"csp.hci.ic.gov",
}

var (
	loadAWSConfig = awsconfig.LoadDefaultConfig
	newEKSClient  = func(cfg aws.Config) eksClient { return eks.NewFromConfig(cfg) }
)

// WithClusterARN enables detection of aws.eks.cluster.arn. The detector calls
// EKS DescribeCluster and reports the ARN only when the returned OIDC issuer
// matches the issuer published by the live Kubernetes API server. The option
// requires cluster.name from the amazon-cloudwatch/cluster-info ConfigMap and
// valid AWS configuration for that cluster. If the AWS configuration has no
// Region, the Region of the live OIDC issuer is used. If the ARN cannot be
// established, Detect returns the other EKS attributes with
// resource.ErrPartialResource.
func WithClusterARN() Option {
	return optionFunc(func(c *config) {
		c.clusterARN = true
	})
}

type oidcIssuerProvider interface {
	getOIDCIssuer(context.Context) (string, error)
}

type eksClient interface {
	DescribeCluster(context.Context, *eks.DescribeClusterInput, ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

type clusterARNDetector struct {
	utils oidcIssuerProvider

	initOnce sync.Once
	sem      chan struct{}
	client   eksClient
}

func (d *clusterARNDetector) lock() chan struct{} {
	d.initOnce.Do(func() { d.sem = make(chan struct{}, 1) })
	return d.sem
}

var (
	_ oidcIssuerProvider = (*eksDetectorUtils)(nil)
	_ eksClient          = (*eks.Client)(nil)
)

func (d *clusterARNDetector) detect(ctx context.Context, clusterName string) (string, error) {
	if clusterName == "" {
		return "", errClusterNameUnavailable
	}

	liveIssuer, err := d.utils.getOIDCIssuer(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errLiveIssuerUnavailable, err)
	}
	liveIssuer = normalizeIssuer(liveIssuer)
	if liveIssuer == "" {
		return "", errLiveIssuerUnavailable
	}

	awsCtx, cancel := context.WithTimeout(ctx, clusterARNTimeout)
	defer cancel()

	client, err := d.getClient(awsCtx, liveIssuer)
	if err != nil {
		if ctxErr := awsCtx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("%w: %w", errAWSConfigUnavailable, err)
	}
	out, err := client.DescribeCluster(awsCtx, &eks.DescribeClusterInput{Name: aws.String(clusterName)})
	if err != nil {
		return "", fmt.Errorf("%w: %w", errDescribeClusterFailed, err)
	}
	if out == nil || out.Cluster == nil {
		return "", errClusterResponseMissing
	}
	if out.Cluster.Identity == nil || out.Cluster.Identity.Oidc == nil {
		return "", errClusterIssuerUnavailable
	}

	describedIssuer := normalizeIssuer(aws.ToString(out.Cluster.Identity.Oidc.Issuer))
	if describedIssuer == "" {
		return "", errClusterIssuerUnavailable
	}
	if describedIssuer != liveIssuer {
		return "", errClusterMismatch
	}

	arn := aws.ToString(out.Cluster.Arn)
	if arn == "" {
		return "", errClusterARNUnavailable
	}
	return arn, nil
}

// getClient loads the AWS configuration synchronously with the caller's ctx.
// Callers are serialized by a lock that is acquired with select so a caller
// waiting on another caller's initialization still honors its own ctx. If the
// configuration has no Region, the Region is taken from issuer.
func (d *clusterARNDetector) getClient(ctx context.Context, issuer string) (eksClient, error) {
	sem := d.lock()
	select {
	case sem <- struct{}{}:
	default:
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	defer func() { <-sem }()

	if d.client != nil {
		return d.client, nil
	}

	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		cfg.Region = regionFromIssuer(issuer)
	}
	if cfg.Region == "" {
		return nil, errAWSRegionUnavailable
	}
	d.client = newEKSClient(cfg)
	return d.client, nil
}

// regionFromIssuer returns the Region of an EKS OIDC issuer URL, or an empty
// string if issuer is not one. An EKS issuer uses the https scheme, the host
// oidc.eks.<region>.<partition DNS suffix>, and the path /id/<id>.
func regionFromIssuer(issuer string) string {
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" ||
		!strings.HasPrefix(u.Path, "/id/") || len(u.Path) == len("/id/") {
		return ""
	}
	rest, ok := strings.CutPrefix(u.Hostname(), "oidc.eks.")
	if !ok {
		return ""
	}
	region, suffix, ok := strings.Cut(rest, ".")
	if !ok || !issuerRegion.MatchString(region) || !slices.Contains(issuerDNSSuffixes, suffix) {
		return ""
	}
	return region
}

func normalizeIssuer(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/")
}

type oidcDiscovery struct {
	Issuer string `json:"issuer"`
}

func (eksUtils eksDetectorUtils) getOIDCIssuer(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, oidcDiscoveryTimeout)
	defer cancel()

	u, err := url.JoinPath(eksUtils.host, "/.well-known/openid-configuration")
	if err != nil {
		return "", fmt.Errorf("failed to build OIDC discovery URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create OIDC discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := eksUtils.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve OIDC discovery document: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("failed to retrieve OIDC discovery document: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOIDCDiscoveryBodySize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC discovery document: %w", err)
	}
	if len(body) > maxOIDCDiscoveryBodySize {
		return "", fmt.Errorf("OIDC discovery document exceeds %d bytes", maxOIDCDiscoveryBodySize)
	}

	var discovery oidcDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return "", fmt.Errorf("failed to decode OIDC discovery document: %w", err)
	}
	return discovery.Issuer, nil
}
