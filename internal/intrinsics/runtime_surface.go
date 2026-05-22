package intrinsics

import (
	"fmt"

	"baziclang/internal/ast"
)

type LLVMBuiltinRuntimeSection string
type LLVMRuntimePreludeSection string

type RuntimeInterfaceSurface struct {
	TypeAliases      map[string]ast.Type
	Functions        []FunctionSpec
	LLVMPreludeDecls []string
}

type LLVMRuntimeSurface struct {
	TypeAliases          map[string]ast.Type
	Functions            []FunctionSpec
	PreludeDecls         []string
	HasStringGlobals     bool
	HasParseIntRuntime   bool
	HasParseFloatRuntime bool
	PreludeSections      []LLVMRuntimePreludeSection
	BuiltinSections      []LLVMBuiltinRuntimeSection
}

const (
	LLVMRuntimePreludeStringGlobals LLVMRuntimePreludeSection = "string_globals"
	LLVMRuntimePreludeRouteTable    LLVMRuntimePreludeSection = "route_table"
	LLVMRuntimePreludeStringRuntime LLVMRuntimePreludeSection = "string_runtime"
	LLVMRuntimePreludeBuiltin       LLVMRuntimePreludeSection = "builtin_runtime"
	LLVMRuntimePreludeAnyRuntime    LLVMRuntimePreludeSection = "any_runtime"
	LLVMRuntimePreludeStdDecls      LLVMRuntimePreludeSection = "std_decls"

	LLVMBuiltinRuntimeContains   LLVMBuiltinRuntimeSection = "contains"
	LLVMBuiltinRuntimeStartsWith LLVMBuiltinRuntimeSection = "starts_with"
	LLVMBuiltinRuntimeEndsWith   LLVMBuiltinRuntimeSection = "ends_with"
	LLVMBuiltinRuntimeToUpper    LLVMBuiltinRuntimeSection = "to_upper"
	LLVMBuiltinRuntimeToLower    LLVMBuiltinRuntimeSection = "to_lower"
	LLVMBuiltinRuntimeTrimSpace  LLVMBuiltinRuntimeSection = "trim_space"
	LLVMBuiltinRuntimeRepeat     LLVMBuiltinRuntimeSection = "repeat"
	LLVMBuiltinRuntimeReplace    LLVMBuiltinRuntimeSection = "replace"
	LLVMBuiltinRuntimeIntToStr   LLVMBuiltinRuntimeSection = "int_to_str"
	LLVMBuiltinRuntimeFloatToStr LLVMBuiltinRuntimeSection = "float_to_str"
	LLVMBuiltinRuntimeParseInt   LLVMBuiltinRuntimeSection = "parse_int"
	LLVMBuiltinRuntimeParseFloat LLVMBuiltinRuntimeSection = "parse_float"
)

func RuntimeTypeAliases(httpResponseType, serverRequestType, serverResponseType string) map[string]ast.Type {
	aliases := map[string]ast.Type{}
	if httpResponseType != "" && httpResponseType != "HttpResponse" {
		aliases["HttpResponse"] = ast.Type(httpResponseType)
	}
	if serverRequestType != "" && serverRequestType != "ServerRequest" {
		aliases["ServerRequest"] = ast.Type(serverRequestType)
	}
	if serverResponseType != "" && serverResponseType != "ServerResponse" {
		aliases["ServerResponse"] = ast.Type(serverResponseType)
	}
	return aliases
}

func BuildRuntimeInterfaceSurface(httpResponseType, serverRequestType, serverResponseType string) RuntimeInterfaceSurface {
	aliases := RuntimeTypeAliases(httpResponseType, serverRequestType, serverResponseType)
	return RuntimeInterfaceSurface{
		TypeAliases:      aliases,
		Functions:        FunctionSpecs(aliases),
		LLVMPreludeDecls: LLVMRuntimePreludeDecls(),
	}
}

func BuildLLVMRuntimeSurface(httpResponseType, serverRequestType, serverResponseType string, hasStringGlobals, hasParseIntRuntime, hasParseFloatRuntime bool) LLVMRuntimeSurface {
	iface := BuildRuntimeInterfaceSurface(httpResponseType, serverRequestType, serverResponseType)
	return LLVMRuntimeSurface{
		TypeAliases:          iface.TypeAliases,
		Functions:            iface.Functions,
		PreludeDecls:         iface.LLVMPreludeDecls,
		HasStringGlobals:     hasStringGlobals,
		HasParseIntRuntime:   hasParseIntRuntime,
		HasParseFloatRuntime: hasParseFloatRuntime,
		PreludeSections:      EnabledLLVMRuntimePreludeSections(hasStringGlobals),
		BuiltinSections:      EnabledLLVMBuiltinRuntimeSections(hasParseIntRuntime, hasParseFloatRuntime),
	}
}

func LLVMRuntimePreludeDecls() []string {
	return []string{
		"declare i32 @printf(ptr, ...)\n",
		"declare i64 @strlen(ptr)\n",
		fmt.Sprintf("declare i64 @%s(ptr)\n", LLVMRuntimeLenFunc),
		"declare i32 @strcmp(ptr, ptr)\n",
		"declare ptr @strstr(ptr, ptr)\n",
		"declare i32 @strncmp(ptr, ptr, i64)\n",
		"declare i32 @toupper(i32)\n",
		"declare i32 @tolower(i32)\n",
		"declare i32 @isspace(i32)\n",
		"declare i64 @strtol(ptr, ptr, i32)\n",
		"declare double @strtod(ptr, ptr)\n",
		"declare i32 @snprintf(ptr, i64, ptr, ...)\n",
		"declare ptr @malloc(i64)\n",
		"declare ptr @memcpy(ptr, ptr, i64)\n",
	}
}

func OrderedLLVMRuntimePreludeSections() []LLVMRuntimePreludeSection {
	return []LLVMRuntimePreludeSection{
		LLVMRuntimePreludeStringGlobals,
		LLVMRuntimePreludeRouteTable,
		LLVMRuntimePreludeStringRuntime,
		LLVMRuntimePreludeBuiltin,
		LLVMRuntimePreludeAnyRuntime,
		LLVMRuntimePreludeStdDecls,
	}
}

func EnabledLLVMRuntimePreludeSections(hasStringGlobals bool) []LLVMRuntimePreludeSection {
	out := make([]LLVMRuntimePreludeSection, 0, len(OrderedLLVMRuntimePreludeSections()))
	for _, section := range OrderedLLVMRuntimePreludeSections() {
		if section == LLVMRuntimePreludeStringGlobals && !hasStringGlobals {
			continue
		}
		out = append(out, section)
	}
	return out
}

func OrderedLLVMBuiltinRuntimeSections() []LLVMBuiltinRuntimeSection {
	return []LLVMBuiltinRuntimeSection{
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
}

func EnabledLLVMBuiltinRuntimeSections(hasParseInt bool, hasParseFloat bool) []LLVMBuiltinRuntimeSection {
	out := make([]LLVMBuiltinRuntimeSection, 0, len(OrderedLLVMBuiltinRuntimeSections()))
	for _, section := range OrderedLLVMBuiltinRuntimeSections() {
		switch section {
		case LLVMBuiltinRuntimeParseInt:
			if !hasParseInt {
				continue
			}
		case LLVMBuiltinRuntimeParseFloat:
			if !hasParseFloat {
				continue
			}
		}
		out = append(out, section)
	}
	return out
}
