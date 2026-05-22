package intrinsics

import (
	"reflect"
	"testing"

	"baziclang/internal/ast"
)

func TestParseHTTPHandlerRecognizesRouteNamingConvention(t *testing.T) {
	fn := &ast.FuncDecl{
		Name:       "get_users_p_id_profile",
		Params:     []ast.Param{{Name: "req", Type: ast.Type("pkg__ServerRequest")}},
		ReturnType: ast.Type("pkg__ServerResponse"),
	}
	got, ok := ParseHTTPHandler(fn)
	if !ok {
		t.Fatalf("expected handler to parse")
	}
	want := HTTPHandlerSpec{
		Method:   "GET",
		FuncName: "get_users_p_id_profile",
		Segments: []HTTPRouteSegmentSpec{
			{Literal: "users"},
			{Param: "id", IsParam: true},
			{Literal: "profile"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed handler:\n got: %#v\nwant: %#v", got, want)
	}
	if route := HTTPRoutePattern(got); route != "/users/:id/profile" {
		t.Fatalf("unexpected route pattern %q", route)
	}
}

func TestCollectHTTPHandlersSortsDeterministically(t *testing.T) {
	prog := &ast.Program{Decls: []ast.Decl{
		&ast.FuncDecl{Name: "post_root", Params: []ast.Param{{Name: "req", Type: ast.Type("ServerRequest")}}, ReturnType: ast.Type("ServerResponse")},
		&ast.FuncDecl{Name: "get_users", Params: []ast.Param{{Name: "req", Type: ast.Type("ServerRequest")}}, ReturnType: ast.Type("ServerResponse")},
		&ast.FuncDecl{Name: "get_root", Params: []ast.Param{{Name: "req", Type: ast.Type("ServerRequest")}}, ReturnType: ast.Type("ServerResponse")},
	}}
	got := CollectHTTPHandlers(prog)
	if len(got) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(got))
	}
	names := []string{got[0].FuncName, got[1].FuncName, got[2].FuncName}
	want := []string{"get_root", "get_users", "post_root"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("unexpected sorted handlers: got %v want %v", names, want)
	}
}

func TestResolveProgramTypeNameUsesInternalNameWhenPresent(t *testing.T) {
	prog := &ast.Program{Decls: []ast.Decl{
		&ast.StructDecl{Name: "HttpResponse", InternalName: "pkg__HttpResponse"},
	}}
	if got := ResolveProgramTypeName(prog, "HttpResponse"); got != "pkg__HttpResponse" {
		t.Fatalf("expected internal name, got %q", got)
	}
	if got := ResolveProgramTypeName(prog, "Missing"); got != "Missing" {
		t.Fatalf("expected missing type name to stay unchanged, got %q", got)
	}
}
