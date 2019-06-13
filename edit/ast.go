package edit

import (
	"go/ast"
	"go/token"
)

func Selector(x ast.Expr, sel *ast.Ident) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   x,
		Sel: sel,
	}
}

func Ident(name string) *ast.Ident {
	return &ast.Ident{
		Name: name,
	}
}

func Call(fun ast.Expr, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

func Unary(op token.Token, x ast.Expr) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		Op: op,
		X:  x,
	}
}
