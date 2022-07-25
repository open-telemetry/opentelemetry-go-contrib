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

package jaegerremote // import "go.opentelemetry.io/contrib/samplers/jaegerremote"

import "github.com/go-logr/logr"

// NullLogger is implementation of the Logger interface that is no-op
var NullLoggger = &nullLogger{}

type nullLogger struct{}

func (n *nullLogger) Init(info logr.RuntimeInfo) {}

func (n *nullLogger) Enabled(level int) bool {
	return false
}

func (n *nullLogger) Info(level int, msg string, keysAndValues ...interface{}) {}

func (n *nullLogger) Error(err error, msg string, keysAndValues ...interface{}) {}

func (n *nullLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return n
}

func (n *nullLogger) WithName(name string) logr.LogSink {
	return n
}
