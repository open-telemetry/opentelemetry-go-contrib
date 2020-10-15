package otels3

import "go.opentelemetry.io/otel/label"

const (
	operationPutObject    = "PutObject"
	operationGetObject    = "GetObject"
	operationDeleteObject = "DeleteObject"

	storageOperationKey   = label.Key("storage.operation")
	storageDestinationKey = label.Key("storage.destination")
	storageSystemKey      = label.Key("storage.system")

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
