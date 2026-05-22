package hir

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
	"baziclang/internal/sema"
)

func TestLowerProducesTypedCallAndMatch(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn User_label(name: string): string { return name; }

fn main(): void {
    let role: Role = Admin;
    let label = match role {
        Guest: "guest",
        Admin: "admin",
    };
    println(User_label(label));
}`
	prog := parseAndCheck(t, src)
	out, err := Lower(prog)
	if err != nil {
		t.Fatalf("lower failed: %v", err)
	}
	mainFn := out.Decls[len(out.Decls)-1].(*FuncDecl)
	letStmt := mainFn.Body.Stmts[1].(*LetStmt)
	matchExpr, ok := letStmt.Init.(*MatchExpr)
	if !ok {
		t.Fatalf("expected lowered match expr, got %T", letStmt.Init)
	}
	if matchExpr.Type.String() != string(ast.TypeString) {
		t.Fatalf("expected match expr type string, got %s", matchExpr.Type)
	}
	callStmt := mainFn.Body.Stmts[2].(*ExprStmt)
	call, ok := callStmt.Expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected lowered call expr, got %T", callStmt.Expr)
	}
	if call.Func != "println" {
		t.Fatalf("expected call func println, got %s", call.Func)
	}
	nested, ok := call.Args[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected nested lowered call expr, got %T", call.Args[0])
	}
	if nested.Func != "User_label" {
		t.Fatalf("expected resolved func call, got %s", nested.Func)
	}
}

func TestLowerRejectsUnresolvedCallExpression(t *testing.T) {
	prog := &ast.Program{
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name:       "main",
				ReturnType: ast.TypeVoid,
				Body: &ast.BlockStmt{
					Stmts: []ast.Stmt{
						&ast.ExprStmt{
							Expr: &ast.CallExpr{
								Receiver: &ast.IdentExpr{Name: "u"},
								Method:   "label",
								Args:     []ast.Expr{},
							},
						},
					},
				},
			},
		},
	}
	_, err := Lower(prog)
	if err == nil {
		t.Fatalf("expected unresolved call rejection")
	}
	if !strings.Contains(err.Error(), "unresolved call expression") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func parseAndCheck(t *testing.T, src string) *ast.Program {
	t.Helper()
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := sema.New().Check(prog); err != nil {
		t.Fatalf("check: %v", err)
	}
	return prog
}
