package backendmeta

import (
	"testing"

	"baziclang/internal/mir"
)

func TestResolveProgramTypeNameUsesMIRDeclNames(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "pkg__HttpResponse"},
			&mir.InterfaceDecl{Name: "pkg__ServerRequest"},
			&mir.EnumDecl{Name: "pkg__Role"},
		},
	}
	if got := ResolveProgramTypeName(mp, "HttpResponse"); got != "pkg__HttpResponse" {
		t.Fatalf("expected internalized struct name, got %q", got)
	}
	if got := ResolveProgramTypeName(mp, "ServerRequest"); got != "pkg__ServerRequest" {
		t.Fatalf("expected internalized interface name, got %q", got)
	}
	if got := ResolveProgramTypeName(mp, "Missing"); got != "Missing" {
		t.Fatalf("expected missing type to remain unchanged, got %q", got)
	}
}

func TestHasProgramTypeNameMatchesExactAndSuffixedNames(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "HttpResponse"},
			&mir.EnumDecl{Name: "pkg__ServerResponse"},
		},
	}
	if !HasProgramTypeName(mp, "HttpResponse") {
		t.Fatalf("expected exact name match")
	}
	if !HasProgramTypeName(mp, "ServerResponse") {
		t.Fatalf("expected suffixed name match")
	}
	if HasProgramTypeName(mp, "Missing") {
		t.Fatalf("expected missing name to return false")
	}
}
