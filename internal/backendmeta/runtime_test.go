package backendmeta

import (
	"reflect"
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

func TestCollectProgramRuntimeMetaAggregatesTypesHandlersAndRuntimeCalls(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "pkg__HttpResponse"},
			&mir.StructDecl{Name: "pkg__ServerRequest"},
			&mir.StructDecl{Name: "pkg__ServerResponse"},
			&mir.FuncDecl{
				Name:       "get_root",
				Params:     []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("pkg__ServerRequest"))}},
				ReturnType: baztypes.MustParse(ast.Type("pkg__ServerResponse")),
				Body: &mir.Block{Stmts: []mir.Stmt{
					&mir.CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "db_exec"},
					&mir.CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "jwt_sign_hs256"},
					&mir.CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "auth_hash_password"},
				}},
			},
		},
	}
	got := CollectProgramRuntimeMeta(mp)
	if !got.Capabilities.HasHTTPResponse || got.Types.HTTPResponseType != "pkg__HttpResponse" || got.Types.ServerRequestType != "pkg__ServerRequest" || got.Types.ServerResponseType != "pkg__ServerResponse" {
		t.Fatalf("unexpected program type metadata: %+v", got)
	}
	if !got.Capabilities.HasHTTPHandlers || !got.Capabilities.HasHTTPResponse || got.Capabilities.HasErrorType {
		t.Fatalf("unexpected runtime capabilities: %+v", got.Capabilities)
	}
	if !got.Capabilities.NeedsDB || !got.Capabilities.NeedsJWT || !got.Capabilities.NeedsHMAC || !got.Capabilities.NeedsBcrypt || got.Capabilities.NeedsSession {
		t.Fatalf("unexpected runtime capability flags: %+v", got.Capabilities)
	}
	for _, want := range []RuntimeFeature{
		RuntimeFeatureGoCore,
		RuntimeFeatureGoStdRuntime,
		RuntimeFeatureGoJSONWeb,
		RuntimeFeatureGoHTTP,
		RuntimeFeatureGoCrypto,
		RuntimeFeatureGoDB,
		RuntimeFeatureGoImportHMAC,
		RuntimeFeatureGoImportBcrypt,
		RuntimeFeatureLLVMStringGlobals,
		RuntimeFeatureLLVMRouteTable,
		RuntimeFeatureLLVMStringRuntime,
		RuntimeFeatureLLVMBuiltin,
		RuntimeFeatureLLVMAnyRuntime,
		RuntimeFeatureLLVMStdDecls,
	} {
		if !HasRuntimeFeature(got.Features, want) {
			t.Fatalf("expected runtime feature %q in %#v", want, got.Features)
		}
	}
	if HasRuntimeFeature(got.Features, RuntimeFeatureGoSession) || HasRuntimeFeature(got.Features, RuntimeFeatureGoImportSync) || HasRuntimeFeature(got.Features, RuntimeFeatureLLVMParseInt) {
		t.Fatalf("did not expect session/sync/parse runtime features in %#v", got.Features)
	}
	if names := []string{got.Routes.Handlers[0].FuncName}; !reflect.DeepEqual(names, []string{"get_root"}) {
		t.Fatalf("unexpected handlers: %+v", got.Routes.Handlers)
	}
	if !reflect.DeepEqual(got.Routes.RouteStrings, []string{"GET", "/"}) {
		t.Fatalf("unexpected route strings: %#v", got.Routes.RouteStrings)
	}
}

func TestProgramShapeHelpersUseSharedRuntimeMetadata(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "pkg__HttpResponse"},
			&mir.StructDecl{Name: "pkg__ServerRequest"},
			&mir.StructDecl{Name: "pkg__ServerResponse"},
			&mir.FuncDecl{
				Name:       "get_root",
				Params:     []mir.Param{{Name: "req", Type: baztypes.MustParse(ast.Type("pkg__ServerRequest"))}},
				ReturnType: baztypes.MustParse(ast.Type("pkg__ServerResponse")),
			},
			&mir.FuncDecl{
				Name:       "main",
				Params:     []mir.Param{{Name: "n", Type: baztypes.MustParse(ast.TypeInt)}},
				ReturnType: baztypes.MustParse(ast.TypeInt),
			},
			&mir.FuncDecl{
				Name:       "main",
				Params:     []mir.Param{{Name: "s", Type: baztypes.MustParse(ast.TypeString)}},
				ReturnType: baztypes.MustParse(ast.TypeString),
			},
		},
	}
	meta := CollectProgramRuntimeMeta(mp)

	runtimeSurface := CollectLLVMRuntimeSurfaceMeta(meta)
	if got := runtimeSurface.TypeAliases; !reflect.DeepEqual(got, map[string]ast.Type{
		"HttpResponse":   ast.Type("pkg__HttpResponse"),
		"ServerRequest":  ast.Type("pkg__ServerRequest"),
		"ServerResponse": ast.Type("pkg__ServerResponse"),
	}) {
		t.Fatalf("unexpected runtime type aliases: %#v", got)
	}

	if got := RuntimeRouteStrings(meta.Routes.Handlers); !reflect.DeepEqual(got, []string{"GET", "/"}) {
		t.Fatalf("unexpected runtime route strings: %#v", got)
	}
	if !reflect.DeepEqual(meta.Routes.RouteStrings, []string{"GET", "/"}) {
		t.Fatalf("unexpected collected runtime route strings: %#v", meta.Routes.RouteStrings)
	}

	ordered := CollectOrderedFuncs(mp)
	if names := []string{ordered[0].Name, ordered[1].Name}; !reflect.DeepEqual(names, []string{"get_root", "main"}) {
		t.Fatalf("unexpected ordered funcs: %#v", names)
	}
	if baztypes.ToAST(ordered[1].Params[0].Type) != ast.TypeString {
		t.Fatalf("expected last duplicate function to win, got %#v", ordered[1].Params)
	}

	if sig := CollectProgramFuncSigs(mp)["main"]; !reflect.DeepEqual(sig, FuncSig{
		Params: []ast.Type{ast.TypeString},
		Ret:    ast.TypeString,
	}) {
		t.Fatalf("unexpected program func sig: %#v", sig)
	}

	runtimeSigs := runtimeSurface.FuncSigs
	if sig, ok := runtimeSigs["__std_http_get_opts_resp"]; !ok || sig.Ret != ast.Type("Result__pkg_HttpResponse__Error") {
		t.Fatalf("unexpected runtime intrinsic sig: %#v", sig)
	}

	shape := CollectProgramShapeMeta(mp)
	if !reflect.DeepEqual(shape.Runtime, meta) {
		t.Fatalf("unexpected runtime metadata in shape: %#v", shape.Runtime)
	}
	if !reflect.DeepEqual(shape.Runtime.Features, CollectRuntimeFeatureSurfaceMeta(meta.Capabilities)) {
		t.Fatalf("unexpected runtime feature surface in shape runtime: %#v", shape.Runtime.Features)
	}
	if !reflect.DeepEqual(shape.RuntimeShape, CollectRuntimeShapeMeta(meta)) {
		t.Fatalf("unexpected runtime shape metadata in shape: %#v", shape.RuntimeShape)
	}
	if shape.RuntimeShape.LLVMRuntimeSurface.HasParseIntRuntime || shape.RuntimeShape.LLVMRuntimeSurface.HasParseFloatRuntime {
		t.Fatalf("did not expect parse runtime capability without Error struct: %#v", shape.RuntimeShape.LLVMRuntimeSurface)
	}
	if !shape.RuntimeShape.LLVMRuntimeSurface.HasStringGlobals {
		t.Fatalf("expected shared llvm runtime surface to include string globals: %#v", shape.RuntimeShape.LLVMRuntimeSurface)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.LLVMRuntimeSurface.PreludeSections, intrinsics.EnabledLLVMRuntimePreludeSections(true)) {
		t.Fatalf("unexpected llvm runtime prelude sections in shape: %#v", shape.RuntimeShape.LLVMRuntimeSurface.PreludeSections)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.LLVMRuntimeSurface.BuiltinSections, intrinsics.EnabledLLVMBuiltinRuntimeSections(false, false)) {
		t.Fatalf("unexpected llvm builtin runtime sections in shape: %#v", shape.RuntimeShape.LLVMRuntimeSurface.BuiltinSections)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.LLVMRuntimeSurface.TypeAliases, runtimeSurface.TypeAliases) {
		t.Fatalf("unexpected type aliases in shape: %#v", shape.RuntimeShape.LLVMRuntimeSurface.TypeAliases)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.RouteStrings, meta.Routes.RouteStrings) {
		t.Fatalf("unexpected route strings in shape: %#v", shape.RuntimeShape.RouteStrings)
	}
	if len(shape.Globals) != 0 {
		t.Fatalf("unexpected globals in shape: %#v", shape.Globals)
	}
	if len(shape.OrderedFuncs) != len(ordered) || shape.OrderedFuncs[1].Name != "main" {
		t.Fatalf("unexpected ordered funcs in shape: %#v", shape.OrderedFuncs)
	}
	if names := []string{shape.StructNodes[0].Name, shape.StructNodes[1].Name, shape.StructNodes[2].Name}; !reflect.DeepEqual(names, []string{"pkg__HttpResponse", "pkg__ServerRequest", "pkg__ServerResponse"}) {
		t.Fatalf("unexpected struct nodes in shape: %#v", shape.StructNodes)
	}
	if len(shape.InterfaceNodes) != 0 || len(shape.EnumNodes) != 0 || len(shape.GlobalNodes) != 0 {
		t.Fatalf("unexpected decl nodes in shape: interfaces=%#v enums=%#v globals=%#v", shape.InterfaceNodes, shape.EnumNodes, shape.GlobalNodes)
	}
	if !reflect.DeepEqual(shape.ProgramFuncSigs["main"], FuncSig{Params: []ast.Type{ast.TypeString}, Ret: ast.TypeString}) {
		t.Fatalf("unexpected program sigs in shape: %#v", shape.ProgramFuncSigs)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.LLVMRuntimeSurface.FuncSigs, runtimeSigs) {
		t.Fatalf("unexpected runtime sigs in shape: %#v", shape.RuntimeShape.LLVMRuntimeSurface.FuncSigs)
	}
	if !reflect.DeepEqual(shape.RuntimeShape.LLVMRuntimeSurface.PreludeDecls, intrinsics.LLVMRuntimePreludeDecls()) {
		t.Fatalf("unexpected runtime prelude decls in shape: %#v", shape.RuntimeShape.LLVMRuntimeSurface.PreludeDecls)
	}
	if !reflect.DeepEqual(shape.Enums, []EnumDecl{}) {
		t.Fatalf("unexpected enums in shape: %#v", shape.Enums)
	}
	if !reflect.DeepEqual(shape.Interfaces, []InterfaceDecl{}) {
		t.Fatalf("unexpected interfaces in shape: %#v", shape.Interfaces)
	}
	if !reflect.DeepEqual(shape.Structs, []StructDecl{
		{Name: "pkg__HttpResponse", Fields: []StructField{}},
		{Name: "pkg__ServerRequest", Fields: []StructField{}},
		{Name: "pkg__ServerResponse", Fields: []StructField{}},
	}) {
		t.Fatalf("unexpected structs in shape: %#v", shape.Structs)
	}
}

func TestCollectLLVMRuntimeSurfaceMetaTracksSharedCapabilitySelection(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "Error"},
			&mir.FuncDecl{Name: "main"},
		},
	}
	got := CollectLLVMRuntimeSurfaceMeta(CollectProgramRuntimeMeta(mp))
	if !got.HasStringGlobals || !got.HasParseIntRuntime || !got.HasParseFloatRuntime {
		t.Fatalf("unexpected llvm runtime surface capabilities: %#v", got)
	}
	if !reflect.DeepEqual(got.PreludeSections, intrinsics.EnabledLLVMRuntimePreludeSections(true)) {
		t.Fatalf("unexpected llvm prelude sections: %#v", got.PreludeSections)
	}
	if !reflect.DeepEqual(got.BuiltinSections, intrinsics.EnabledLLVMBuiltinRuntimeSections(true, true)) {
		t.Fatalf("unexpected llvm builtin sections: %#v", got.BuiltinSections)
	}
	if !reflect.DeepEqual(got.PreludeSections, collectLLVMRuntimePreludeSections(CollectProgramRuntimeMeta(mp).Features)) {
		t.Fatalf("expected llvm prelude sections to derive from shared feature surface: %#v", got.PreludeSections)
	}
	if !reflect.DeepEqual(got.BuiltinSections, collectLLVMBuiltinRuntimeSections(CollectProgramRuntimeMeta(mp).Features)) {
		t.Fatalf("expected llvm builtin sections to derive from shared feature surface: %#v", got.BuiltinSections)
	}
}

func TestCollectRuntimeFeatureSurfaceMetaTracksSharedBackendFeatureSelection(t *testing.T) {
	got := CollectRuntimeFeatureSurfaceMeta(RuntimeCapabilityMeta{
		HasErrorType:      true,
		NeedsSession:      true,
		NeedsDB:           true,
		NeedsHTTPServeApp: true,
	})
	for _, want := range []RuntimeFeature{
		RuntimeFeatureGoCore,
		RuntimeFeatureGoStdRuntime,
		RuntimeFeatureGoJSONWeb,
		RuntimeFeatureGoHTTP,
		RuntimeFeatureGoCrypto,
		RuntimeFeatureGoHTTPServe,
		RuntimeFeatureGoSession,
		RuntimeFeatureGoDB,
		RuntimeFeatureGoImportSync,
		RuntimeFeatureLLVMStringGlobals,
		RuntimeFeatureLLVMRouteTable,
		RuntimeFeatureLLVMStringRuntime,
		RuntimeFeatureLLVMBuiltin,
		RuntimeFeatureLLVMAnyRuntime,
		RuntimeFeatureLLVMStdDecls,
		RuntimeFeatureLLVMParseInt,
		RuntimeFeatureLLVMParseFloat,
	} {
		if !HasRuntimeFeature(got, want) {
			t.Fatalf("expected runtime feature %q in %#v", want, got)
		}
	}
	if HasRuntimeFeature(got, RuntimeFeatureGoImportHMAC) || HasRuntimeFeature(got, RuntimeFeatureGoImportBcrypt) {
		t.Fatalf("did not expect hmac/bcrypt import features in %#v", got)
	}
	if !reflect.DeepEqual(collectLLVMRuntimePreludeSections(got), []intrinsics.LLVMRuntimePreludeSection{
		intrinsics.LLVMRuntimePreludeStringGlobals,
		intrinsics.LLVMRuntimePreludeRouteTable,
		intrinsics.LLVMRuntimePreludeStringRuntime,
		intrinsics.LLVMRuntimePreludeBuiltin,
		intrinsics.LLVMRuntimePreludeAnyRuntime,
		intrinsics.LLVMRuntimePreludeStdDecls,
	}) {
		t.Fatalf("unexpected llvm prelude projection from shared runtime features: %#v", collectLLVMRuntimePreludeSections(got))
	}
	if !reflect.DeepEqual(collectLLVMBuiltinRuntimeSections(got), []intrinsics.LLVMBuiltinRuntimeSection{
		intrinsics.LLVMBuiltinRuntimeContains,
		intrinsics.LLVMBuiltinRuntimeStartsWith,
		intrinsics.LLVMBuiltinRuntimeEndsWith,
		intrinsics.LLVMBuiltinRuntimeToUpper,
		intrinsics.LLVMBuiltinRuntimeToLower,
		intrinsics.LLVMBuiltinRuntimeTrimSpace,
		intrinsics.LLVMBuiltinRuntimeRepeat,
		intrinsics.LLVMBuiltinRuntimeReplace,
		intrinsics.LLVMBuiltinRuntimeIntToStr,
		intrinsics.LLVMBuiltinRuntimeFloatToStr,
		intrinsics.LLVMBuiltinRuntimeParseInt,
		intrinsics.LLVMBuiltinRuntimeParseFloat,
	}) {
		t.Fatalf("unexpected llvm builtin projection from shared runtime features: %#v", collectLLVMBuiltinRuntimeSections(got))
	}
}

func TestBuildGoRuntimePlanAggregatesImportsPreludeAndHelpers(t *testing.T) {
	meta := ProgramRuntimeMeta{
		Capabilities: RuntimeCapabilityMeta{
			HasHTTPHandlers:   true,
			HasHTTPResponse:   true,
			NeedsSession:      true,
			NeedsDB:           true,
			NeedsBcrypt:       true,
			NeedsJWT:          true,
			NeedsHMAC:         true,
			NeedsHTTPServeApp: true,
			NeedsHeaderString: true,
		},
		Types: RuntimeTypeMeta{
			HTTPResponseType:   "pkg__HttpResponse",
			ServerRequestType:  "pkg__ServerRequest",
			ServerResponseType: "pkg__ServerResponse",
		},
		Routes: RuntimeRouteMeta{
			Handlers:     []intrinsics.HTTPHandlerSpec{{Method: "GET", FuncName: "get_root"}},
			RouteStrings: []string{"GET", "/"},
		},
	}
	meta.Features = CollectRuntimeFeatureSurfaceMeta(meta.Capabilities)

	surface := CollectGoRuntimeSurfaceMeta(meta, "  WASM ")
	if surface.Target != "wasm" {
		t.Fatalf("unexpected normalized go runtime target: %#v", surface)
	}
	if !surface.HasSessionPrelude {
		t.Fatalf("expected session prelude capability in go surface: %#v", surface)
	}
	for _, want := range []GoRuntimeHelperSection{
		GoRuntimeHelperCore,
		GoRuntimeHelperStdRuntime,
		GoRuntimeHelperJSONWeb,
		GoRuntimeHelperHTTP,
		GoRuntimeHelperCrypto,
		GoRuntimeHelperHTTPServe,
		GoRuntimeHelperSession,
		GoRuntimeHelperDB,
	} {
		if !containsHelperSection(surface.HelperSections, want) {
			t.Fatalf("expected helper section %q in %#v", want, surface.HelperSections)
		}
	}
	if !reflect.DeepEqual(surface.HelperSections, collectGoRuntimeHelperSections(meta.Features)) {
		t.Fatalf("expected go helper sections to derive from shared feature surface: %#v", surface.HelperSections)
	}
	for _, want := range []string{
		"\"crypto/hmac\"",
		"\"golang.org/x/crypto/bcrypt\"",
		"\"sync\"",
		"\"syscall/js\"",
		"\"database/sql\"",
	} {
		if !containsString(surface.Imports, want) {
			t.Fatalf("expected import %q in %#v", want, surface.Imports)
		}
	}
	if !reflect.DeepEqual(surface.Imports, collectGoRuntimeImports(meta.Features, "wasm")) {
		t.Fatalf("expected go imports to derive from shared feature surface: %#v", surface.Imports)
	}
	if containsString(surface.Imports, "_ \"github.com/go-sql-driver/mysql\"") {
		t.Fatalf("did not expect host db driver imports in wasm surface: %#v", surface.Imports)
	}
	if !strings.Contains(surface.SessionPrelude, "__bazic_session_store") {
		t.Fatalf("expected session prelude directly on go surface, got %q", surface.SessionPrelude)
	}
	surfaceJoined := strings.Join(surface.HelperSnippets, "\n")
	if !reflect.DeepEqual(surface.HelperSnippets, collectGoRuntimeHelperSnippets(meta, "wasm", surface.HelperSections)) {
		t.Fatalf("expected go helper snippets to derive from shared helper registry: %#v", surface.HelperSnippets)
	}
	for _, want := range []string{
		"func __std_http_serve_app(",
		"func __std_session_init(",
		"func __std_db_exec(",
		"func __std_bcrypt_hash(",
		"func __std_jwt_sign_hs256(",
		"func headerString(",
	} {
		if !strings.Contains(surfaceJoined, want) {
			t.Fatalf("expected go surface helper snippet %q", want)
		}
	}
	if !strings.Contains(surface.Prelude, "package main") || !strings.Contains(surface.Prelude, "func __std_http_serve_app(") {
		t.Fatalf("unexpected rendered prelude on go surface: %q", surface.Prelude)
	}

	wasm := BuildGoRuntimePlan(meta, "  WASM ")
	if !reflect.DeepEqual(wasm, surface) {
		t.Fatalf("expected go runtime plan wrapper to match go runtime surface:\nplan=%#v\nsurface=%#v", wasm, surface)
	}
	if wasm.Target != "wasm" {
		t.Fatalf("unexpected normalized target: %q", wasm.Target)
	}
	for _, want := range []string{
		"\"crypto/hmac\"",
		"\"golang.org/x/crypto/bcrypt\"",
		"\"sync\"",
		"\"syscall/js\"",
		"\"database/sql\"",
	} {
		if !containsString(wasm.Imports, want) {
			t.Fatalf("expected import %q in %#v", want, wasm.Imports)
		}
	}
	if containsString(wasm.Imports, "_ \"github.com/go-sql-driver/mysql\"") {
		t.Fatalf("did not expect host db driver imports in wasm plan: %#v", wasm.Imports)
	}
	if !strings.Contains(wasm.SessionPrelude, "__bazic_session_store") {
		t.Fatalf("expected session prelude, got %q", wasm.SessionPrelude)
	}
	joined := strings.Join(wasm.HelperSnippets, "\n")
	for _, want := range []string{
		"func __std_http_serve_app(",
		"func __std_session_init(",
		"func __std_db_exec(",
		"func __std_bcrypt_hash(",
		"func __std_jwt_sign_hs256(",
		"func headerString(",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected helper snippet %q", want)
		}
	}

	host := BuildGoRuntimePlan(meta, "native")
	for _, want := range []string{
		"_ \"github.com/go-sql-driver/mysql\"",
		"_ \"github.com/lib/pq\"",
		"_ \"modernc.org/sqlite\"",
	} {
		if !containsString(host.Imports, want) {
			t.Fatalf("expected host driver import %q in %#v", want, host.Imports)
		}
	}

	rendered := RenderGoPrelude(host)
	for _, want := range []string{
		"package main\n\nimport (\n",
		"\t\"database/sql\"\n",
		"\t_ \"github.com/go-sql-driver/mysql\"\n",
		"type __bazic_session_entry struct",
		"func __std_http_serve_app(",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered prelude to contain %q:\n%s", want, rendered)
		}
	}

	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.StructDecl{Name: "pkg__HttpResponse"},
			&mir.FuncDecl{Name: "main"},
		},
	}
	goPlan := CollectGoProgramPlan(mp, "WASM")
	if goPlan.Shape.Runtime.Types.HTTPResponseType != "pkg__HttpResponse" {
		t.Fatalf("unexpected go plan shape: %#v", goPlan.Shape)
	}
	if goPlan.RuntimePlan.Target != "wasm" {
		t.Fatalf("unexpected go plan runtime target: %#v", goPlan.RuntimePlan)
	}
	if goPlan.Prelude != goPlan.RuntimePlan.Prelude {
		t.Fatalf("expected go plan prelude to come from shared runtime surface:\nplan=%q\nruntime=%q", goPlan.Prelude, goPlan.RuntimePlan.Prelude)
	}
}

func containsHelperSection(items []GoRuntimeHelperSection, want GoRuntimeHelperSection) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
