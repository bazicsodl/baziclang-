package parser

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
	"baziclang/internal/source"
)

func TestParseProgramPopulatesNodeSpans(t *testing.T) {
	src := "fn main() { let x = 1 + 2; println(x) }"

	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	checkSpan(t, prog.Span(), 0, len(src))

	fn, ok := prog.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected func decl, got %T", prog.Decls[0])
	}
	checkSpan(t, fn.Span(), 0, len(src))

	checkSpan(t, fn.Body.Span(), strings.Index(src, "{"), len(src))

	letStmt, ok := fn.Body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected let stmt, got %T", fn.Body.Stmts[0])
	}
	checkSpan(t, letStmt.Span(), strings.Index(src, "let"), strings.Index(src, "2")+1)

	binary, ok := letStmt.Init.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected binary expr, got %T", letStmt.Init)
	}
	checkSpan(t, binary.Span(), strings.Index(src, "1"), strings.Index(src, "2")+1)

	exprStmt, ok := fn.Body.Stmts[1].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected expr stmt, got %T", fn.Body.Stmts[1])
	}
	call, ok := exprStmt.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", exprStmt.Expr)
	}
	callStart := strings.Index(src, "println")
	callEnd := callStart + len("println(x)")
	checkSpan(t, call.Span(), callStart, callEnd)
}

func TestParseProgramCapturesPackageDecl(t *testing.T) {
	src := "package main;\nfn main() { }"
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if prog.Package == nil {
		t.Fatalf("expected package declaration")
	}
	if prog.Package.Name != "main" {
		t.Fatalf("unexpected package name: %q", prog.Package.Name)
	}
	checkSpan(t, prog.Package.Span(), 0, len("package main"))
}

func checkSpan(t *testing.T, got source.Span, wantStart, wantEnd int) {
	t.Helper()
	if got.Start.Offset != wantStart || got.End.Offset != wantEnd {
		t.Fatalf("unexpected span offsets: got [%d,%d), want [%d,%d)", got.Start.Offset, got.End.Offset, wantStart, wantEnd)
	}
	if got.Start.Line != 1 || got.End.Line != 1 {
		t.Fatalf("unexpected span lines: got start=%d end=%d", got.Start.Line, got.End.Line)
	}
	if got.Start.Col != wantStart+1 || got.End.Col != wantEnd+1 {
		t.Fatalf("unexpected span cols: got start=%d end=%d, want start=%d end=%d", got.Start.Col, got.End.Col, wantStart+1, wantEnd+1)
	}
}
