// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lib // import "go.opentelemetry.io/contrib/instrgen/lib"

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func isFunPartOfCallGraph(fun FuncDescriptor, callgraph map[FuncDescriptor][]FuncDescriptor) bool {
	// TODO this is not optimap o(n)
	for k, v := range callgraph {
		if k.TypeHash() == fun.TypeHash() {
			return true
		}
		for _, e := range v {
			if fun.TypeHash() == e.TypeHash() {
				return true
			}
		}
	}
	return false
}

// ContextPropagationPass.
type ContextPropagationPass struct{}

// Execute.
func (pass *ContextPropagationPass) Execute(
	node *ast.File,
	analysis *PackageAnalysis,
	pkg *packages.Package,
	pkgs []*packages.Package,
) []Import {
	var imports []Import
	addImports := false
	// below variable is used
	// when callexpr is inside var decl
	// instead of functiondecl
	currentFun := FuncDescriptor{}
	emitEmptyContext := func(callExpr *ast.CallExpr, ctxArg *ast.Ident) {
		addImports = true
		if currentFun != (FuncDescriptor{}) {
			visited := map[FuncDescriptor]bool{}
			if isPath(analysis.Callgraph, currentFun, analysis.RootFunctions[0], visited) {
				callExpr.Args = append([]ast.Expr{ctxArg}, callExpr.Args...)
			} else {
				contextTodo := &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "__atel_context",
						},
						Sel: &ast.Ident{
							Name: "TODO",
						},
					},
					Lparen:   62,
					Ellipsis: 0,
				}
				callExpr.Args = append([]ast.Expr{contextTodo}, callExpr.Args...)
			}
			return
		}
		contextTodo := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "__atel_context",
				},
				Sel: &ast.Ident{
					Name: "TODO",
				},
			},
			Lparen:   62,
			Ellipsis: 0,
		}
		callExpr.Args = append([]ast.Expr{contextTodo}, callExpr.Args...)
	}
	emitCallExpr := func(ident *ast.Ident, n ast.Node, ctxArg *ast.Ident, pkgPath string) {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			funId := pkgPath + "." + pkg.TypesInfo.Uses[ident].Name()
			fun := FuncDescriptor{
				Id:              funId,
				DeclType:        pkg.TypesInfo.Uses[ident].Type().String(),
				CustomInjection: false,
			}
			found := analysis.FuncDecls[fun]

			// inject context parameter only
			// to these functions for which function decl
			// exists

			if found {
				visited := map[FuncDescriptor]bool{}
				if isPath(analysis.Callgraph, fun, analysis.RootFunctions[0], visited) {
					fmt.Println("\t\t\tContextPropagation FuncCall:", funId, pkg.TypesInfo.Uses[ident].Type().String())
					emitEmptyContext(callExpr, ctxArg)
				}
			}
		}
	}
	ast.Inspect(node, func(n ast.Node) bool {
		ctxArg := &ast.Ident{
			Name: "__atel_child_tracing_ctx",
		}
		ctxField := &ast.Field{
			Names: []*ast.Ident{
				{
					Name: "__atel_tracing_ctx",
				},
			},
			Type: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "__atel_context",
				},
				Sel: &ast.Ident{
					Name: "Context",
				},
			},
		}
		switch xNode := n.(type) {
		case *ast.FuncDecl:
			pkgPath := GetPkgPathForFunction(pkg, pkgs, xNode, analysis.Interfaces)
			funId := pkgPath + "." + pkg.TypesInfo.Defs[xNode.Name].Name()
			fun := FuncDescriptor{
				Id:              funId,
				DeclType:        pkg.TypesInfo.Defs[xNode.Name].Type().String(),
				CustomInjection: false,
			}
			currentFun = fun
			// inject context only
			// functions available in the call graph
			if !isFunPartOfCallGraph(fun, analysis.Callgraph) {
				break
			}

			if Contains(analysis.RootFunctions, fun) {
				break
			}
			visited := map[FuncDescriptor]bool{}

			if isPath(analysis.Callgraph, fun, analysis.RootFunctions[0], visited) {
				fmt.Println("\t\t\tContextPropagation FuncDecl:", funId,
					pkg.TypesInfo.Defs[xNode.Name].Type().String())
				addImports = true
				xNode.Type.Params.List = append([]*ast.Field{ctxField}, xNode.Type.Params.List...)
			}
		case *ast.CallExpr:
			if ident, ok := xNode.Fun.(*ast.Ident); ok {
				if pkg.TypesInfo.Uses[ident] == nil {
					return false
				}
				pkgPath := GetPkgNameFromUsesTable(pkg, ident)
				emitCallExpr(ident, n, ctxArg, pkgPath)
			}

			if sel, ok := xNode.Fun.(*ast.SelectorExpr); ok {
				if pkg.TypesInfo.Uses[sel.Sel] == nil {
					return false
				}
				pkgPath := GetPkgNameFromUsesTable(pkg, sel.Sel)
				if sel.X != nil {
					pkgPath = GetSelectorPkgPath(sel, pkg, pkgPath)
				}
				emitCallExpr(sel.Sel, n, ctxArg, pkgPath)
			}

		case *ast.TypeSpec:
			iname := xNode.Name
			iface, ok := xNode.Type.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, method := range iface.Methods.List {
				funcType, ok := method.Type.(*ast.FuncType)
				if !ok {
					return true
				}
				visited := map[FuncDescriptor]bool{}
				pkgPath := GetPkgNameFromDefsTable(pkg, method.Names[0])
				funId := pkgPath + "." + iname.Name + "." + pkg.TypesInfo.Defs[method.Names[0]].Name()
				fun := FuncDescriptor{
					Id:              funId,
					DeclType:        pkg.TypesInfo.Defs[method.Names[0]].Type().String(),
					CustomInjection: false,
				}
				if isPath(analysis.Callgraph, fun, analysis.RootFunctions[0], visited) {
					fmt.Println("\t\t\tContext Propagation InterfaceType", fun.Id, fun.DeclType)
					addImports = true
					funcType.Params.List = append([]*ast.Field{ctxField}, funcType.Params.List...)
				}
			}
		}
		return true
	})
	if addImports {
		imports = append(imports, Import{"__atel_context", "context", Add})
	}
	return imports
}
