// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

// S3AttributeBuilder sets S3 specific attributes depending on the S3 operation being performed.
func S3AttributeBuilder(_ context.Context, in middleware.InitializeInput, _ middleware.InitializeOutput) []attribute.KeyValue {
	s3Attributes := []attribute.KeyValue{semconv.RPCSystemNameKey.String(AWSSystemVal)}

	switch v := in.Parameters.(type) {
	case *s3.CopyObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.CopySource != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3CopySource(*v.CopySource))
		}

	case *s3.DeleteObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.DeleteObjectsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		if v.Delete != nil {
			d, _ := json.Marshal(v.Delete)
			s3Attributes = append(s3Attributes, semconv.AWSS3Delete(string(d)))
		}

	case *s3.GetObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.HeadObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.PutObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.RestoreObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.SelectObjectContentInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.ListBucketsInput:
		// No S3-specific attributes for list-buckets.

	case *s3.AbortMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.CompleteMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.CreateMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))

	case *s3.ListPartsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.UploadPartInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}
		if v.PartNumber != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3PartNumber(int(*v.PartNumber)))
		}

	case *s3.UploadPartCopyInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(*v.Bucket))
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(*v.Key))
		if v.CopySource != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3CopySource(*v.CopySource))
		}
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}
		if v.PartNumber != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3PartNumber(int(*v.PartNumber)))
		}
	}

	return s3Attributes
}
