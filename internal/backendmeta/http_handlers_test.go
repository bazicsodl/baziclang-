package backendmeta

import (
	"reflect"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

func TestParseHTTPHandlerRecognizesMIRRouteNamingConvention(t *testing.T) {
	fn := &mir.FuncDecl{
		Name:       "get_users_p_id_profile",
		Params:     []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("pkg__ServerRequest"))}},
		ReturnType: baztypes.MustParse(ast.Type("pkg__ServerResponse")),
	}
	got, ok := ParseHTTPHandler(fn)
	if !ok {
		t.Fatalf("expected handler to parse")
	}
	want := intrinsics.HTTPHandlerSpec{
		Method:   "GET",
		FuncName: "get_users_p_id_profile",
		Segments: []intrinsics.HTTPRouteSegmentSpec{
			{Literal: "users"},
			{Param: "id", IsParam: true},
			{Literal: "profile"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parsed handler:\n got: %#v\nwant: %#v", got, want)
	}
	if route := intrinsics.HTTPRoutePattern(got); route != "/users/:id/profile" {
		t.Fatalf("unexpected route pattern %q", route)
	}
}

func TestCollectHTTPHandlersSortsDeterministicallyFromMIR(t *testing.T) {
	mp := &mir.Program{Decls: []mir.Decl{
		&mir.FuncDecl{Name: "post_root", Params: []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("ServerRequest"))}}, ReturnType: baztypes.MustParse(ast.Type("ServerResponse"))},
		&mir.FuncDecl{Name: "get_users", Params: []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("ServerRequest"))}}, ReturnType: baztypes.MustParse(ast.Type("ServerResponse"))},
		&mir.FuncDecl{Name: "get_root", Params: []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("ServerRequest"))}}, ReturnType: baztypes.MustParse(ast.Type("ServerResponse"))},
	}}
	got := CollectHTTPHandlers(mp)
	if len(got) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(got))
	}
	names := []string{got[0].FuncName, got[1].FuncName, got[2].FuncName}
	want := []string{"get_root", "get_users", "post_root"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("unexpected sorted handlers: got %v want %v", names, want)
	}
}
