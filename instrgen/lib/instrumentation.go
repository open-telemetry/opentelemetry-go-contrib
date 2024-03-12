// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lib // import "go.opentelemetry.io/contrib/instrgen/lib"

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// InstrumentationPass.
type InstrumentationPass struct{}

func makeInitStmts(name string) []ast.Stmt {
	childTracingSupress := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "_",
			},
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_child_tracing_ctx",
			},
		},
	}
	s1 := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_ts",
			},
		},
		Tok: token.DEFINE,

		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "rtlib",
					},
					Sel: &ast.Ident{
						Name: "NewTracingState",
					},
				},
				Lparen:   54,
				Ellipsis: 0,
			},
		},
	}
	s2 := &ast.DeferStmt{
		Defer: 27,
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "rtlib",
				},
				Sel: &ast.Ident{
					Name: "Shutdown",
				},
			},
			Lparen: 48,
			Args: []ast.Expr{
				&ast.Ident{
					Name: "__atel_ts",
				},
			},
			Ellipsis: 0,
		},
	}

	s3 := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "__atel_otel",
				},
				Sel: &ast.Ident{
					Name: "SetTracerProvider",
				},
			},
			Lparen: 49,
			Args: []ast.Expr{
				&ast.SelectorExpr{
					X: &ast.Ident{
						Name: "__atel_ts",
					},
					Sel: &ast.Ident{
						Name: "Tp",
					},
				},
			},
			Ellipsis: 0,
		},
	}
	s4 := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_ctx",
			},
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "__atel_context",
					},
					Sel: &ast.Ident{
						Name: "Background",
					},
				},
				Lparen:   52,
				Ellipsis: 0,
			},
		},
	}
	s5 := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_child_tracing_ctx",
			},
			&ast.Ident{
				Name: "__atel_span",
			},
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "__atel_otel",
							},
							Sel: &ast.Ident{
								Name: "Tracer",
							},
						},
						Lparen: 50,
						Args: []ast.Expr{
							&ast.Ident{
								Name: `"` + name + `"`,
							},
						},
						Ellipsis: 0,
					},
					Sel: &ast.Ident{
						Name: "Start",
					},
				},
				Lparen: 62,
				Args: []ast.Expr{
					&ast.Ident{
						Name: "__atel_ctx",
					},
					&ast.Ident{
						Name: `"` + name + `"`,
					},
				},
				Ellipsis: 0,
			},
		},
	}

	s6 := &ast.DeferStmt{
		Defer: 27,
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "__atel_span",
				},
				Sel: &ast.Ident{
					Name: "End",
				},
			},
			Lparen:   41,
			Ellipsis: 0,
		},
	}
	stmts := []ast.Stmt{s1, s2, s3, s4, s5, childTracingSupress, s6}
	return stmts
}

func makeSpanStmts(name string, paramName string) []ast.Stmt {
	s1 := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_child_tracing_ctx",
			},
			&ast.Ident{
				Name: "__atel_span",
			},
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "__atel_otel",
							},
							Sel: &ast.Ident{
								Name: "Tracer",
							},
						},
						Lparen: 50,
						Args: []ast.Expr{
							&ast.Ident{
								Name: `"` + name + `"`,
							},
						},
						Ellipsis: 0,
					},
					Sel: &ast.Ident{
						Name: "Start",
					},
				},
				Lparen: 62,
				Args: []ast.Expr{
					&ast.Ident{
						Name: paramName,
					},
					&ast.Ident{
						Name: `"` + name + `"`,
					},
				},
				Ellipsis: 0,
			},
		},
	}

	s2 := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: "_",
			},
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			&ast.Ident{
				Name: "__atel_child_tracing_ctx",
			},
		},
	}

	s3 := &ast.DeferStmt{
		Defer: 27,
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "__atel_span",
				},
				Sel: &ast.Ident{
					Name: "End",
				},
			},
			Lparen:   41,
			Ellipsis: 0,
		},
	}
	stmts := []ast.Stmt{s1, s2, s3}
	return stmts
}

// Execute.
func (pass *InstrumentationPass) Execute(
	node *ast.File,
	analysis *PackageAnalysis,
	pkg *packages.Package,
	pkgs []*packages.Package,
) []Import {
	var imports []Import
	addImports := false
	addContext := false
	// store all function literals positions
	// that are part of assignment statement
	// it's used to avoid injection into literal
	// more than once
	var functionLiteralPositions []token.Pos
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			pkgPath := GetPkgPathForFunction(pkg, pkgs, x, analysis.Interfaces)
			fundId := pkgPath + "." + pkg.TypesInfo.Defs[x.Name].Name()
			fun := FuncDescriptor{
				Id:              fundId,
				DeclType:        pkg.TypesInfo.Defs[x.Name].Type().String(),
				CustomInjection: false,
			}
			// check if it's root function or
			// one of function in call graph
			// and emit proper ast nodes
			_, exists := analysis.Callgraph[fun]
			if !exists {
				if !Contains(analysis.RootFunctions, fun) {
					return false
				}
			}
			for _, root := range analysis.RootFunctions {
				visited := map[FuncDescriptor]bool{}
				fmt.Println("\t\t\tInstrumentation FuncDecl:", fundId, pkg.TypesInfo.Defs[x.Name].Type().String())
				if isPath(analysis.Callgraph, fun, root, visited) && fun.TypeHash() != root.TypeHash() {
					x.Body.List = append(makeSpanStmts(x.Name.Name, "__atel_tracing_ctx"), x.Body.List...)
					addContext = true
					addImports = true
				} else {
					// check whether this function is root function
					if !Contains(analysis.RootFunctions, fun) {
						return false
					}
					x.Body.List = append(makeInitStmts(x.Name.Name), x.Body.List...)
					addContext = true
					addImports = true
				}
			}
		case *ast.AssignStmt:
			for _, e := range x.Lhs {
				if ident, ok := e.(*ast.Ident); ok {
					_ = ident
					pkgPath := ""
					pkgPath = GetPkgNameFromDefsTable(pkg, ident)
					if pkg.TypesInfo.Defs[ident] == nil {
						return false
					}
					fundId := pkgPath + "." + pkg.TypesInfo.Defs[ident].Name()
					fun := FuncDescriptor{
						Id:              fundId,
						DeclType:        pkg.TypesInfo.Defs[ident].Type().String(),
						CustomInjection: true,
					}
					_, exists := analysis.Callgraph[fun]
					if exists {
						return false
					}
				}
			}
			for _, e := range x.Rhs {
				if funLit, ok := e.(*ast.FuncLit); ok {
					functionLiteralPositions = append(functionLiteralPositions, funLit.Pos())
					funLit.Body.List = append(makeSpanStmts("anonymous", "__atel_child_tracing_ctx"), funLit.Body.List...)
					addImports = true
					addContext = true
				}
			}
		case *ast.FuncLit:
			for _, pos := range functionLiteralPositions {
				if pos == x.Pos() {
					return false
				}
			}
			x.Body.List = append(makeSpanStmts("anonymous", "__atel_child_tracing_ctx"), x.Body.List...)
			addImports = true
			addContext = true
		}

		return true
	})
	if addContext {
		imports = append(imports, Import{"__atel_context", "context", Add})
	}
	if addImports {
		imports = append(imports, Import{"__atel_otel", "go.opentelemetry.io/otel", Add})
	}
	return imports
}
