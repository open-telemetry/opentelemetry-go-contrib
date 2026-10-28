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

var (
	loadAWSConfig = awsconfig.LoadDefaultConfig
	newEKSClient  = func(cfg aws.Config) eksClient { return eks.NewFromConfig(cfg) }
)

// WithClusterARN enables detection of aws.eks.cluster.arn. The detector calls
// EKS DescribeCluster and reports the ARN only when the returned OIDC issuer
// matches the issuer published by the live Kubernetes API server. The option
// requires cluster.name from the amazon-cloudwatch/cluster-info ConfigMap and
// valid AWS configuration for that cluster. If the ARN cannot be established,
// Detect returns the other EKS attributes with resource.ErrPartialResource.
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

	mu     sync.Mutex
	client eksClient
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

	client, err := d.getClient(awsCtx)
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

func (d *clusterARNDetector) getClient(ctx context.Context) (eksClient, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client != nil {
		return d.client, nil
	}

	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Region == "" {
		return nil, errAWSRegionUnavailable
	}
	d.client = newEKSClient(cfg)
	return d.client, nil
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
