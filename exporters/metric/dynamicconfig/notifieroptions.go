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

package dynamicconfig

import (
	"time"

	"go.opentelemetry.io/otel/sdk/resource"
)

// Option is the interface that applies the value to a configuration option.
type Option interface {
	// Apply sets the Option value of a Config.
	Apply(*Notifier)
}

// WithCheckFrequency sets the checkFrequency configuration option of a Notifier.
func WithCheckFrequency(checkFrequency time.Duration) Option {
	return checkFrequencyOption(checkFrequency)
}

type checkFrequencyOption time.Duration

func (o checkFrequencyOption) Apply(notifier *Notifier) {
	notifier.checkFrequency = time.Duration(o)
}

// WithConfigHost sets the configHost configuration option of a Notifier.
func WithConfigHost(host string) Option {
	return configHostOption(host)
}

type configHostOption string

func (o configHostOption) Apply(notifier *Notifier) {
	notifier.configHost = string(o)
}

// WithResource sets the resource configuration option of a gNotifier
func WithResource(r *resource.Resource) Option {
	return resourceOption{r}
}

type resourceOption struct{ *resource.Resource }

func (o resourceOption) Apply(notifier *Notifier) {
	notifier.resource = o.Resource
}
