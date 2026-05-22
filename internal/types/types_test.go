package types

import (
	"testing"

	"baziclang/internal/ast"
)

func TestParseNestedGenericType(t *testing.T) {
	got, err := Parse(ast.Type("Result[Option[int],Error]"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got.Name != "Result" || len(got.Args) != 2 {
		t.Fatalf("unexpected parse result: %#v", got)
	}
	if got.Args[0].Name != "Option" || len(got.Args[0].Args) != 1 || got.Args[0].Args[0].Name != "int" {
		t.Fatalf("unexpected nested arg parse: %#v", got.Args[0])
	}
	if ToAST(got) != ast.Type("Result[Option[int],Error]") {
		t.Fatalf("round trip mismatch: %s", ToAST(got))
	}
}

func TestSubstituteAndUnify(t *testing.T) {
	mapping := map[string]ast.Type{}
	if !Unify(ast.Type("Result[T,Error]"), ast.Type("Result[string,Error]"), mapping, []string{"T"}) {
		t.Fatalf("expected unify to succeed")
	}
	if mapping["T"] != ast.TypeString {
		t.Fatalf("expected T=string, got %s", mapping["T"])
	}
	subst := Substitute(ast.Type("Option[T]"), mapping)
	if subst != ast.Type("Option[string]") {
		t.Fatalf("unexpected substitution: %s", subst)
	}
	if !ContainsParam(ast.Type("Result[Option[T],Error]"), "T") {
		t.Fatalf("expected contains param")
	}
}
