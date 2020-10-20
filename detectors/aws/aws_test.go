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
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

func TestAWS_Detect(t *testing.T) {
	type fields struct {
		Client client
	}

	type want struct {
		Error    string
		Resource *resource.Resource
	}

	testTable := map[string]struct {
		Fields fields
		Want   want
	}{
		"Unavailable": {
			Fields: fields{Client: &clientMock{}},
		},
		"Instance ID Error": {
			Fields: fields{
				Client: &clientMock{available: true, idDoc: func() (ec2metadata.EC2InstanceIdentityDocument, error) {
					return ec2metadata.EC2InstanceIdentityDocument{}, errors.New("id not available")
				}},
			},
			Want: want{Error: "id not available"},
		},
		"Instance ID Available": {
			Fields: fields{
				Client: &clientMock{available: true, idDoc: func() (ec2metadata.EC2InstanceIdentityDocument, error) {
					// Example from https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
					doc := ec2metadata.EC2InstanceIdentityDocument{
						MarketplaceProductCodes: []string{"1abc2defghijklm3nopqrs4tu"},
						AvailabilityZone:        "us-west-2b",
						PrivateIP:               "10.158.112.84",
						Version:                 "2017-09-30",
						Region:                  "us-west-2",
						InstanceID:              "i-1234567890abcdef0",
						InstanceType:            "t2.micro",
						AccountID:               "123456789012",
						PendingTime:             time.Date(2016, time.November, 19, 16, 32, 11, 0, time.UTC),
						ImageID:                 "ami-5fb8c835",
						Architecture:            "x86_64",
					}

					return doc, nil
				}},
			},
			Want: want{Resource: resource.New(
				semconv.CloudProviderAWS,
				semconv.CloudRegionKey.String("us-west-2"),
				semconv.CloudAccountIDKey.String("123456789012"),
				semconv.HostIDKey.String("i-1234567890abcdef0"),
			)},
		},
	}

	for name, tt := range testTable {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			aws := AWS{c: tt.Fields.Client}

			r, err := aws.Detect(context.Background())

			if tt.Want.Error != "" {
				require.EqualError(t, err, tt.Want.Error, "Error")
				return
			}

			require.NoError(t, err, "Error")
			assert.Equal(t, tt.Want.Resource, r, "Resource")
		})
	}
}

type clientMock struct {
	available bool
	idDoc     func() (ec2metadata.EC2InstanceIdentityDocument, error)
}

func (c *clientMock) Available() bool {
	return c.available
}

func (c *clientMock) GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return c.idDoc()
}
