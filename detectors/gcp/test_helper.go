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

import "fmt"

var (
	notOnGCE = func() bool { return false }
	onGCE    = func() bool { return true }
)

type client struct {
	m map[string]string
}

func getenv(m map[string]string) func(string) string {
	return func(s string) string {
		if m == nil {
			return ""
		}
		return m[s]
	}
}

func (c *client) Get(s string) (string, error) {
	got, ok := c.m[s]
	if !ok {
		return "", fmt.Errorf("%q do not exist", s)
	} else if got == "" {
		return "", fmt.Errorf("%q is empty", s)
	}
	return got, nil
}

func (c *client) InstanceID() (string, error) {
	return c.Get("instance/id")
}

func (c *client) ProjectID() (string, error) {
	return c.Get("project/project-id")
}

func (c *client) Zone() (string, error) {
	return c.Get("instance/zone")
}

func (c *client) InstanceName() (string, error) {
	return c.Get("instance/name")
}

func (c *client) InstanceAttributeValue(s string) (string, error) {
	return c.Get(fmt.Sprintf("instance/attributes/%s", s))
}
