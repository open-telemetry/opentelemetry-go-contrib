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

package lib // import "go.opentelemetry.io/contrib/instrgen/lib"

import (
	"go/ast"
	"go/token"
	"os"
)

// PackageRewriter interface does actual input package
// rewriting according to specific criteria.
type PackageRewriter interface {
	// ID Dumps rewriter id.
	Id() string
	// Inject tells whether package should be rewritten.
	Inject(pkg string, filepath string) bool
	// ReplaceSource decides whether input sources should be replaced
	// or all rewriting work should be done in temporary location.
	ReplaceSource(pkg string, filePath string) bool
	// Rewrite does actual package rewriting.
	Rewrite(pkg string, file *ast.File, fset *token.FileSet, trace *os.File)
	// WriteExtraFiles generate additional files that will be linked
	// together to input package.
	// Additional files have to be returned as array of file names.
	WriteExtraFiles(pkg string, destPath string) []string
}
