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

package otels3

import "go.opentelemetry.io/otel/label"

const (
	operationPutObject    = "PutObject"
	operationGetObject    = "GetObject"
	operationDeleteObject = "DeleteObject"

	storageOperationKey   = label.Key("aws.s3.operation")
	storageDestinationKey = label.Key("aws.s3.destination")
	storageSystemKey      = label.Key("aws.s3.system")

	s3StorageSystemValue = "s3"

	labelKeyStatus    = "status"
	labelValueSuccess = "success"
	labelValueFailure = "failure"
)

var (
	labelStatusSuccess = label.String(labelKeyStatus, labelValueSuccess)
	labelStatusFailure = label.String(labelKeyStatus, labelValueFailure)
)

func s3StorageOperation(operation string) label.KeyValue {
	return storageOperationKey.String(operation)
}

func s3StorageDestination(destination string) label.KeyValue {
	return storageDestinationKey.String(destination)
}

func s3StorageSystem() label.KeyValue {
	return storageSystemKey.String(s3StorageSystemValue)
}

func createAttributes(destination, operation string) []label.KeyValue {
	return []label.KeyValue{
		s3StorageSystem(),
		s3StorageDestination(destination),
		s3StorageOperation(operation),
	}
}
