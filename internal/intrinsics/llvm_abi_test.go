package intrinsics

import (
	"runtime"
	"strings"
	"testing"

	"baziclang/internal/ast"
)

func wantLLVMCompactBoolRetType() string {
	if runtime.GOOS == "darwin" {
		return "[2 x i64]"
	}
	return "{ i64, ptr }"
}

func TestIsLLVMResultStructReturnRecognizesLoweredResultStructs(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__string__Error"
	}
	if !IsLLVMResultStructReturn(ast.Type("Result__string__Error"), hasStruct) {
		t.Fatalf("expected lowered result struct return to use sret")
	}
	if IsLLVMResultStructReturn(ast.Type("Result[string,Error]"), hasStruct) {
		t.Fatalf("did not expect generic result syntax to count as lowered result struct")
	}
	if IsLLVMResultStructReturn(ast.Type("Result__missing__Error"), hasStruct) {
		t.Fatalf("did not expect unknown lowered result struct to count as sret")
	}
}

func TestNormalizeLLVMTypeLowersGenericResultToNativeResultStruct(t *testing.T) {
	got := NormalizeLLVMType(ast.Type("Result[HttpResponse,Error]"))
	want := ast.Type("Result__HttpResponse__Error")
	if got != want {
		t.Fatalf("expected normalized llvm result type %q, got %q", want, got)
	}
	if unchanged := NormalizeLLVMType(ast.TypeString); unchanged != ast.TypeString {
		t.Fatalf("expected non-generic type to stay unchanged, got %q", unchanged)
	}
}

func TestLLVMGenericBaseUsesNormalizedLLVMTypeNames(t *testing.T) {
	got, ok := LLVMGenericBase(ast.Type("Result[HttpResponse,Error]"))
	if !ok || got != "Result__HttpResponse__Error" {
		t.Fatalf("expected normalized llvm generic base, got (%q, %v)", got, ok)
	}
	got, ok = LLVMGenericBase(ast.Type("Pair[int,string]"))
	if !ok || got != "Pair" {
		t.Fatalf("expected plain generic base Pair, got (%q, %v)", got, ok)
	}
}

func TestUsesLLVMIntrinsicSRetOnlyForStdIntrinsicsReturningResultStructs(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__bool__Error"
	}
	gotSRet := UsesLLVMIntrinsicSRet("__std_write_file", ast.Type("Result__bool__Error"), hasStruct)
	wantSRet := runtime.GOOS == "windows"
	if gotSRet != wantSRet {
		t.Fatalf("UsesLLVMIntrinsicSRet(bool result) = %v, want %v on %s", gotSRet, wantSRet, runtime.GOOS)
	}
	if UsesLLVMIntrinsicSRet("write_file", ast.Type("Result__bool__Error"), hasStruct) {
		t.Fatalf("did not expect non-intrinsic call to use sret")
	}
	if UsesLLVMIntrinsicSRet("__std_write_file", ast.TypeString, hasStruct) {
		t.Fatalf("did not expect non-result return to use sret")
	}
}

func TestClassifyLLVMCallConventionTracksVoidValueAndSRet(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__bool__Error"
	}
	if got := ClassifyLLVMCallConvention("print", ast.TypeVoid, hasStruct); got != LLVMCallVoid {
		t.Fatalf("expected void llvm call convention, got %q", got)
	}
	want := LLVMCallValue
	if runtime.GOOS == "windows" {
		want = LLVMCallSRet
	}
	if got := ClassifyLLVMCallConvention("__std_write_file", ast.Type("Result__bool__Error"), hasStruct); got != want {
		t.Fatalf("unexpected llvm call convention for bool result on %s: got %q want %q", runtime.GOOS, got, want)
	}
	if got := ClassifyLLVMCallConvention("len", ast.TypeInt, hasStruct); got != LLVMCallValue {
		t.Fatalf("expected value llvm call convention, got %q", got)
	}
}

func TestClassifyLLVMCallABITracksNormalizedReturnTypeAndConvention(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__string__Error" || name == "Result__bool__Error"
	}
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeVoid:
			return "void", true
		case ast.TypeInt:
			return "i64", true
		case ast.Type("Result__bool__Error"):
			return "%Result__bool__Error", true
		case ast.Type("Result__string__Error"):
			return "%Result__string__Error", true
		default:
			return "", false
		}
	}
	abi, ok := ClassifyLLVMCallABI("__std_read_file", ast.Type("Result[string,Error]"), mapType, hasStruct)
	if !ok {
		t.Fatalf("expected call abi classification to succeed")
	}
	if abi.Convention != LLVMCallSRet || abi.NormalizedRet != ast.Type("Result__string__Error") || abi.LLVMRetType != "%Result__string__Error" {
		t.Fatalf("unexpected llvm call abi: %+v", abi)
	}
	abi, ok = ClassifyLLVMCallABI("__std_write_file", ast.Type("Result[bool,Error]"), mapType, hasStruct)
	if !ok {
		t.Fatalf("expected compact bool call abi classification to succeed")
	}
	wantBoolConvention := LLVMCallValue
	if runtime.GOOS == "windows" {
		wantBoolConvention = LLVMCallSRet
	}
	wantBoolRetType := "%Result__bool__Error"
	if runtime.GOOS != "windows" {
		wantBoolRetType = wantLLVMCompactBoolRetType()
	}
	if abi.Convention != wantBoolConvention || abi.NormalizedRet != ast.Type("Result__bool__Error") || abi.LLVMRetType != wantBoolRetType {
		t.Fatalf("unexpected compact bool llvm call abi: %+v", abi)
	}
	abi, ok = ClassifyLLVMCallABI("len", ast.TypeInt, mapType, hasStruct)
	if !ok {
		t.Fatalf("expected plain call abi classification to succeed")
	}
	if abi.Convention != LLVMCallValue || abi.NormalizedRet != ast.TypeInt || abi.LLVMRetType != "i64" {
		t.Fatalf("unexpected plain llvm call abi: %+v", abi)
	}
}

func TestBuildLLVMFunctionABITracksNormalizedReturnAndParams(t *testing.T) {
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeVoid:
			return "void", true
		case ast.TypeInt:
			return "i64", true
		case ast.TypeString:
			return "ptr", true
		case ast.Type("Result__string__Error"):
			return "%Result__string__Error", true
		default:
			return "", false
		}
	}
	abi, ok := BuildLLVMFunctionABI(
		ast.Type("Result[string,Error]"),
		[]LLVMNamedType{
			{Name: "src", Type: ast.TypeString},
			{Name: "count", Type: ast.TypeInt},
		},
		mapType,
	)
	if !ok {
		t.Fatalf("expected function abi construction to succeed")
	}
	if abi.NormalizedRet != ast.Type("Result__string__Error") || abi.LLVMRetType != "%Result__string__Error" {
		t.Fatalf("unexpected function abi return: %+v", abi)
	}
	if len(abi.Params) != 2 || abi.Params[0].Name != "src" || abi.Params[0].LLVMType != "ptr" || abi.Params[1].Name != "count" || abi.Params[1].LLVMType != "i64" {
		t.Fatalf("unexpected function abi params: %+v", abi.Params)
	}
}

func TestBuildLLVMStorageABITracksNormalizedTypeMappedTypeAndDefault(t *testing.T) {
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeInt:
			return "i64", true
		case ast.Type("User"):
			return "%User", true
		default:
			return "", false
		}
	}
	defaultValue := func(t ast.Type) string {
		switch t {
		case ast.TypeInt:
			return "0"
		case ast.Type("User"):
			return "zeroinitializer"
		default:
			return "?"
		}
	}
	abi, ok := BuildLLVMStorageABI(ast.TypeInt, mapType, defaultValue)
	if !ok || abi.NormalizedType != ast.TypeInt || abi.LLVMType != "i64" || abi.DefaultValue != "0" {
		t.Fatalf("unexpected int storage abi: %+v, ok=%v", abi, ok)
	}
	abi, ok = BuildLLVMStorageABI(ast.Type("User"), mapType, defaultValue)
	if !ok || abi.NormalizedType != ast.Type("User") || abi.LLVMType != "%User" || abi.DefaultValue != "zeroinitializer" {
		t.Fatalf("unexpected struct storage abi: %+v, ok=%v", abi, ok)
	}
}

func TestBuildLLVMValueABITracksNormalizedTypeAndMappedType(t *testing.T) {
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeInt:
			return "i64", true
		case ast.Type("Result__string__Error"):
			return "%Result__string__Error", true
		default:
			return "", false
		}
	}
	abi, ok := BuildLLVMValueABI(ast.TypeInt, mapType)
	if !ok || abi.NormalizedType != ast.TypeInt || abi.LLVMType != "i64" {
		t.Fatalf("unexpected int value abi: %+v, ok=%v", abi, ok)
	}
	abi, ok = BuildLLVMValueABI(ast.Type("Result[string,Error]"), mapType)
	if !ok || abi.NormalizedType != ast.Type("Result__string__Error") || abi.LLVMType != "%Result__string__Error" {
		t.Fatalf("unexpected result value abi: %+v, ok=%v", abi, ok)
	}
}

func TestBuildLLVMSortedNamedStorageABIsSortsAndBuildsStorage(t *testing.T) {
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeInt:
			return "i64", true
		case ast.TypeString:
			return "ptr", true
		default:
			return "", false
		}
	}
	defaultValue := func(t ast.Type) string {
		switch t {
		case ast.TypeInt:
			return "0"
		case ast.TypeString:
			return "null"
		default:
			return "?"
		}
	}
	abis, ok := BuildLLVMSortedNamedStorageABIs(map[string]ast.Type{
		"z": ast.TypeString,
		"a": ast.TypeInt,
	}, mapType, defaultValue)
	if !ok {
		t.Fatalf("expected named storage abi construction to succeed")
	}
	if len(abis) != 2 || abis[0].Name != "a" || abis[0].Storage.LLVMType != "i64" || abis[1].Name != "z" || abis[1].Storage.LLVMType != "ptr" {
		t.Fatalf("unexpected named storage abis: %+v", abis)
	}
}

func TestFormatLLVMDefaultReturnTracksVoidAndValue(t *testing.T) {
	if got := FormatLLVMDefaultReturn(ast.TypeVoid, "void", ""); got != "  ret void\n" {
		t.Fatalf("unexpected void default return: %q", got)
	}
	if got := FormatLLVMDefaultReturn(ast.TypeInt, "i64", "0"); got != "  ret i64 0\n" {
		t.Fatalf("unexpected value default return: %q", got)
	}
}

func TestClassifyLLVMValueCoercionTracksDirectAndAnyBoxing(t *testing.T) {
	tests := []struct {
		target ast.Type
		value  ast.Type
		want   LLVMValueCoercion
		out    ast.Type
		ok     bool
	}{
		{target: ast.TypeInt, value: ast.TypeInt, want: LLVMValueDirect, out: ast.TypeInt, ok: true},
		{target: ast.TypeAny, value: ast.TypeInt, want: LLVMValueBoxAny, out: ast.TypeAny, ok: true},
		{target: ast.Type("Result__string__Error"), value: ast.Type("Result[string,Error]"), want: LLVMValueDirect, out: ast.Type("Result__string__Error"), ok: true},
		{target: ast.TypeString, value: ast.TypeInt, want: "", out: ast.TypeInvalid, ok: false},
	}
	for _, tc := range tests {
		got, out, ok := ClassifyLLVMValueCoercion(tc.target, tc.value)
		if got != tc.want || out != tc.out || ok != tc.ok {
			t.Fatalf("ClassifyLLVMValueCoercion(%q, %q) = (%q, %q, %v), want (%q, %q, %v)", tc.target, tc.value, got, out, ok, tc.want, tc.out, tc.ok)
		}
	}
}

func TestLLVMResultStructNameSanitizesNativeABIName(t *testing.T) {
	got := LLVMResultStructName("Result[string,Error]", "My Error")
	want := "Result__Result_string_Error__My_Error"
	if got != want {
		t.Fatalf("expected sanitized llvm result struct name %q, got %q", want, got)
	}
}

func TestFormatLLVMIntrinsicDeclUsesSRetForLoweredResultStructs(t *testing.T) {
	spec := FunctionSpec{
		Name:   "__std_read_file",
		Ret:    ast.Type("Result__string__Error"),
		Params: []ast.Type{ast.TypeString},
	}
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeString:
			return "ptr", true
		case ast.Type("Result__string__Error"):
			return "%Result__string__Error", true
		default:
			return "", false
		}
	}
	decl, ok := FormatLLVMIntrinsicDecl(spec, mapType, func(name string) bool {
		return name == "Result__string__Error"
	})
	if !ok {
		t.Fatalf("expected declaration formatting to succeed")
	}
	if !strings.Contains(decl, "declare void @__std_read_file(ptr sret(%Result__string__Error) align 8, ptr)") {
		t.Fatalf("expected sret intrinsic declaration, got %q", decl)
	}
}

func TestFormatLLVMIntrinsicDeclKeepsPlainReturnsForNonStructResults(t *testing.T) {
	spec := FunctionSpec{
		Name:   "__std_exists",
		Ret:    ast.TypeBool,
		Params: []ast.Type{ast.TypeString, ast.TypeBool},
	}
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeBool:
			return "i1", true
		case ast.TypeString:
			return "ptr", true
		default:
			return "", false
		}
	}
	decl, ok := FormatLLVMIntrinsicDecl(spec, mapType, func(string) bool { return false })
	if !ok {
		t.Fatalf("expected declaration formatting to succeed")
	}
	if decl != "declare zeroext i1 @__std_exists(ptr, i1 zeroext)\n" {
		t.Fatalf("unexpected plain intrinsic declaration: %q", decl)
	}
}

func TestFormatLLVMIntrinsicDeclUsesCompactRegisterReturnForBoolResult(t *testing.T) {
	spec := FunctionSpec{
		Name:   "__std_write_file",
		Ret:    ast.Type("Result__bool__Error"),
		Params: []ast.Type{ast.TypeString, ast.TypeString},
	}
	mapType := func(t ast.Type) (string, bool) {
		switch t {
		case ast.TypeString:
			return "ptr", true
		case ast.Type("Result__bool__Error"):
			return "%Result__bool__Error", true
		default:
			return "", false
		}
	}
	decl, ok := FormatLLVMIntrinsicDecl(spec, mapType, func(name string) bool {
		return name == "Result__bool__Error"
	})
	if !ok {
		t.Fatalf("expected bool result declaration formatting to succeed")
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(decl, "declare void @__std_write_file(ptr sret(%Result__bool__Error) align 8, ptr, ptr)") {
			t.Fatalf("expected windows sret bool intrinsic declaration, got %q", decl)
		}
		return
	}
	want := "declare " + wantLLVMCompactBoolRetType() + " @__std_write_file(ptr, ptr)\n"
	if decl != want {
		t.Fatalf("unexpected compact bool intrinsic declaration: got %q want %q", decl, want)
	}
}

func TestMapLLVMDeclTypeTracksSharedNativeDeclTypes(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__string__Error"
	}
	tests := []struct {
		typ  ast.Type
		want string
		ok   bool
	}{
		{typ: ast.TypeVoid, want: "void", ok: true},
		{typ: ast.TypeBool, want: "i8", ok: true},
		{typ: ast.TypeInt, want: "i64", ok: true},
		{typ: ast.TypeFloat, want: "double", ok: true},
		{typ: ast.TypeString, want: "ptr", ok: true},
		{typ: ast.Type("Result__string__Error"), want: "%Result__string__Error", ok: true},
		{typ: ast.Type("Missing"), want: "", ok: false},
	}
	for _, tc := range tests {
		got, ok := MapLLVMDeclType(tc.typ, hasStruct)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("MapLLVMDeclType(%q) = (%q, %v), want (%q, %v)", tc.typ, got, ok, tc.want, tc.ok)
		}
	}
}

func TestMapLLVMCABITypeTracksScalarBoolSeparately(t *testing.T) {
	hasStruct := func(name string) bool {
		return name == "Result__string__Error"
	}
	tests := []struct {
		typ  ast.Type
		want string
		ok   bool
	}{
		{typ: ast.TypeVoid, want: "void", ok: true},
		{typ: ast.TypeBool, want: "i1", ok: true},
		{typ: ast.TypeInt, want: "i64", ok: true},
		{typ: ast.TypeFloat, want: "double", ok: true},
		{typ: ast.TypeString, want: "ptr", ok: true},
		{typ: ast.Type("Result__string__Error"), want: "%Result__string__Error", ok: true},
		{typ: ast.Type("Missing"), want: "", ok: false},
	}
	for _, tc := range tests {
		got, ok := MapLLVMCABIType(tc.typ, hasStruct)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("MapLLVMCABIType(%q) = (%q, %v), want (%q, %v)", tc.typ, got, ok, tc.want, tc.ok)
		}
	}
}

func TestMapLLVMRuntimeTypeTracksEnumsStructsIfacesAndAny(t *testing.T) {
	isEnum := func(name string) bool { return name == "Role" }
	hasStruct := func(name string) bool { return name == "User" }
	hasIface := func(name string) bool { return name == "Greeter" }
	tests := []struct {
		typ  ast.Type
		want string
		ok   bool
	}{
		{typ: ast.TypeAny, want: "%Any", ok: true},
		{typ: ast.Type("Role"), want: "i64", ok: true},
		{typ: ast.Type("User"), want: "%User", ok: true},
		{typ: ast.Type("Greeter"), want: "%Greeter", ok: true},
		{typ: ast.Type("Pair[int,string]"), want: "", ok: false},
	}
	for _, tc := range tests {
		got, ok := MapLLVMRuntimeType(tc.typ, isEnum, hasStruct, hasIface)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("MapLLVMRuntimeType(%q) = (%q, %v), want (%q, %v)", tc.typ, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDefaultLLVMValueTracksEnumsStructsIfacesAndAny(t *testing.T) {
	isEnum := func(name string) bool { return name == "Role" }
	hasStruct := func(name string) bool { return name == "User" }
	hasIface := func(name string) bool { return name == "Greeter" }
	tests := []struct {
		typ  ast.Type
		want string
	}{
		{typ: ast.TypeAny, want: "zeroinitializer"},
		{typ: ast.Type("Role"), want: "0"},
		{typ: ast.Type("User"), want: "zeroinitializer"},
		{typ: ast.Type("Greeter"), want: "zeroinitializer"},
		{typ: ast.Type("Missing"), want: "0"},
	}
	for _, tc := range tests {
		if got := DefaultLLVMValue(tc.typ, isEnum, hasStruct, hasIface); got != tc.want {
			t.Fatalf("DefaultLLVMValue(%q) = %q, want %q", tc.typ, got, tc.want)
		}
	}
}

func TestClassifyLLVMAnyBoxingTracksTagsAndHeapCopyPolicy(t *testing.T) {
	isEnum := func(name string) bool { return name == "Role" }
	hasStruct := func(name string) bool { return name == "User" }
	hasIface := func(name string) bool { return name == "Greeter" }
	tests := []struct {
		typ      ast.Type
		wantTag  int
		wantKind LLVMAnyPayloadKind
		ok       bool
	}{
		{typ: ast.TypeInt, wantTag: LLVMAnyTagInt, wantKind: LLVMAnyPayloadIntToPtr, ok: true},
		{typ: ast.Type("Role"), wantTag: LLVMAnyTagInt, wantKind: LLVMAnyPayloadIntToPtr, ok: true},
		{typ: ast.TypeFloat, wantTag: LLVMAnyTagFloat, wantKind: LLVMAnyPayloadFloatBits, ok: true},
		{typ: ast.TypeBool, wantTag: LLVMAnyTagBool, wantKind: LLVMAnyPayloadBoolToPtr, ok: true},
		{typ: ast.TypeString, wantTag: LLVMAnyTagString, wantKind: LLVMAnyPayloadPtr, ok: true},
		{typ: ast.Type("User"), wantTag: LLVMAnyTagOther, wantKind: LLVMAnyPayloadHeapCopy, ok: true},
		{typ: ast.Type("Greeter"), wantTag: LLVMAnyTagOther, wantKind: LLVMAnyPayloadHeapCopy, ok: true},
		{typ: ast.Type("Missing"), wantTag: 0, wantKind: "", ok: false},
	}
	for _, tc := range tests {
		got, ok := ClassifyLLVMAnyBoxing(tc.typ, isEnum, hasStruct, hasIface)
		if ok != tc.ok || got.Tag != tc.wantTag || got.Payload != tc.wantKind {
			t.Fatalf("ClassifyLLVMAnyBoxing(%q) = (%+v, %v), want (tag=%d kind=%q ok=%v)", tc.typ, got, ok, tc.wantTag, tc.wantKind, tc.ok)
		}
	}
}

func TestMapLLVMAnyHeapCopyTypeOnlyForHeapCopiedPayloads(t *testing.T) {
	isEnum := func(name string) bool { return name == "Role" }
	hasStruct := func(name string) bool { return name == "User" }
	hasIface := func(name string) bool { return name == "Greeter" }

	if got, ok := MapLLVMAnyHeapCopyType(ast.Type("User"), isEnum, hasStruct, hasIface); !ok || got != "%User" {
		t.Fatalf("expected heap-copy struct payload type, got (%q, %v)", got, ok)
	}
	if got, ok := MapLLVMAnyHeapCopyType(ast.Type("Greeter"), isEnum, hasStruct, hasIface); !ok || got != "%Greeter" {
		t.Fatalf("expected heap-copy iface payload type, got (%q, %v)", got, ok)
	}
	if got, ok := MapLLVMAnyHeapCopyType(ast.TypeString, isEnum, hasStruct, hasIface); ok || got != "" {
		t.Fatalf("expected string payload to skip heap-copy type mapping, got (%q, %v)", got, ok)
	}
}

func TestFormatLLVMStdDeclsUsesAliasesAndAvailableStructs(t *testing.T) {
	httpResponseType := "pkg__HttpResponse"
	resultHttpResponseErr := LLVMResultStructName(httpResponseType, "Error")
	hasStruct := func(name string) bool {
		switch name {
		case LLVMResultStructName("string", "Error"),
			LLVMResultStructName("bool", "Error"),
			LLVMResultStructName("int", "Error"),
			httpResponseType,
			resultHttpResponseErr:
			return true
		default:
			return false
		}
	}
	out := FormatLLVMStdDecls(map[string]ast.Type{"HttpResponse": ast.Type(httpResponseType)}, hasStruct)
	if !strings.Contains(out, "declare void @__std_json_get_int(ptr sret(%Result__int__Error) align 8, ptr, ptr)\n") {
		t.Fatalf("expected int result intrinsic decl in generated std decls:\n%s", out)
	}
	if strings.Contains(out, "__std_json_get_float") {
		t.Fatalf("did not expect float result intrinsic decl without matching result struct:\n%s", out)
	}
	expectedHTTP := "declare void @__std_http_get_opts_resp(ptr sret(%" + resultHttpResponseErr + ") align 8, ptr, i64, i64, ptr, ptr, i1 zeroext, ptr)\n"
	if !strings.Contains(out, expectedHTTP) {
		t.Fatalf("expected HttpResponse result decl using aliased internal type name:\n%s", out)
	}
	if !strings.Contains(out, "declare zeroext i1 @__std_json_validate(ptr)\n") {
		t.Fatalf("expected scalar bool std decl to use c abi bool:\n%s", out)
	}
	if !strings.Contains(out, "declare void @__bazic_set_args(i32, ptr)\n") {
		t.Fatalf("expected argv bridge decl in std decls:\n%s", out)
	}
}
