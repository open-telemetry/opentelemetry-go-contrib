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

// This package provides an implementation of the tracing database
// driver. The driver itself is not standalone - it is an exact
// wrapper around the actual database driver. All the wrapper does is
// starting and ending the spans for some database-related operations.
//
// The driver is created with the NewDriver function. It can be
// configured to use custom span tracer instead of using a global one.
//
// This package does not provide any driver registration functionality
// - for that, either use the parent package of this package or the
// standard "database/sql" package.
package driver // import "go.opentelemetry.io/contrib/plugins/database/sql/driver"
