package codegen

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
)

func TestGenerateGoRejectsUnresolvedCallViaHIRLowering(t *testing.T) {
	prog := &ast.Program{
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name:       "main",
				ReturnType: ast.TypeVoid,
				Body: &ast.BlockStmt{
					Stmts: []ast.Stmt{
						&ast.ExprStmt{
							Expr: &ast.CallExpr{
								Receiver: &ast.IdentExpr{Name: "user"},
								Method:   "label",
							},
						},
					},
				},
			},
		},
	}
	_, err := GenerateGo(prog)
	if err == nil {
		t.Fatalf("expected unresolved call error")
	}
	if !strings.Contains(err.Error(), "unresolved call expression") {
		t.Fatalf("unexpected error: %v", err)
	}
}
