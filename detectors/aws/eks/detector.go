// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package eks provides a resource detector for AWS EKS.
package eks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	k8sTokenPath                    = "/var/run/secrets/kubernetes.io/serviceaccount/token" //nolint:gosec // False positive G101: Potential hardcoded credentials. The detector only check if the token exists.
	k8sCertPath                     = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	authConfigmapNS                 = "kube-system"
	authConfigmapName               = "aws-auth"
	cwConfigmapNS                   = "amazon-cloudwatch"
	cwConfigmapName                 = "cluster-info"
	defaultCgroupPath               = "/proc/self/cgroup"
	containerIDLength               = 64
	configMapTimeout                = 10 * time.Second
	kubernetesServiceHostEnvVar     = "KUBERNETES_SERVICE_HOST"
	eksClusterStringIdentifier      = "-eks-"
	eksIRSATokenPathIdentifier      = "eks.amazonaws.com" //nolint:gosec // not a credential
	eksPodIdentityPathIdentifier    = "eks-pod-identity"
	awsWebIdentityTokenFileEnvVar   = "AWS_WEB_IDENTITY_TOKEN_FILE"            //nolint:gosec // env var name, not a credential
	awsContainerAuthTokenFileEnvVar = "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE" //nolint:gosec // env var name, not a credential
	defaultNodeEnvVar               = "K8S_NODE_NAME"
)

// detectorUtils is used for testing the resourceDetector by abstracting functions that rely on external systems.
type detectorUtils interface {
	fileExists(filename string) bool
	getConfigMap(ctx context.Context, namespace, name string) (map[string]string, error)
	getContainerID() (string, error)
	getServerVersion() (string, error)
}

// This struct will implement the detectorUtils interface.
type eksDetectorUtils struct {
	host       string
	client     *http.Client
	kubeClient kubernetes.Interface
}

// configMap is the subset of a Kubernetes ConfigMap response needed by the detector.
type configMap struct {
	Data map[string]string `json:"data"`
}

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

// WithAttributeFilter sets a filter that controls which detected attributes
// are included in the returned resource. Only attributes for which filter
// returns true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// WithNodeEnvVar sets the environment variable name from which the Kubernetes
// node name is read. Defaults to "K8S_NODE_NAME".
func WithNodeEnvVar(name string) Option {
	return optionFunc(func(c *config) { c.nodeEnvVar = name })
}

// WithKubeClient sets the Kubernetes client used to query the Kubernetes API.
// If not set, an in-cluster client is created automatically during
// [ResourceDetector.Detect].
func WithKubeClient(client kubernetes.Interface) Option {
	return optionFunc(func(c *config) { c.kubeClient = client })
}

// ResourceDetector for detecting resources running on Amazon EKS.
type ResourceDetector struct {
	cfg   config
	utils detectorUtils
	err   error
}

// Backwards-compatible type alias.
type resourceDetector = ResourceDetector

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// Compile time assertion that eksDetectorUtils implements the detectorUtils interface.
var _ detectorUtils = (*eksDetectorUtils)(nil)

// is this going to stop working with 1.20 when Docker is deprecated?
var containerIDRegex = regexp.MustCompile(`^.*/docker/(.+)$`)

// NewResourceDetector returns a resource detector that will detect AWS EKS resources.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{nodeEnvVar: defaultNodeEnvVar}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	var (
		utils detectorUtils
		err   error
	)
	if cfg.kubeClient != nil {
		utils = &eksDetectorUtils{
			kubeClient: cfg.kubeClient,
		}
	} else {
		utils, err = newK8sDetectorUtils()
	}

	return &ResourceDetector{cfg: cfg, utils: utils, err: err}
}

// Detect returns a Resource describing the Amazon EKS environment being run in.
//
// Detect performs up to two Kubernetes API requests, each bounded by a
// 10-second timeout, so a single call can take up to ~20 seconds when the
// API server is unresponsive. Provide a ctx with a shorter deadline to bound
// the total duration.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	if detector.err != nil {
		if errors.Is(detector.err, rest.ErrNotInCluster) {
			return resource.Empty(), nil
		}

		return nil, detector.err
	}

	isEks, err := isEKS(ctx, detector.utils)
	if err != nil {
		return nil, err
	}

	// Return empty resource object if not running in EKS
	if !isEks {
		return resource.Empty(), nil
	}

	// Create variable to hold resource attributes
	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
	}

	// Get clusterName and append to attributes
	clusterName, err := getClusterName(ctx, detector.utils)
	if err != nil {
		return nil, err
	}
	if clusterName != "" {
		attributes = append(attributes, semconv.K8SClusterName(clusterName))
	}

	// Get containerID and append to attributes
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return nil, err
	}
	if containerID != "" {
		attributes = append(attributes, semconv.ContainerID(containerID))
	}

	if detector.cfg.filter != nil {
		filtered := attributes[:0]
		for _, kv := range attributes {
			if detector.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attributes = filtered
	}

	// Return new resource object with clusterName and containerID as attributes
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

// isEKS checks if the current environment is running in EKS.
func isEKS(ctx context.Context, utils detectorUtils) (bool, error) {
	if !isK8s(utils) {
		return false, nil
	}

	// Check for EKS-specific IRSA token file path
	if tokenFile := os.Getenv(awsWebIdentityTokenFileEnvVar); strings.Contains(tokenFile, eksIRSATokenPathIdentifier) {
		return true, nil
	}

	// Check for EKS Pod Identity token file path
	if authFile := os.Getenv(awsContainerAuthTokenFileEnvVar); strings.Contains(authFile, eksPodIdentityPathIdentifier) {
		return true, nil
	}

	// Check cluster server version for -eks- identifier
	if version, err := utils.getServerVersion(); err == nil && strings.Contains(version, eksClusterStringIdentifier) {
		return true, nil
	}

	// Fallback: check for aws-auth ConfigMap in kube-system
	awsAuth, err := utils.getConfigMap(ctx, authConfigmapNS, authConfigmapName)
	if err != nil {
		return false, fmt.Errorf("isEks() error retrieving auth configmap: %w", err)
	}

	return awsAuth != nil, nil
}

// newK8sDetectorUtils creates utilities that fetch ConfigMaps over the in-cluster HTTP client.
func newK8sDetectorUtils() (*eksDetectorUtils, error) {
	// Get cluster configuration
	confs, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	client, err := rest.HTTPClientFor(confs)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for Kubernetes: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(confs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &eksDetectorUtils{host: confs.Host, client: client, kubeClient: kubeClient}, nil
}

// isK8s checks if the current environment is running in a Kubernetes environment.
func isK8s(utils detectorUtils) bool {
	if os.Getenv(kubernetesServiceHostEnvVar) != "" {
		return true
	}
	return utils.fileExists(k8sTokenPath) && utils.fileExists(k8sCertPath)
}

// fileExists checks if a file with a given filename exists.
func (eksDetectorUtils) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

// getConfigMap retrieves the configuration map from the k8s API.
func (eksUtils eksDetectorUtils) getConfigMap(ctx context.Context, namespace, name string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, configMapTimeout)
	defer cancel()

	if eksUtils.kubeClient != nil {
		cm, err := eksUtils.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve ConfigMap %s/%s: %w", namespace, name, err)
		}
		return cm.Data, nil
	}

	u, err := url.JoinPath(eksUtils.host, "api", "v1", "namespaces", namespace, "configmaps", name)
	if err != nil {
		return nil, fmt.Errorf("failed to build ConfigMap URL for %s/%s: %w", namespace, name, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create ConfigMap request for %s/%s: %w", namespace, name, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := eksUtils.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ConfigMap %s/%s: %w", namespace, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to retrieve ConfigMap %s/%s: unexpected status %s", namespace, name, resp.Status)
	}

	var cm configMap
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		return nil, fmt.Errorf("failed to decode ConfigMap %s/%s: %w", namespace, name, err)
	}

	return cm.Data, nil
}

// getServerVersion retrieves the Kubernetes cluster server version.
func (eksUtils eksDetectorUtils) getServerVersion() (string, error) {
	if eksUtils.kubeClient == nil {
		return "", errors.New("kubernetes client not configured")
	}
	info, err := eksUtils.kubeClient.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return info.GitVersion, nil
}

// getClusterName retrieves the clusterName resource attribute.
func getClusterName(ctx context.Context, utils detectorUtils) (string, error) {
	resp, err := utils.getConfigMap(ctx, cwConfigmapNS, cwConfigmapName)
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("getClusterName() error: %w", err)
	}

	return resp["cluster.name"], nil
}

func isNotFound(err error) bool {
	if apierrors.IsNotFound(err) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") || strings.Contains(errStr, "not found")
}

// getContainerID returns the containerID if currently running within a container.
func (eksDetectorUtils) getContainerID() (string, error) {
	if runtime.GOOS != "linux" {
		return "", nil
	}

	fileData, err := os.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", nil
	}

	// Retrieve containerID from file
	splitData := strings.SplitSeq(strings.TrimSpace(string(fileData)), "\n")
	for str := range splitData {
		if containerIDRegex.MatchString(str) {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", nil
}
