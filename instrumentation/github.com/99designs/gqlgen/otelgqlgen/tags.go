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

package otelgqlgen

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"go.opentelemetry.io/otel/attribute"
)

const (
	RequestVariablesPrefix        = "request.variables"
	ResolverArgsPrefix            = "resolver.args"
	ResolverErrorPrefix           = "resolver.error"
	ServiceNameKey                = attribute.Key("service.name")
	RequestQueryKey               = attribute.Key("request.query")
	RequestComplexityLimitKey     = attribute.Key("request.complexityLimit")
	RequestOperationComplexityKey = attribute.Key("request.operationComplexity")
	ResolverPathKey               = attribute.Key("resolver.path")
	ResolverObjectKey             = attribute.Key("resolver.object")
	ResolverFieldKey              = attribute.Key("resolver.field")
	ResolverAliasKey              = attribute.Key("resolver.alias")
	ResolverHasErrorKey           = attribute.Key("resolver.hasError")
	ResolverErrorCountKey         = attribute.Key("resolver.errorCount")
)

// ServiceName defines the service name for this span.
func ServiceName(serviceName string) attribute.KeyValue {
	return ServiceNameKey.String(serviceName)
}

// RequestQuery sets the request query.
func RequestQuery(requestQuery string) attribute.KeyValue {
	return RequestQueryKey.String(requestQuery)
}

// RequestComplexityLimit sets the complexity limit.
func RequestComplexityLimit(complexityLimit int64) attribute.KeyValue {
	return RequestComplexityLimitKey.Int64(complexityLimit)
}

// RequestOperationComplexity sets the operation complexity.
func RequestOperationComplexity(complexityLimit int64) attribute.KeyValue {
	return RequestOperationComplexityKey.Int64(complexityLimit)
}

// RequestVariables sets request variables.
func RequestVariables(requestVariables map[string]interface{}) []attribute.KeyValue {
	variables := make([]attribute.KeyValue, 0, len(requestVariables))
	for k, v := range requestVariables {
		variables = append(variables,
			attribute.String(fmt.Sprintf("%s.%s", RequestVariablesPrefix, k), fmt.Sprintf("%+v", v)),
		)
	}
	return variables
}

// ResolverPath sets resolver path.
func ResolverPath(resolverPath string) attribute.KeyValue {
	return ResolverPathKey.String(resolverPath)
}

// ResolverObject sets resolver object.
func ResolverObject(resolverObject string) attribute.KeyValue {
	return ResolverObjectKey.String(resolverObject)
}

// ResolverField sets resolver field.
func ResolverField(resolverField string) attribute.KeyValue {
	return ResolverFieldKey.String(resolverField)
}

// ResolverAlias sets resolver alias.
func ResolverAlias(resolverAlias string) attribute.KeyValue {
	return ResolverAliasKey.String(resolverAlias)
}

// ResolverArgs sets resolver args.
func ResolverArgs(argList ast.ArgumentList) []attribute.KeyValue {
	args := make([]attribute.KeyValue, 0, len(argList))

	for _, arg := range argList {
		if arg.Value != nil {
			args = append(args, attribute.String(fmt.Sprintf("%s.%s", ResolverArgsPrefix, arg.Name), arg.Value.String()))
		}
	}

	return args
}

// ResolverErrors sets errors.
func ResolverErrors(errorList gqlerror.List) []attribute.KeyValue {
	errors := make([]attribute.KeyValue, 0, len(errorList)*4)
	for idx, err := range errorList {
		errors = append(
			errors,
			ResolverHasErrorKey.Bool(true),
			ResolverHasErrorKey.Int64(int64(len(errorList))),
			attribute.String(fmt.Sprintf("%s.%d.message", ResolverErrorPrefix, idx), err.Error()),
			attribute.String(fmt.Sprintf("%s.%d.kind", ResolverErrorPrefix, idx), fmt.Sprintf("%T", err)),
		)
	}

	return errors
}
