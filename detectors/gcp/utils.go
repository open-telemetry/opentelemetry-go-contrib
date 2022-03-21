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

package gcp

import "cloud.google.com/go/compute/metadata"

// The minimal list of metadata.Client methods we use. Use an interface so we
// can replace it with a fake implementation in the unit test.
type metadataClient interface {
	ProjectID() (string, error)
	Zone() (string, error)
	InstanceID() (string, error)
	InstanceName() (string, error)
	Get(string) (string, error)
	InstanceAttributeValue(string) (string, error)
}

// hasProblem checks if the err is not nil or for missing resources
func hasProblem(err error) bool {
	if err == nil {
		return false
	}
	if _, undefined := err.(metadata.NotDefinedError); undefined {
		return false
	}
	return true
}
