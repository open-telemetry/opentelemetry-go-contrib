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

//nolint:all // Linter is executed at the same time as tests which leads to race conditions and failures.
package main

import (
	. "go.opentelemetry.io/contrib/instrgen/testdata/interface/app"
	. "go.opentelemetry.io/contrib/instrgen/testdata/interface/serializer"
	"go.opentelemetry.io/contrib/instrgen/rtlib"
)

func main() {

	rtlib.AutotelEntryPoint()
	bs := BasicSerializer{}
	var s Serializer
	s = bs
	s.Serialize()
}
