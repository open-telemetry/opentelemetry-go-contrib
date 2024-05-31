// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package lib // import "go.opentelemetry.io/contrib/instrgen/lib"

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// PackageAnalysis analyze all package set according to passed
// pattern. It requires an information about path, pattern,
// root functions - entry points, function declarations,
// and so on.
type PackageAnalysis struct {
	ProjectPath    string
	PackagePattern string
	RootFunctions  []FuncDescriptor
	FuncDecls      map[FuncDescriptor]bool
	Callgraph      map[FuncDescriptor][]FuncDescriptor
	Interfaces     map[string]bool
	Debug          bool
}

type importaction int

const (
	// const that tells whether package should be imported.
	Add importaction = iota
	// or removed.
	Remove
)

// Stores an information about operations on packages.
// Currently packages can be imported with an aliases
// or without.
type Import struct {
	NamedPackage string
	Package      string
	ImportAction importaction
}

// FileAnalysisPass executes an analysis for
// specific file node - translation unit.
type FileAnalysisPass interface {
	Execute(node *ast.File,
		analysis *PackageAnalysis,
		pkg *packages.Package,
		pkgs []*packages.Package) []Import
}

func createFile(name string) (*os.File, error) {
	var out *os.File
	out, err := os.Create(name)
	if err != nil {
		defer out.Close()
	}
	return out, err
}

func addImports(imports []Import, fset *token.FileSet, fileNode *ast.File) {
	for _, imp := range imports {
		if imp.ImportAction == Add {
			if len(imp.NamedPackage) > 0 {
				astutil.AddNamedImport(fset, fileNode, imp.NamedPackage, imp.Package)
			} else {
				astutil.AddImport(fset, fileNode, imp.Package)
			}
		} else {
			if len(imp.NamedPackage) > 0 {
				astutil.DeleteNamedImport(fset, fileNode, imp.NamedPackage, imp.Package)
			} else {
				astutil.DeleteImport(fset, fileNode, imp.Package)
			}
		}
	}
}

// Execute function, main entry point to analysis process.
func (analysis *PackageAnalysis) Execute(pass FileAnalysisPass, fileSuffix string) ([]*ast.File, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{Fset: fset, Mode: LoadMode, Dir: analysis.ProjectPath}
	pkgs, err := packages.Load(cfg, analysis.PackagePattern)
	if err != nil {
		return nil, err
	}
	var fileNodeSet []*ast.File
	for _, pkg := range pkgs {
		fmt.Println("\t", pkg)
		// fileNode represents a translationUnit
		var fileNode *ast.File
		for _, fileNode = range pkg.Syntax {
			fmt.Println("\t\t", fset.File(fileNode.Pos()).Name())
			var out *os.File
			out, err = createFile(fset.File(fileNode.Pos()).Name() + fileSuffix)
			if err != nil {
				return nil, err
			}
			if len(analysis.RootFunctions) == 0 {
				e := printer.Fprint(out, fset, fileNode)
				if e != nil {
					return nil, e
				}
				continue
			}
			imports := pass.Execute(fileNode, analysis, pkg, pkgs)
			addImports(imports, fset, fileNode)
			e := printer.Fprint(out, fset, fileNode)
			if e != nil {
				return nil, e
			}
			if !analysis.Debug {
				oldFileName := fset.File(fileNode.Pos()).Name() + fileSuffix
				newFileName := fset.File(fileNode.Pos()).Name()
				e = os.Rename(oldFileName, newFileName)
				if e != nil {
					return nil, e
				}
			}
			fileNodeSet = append(fileNodeSet, fileNode)
		}
	}
	return fileNodeSet, nil
}
