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

import "go.opentelemetry.io/otel/label"

const (
	TargetHostKey  = label.Key("out.host")
	TargetPortKey  = label.Key("out.port")
	ServiceNameKey = label.Key("service.name")
	DBOperationKey = label.Key("db.operation")
	ErrorKey       = label.Key("error")
	ErrorMsgKey    = label.Key("error.msg")
)

// TargetHost sets the target host address.
func TargetHost(targetHost string) label.KeyValue {
	return TargetHostKey.String(targetHost)
}

// TargetPort sets the target host port.
func TargetPort(targetPort string) label.KeyValue {
	return TargetPortKey.String(targetPort)
}

// ServiceName defines the Service name for this Span.
func ServiceName(serviceName string) label.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// DBOperation defines the name of the operation.
func DBOperation(operation string) label.KeyValue {
	return DBOperationKey.String(operation)
}

// Error specifies whether an error occurred.
func Error(err bool) label.KeyValue {
	return ErrorKey.Bool(err)
}

// ErrorMsg specifies the error message.
func ErrorMsg(errorMsg string) label.KeyValue {
	return ErrorMsgKey.String(errorMsg)
}
