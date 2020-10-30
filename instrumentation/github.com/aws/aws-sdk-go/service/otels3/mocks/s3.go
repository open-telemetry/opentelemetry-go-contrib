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

// package mocks provides a mock interface to the S3 client
package mocks

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type MockS3Client struct {
	s3iface.S3API
}

// PutObjectWithContext implements the mocked S3 client successfully returning an empty PutObjectOutput type
func (s *MockS3Client) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

// GetObjectWithContext implements the mocked S3 client successfully returning an empty GetObjectOutput type
func (s *MockS3Client) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

// DeleteObjectWithContext implements the mocked S3 client successfully returning an empty DeleteObjectOutput type
func (s *MockS3Client) DeleteObjectWithContext(ctx aws.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

// DeleteObject implements the mocked S3 client successfully returning an empty DeleteObjectOutput type
func (s *MockS3Client) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	fmt.Printf("UnInstrumentedMethod `DeleteObject` called")
	return &s3.DeleteObjectOutput{}, nil
}
