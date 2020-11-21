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
	"log"
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

type eksDetectorUtils struct{}

// ResourceDetector for detecting resources running on EKS
type ResourceDetector struct {
	utils detectorUtils
}

// JSONResponse is used to parse the JSON response returned from calling HTTP GET to EKS
type JSONResponse struct {
	Data DataObject `json:"data"`
}

// DataObject is used to parse the data attribute inside the JSON response returned from calling HTTP GET to EKS
type DataObject struct {
	ClusterName string `json:"cluster.name"`
}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// Compile time assertion that eksDetectorUtils implements the detectorUtils interface.
var _ detectorUtils = (*eksDetectorUtils)(nil)

// Detect detects associated resources when running with AWS EKS.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {

	labels := []label.KeyValue{}

	isEks, err := isEks(detector.utils)
	if hasProblem(err) {
		return nil, err
	}

	if !isEks {
		return resource.New(labels...), nil
	}

	clusterName, err := getClusterName(detector.utils)
	if hasProblem(err) {
		return nil, err
	}
	if clusterName != "" {
		labels = append(labels, semconv.K8SClusterNameKey.String(clusterName))
	}

	containerID, err := detector.utils.getContainerID()
	if hasProblem(err) {
		return nil, err
	}
	if containerID != "" {
		labels = append(labels, semconv.ContainerIDKey.String(containerID))
	}

	return resource.New(labels...), nil

}

func isEks(utils detectorUtils) (bool, error) {
	if !isK8s(utils) {
		return false, nil
	}

	awsAuth, err := utils.fetchString("GET", k8sSvcURL+authConfigmapPath)
	if hasProblem(err) {
		return false, err
	}

	return awsAuth != "", nil
}

func isK8s(utils detectorUtils) bool {
	return utils.fileExists(k8sTokenPath) && utils.fileExists(k8sCertPath)
}

// eksDetectorUtils is implementing the detectorUtils interface
func (eksUtils eksDetectorUtils) fileExists(filename string) bool {
	fmt.Println("REAL HIT")
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// fetchString is implementing the detectorUtils interface
func (eksUtils eksDetectorUtils) fetchString(httpMethod string, URL string) (string, error) {
	request, err := http.NewRequest(httpMethod, URL, nil)
	if hasProblem(err) {
		return "", err
	}

	authHeader, err := getK8sCredHeader()
	if hasProblem(err) {
		return "", err
	}
	request.Header.Set("Authorization", authHeader)

	caCert, err := ioutil.ReadFile(k8sCertPath)
	if hasProblem(err) {
		return "", err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Timeout: millisecondTimeOut * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	response, err := client.Do(request)
	if hasProblem(err) {
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	if hasProblem(err) {
		return "", err
	}

	return string(body), nil
}

func getK8sCredHeader() (string, error) {
	content, err := ioutil.ReadFile(k8sTokenPath)
	if hasProblem(err) {
		return "", err
	}

	return "Bearer " + string(content), nil
}

func getClusterName(utils detectorUtils) (string, error) {
	resp, err := utils.fetchString("GET", k8sSvcURL+cwConfigmapPath)
	if hasProblem(err) {
		return "", err
	}

	var parsedResp JSONResponse
	err = json.Unmarshal([]byte(resp), &parsedResp)
	if hasProblem(err) {
		return "", err
	}
	clusterName := parsedResp.Data.ClusterName

	return clusterName, nil
}

// getContainerID is implementing the detectorUtils interface
func (eksUtils eksDetectorUtils) getContainerID() (string, error) {
	fileData, err := ioutil.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", err
	}
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if len(str) > containerIDLength {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", err
}

func hasProblem(err error) bool {
	if err == nil {
		return false
	}
	log.Fatalln(err)
	return true
}
