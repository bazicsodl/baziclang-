package intrinsics

import (
	"reflect"
	"strings"
	"testing"

	"baziclang/internal/ast"
)

func TestSurfaceFunctionSpecsIncludePreludeAndExcludeInternalIntrinsics(t *testing.T) {
	specs := SurfaceFunctionSpecs()
	seen := map[string]bool{}
	for _, spec := range specs {
		seen[spec.Name] = true
		if strings.HasPrefix(spec.Name, "__std_") {
			t.Fatalf("did not expect internal intrinsic %q in user-visible surface", spec.Name)
		}
	}
	for _, want := range []string{"print", "parse_int", "some", "assert_msg", "unwrap_or"} {
		if !seen[want] {
			t.Fatalf("expected user-visible builtin %q", want)
		}
	}
}

func TestLookupSurfaceFunctionReturnsSignatureBackedHover(t *testing.T) {
	spec, ok := LookupSurfaceFunction("assert_msg")
	if !ok {
		t.Fatalf("expected assert_msg in surface registry")
	}
	hover := spec.Hover()
	if !strings.Contains(hover, "fn assert_msg(bool, string): void") {
		t.Fatalf("expected signature in hover, got %q", hover)
	}
	if !strings.Contains(hover, "custom failure message") {
		t.Fatalf("expected doc text in hover, got %q", hover)
	}
}

func TestRuntimeCallFamiliesRecognizeWrapperAndIntrinsicNames(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{name: "db_exec", ok: IsDBCall("db_exec")},
		{name: "__std_db_query", ok: IsDBCall("__std_db_query")},
		{name: "session_put", ok: IsSessionCall("session_put")},
		{name: "__std_session_delete", ok: IsSessionCall("__std_session_delete")},
		{name: "crypto_hmac_sha256_hex", ok: IsHMACCall("crypto_hmac_sha256_hex")},
		{name: "jwt_sign_hs256", ok: IsJWTCall("jwt_sign_hs256")},
		{name: "http_serve_app", ok: IsHTTPServeAppCall("http_serve_app")},
		{name: "auth_hash_password", ok: IsBcryptCall("auth_hash_password")},
	}
	for _, tc := range tests {
		if !tc.ok {
			t.Fatalf("expected call family registry to recognize %q", tc.name)
		}
	}
	if IsDBCall("print") {
		t.Fatalf("did not expect unrelated builtin to be classified as db call")
	}
}

func TestIntrinsicTargetNameCanonicalizesWrapperChains(t *testing.T) {
	tests := map[string]string{
		"db_exec":                "__std_db_exec",
		"session_put":            "__std_session_put",
		"jwt_sign_hs256":         "__std_jwt_sign_hs256",
		"crypto_hmac_sha256_hex": "__std_hmac_sha256_hex",
		"auth_hash_password":     "__std_bcrypt_hash",
		"web_get_json":           "__std_web_get_json",
	}
	for name, want := range tests {
		got, ok := IntrinsicTargetName(name)
		if !ok {
			t.Fatalf("expected intrinsic target for %q", name)
		}
		if got != want {
			t.Fatalf("expected %q -> %q, got %q", name, want, got)
		}
	}
	if got := CanonicalRuntimeCallName("print"); got != "print" {
		t.Fatalf("expected unrelated call name to stay unchanged, got %q", got)
	}
	if _, ok := IntrinsicTargetName("print"); ok {
		t.Fatalf("did not expect intrinsic target for plain builtin")
	}
}

func TestLookupLoweredBuiltinProvidesSharedBackendMetadata(t *testing.T) {
	lenSpec, ok := LookupLoweredBuiltin("len")
	if !ok {
		t.Fatalf("expected lowered builtin metadata for len")
	}
	if lenSpec.GoTarget != LLVMRuntimeLenFunc || lenSpec.LLVMTarget != LLVMRuntimeLenFunc {
		t.Fatalf("unexpected len lowering metadata: %+v", lenSpec)
	}
	if lenSpec.Category != LoweredBuiltinLen || lenSpec.Arity != 1 {
		t.Fatalf("unexpected len lowering shape: %+v", lenSpec)
	}
	parseSpec, ok := LookupLoweredBuiltin("parse_float")
	if !ok {
		t.Fatalf("expected lowered builtin metadata for parse_float")
	}
	if parseSpec.Category != LoweredBuiltinParseString || parseSpec.ParseKind != "float" {
		t.Fatalf("unexpected parse_float lowering metadata: %+v", parseSpec)
	}
	strSpec, ok := LookupLoweredBuiltin("str")
	if !ok {
		t.Fatalf("expected lowered builtin metadata for str")
	}
	if strSpec.Category != LoweredBuiltinStringify || strSpec.ReturnType != ast.TypeString {
		t.Fatalf("unexpected str lowering metadata: %+v", strSpec)
	}
	if !IsBuiltinVoidCall("println") || IsBuiltinVoidCall("len") {
		t.Fatalf("unexpected builtin void call classification")
	}
}

func TestLLVMPrintfFormatTracksSharedBuiltinPrintRules(t *testing.T) {
	tests := []struct {
		name      string
		isPrintln bool
		argType   ast.Type
		isEnum    bool
		want      string
	}{
		{name: "int", argType: ast.TypeInt, want: "%ld"},
		{name: "enum", argType: ast.Type("Role"), isEnum: true, want: "%ld"},
		{name: "float", argType: ast.TypeFloat, want: "%g"},
		{name: "string", argType: ast.TypeString, want: "%s"},
		{name: "bool_newline", isPrintln: true, argType: ast.TypeBool, want: "%s\n"},
		{name: "any", argType: ast.TypeAny, want: "%s"},
	}
	for _, tc := range tests {
		got, ok := LLVMPrintfFormat(tc.isPrintln, tc.argType, tc.isEnum)
		if !ok {
			t.Fatalf("expected printf format for %s", tc.name)
		}
		if got != tc.want {
			t.Fatalf("expected %s -> %q, got %q", tc.name, tc.want, got)
		}
	}
	if _, ok := LLVMPrintfFormat(false, ast.TypeVoid, false); ok {
		t.Fatalf("did not expect printf format for void")
	}
}

func TestLLVMRuntimePreludeDeclsTrackSharedRuntimeSurface(t *testing.T) {
	decls := strings.Join(LLVMRuntimePreludeDecls(), "")
	for _, want := range []string{
		"declare i32 @printf(ptr, ...)\n",
		"declare i64 @strlen(ptr)\n",
		"declare i64 @" + LLVMRuntimeLenFunc + "(ptr)\n",
		"declare i32 @strcmp(ptr, ptr)\n",
		"declare ptr @malloc(i64)\n",
	} {
		if !strings.Contains(decls, want) {
			t.Fatalf("expected runtime prelude declarations to include %q, got:\n%s", want, decls)
		}
	}
}

func TestBuildRuntimeInterfaceSurfaceTracksAliasesFunctionsAndPreludeDecls(t *testing.T) {
	surface := BuildRuntimeInterfaceSurface("pkg__HttpResponse", "pkg__ServerRequest", "pkg__ServerResponse")
	if got := surface.TypeAliases["HttpResponse"]; got != ast.Type("pkg__HttpResponse") {
		t.Fatalf("unexpected http response alias: %#v", surface.TypeAliases)
	}
	if len(surface.Functions) == 0 {
		t.Fatalf("expected runtime interface functions")
	}
	found := false
	for _, fn := range surface.Functions {
		if fn.Name == "__std_http_get_opts_resp" {
			found = true
			if fn.Ret != ast.Type("Result[pkg__HttpResponse,Error]") {
				t.Fatalf("unexpected aliased runtime return type: %#v", fn)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected aliased http response runtime function in surface")
	}
	if strings.Join(surface.LLVMPreludeDecls, "") != strings.Join(LLVMRuntimePreludeDecls(), "") {
		t.Fatalf("unexpected runtime prelude decl surface: %#v", surface.LLVMPreludeDecls)
	}
}

func TestBuildLLVMRuntimeSurfaceTracksIntegratedBackendSurface(t *testing.T) {
	surface := BuildLLVMRuntimeSurface("pkg__HttpResponse", "pkg__ServerRequest", "pkg__ServerResponse", true, true, false)
	if got := surface.TypeAliases["HttpResponse"]; got != ast.Type("pkg__HttpResponse") {
		t.Fatalf("unexpected llvm runtime alias surface: %#v", surface.TypeAliases)
	}
	if !surface.HasStringGlobals || !surface.HasParseIntRuntime || surface.HasParseFloatRuntime {
		t.Fatalf("unexpected llvm runtime capability surface: %#v", surface)
	}
	if !reflect.DeepEqual(surface.PreludeDecls, LLVMRuntimePreludeDecls()) {
		t.Fatalf("unexpected llvm runtime prelude decl surface: %#v", surface.PreludeDecls)
	}
	if !reflect.DeepEqual(surface.PreludeSections, EnabledLLVMRuntimePreludeSections(true)) {
		t.Fatalf("unexpected llvm runtime prelude sections: %#v", surface.PreludeSections)
	}
	if !reflect.DeepEqual(surface.BuiltinSections, EnabledLLVMBuiltinRuntimeSections(true, false)) {
		t.Fatalf("unexpected llvm builtin runtime sections: %#v", surface.BuiltinSections)
	}
	found := false
	for _, fn := range surface.Functions {
		if fn.Name == "__std_http_get_opts_resp" {
			found = true
			if fn.Ret != ast.Type("Result[pkg__HttpResponse,Error]") {
				t.Fatalf("unexpected llvm runtime function aliasing: %#v", fn)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected integrated llvm runtime surface to include aliased runtime functions")
	}
}

func TestOrderedLLVMBuiltinRuntimeSectionsTrackSharedBuiltinRuntimeOrder(t *testing.T) {
	got := OrderedLLVMBuiltinRuntimeSections()
	want := []LLVMBuiltinRuntimeSection{
		LLVMBuiltinRuntimeContains,
		LLVMBuiltinRuntimeStartsWith,
		LLVMBuiltinRuntimeEndsWith,
		LLVMBuiltinRuntimeToUpper,
		LLVMBuiltinRuntimeToLower,
		LLVMBuiltinRuntimeTrimSpace,
		LLVMBuiltinRuntimeRepeat,
		LLVMBuiltinRuntimeReplace,
		LLVMBuiltinRuntimeIntToStr,
		LLVMBuiltinRuntimeFloatToStr,
		LLVMBuiltinRuntimeParseInt,
		LLVMBuiltinRuntimeParseFloat,
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected builtin runtime section count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected builtin runtime section order at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestOrderedLLVMRuntimePreludeSectionsTrackSharedRuntimePreludeOrder(t *testing.T) {
	got := OrderedLLVMRuntimePreludeSections()
	want := []LLVMRuntimePreludeSection{
		LLVMRuntimePreludeStringGlobals,
		LLVMRuntimePreludeRouteTable,
		LLVMRuntimePreludeStringRuntime,
		LLVMRuntimePreludeBuiltin,
		LLVMRuntimePreludeAnyRuntime,
		LLVMRuntimePreludeStdDecls,
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected runtime prelude section count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected runtime prelude section order at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestEnabledLLVMRuntimePreludeSectionsTrackSharedFeatureSelection(t *testing.T) {
	got := EnabledLLVMRuntimePreludeSections(false)
	for _, section := range got {
		if section == LLVMRuntimePreludeStringGlobals {
			t.Fatalf("did not expect string globals section when disabled: %v", got)
		}
	}
	got = EnabledLLVMRuntimePreludeSections(true)
	if len(got) == 0 || got[0] != LLVMRuntimePreludeStringGlobals {
		t.Fatalf("expected string globals section to lead enabled runtime prelude, got %v", got)
	}
}

func TestEnabledLLVMBuiltinRuntimeSectionsTrackSharedFeatureSelection(t *testing.T) {
	got := EnabledLLVMBuiltinRuntimeSections(false, true)
	for _, section := range got {
		if section == LLVMBuiltinRuntimeParseInt {
			t.Fatalf("did not expect parse_int section when disabled: %v", got)
		}
	}
	last := got[len(got)-1]
	if last != LLVMBuiltinRuntimeParseFloat {
		t.Fatalf("expected parse_float section to remain enabled at tail, got %v", got)
	}
}
