package compiler

import "github.com/arcgolabs/plano/ast"

func callName(expr ast.Expr) (string, bool) {
	switch node := expr.(type) {
	case *ast.IdentExpr:
		return node.Name.Name, true
	case *ast.SelectorExpr:
		prefix, ok := callName(node.X)
		if !ok {
			return "", false
		}
		return prefix + "." + node.Sel.Name, true
	default:
		return "", false
	}
}
