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
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestS3AttributeBuilderBucketOperations(t *testing.T) {
	tests := []struct {
		name       string
		params     any
		wantBucket string
	}{
		{
			name:       "ListObjectsV2",
			params:     &s3.ListObjectsV2Input{Bucket: aws.String("test-bucket")},
			wantBucket: "test-bucket",
		},
		{
			name:       "CreateBucket",
			params:     &s3.CreateBucketInput{Bucket: aws.String("test-bucket")},
			wantBucket: "test-bucket",
		},
		{
			name:       "HeadBucket",
			params:     &s3.HeadBucketInput{Bucket: aws.String("test-bucket")},
			wantBucket: "test-bucket",
		},
		{
			name:       "DeleteBucket",
			params:     &s3.DeleteBucketInput{Bucket: aws.String("test-bucket")},
			wantBucket: "test-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := middleware.InitializeInput{Parameters: tt.params}
			attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

			assert.Equal(t, []attribute.KeyValue{semconv.AWSS3Bucket(tt.wantBucket)}, attrs)
		})
	}
}

func TestS3AttributeBuilderObjectOperations(t *testing.T) {
	tests := []struct {
		name   string
		params any
	}{
		{
			name:   "GetObject",
			params: &s3.GetObjectInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "PutObject",
			params: &s3.PutObjectInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "DeleteObject",
			params: &s3.DeleteObjectInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "HeadObject",
			params: &s3.HeadObjectInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "RestoreObject",
			params: &s3.RestoreObjectInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "SelectObjectContent",
			params: &s3.SelectObjectContentInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
		{
			name:   "CreateMultipartUpload",
			params: &s3.CreateMultipartUploadInput{Bucket: aws.String("test-bucket"), Key: aws.String("test-key")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := middleware.InitializeInput{Parameters: tt.params}
			attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

			assert.Contains(t, attrs, semconv.AWSS3Bucket("test-bucket"))
			assert.Contains(t, attrs, semconv.AWSS3Key("test-key"))
		})
	}
}

func TestS3AttributeBuilderSpecialOperations(t *testing.T) {
	del := &s3types.Delete{
		Objects: []s3types.ObjectIdentifier{
			{Key: aws.String("key1")},
			{Key: aws.String("key2")},
		},
	}
	delJSON, _ := json.Marshal(del)

	tests := []struct {
		name      string
		params    any
		wantAttrs []attribute.KeyValue
	}{
		{
			name: "CopyObject",
			params: &s3.CopyObjectInput{
				Bucket:     aws.String("test-bucket"),
				Key:        aws.String("test-key"),
				CopySource: aws.String("src-bucket/src-key"),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3CopySource("src-bucket/src-key"),
			},
		},
		{
			name: "DeleteObjects",
			params: &s3.DeleteObjectsInput{
				Bucket: aws.String("test-bucket"),
				Delete: del,
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Delete(string(delJSON)),
			},
		},
		{
			name: "AbortMultipartUpload",
			params: &s3.AbortMultipartUploadInput{
				Bucket:   aws.String("test-bucket"),
				Key:      aws.String("test-key"),
				UploadId: aws.String("upload-id-123"),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3UploadID("upload-id-123"),
			},
		},
		{
			name: "CompleteMultipartUpload",
			params: &s3.CompleteMultipartUploadInput{
				Bucket:   aws.String("test-bucket"),
				Key:      aws.String("test-key"),
				UploadId: aws.String("upload-id-123"),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3UploadID("upload-id-123"),
			},
		},
		{
			name: "ListParts",
			params: &s3.ListPartsInput{
				Bucket:   aws.String("test-bucket"),
				Key:      aws.String("test-key"),
				UploadId: aws.String("upload-id-123"),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3UploadID("upload-id-123"),
			},
		},
		{
			name: "UploadPart",
			params: &s3.UploadPartInput{
				Bucket:     aws.String("test-bucket"),
				Key:        aws.String("test-key"),
				UploadId:   aws.String("upload-id-123"),
				PartNumber: aws.Int32(5),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3UploadID("upload-id-123"),
				semconv.AWSS3PartNumber(5),
			},
		},
		{
			name: "UploadPartCopy",
			params: &s3.UploadPartCopyInput{
				Bucket:     aws.String("test-bucket"),
				Key:        aws.String("test-key"),
				CopySource: aws.String("src-bucket/src-key"),
				UploadId:   aws.String("upload-id-123"),
				PartNumber: aws.Int32(3),
			},
			wantAttrs: []attribute.KeyValue{
				semconv.AWSS3Bucket("test-bucket"),
				semconv.AWSS3Key("test-key"),
				semconv.AWSS3CopySource("src-bucket/src-key"),
				semconv.AWSS3UploadID("upload-id-123"),
				semconv.AWSS3PartNumber(3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := middleware.InitializeInput{Parameters: tt.params}
			attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

			for _, want := range tt.wantAttrs {
				assert.Contains(t, attrs, want)
			}
		})
	}
}

func TestS3AttributeBuilderEdgeCases(t *testing.T) {
	t.Run("ListBuckets", func(t *testing.T) {
		input := middleware.InitializeInput{Parameters: &s3.ListBucketsInput{}}
		attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})
		assert.Empty(t, attrs)
	})

	t.Run("NilBucketAndKey", func(t *testing.T) {
		input := middleware.InitializeInput{Parameters: &s3.GetObjectInput{}}
		attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})
		assert.Empty(t, attrs)
	})

	t.Run("NilParameters", func(t *testing.T) {
		input := middleware.InitializeInput{Parameters: nil}
		attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})
		assert.Empty(t, attrs)
	})

	t.Run("TypedNilParameters", func(t *testing.T) {
		input := middleware.InitializeInput{Parameters: (*s3.CopyObjectInput)(nil)}
		attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})
		assert.Empty(t, attrs)
	})

	t.Run("NilMultipartFields", func(t *testing.T) {
		input := middleware.InitializeInput{
			Parameters: &s3.UploadPartInput{
				Bucket: aws.String("test-bucket"),
				Key:    aws.String("test-key"),
			},
		}
		attrs := S3AttributeBuilder(t.Context(), input, middleware.InitializeOutput{})

		assert.Contains(t, attrs, semconv.AWSS3Bucket("test-bucket"))
		assert.Contains(t, attrs, semconv.AWSS3Key("test-key"))
		assert.Len(t, attrs, 2)
	})
}
