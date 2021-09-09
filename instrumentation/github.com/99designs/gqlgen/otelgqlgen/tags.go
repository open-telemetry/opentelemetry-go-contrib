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
	requestVariablesPrefix        = "gql.request.variables"
	resolverArgsPrefix            = "gql.resolver.args"
	resolverErrorPrefix           = "gql.resolver.error"
	requestQueryKey               = attribute.Key("gql.request.query")
	requestComplexityLimitKey     = attribute.Key("gql.request.complexityLimit")
	requestOperationComplexityKey = attribute.Key("gql.request.operationComplexity")
	resolverPathKey               = attribute.Key("gql.resolver.path")
	resolverObjectKey             = attribute.Key("gql.resolver.object")
	resolverFieldKey              = attribute.Key("gql.resolver.field")
	resolverAliasKey              = attribute.Key("gql.resolver.alias")
	resolverHasErrorKey           = attribute.Key("gql.resolver.hasError")
)

// RequestQuery sets the request query.
func RequestQuery(requestQuery string) attribute.KeyValue {
	return requestQueryKey.String(requestQuery)
}

// RequestComplexityLimit sets the complexity limit.
func RequestComplexityLimit(complexityLimit int64) attribute.KeyValue {
	return requestComplexityLimitKey.Int64(complexityLimit)
}

// RequestOperationComplexity sets the operation complexity.
func RequestOperationComplexity(complexityLimit int64) attribute.KeyValue {
	return requestOperationComplexityKey.Int64(complexityLimit)
}

// RequestVariables sets request variables.
func RequestVariables(requestVariables map[string]interface{}) []attribute.KeyValue {
	variables := make([]attribute.KeyValue, 0, len(requestVariables))
	for k, v := range requestVariables {
		variables = append(variables,
			attribute.String(fmt.Sprintf("%s.%s", requestVariablesPrefix, k), fmt.Sprintf("%+v", v)),
		)
	}
	return variables
}

// ResolverPath sets resolver path.
func ResolverPath(resolverPath string) attribute.KeyValue {
	return resolverPathKey.String(resolverPath)
}

// ResolverObject sets resolver object.
func ResolverObject(resolverObject string) attribute.KeyValue {
	return resolverObjectKey.String(resolverObject)
}

// ResolverField sets resolver field.
func ResolverField(resolverField string) attribute.KeyValue {
	return resolverFieldKey.String(resolverField)
}

// ResolverAlias sets resolver alias.
func ResolverAlias(resolverAlias string) attribute.KeyValue {
	return resolverAliasKey.String(resolverAlias)
}

// ResolverArgs sets resolver args.
func ResolverArgs(argList ast.ArgumentList) []attribute.KeyValue {
	args := make([]attribute.KeyValue, 0, len(argList))

	for _, arg := range argList {
		if arg.Value != nil {
			args = append(args, attribute.String(fmt.Sprintf("%s.%s", resolverArgsPrefix, arg.Name), arg.Value.String()))
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
			resolverHasErrorKey.Bool(true),
			resolverHasErrorKey.Int64(int64(len(errorList))),
			attribute.String(fmt.Sprintf("%s.%d.message", resolverErrorPrefix, idx), err.Error()),
			attribute.String(fmt.Sprintf("%s.%d.kind", resolverErrorPrefix, idx), fmt.Sprintf("%T", err)),
		)
	}

	return errors
}
