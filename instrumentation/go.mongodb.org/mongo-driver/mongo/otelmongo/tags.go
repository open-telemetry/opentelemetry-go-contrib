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

package otelmongo

import "go.opentelemetry.io/otel/attribute"

const (
	TargetHostKey  = attribute.Key("out.host")
	TargetPortKey  = attribute.Key("out.port")
	ServiceNameKey = attribute.Key("service.name")
	DBOperationKey = attribute.Key("db.operation")
	ErrorKey       = attribute.Key("error")
	ErrorMsgKey    = attribute.Key("error.msg")
)

// TargetHost sets the target host address.
func TargetHost(targetHost string) attribute.KeyValue {
	return TargetHostKey.String(targetHost)
}

// TargetPort sets the target host port.
func TargetPort(targetPort string) attribute.KeyValue {
	return TargetPortKey.String(targetPort)
}

// ServiceName defines the Service name for this Span.
func ServiceName(serviceName string) attribute.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// DBOperation defines the name of the operation.
func DBOperation(operation string) attribute.KeyValue {
	return DBOperationKey.String(operation)
}

// Error specifies whether an error occurred.
func Error(err bool) attribute.KeyValue {
	return ErrorKey.Bool(err)
}

// ErrorMsg specifies the error message.
func ErrorMsg(errorMsg string) attribute.KeyValue {
	return ErrorMsgKey.String(errorMsg)
}
