// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eks

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

const (
	k8sTokenPath      = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	k8sCertPath       = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	authConfigmapPath = "/api/v1/namespaces/kube-system/configmaps/aws-auth"
	cwConfigmapPath   = "/api/v1/namespaces/amazon-cloudwatch/configmaps/cluster-info"
	defaultCgroupPath = "/proc/self/cgroup"
	containerIDLength = 64
)

// detectorUtils is used for testing the resourceDetector by abstracting functions that rely on external systems.
type detectorUtils interface {
	fileExists(filename string) bool
	getConfigMap(httpMethod string, API string) (string, error)
	getContainerID() (string, error)
}

// This struct will implement the detectorUtils interface
type eksDetectorUtils struct {
	clientset *kubernetes.Clientset
}

// resourceDetector for detecting resources running on Amazon EKS
type resourceDetector struct {
	utils detectorUtils
}

// This struct will help unmarshal clustername from JSON response
type data struct {
	ClusterName string `json:"cluster.name"`
}

// Compile time assertion that resourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// Compile time assertion that eksDetectorUtils implements the detectorUtils interface.
var _ detectorUtils = (*eksDetectorUtils)(nil)

// NewResourceDetector returns a resource detector that will detect AWS EKS resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{utils: eksDetectorUtils{}}
}

// Detect returns a Resource describing the Amazon EKS environment being run in.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	isEks, err := isEKS(detector.utils)
	if err != nil {
		return nil, err
	}

	// Return empty resource object if not running in EKS
	if !isEks {
		return resource.Empty(), nil
	}

	// Create variable to hold resource attributes
	attributes := []attribute.KeyValue{}

	// Get clusterName and append to attributes
	clusterName, err := getClusterName(detector.utils)
	if err != nil {
		return nil, err
	}
	if clusterName != "" {
		attributes = append(attributes, semconv.K8SClusterNameKey.String(clusterName))
	}

	// Get containerID and append to attributes
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return nil, err
	}
	if containerID != "" {
		attributes = append(attributes, semconv.ContainerIDKey.String(containerID))
	}

	// Return new resource object with clusterName and containerID as attributes
	return resource.NewWithAttributes(attributes...), nil

}

// isEKS checks if the current environment is running in EKS.
func isEKS(utils detectorUtils) (bool, error) {
	if !isK8s(utils) {
		return false, nil
	}

	// Make HTTP GET request
	awsAuth, err := utils.getConfigMap(http.MethodGet, authConfigmapPath)
	if err != nil {
		return false, fmt.Errorf("isEks() error retrieving auth configmap: %w", err)
	}

	return awsAuth != "", nil
}

// getClientset creates the Kubernetes clientset
func getClientset() (*kubernetes.Clientset, error) {
	// Get cluster configuration
	confs, err := rest.InClusterConfig()
	if err != nil {
		return &kubernetes.Clientset{}, fmt.Errorf("failed to create config: %w", err)
	}

	// Create clientset using generated configuration
	clientset, err := kubernetes.NewForConfig(confs)
	if err != nil {
		return &kubernetes.Clientset{}, fmt.Errorf("failed to create clientset for Kubernetes client")
	}

	return clientset, nil
}

// isK8s checks if the current environment is running in a Kubernetes environment
func isK8s(utils detectorUtils) bool {
	return utils.fileExists(k8sTokenPath) && utils.fileExists(k8sCertPath)
}

// fileExists checks if a file with a given filename exists.
func (eksUtils eksDetectorUtils) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

// getConfigMap retieves the configuration map from the config map API path
func (eksUtils eksDetectorUtils) getConfigMap(httpMethod string, API string) (string, error) {
	// Create Kubernetes clientset if not created
	if eksUtils.clientset == nil {
		clientset, err := getClientset()
		if err != nil {
			return "", fmt.Errorf("failed to create clientset: %w", err)
		}
		eksUtils.clientset = clientset
	}

	// Execute HTTP request
	if httpMethod == "GET" {
		body, err := eksUtils.clientset.RESTClient().
			Get().
			AbsPath(API).
			DoRaw(context.TODO())
		if err != nil {
			return "", fmt.Errorf("failed to execute HTTP request with method=%s, API=%s: %w", httpMethod, API, err)
		}

		return string(body), nil
	}

	return "", fmt.Errorf("invalid HTTP request with method=%s", httpMethod)
}

// getClusterName retrieves the clusterName resource attribute
func getClusterName(utils detectorUtils) (string, error) {
	resp, err := utils.getConfigMap("GET", cwConfigmapPath)
	if err != nil {
		return "", fmt.Errorf("getClusterName() error: %w", err)
	}

	// parse JSON object returned from HTTP request
	var respmap map[string]json.RawMessage
	err = json.Unmarshal([]byte(resp), &respmap)
	if err != nil {
		return "", fmt.Errorf("getClusterName() error: cannot parse JSON: %w", err)
	}
	var d data
	err = json.Unmarshal(respmap["data"], &d)
	if err != nil {
		return "", fmt.Errorf("getClusterName() error: cannot parse JSON: %w", err)
	}

	clusterName := d.ClusterName

	return clusterName, nil
}

// getContainerID returns the containerID if currently running within a container.
func (eksUtils eksDetectorUtils) getContainerID() (string, error) {
	fileData, err := ioutil.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", fmt.Errorf("getContainerID() error: cannot read file with path %s: %w", defaultCgroupPath, err)
	}

	r, err := regexp.Compile(`^.*/docker/(.+)$`)
	if err != nil {
		return "", err
	}

	// Retrieve containerID from file
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if r.MatchString(str) {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", fmt.Errorf("getContainerID() error: cannot read containerID from file %s", defaultCgroupPath)
}
