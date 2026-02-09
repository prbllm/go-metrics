package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var PanicCheckAnalyzer = &analysis.Analyzer{
	Name: "paniccheck",
	Doc:  "check for panic, log.Fatal and os.Exit out of the main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Println("paniccheck: running on package", pass.Pkg.Path())
	for _, file := range pass.Files {
		var currentFunc *ast.FuncDecl
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				currentFunc = node
			case *ast.CallExpr:
				inspectCall(pass, node, currentFunc)

			}
			return true
		})
	}
	return nil, nil
}

func isMainFunction(pass *analysis.Pass, currentFunc *ast.FuncDecl) bool {
	if currentFunc == nil {
		return false
	}
	return pass.Pkg.Name() == "main" && currentFunc.Recv == nil && currentFunc.Name.Name == "main"
}

func functionIdent(expression ast.Expr) *ast.Ident {
	switch typ := expression.(type) {
	case *ast.Ident:
		return typ
	case *ast.SelectorExpr:
		return typ.Sel
	default:
		return nil
	}
}

func inspectCall(pass *analysis.Pass, node *ast.CallExpr, currentFunc *ast.FuncDecl) {
	if isMainFunction(pass, currentFunc) {
		return
	}

	ident := functionIdent(node.Fun)
	if ident == nil {
		return
	}

	obj := pass.TypesInfo.Uses[ident]
	if obj == nil {
		return
	}

	if b, ok := obj.(*types.Builtin); ok && b.Name() == "panic" {
		pass.Reportf(node.Pos(), "paniccheck: panic вызывается вне функции main")
		return
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}

	if pkg := fn.Pkg(); pkg != nil {
		switch pkg.Path() {
		case "log":
			if strings.HasPrefix(fn.Name(), "Fatal") {
				pass.Reportf(node.Pos(), "paniccheck: log.%s вызывается вне функции main", fn.Name())
				return
			}
		case "os":
			if fn.Name() == "Exit" {
				pass.Reportf(node.Pos(), "paniccheck: os.Exit вызывается вне функции main")
				return
			}
		}
	}
}
