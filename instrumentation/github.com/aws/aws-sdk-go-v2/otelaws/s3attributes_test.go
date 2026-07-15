// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

func TestS3CopyObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CopyObjectInput{
			Bucket:     aws.String("test-bucket"),
			Key:        aws.String("test-key"),
			CopySource: aws.String("src-bucket/src-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3CopySource("src-bucket/src-key"))
}

func TestS3DeleteObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3DeleteObjectsInput(t *testing.T) {
	del := &s3types.Delete{
		Objects: []s3types.ObjectIdentifier{
			{Key: aws.String("key1")},
			{Key: aws.String("key2")},
		},
	}
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteObjectsInput{
			Bucket: aws.String("test-bucket"),
			Delete: del,
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	d, _ := json.Marshal(del)
	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Delete(string(d)))
}

func TestS3GetObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3HeadObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.HeadObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3PutObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3RestoreObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.RestoreObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3SelectObjectContentInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.SelectObjectContentInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3ListBucketsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketsInput{},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.RPCSystemNameKey.String(AWSSystemVal))
	assert.Len(t, attributes, 1)
}

func TestS3AbortMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.AbortMultipartUploadInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("upload-id-123"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3UploadID("upload-id-123"))
}

func TestS3CompleteMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("upload-id-123"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3UploadID("upload-id-123"))
}

func TestS3CreateMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CreateMultipartUploadInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
}

func TestS3ListPartsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListPartsInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("upload-id-123"),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3UploadID("upload-id-123"))
}

func TestS3UploadPartInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.UploadPartInput{
			Bucket:     aws.String("test-bucket"),
			Key:        aws.String("test-key"),
			UploadId:   aws.String("upload-id-123"),
			PartNumber: aws.Int32(5),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3UploadID("upload-id-123"))
	assert.Contains(t, attributes, semconv.AWSS3PartNumber(5))
}

func TestS3UploadPartCopyInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.UploadPartCopyInput{
			Bucket:     aws.String("test-bucket"),
			Key:        aws.String("test-key"),
			CopySource: aws.String("src-bucket/src-key"),
			UploadId:   aws.String("upload-id-123"),
			PartNumber: aws.Int32(3),
		},
	}

	attributes := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, semconv.AWSS3Bucket("test-bucket"))
	assert.Contains(t, attributes, semconv.AWSS3Key("test-key"))
	assert.Contains(t, attributes, semconv.AWSS3CopySource("src-bucket/src-key"))
	assert.Contains(t, attributes, semconv.AWSS3UploadID("upload-id-123"))
	assert.Contains(t, attributes, semconv.AWSS3PartNumber(3))
}
