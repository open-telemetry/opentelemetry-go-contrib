// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

// S3AttributeBuilder sets S3 specific attributes depending on the S3 operation being performed.
func S3AttributeBuilder(_ context.Context, in middleware.InitializeInput, _ middleware.InitializeOutput) []attribute.KeyValue {
	s3Attributes := []attribute.KeyValue{semconv.RPCSystemNameKey.String(AWSSystemVal)}

	if in.Parameters == nil {
		return s3Attributes
	}

	// Extract aws.s3.bucket and aws.s3.key from any S3 input struct.
	// The S3 semantic convention applies bucket to all operations except
	// list-buckets and key to all object-related operations. Using
	// reflection avoids enumerating 80+ S3 API input types.
	if bucket, ok := stringPtrField(in.Parameters, "Bucket"); ok && bucket != "" {
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(bucket))
	}
	if key, ok := stringPtrField(in.Parameters, "Key"); ok && key != "" {
		s3Attributes = append(s3Attributes, semconv.AWSS3Key(key))
	}

	// Operation-specific attributes.
	switch v := in.Parameters.(type) {
	case *s3.CopyObjectInput:
		if v.CopySource != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3CopySource(*v.CopySource))
		}

	case *s3.DeleteObjectsInput:
		if v.Delete != nil {
			d, _ := json.Marshal(v.Delete)
			s3Attributes = append(s3Attributes, semconv.AWSS3Delete(string(d)))
		}

	case *s3.AbortMultipartUploadInput:
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.CompleteMultipartUploadInput:
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.ListPartsInput:
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}

	case *s3.UploadPartInput:
		if v.UploadId != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(*v.UploadId))
		}
		if v.PartNumber != nil {
			s3Attributes = append(s3Attributes, semconv.AWSS3PartNumber(int(*v.PartNumber)))
		}

	case *s3.UploadPartCopyInput:
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

// stringPtrField extracts a *string field by name from a struct pointer.
// Returns the dereferenced string and true if the field exists, is a
// non-nil *string, and the input is a pointer to a struct.
func stringPtrField(v any, name string) (string, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}
	f := rv.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Ptr || f.IsNil() {
		return "", false
	}
	s, ok := f.Elem().Interface().(string)
	return s, ok
}
