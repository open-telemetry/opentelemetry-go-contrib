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

//go:build go1.18 && !go1.19

package runtime

var expectRuntimeMetrics = map[string]int{
	"gc.cycles":              2,
	"gc.heap.allocs":         1,
	"gc.heap.allocs.objects": 1,
	"gc.heap.frees":          1,
	"gc.heap.frees.objects":  1,
	"gc.heap.goal":           1,
	"gc.heap.objects":        1,
	"gc.heap.tiny.allocs":    1,
	"memory.classes":         13,
	"sched.goroutines":       1,
}
