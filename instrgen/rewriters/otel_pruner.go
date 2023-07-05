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

package rewriters // import "go.opentelemetry.io/contrib/instrgen/rewriters"

import (
	"go/ast"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func removeStmt(slice []ast.Stmt, s int) []ast.Stmt {
	return append(slice[:s], slice[s+1:]...)
}

func removeField(slice []*ast.Field, s int) []*ast.Field {
	return append(slice[:s], slice[s+1:]...)
}

func removeExpr(slice []ast.Expr, s int) []ast.Expr {
	return append(slice[:s], slice[s+1:]...)
}

func inspectFuncContent(fType *ast.FuncType, fBody *ast.BlockStmt, remove bool) bool {
	instrgenCode := false
	for index := 0; index < len(fType.Params.List); index++ {
		param := fType.Params.List[index]
		for _, ident := range param.Names {
			if strings.Contains(ident.Name, "__atel_") {
				if remove == true {
					fType.Params.List = removeField(fType.Params.List, index)
					index--
				}
				instrgenCode = true
			}
		}
	}
	for index := 0; index < len(fBody.List); index++ {
		stmt := fBody.List[index]
		switch bodyStmt := stmt.(type) {
		case *ast.IfStmt:
			if assigment, ok := bodyStmt.Init.(*ast.AssignStmt); ok {
				if ident, ok := assigment.Lhs[0].(*ast.Ident); ok {
					if strings.Contains(ident.Name, "__atel_") {
						if remove == true {
							fBody.List = removeStmt(fBody.List, index)
							index--
						}
						instrgenCode = true
					}
				}
			}
		case *ast.AssignStmt:
			if ident, ok := bodyStmt.Lhs[0].(*ast.Ident); ok {
				if strings.Contains(ident.Name, "__atel_") {
					if remove == true {
						fBody.List = removeStmt(fBody.List, index)
						index--
					}
					instrgenCode = true
				}
			}
			if ident, ok := bodyStmt.Rhs[0].(*ast.Ident); ok {
				if strings.Contains(ident.Name, "__atel_") {
					if remove == true {
						fBody.List = removeStmt(fBody.List, index)
						index--
					}
					instrgenCode = true
				}
			}
		case *ast.ExprStmt:
			if call, ok := bodyStmt.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if strings.Contains(sel.Sel.Name, "SetTracerProvider") {
						if remove == true {
							fBody.List = removeStmt(fBody.List, index)
							index--
						}
						instrgenCode = true
					}
					if strings.Contains(sel.Sel.Name, "InstrgenSetTls") {
						if remove == true {
							fBody.List = removeStmt(fBody.List, index)
							index--
						}
						instrgenCode = true
					}
				}
			}
		case *ast.DeferStmt:
			if sel, ok := bodyStmt.Call.Fun.(*ast.SelectorExpr); ok {
				if strings.Contains(sel.Sel.Name, "Shutdown") {
					if ident, ok := sel.X.(*ast.Ident); ok {
						if strings.Contains(ident.Name, "rtlib") {
							if remove == true {
								fBody.List = removeStmt(fBody.List, index)
								index--
							}
							instrgenCode = true
						}
					}
				}
				if ident, ok := sel.X.(*ast.Ident); ok {
					if strings.Contains(ident.Name, "__atel_") {
						if remove == true {
							fBody.List = removeStmt(fBody.List, index)
							index--
						}
						instrgenCode = true
					}
				}
			}
		}
	}
	return instrgenCode
}

func inspect(file *ast.File, remove bool) bool {
	instrgenCode := false
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body != nil {
				instrgenCode = inspectFuncContent(x.Type, x.Body, remove)
			}
		case *ast.CallExpr:
			for argIndex := 0; argIndex < len(x.Args); argIndex++ {
				if ident, ok := x.Args[argIndex].(*ast.Ident); ok {
					if strings.Contains(ident.Name, "__atel_") {
						if remove == true {
							x.Args = removeExpr(x.Args, argIndex)
							argIndex--
						}
						instrgenCode = true
					}
				}
			}
			for argIndex := 0; argIndex < len(x.Args); argIndex++ {
				if c, ok := x.Args[argIndex].(*ast.CallExpr); ok {
					if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							if strings.Contains(ident.Name, "__atel_") {
								if remove == true {
									x.Args = removeExpr(x.Args, argIndex)
									argIndex--
								}
								instrgenCode = true
							}
						}
					}
				}
			}
		case *ast.FuncLit:
			instrgenCode = inspectFuncContent(x.Type, x.Body, remove)
		case *ast.TypeSpec:
			iface, ok := x.Type.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, method := range iface.Methods.List {
				funcType, ok := method.Type.(*ast.FuncType)
				if !ok {
					continue
				}
				for argIndex := 0; argIndex < len(funcType.Params.List); argIndex++ {
					for _, ident := range funcType.Params.List[argIndex].Names {
						if strings.Contains(ident.Name, "__atel_") {
							if remove == true {
								funcType.Params.List = removeField(funcType.Params.List, argIndex)
								argIndex--
							}
							instrgenCode = true
						}
					}
				}
			}
		}
		return true
	})
	return instrgenCode
}

// OtelPruner.
type OtelPruner struct {
	FilePattern string
	Replace     bool
}

// Id.
func (OtelPruner) Id() string {
	return "Pruner"
}

// Inject.
func (pruner OtelPruner) Inject(pkg string, filepath string) bool {
	return strings.Contains(filepath, pruner.FilePattern)
}

// ReplaceSource.
func (pruner OtelPruner) ReplaceSource(pkg string, filePath string) bool {
	return pruner.Replace
}

// Rewrite.
func (OtelPruner) Rewrite(pkg string, file *ast.File, fset *token.FileSet, trace *os.File) {
	inspect(file, true)
	astutil.DeleteNamedImport(fset, file, "__atel_context", "context")
	astutil.DeleteNamedImport(fset, file, "__atel_otel", "go.opentelemetry.io/otel")
	astutil.DeleteNamedImport(fset, file, "__atel_runtime", "runtime")
	astutil.DeleteNamedImport(fset, file, "__atel_trace", "go.opentelemetry.io/otel/trace")
	astutil.DeleteNamedImport(fset, file, "__atel_sdktrace", "go.opentelemetry.io/otel/sdk/trace")
	astutil.DeleteImport(fset, file, "go.opentelemetry.io/contrib/instrgen/rtlib")
}

// WriteExtraFiles.
func (OtelPruner) WriteExtraFiles(pkg string, destPath string) []string {
	return nil
}
