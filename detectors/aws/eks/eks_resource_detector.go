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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

const (
	k8sSvcURL          = "https://kubernetes.default.svc"
	k8sTokenPath       = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	k8sCertPath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	authConfigmapPath  = "/api/v1/namespaces/kube-system/configmaps/aws-auth"
	cwConfigmapPath    = "/api/v1/namespaces/amazon-cloudwatch/configmaps/cluster-info"
	defaultCgroupPath  = "/proc/self/cgroup"
	containerIDLength  = 64
	millisecondTimeOut = 2000
)

// Create interface for functions that need to be mocked
type detectorUtils interface {
	fileExists(filename string) bool
	fetchString(httpMethod string, URL string) (string, error)
	getContainerID() (string, error)
}

// This struct will implement the detectorUtils interface
type eksDetectorUtils struct{}

// ResourceDetector for detecting resources running on Amazon EKS
type ResourceDetector struct {
	utils detectorUtils
}

// JSONResponse is used to parse the JSON response returned from calling HTTP GET to EKS
// eg. {"data":{"cluster.name":"my-cluster"}}
type JSONResponse struct {
	Data DataObject `json:"data"`
}

// DataObject is used to parse the data attribute inside the JSON response returned from calling HTTP GET to EKS
// eg. {"data":{"cluster.name":"my-cluster"}}
type DataObject struct {
	ClusterName string `json:"cluster.name"`
}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// Compile time assertion that eksDetectorUtils implements the detectorUtils interface.
var _ detectorUtils = (*eksDetectorUtils)(nil)

// Detect function collects associated resource attributes when running in Amazon EKS environment.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {

	isEks, err := isEks(detector.utils)
	if err != nil {
		return nil, err
	}

	// Return empty resource object if not running in EKS
	if !isEks {
		return resource.Empty(), nil
	}

	// Create variable to hold resource attributes
	labels := []label.KeyValue{}

	// Get clusterName and append to labels
	clusterName, err := getClusterName(detector.utils)
	if err != nil {
		return nil, err
	}
	if clusterName != "" {
		labels = append(labels, semconv.K8SClusterNameKey.String(clusterName))
	}

	// Get containerID and append to labels
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return nil, err
	}
	if containerID != "" {
		labels = append(labels, semconv.ContainerIDKey.String(containerID))
	}

	// Return new resource object with clusterName and containerID as attributes
	return resource.New(labels...), nil

}

// isEks checks if the current environment is running in EKS
func isEks(utils detectorUtils) (bool, error) {
	if !isK8s(utils) {
		return false, nil
	}

	// Make HTTP GET request
	awsAuth, err := utils.fetchString(http.MethodGet, k8sSvcURL+authConfigmapPath)
	if err != nil {
		return false, fmt.Errorf("isEks() error retrieving auth configmap: %w", err.Error())
	}

	return awsAuth != "", nil
}

// isK8s checks if the current environment is running in a Kubernetes environment
func isK8s(utils detectorUtils) bool {
	return utils.fileExists(k8sTokenPath) && utils.fileExists(k8sCertPath)
}

// fileExists checks if a file with a given filename exists
// this function implements the detectorUtils interface
func (eksUtils eksDetectorUtils) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// fetchString executes an HTTP request with a given HTTP Method and URL string
// this function implements the detectorUtils interface
func (eksUtils eksDetectorUtils) fetchString(httpMethod string, URL string) (string, error) {

	// Create new HTTP request object
	request, err := http.NewRequest(httpMethod, URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new HTTP request with method=%s, URL=%s: %w", httpMethod, URL, err)
	}

	// Set HTTP request header with authentication credentials
	authHeader, err := getK8sCredHeader()
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", authHeader)

	// Get certificate
	caCert, err := ioutil.ReadFile(k8sCertPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file with path %s", k8sCertPath)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Set HTTP request timeout and add certificate
	client := &http.Client{
		Timeout: millisecondTimeOut * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// Execute HTTP request
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request with method=%s, URL=%s: %w", httpMethod, URL, err)
	}

	// Retrieve response body from HTTP request
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response from HTTP request with method=%s, URL=%s: %w", httpMethod, URL, err)
	}

	return string(body), nil
}

// getK8sCredHeader retrieves the kubernetes credential information
func getK8sCredHeader() (string, error) {
	content, err := ioutil.ReadFile(k8sTokenPath)
	if err != nil {
		return "", fmt.Errorf("getK8sCredHeader() error: cannot read file with path %s", k8sTokenPath)
	}

	return "Bearer " + string(content), nil
}

// getClusterName retrieves the clusterName resource attribute
func getClusterName(utils detectorUtils) (string, error) {
	resp, err := utils.fetchString("GET", k8sSvcURL+cwConfigmapPath)
	if err != nil {
		return "", fmt.Errorf("getClusterName() error: %w", err)
	}

	// parse JSON object returned from HTTP request
	var parsedResp JSONResponse
	err = json.Unmarshal([]byte(resp), &parsedResp)
	if err != nil {
		return "", fmt.Errorf("getClusterName() error: cannot parse JSON: %w", w)
	}
	clusterName := parsedResp.Data.ClusterName

	return clusterName, nil
}

// getContainerID retrieves the containerID resource attribute
// this function implements the detectorUtils interface
func (eksUtils eksDetectorUtils) getContainerID() (string, error) {

	// Read file
	fileData, err := ioutil.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", fmt.Errorf("getContainerID() error: cannot read file with path %s", defaultCgroupPath)
	}

	// Retrieve containerID from file
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if len(str) > containerIDLength {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", fmt.Errorf("getContainerID() error: cannot read containerID from file %s", defaultCgroupPath)
}
