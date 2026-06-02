package noosexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports direct calls to os.Exit in main.main.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "forbids direct os.Exit calls inside main.main of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				return true
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "Exit" {
					return true
				}

				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "os" {
					return true
				}

				if obj := pass.TypesInfo.Uses[pkgIdent]; obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != "os" {
					return true
				}

				pass.Reportf(call.Pos(), "direct os.Exit call in main.main is forbidden")
				return true
			})

			return false
		})
	}

	return nil, nil
}
