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

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

// AWS collects resource information of AWS computing instances
type AWS struct {
	c client
}

type client interface {
	Available() bool
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

// compile time assertion that AWS implements the resource.Detector interface.
var _ resource.Detector = (*AWS)(nil)

// Detect detects associated resources when running in AWS environment.
func (aws *AWS) Detect(ctx context.Context) (*resource.Resource, error) {
	client, err := aws.client()
	if err != nil {
		return nil, err
	}

	if !client.Available() {
		return nil, nil
	}

	doc, err := client.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}

	labels := []label.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudRegionKey.String(doc.Region),
		semconv.CloudAccountIDKey.String(doc.AccountID),
		semconv.HostIDKey.String(doc.InstanceID),
	}

	return resource.New(labels...), nil
}

func (aws *AWS) client() (client, error) {
	if aws.c != nil {
		return aws.c, nil
	}

	s, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return ec2metadata.New(s), nil
}
