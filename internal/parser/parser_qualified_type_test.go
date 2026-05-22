package parser

import (
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
)

func TestParseQualifiedImportedTypesAndStructLiterals(t *testing.T) {
	src := `fn main(): void {
    let v: util.Visible = util.Visible { value: 1 };
    let b: util.Box[int] = util.Box[int] { value: 2 };
}`
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := prog.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected first decl to be function, got %T", prog.Decls[0])
	}
	if len(fn.Body.Stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(fn.Body.Stmts))
	}
	first, ok := fn.Body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected first stmt to be let, got %T", fn.Body.Stmts[0])
	}
	if first.Type != ast.Type("util.Visible") {
		t.Fatalf("expected qualified type util.Visible, got %s", first.Type)
	}
	firstLit, ok := first.Init.(*ast.StructLitExpr)
	if !ok {
		t.Fatalf("expected qualified struct literal, got %T", first.Init)
	}
	if firstLit.TypeName != "util.Visible" {
		t.Fatalf("expected qualified struct literal type util.Visible, got %s", firstLit.TypeName)
	}
	second, ok := fn.Body.Stmts[1].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected second stmt to be let, got %T", fn.Body.Stmts[1])
	}
	if second.Type != ast.Type("util.Box[int]") {
		t.Fatalf("expected qualified generic type util.Box[int], got %s", second.Type)
	}
	secondLit, ok := second.Init.(*ast.StructLitExpr)
	if !ok {
		t.Fatalf("expected qualified generic struct literal, got %T", second.Init)
	}
	if secondLit.TypeName != "util.Box[int]" {
		t.Fatalf("expected qualified generic struct literal type util.Box[int], got %s", secondLit.TypeName)
	}
}

func TestParseImportAlias(t *testing.T) {
	src := `package main;
import "util" as tools;
fn main(): void { println(tools.answer); }`
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	imp, ok := prog.Decls[0].(*ast.ImportDecl)
	if !ok {
		t.Fatalf("expected first decl to be import, got %T", prog.Decls[0])
	}
	if imp.Path != "util" {
		t.Fatalf("expected import path util, got %q", imp.Path)
	}
	if imp.Alias != "tools" {
		t.Fatalf("expected import alias tools, got %q", imp.Alias)
	}
	if !imp.ExplicitAlias {
		t.Fatalf("expected import alias to be marked explicit")
	}
}

func TestParseQualifiedMatchVariants(t *testing.T) {
	src := `fn main(): void {
    let label = match util.Admin {
        util.Guest: "guest",
        util.Admin: "admin",
    };
    println(label);
}`
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := prog.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected first decl to be function, got %T", prog.Decls[0])
	}
	letStmt, ok := fn.Body.Stmts[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("expected first stmt to be let, got %T", fn.Body.Stmts[0])
	}
	matchExpr, ok := letStmt.Init.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("expected let init to be match expr, got %T", letStmt.Init)
	}
	if len(matchExpr.Arms) != 2 {
		t.Fatalf("expected 2 match arms, got %d", len(matchExpr.Arms))
	}
	if matchExpr.Arms[0].Variant != "util.Guest" || matchExpr.Arms[1].Variant != "util.Admin" {
		t.Fatalf("unexpected qualified match variants: %+v", matchExpr.Arms)
	}
}
