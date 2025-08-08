// Package analyzers contains custom analyzers for static analysis.
package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// NoOsExitMainAnalyzer disallows direct calls to os.Exit in the main function
// of the main package.
//
// This is done to prevent abrupt program termination without proper cleanup,
// for example, to enforce centralized exit handling.
var NoOsExitMainAnalyzer = &analysis.Analyzer{
	Name: "noosexitmain",
	Doc:  "disallow direct calls to os.Exit in main.main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Check if the file belongs to the main package
		if pass.Pkg.Name() != "main" {
			continue
		}

		// Find the main function
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "main" {
				continue
			}

			// Inspect the body of the main function
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Check if the call is os.Exit
				selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := selectorExpr.X.(*ast.Ident)
				if !ok {
					return true
				}

				if ident.Name == "os" && selectorExpr.Sel.Name == "Exit" {
					// Make sure the identifier "os" is imported from the "os" package
					obj := pass.TypesInfo.Uses[ident]
					if obj == nil {
						return true
					}
					if pkgName, ok := obj.(*types.PkgName); ok {
						if pkgName.Imported().Path() == "os" {
							pass.Reportf(callExpr.Pos(), "direct call to os.Exit in main.main is forbidden")
						}
					}
				}
				return true
			})
		}
	}
	return nil, nil
}
