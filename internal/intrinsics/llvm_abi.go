package intrinsics

import (
	"fmt"
	"sort"
	"strings"

	"baziclang/internal/ast"
	baztypes "baziclang/internal/types"
)

type LLVMCallConvention string

const (
	LLVMCallValue LLVMCallConvention = "value"
	LLVMCallVoid  LLVMCallConvention = "void"
	LLVMCallSRet  LLVMCallConvention = "sret"
)

type LLVMCallABI struct {
	Convention    LLVMCallConvention
	NormalizedRet ast.Type
	LLVMRetType   string
}

type LLVMNamedType struct {
	Name string
	Type ast.Type
}

type LLVMNamedTypeABI struct {
	Name           string
	NormalizedType ast.Type
	LLVMType       string
}

type LLVMFunctionABI struct {
	NormalizedRet ast.Type
	LLVMRetType   string
	Params        []LLVMNamedTypeABI
}

type LLVMStorageABI struct {
	NormalizedType ast.Type
	LLVMType       string
	DefaultValue   string
}

type LLVMNamedStorageABI struct {
	Name    string
	Storage LLVMStorageABI
}

type LLVMValueABI struct {
	NormalizedType ast.Type
	LLVMType       string
}

type LLVMValueCoercion string

const (
	LLVMValueDirect LLVMValueCoercion = "direct"
	LLVMValueBoxAny LLVMValueCoercion = "box_any"
)

const (
	LLVMAnyTagInt    = 1
	LLVMAnyTagFloat  = 2
	LLVMAnyTagBool   = 3
	LLVMAnyTagString = 4
	LLVMAnyTagOther  = 5
)

type LLVMAnyBoxClass struct {
	Tag     int
	Payload LLVMAnyPayloadKind
}

type LLVMAnyPayloadKind string

const (
	LLVMAnyPayloadPtr       LLVMAnyPayloadKind = "ptr"
	LLVMAnyPayloadIntToPtr  LLVMAnyPayloadKind = "int_to_ptr"
	LLVMAnyPayloadBoolToPtr LLVMAnyPayloadKind = "bool_to_ptr"
	LLVMAnyPayloadFloatBits LLVMAnyPayloadKind = "float_bits_to_ptr"
	LLVMAnyPayloadHeapCopy  LLVMAnyPayloadKind = "heap_copy"
)

func LLVMResultStructName(okType string, errType string) string {
	return fmt.Sprintf("Result__%s__%s", llvmSanitizeName(okType), llvmSanitizeName(errType))
}

func NormalizeLLVMType(t ast.Type) ast.Type {
	if !baztypes.IsGeneric(t) {
		return t
	}
	parsed := baztypes.MustParse(t)
	if parsed.Name == "Result" && len(parsed.Args) == 2 {
		return ast.Type(LLVMResultStructName(parsed.Args[0].String(), parsed.Args[1].String()))
	}
	return t
}

func LLVMGenericBase(t ast.Type) (string, bool) {
	normalized := NormalizeLLVMType(t)
	if base, ok := baztypes.Base(normalized); ok {
		return base, true
	}
	if normalized != t {
		return string(normalized), true
	}
	return "", false
}

func IsLLVMResultStructReturn(ret ast.Type, hasStruct func(string) bool) bool {
	name := string(ret)
	return strings.HasPrefix(name, "Result__") && hasStruct(name)
}

func UsesLLVMIntrinsicSRet(callee string, ret ast.Type, hasStruct func(string) bool) bool {
	return strings.HasPrefix(callee, "__std_") && IsLLVMResultStructReturn(ret, hasStruct)
}

func ClassifyLLVMCallConvention(callee string, ret ast.Type, hasStruct func(string) bool) LLVMCallConvention {
	switch {
	case ret == ast.TypeVoid:
		return LLVMCallVoid
	case UsesLLVMIntrinsicSRet(callee, ret, hasStruct):
		return LLVMCallSRet
	default:
		return LLVMCallValue
	}
}

func ClassifyLLVMCallABI(callee string, ret ast.Type, mapType func(ast.Type) (string, bool), hasStruct func(string) bool) (LLVMCallABI, bool) {
	normalizedRet := NormalizeLLVMType(ret)
	llvmRetType, ok := mapType(normalizedRet)
	if !ok {
		return LLVMCallABI{}, false
	}
	return LLVMCallABI{
		Convention:    ClassifyLLVMCallConvention(callee, normalizedRet, hasStruct),
		NormalizedRet: normalizedRet,
		LLVMRetType:   llvmRetType,
	}, true
}

func BuildLLVMFunctionABI(ret ast.Type, params []LLVMNamedType, mapType func(ast.Type) (string, bool)) (LLVMFunctionABI, bool) {
	normalizedRet := NormalizeLLVMType(ret)
	llvmRetType, ok := mapType(normalizedRet)
	if !ok {
		return LLVMFunctionABI{}, false
	}
	out := LLVMFunctionABI{
		NormalizedRet: normalizedRet,
		LLVMRetType:   llvmRetType,
		Params:        make([]LLVMNamedTypeABI, 0, len(params)),
	}
	for _, param := range params {
		normalizedType := NormalizeLLVMType(param.Type)
		llvmType, ok := mapType(normalizedType)
		if !ok {
			return LLVMFunctionABI{}, false
		}
		out.Params = append(out.Params, LLVMNamedTypeABI{
			Name:           param.Name,
			NormalizedType: normalizedType,
			LLVMType:       llvmType,
		})
	}
	return out, true
}

func BuildLLVMStorageABI(t ast.Type, mapType func(ast.Type) (string, bool), defaultValue func(ast.Type) string) (LLVMStorageABI, bool) {
	normalizedType := NormalizeLLVMType(t)
	llvmType, ok := mapType(normalizedType)
	if !ok {
		return LLVMStorageABI{}, false
	}
	return LLVMStorageABI{
		NormalizedType: normalizedType,
		LLVMType:       llvmType,
		DefaultValue:   defaultValue(normalizedType),
	}, true
}

func BuildLLVMValueABI(t ast.Type, mapType func(ast.Type) (string, bool)) (LLVMValueABI, bool) {
	normalizedType := NormalizeLLVMType(t)
	llvmType, ok := mapType(normalizedType)
	if !ok {
		return LLVMValueABI{}, false
	}
	return LLVMValueABI{
		NormalizedType: normalizedType,
		LLVMType:       llvmType,
	}, true
}

func BuildLLVMSortedNamedStorageABIs(types map[string]ast.Type, mapType func(ast.Type) (string, bool), defaultValue func(ast.Type) string) ([]LLVMNamedStorageABI, bool) {
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]LLVMNamedStorageABI, 0, len(names))
	for _, name := range names {
		storage, ok := BuildLLVMStorageABI(types[name], mapType, defaultValue)
		if !ok {
			return nil, false
		}
		out = append(out, LLVMNamedStorageABI{Name: name, Storage: storage})
	}
	return out, true
}

func FormatLLVMDefaultReturn(ret ast.Type, llvmRetType string, defaultValue string) string {
	if NormalizeLLVMType(ret) == ast.TypeVoid {
		return "  ret void\n"
	}
	return fmt.Sprintf("  ret %s %s\n", llvmRetType, defaultValue)
}

func ClassifyLLVMValueCoercion(targetType ast.Type, valueType ast.Type) (LLVMValueCoercion, ast.Type, bool) {
	targetType = NormalizeLLVMType(targetType)
	valueType = NormalizeLLVMType(valueType)
	switch {
	case targetType == ast.TypeAny && valueType != ast.TypeAny:
		return LLVMValueBoxAny, ast.TypeAny, true
	case valueType == targetType:
		return LLVMValueDirect, targetType, true
	default:
		return "", ast.TypeInvalid, false
	}
}

func MapLLVMDeclType(t ast.Type, hasStruct func(string) bool) (string, bool) {
	switch normalized := NormalizeLLVMType(t); normalized {
	case ast.TypeVoid:
		return "void", true
	case ast.TypeBool:
		return "i8", true
	case ast.TypeInt:
		return "i64", true
	case ast.TypeFloat:
		return "double", true
	case ast.TypeString:
		return "ptr", true
	default:
		if hasStruct(string(normalized)) {
			return "%" + string(normalized), true
		}
		return "", false
	}
}

func MapLLVMRuntimeType(t ast.Type, isEnum func(string) bool, hasStruct func(string) bool, hasIface func(string) bool) (string, bool) {
	switch normalized := NormalizeLLVMType(t); normalized {
	case ast.TypeVoid:
		return "void", true
	case ast.TypeBool:
		return "i8", true
	case ast.TypeInt:
		return "i64", true
	case ast.TypeFloat:
		return "double", true
	case ast.TypeString:
		return "ptr", true
	case ast.TypeAny:
		return "%Any", true
	default:
		name := string(normalized)
		if isEnum != nil && isEnum(name) {
			return "i64", true
		}
		if baztypes.IsGeneric(normalized) {
			return "", false
		}
		if hasStruct != nil && hasStruct(name) {
			return "%" + name, true
		}
		if hasIface != nil && hasIface(name) {
			return "%" + name, true
		}
		return "", false
	}
}

func DefaultLLVMValue(t ast.Type, isEnum func(string) bool, hasStruct func(string) bool, hasIface func(string) bool) string {
	switch normalized := NormalizeLLVMType(t); normalized {
	case ast.TypeInt:
		return "0"
	case ast.TypeFloat:
		return "0.0"
	case ast.TypeBool:
		return "0"
	case ast.TypeString:
		return "null"
	case ast.TypeAny:
		return "zeroinitializer"
	default:
		name := string(normalized)
		if isEnum != nil && isEnum(name) {
			return "0"
		}
		if hasStruct != nil && hasStruct(name) {
			return "zeroinitializer"
		}
		if hasIface != nil && hasIface(name) {
			return "zeroinitializer"
		}
		return "0"
	}
}

func ClassifyLLVMAnyBoxing(t ast.Type, isEnum func(string) bool, hasStruct func(string) bool, hasIface func(string) bool) (LLVMAnyBoxClass, bool) {
	switch normalized := NormalizeLLVMType(t); normalized {
	case ast.TypeInt:
		return LLVMAnyBoxClass{Tag: LLVMAnyTagInt, Payload: LLVMAnyPayloadIntToPtr}, true
	case ast.TypeBool:
		return LLVMAnyBoxClass{Tag: LLVMAnyTagBool, Payload: LLVMAnyPayloadBoolToPtr}, true
	case ast.TypeFloat:
		return LLVMAnyBoxClass{Tag: LLVMAnyTagFloat, Payload: LLVMAnyPayloadFloatBits}, true
	case ast.TypeString:
		return LLVMAnyBoxClass{Tag: LLVMAnyTagString, Payload: LLVMAnyPayloadPtr}, true
	default:
		name := string(normalized)
		if isEnum != nil && isEnum(name) {
			return LLVMAnyBoxClass{Tag: LLVMAnyTagInt, Payload: LLVMAnyPayloadIntToPtr}, true
		}
		if hasStruct != nil && hasStruct(name) {
			return LLVMAnyBoxClass{Tag: LLVMAnyTagOther, Payload: LLVMAnyPayloadHeapCopy}, true
		}
		if hasIface != nil && hasIface(name) {
			return LLVMAnyBoxClass{Tag: LLVMAnyTagOther, Payload: LLVMAnyPayloadHeapCopy}, true
		}
		return LLVMAnyBoxClass{}, false
	}
}

func MapLLVMAnyHeapCopyType(t ast.Type, isEnum func(string) bool, hasStruct func(string) bool, hasIface func(string) bool) (string, bool) {
	class, ok := ClassifyLLVMAnyBoxing(t, isEnum, hasStruct, hasIface)
	if !ok || class.Payload != LLVMAnyPayloadHeapCopy {
		return "", false
	}
	return MapLLVMRuntimeType(t, isEnum, hasStruct, hasIface)
}

func FormatLLVMIntrinsicDecl(fn FunctionSpec, mapType func(ast.Type) (string, bool), hasStruct func(string) bool) (string, bool) {
	params := make([]string, 0, len(fn.Params)+1)
	if ClassifyLLVMCallConvention(fn.Name, fn.Ret, hasStruct) == LLVMCallSRet {
		params = append(params, fmt.Sprintf("ptr sret(%%%s)", fn.Ret))
		for _, p := range fn.Params {
			llvmParam, ok := mapType(p)
			if !ok {
				return "", false
			}
			params = append(params, llvmParam)
		}
		return fmt.Sprintf("declare void @%s(%s)\n", fn.Name, strings.Join(params, ", ")), true
	}
	llvmRet, ok := mapType(fn.Ret)
	if !ok {
		return "", false
	}
	for _, p := range fn.Params {
		llvmParam, ok := mapType(p)
		if !ok {
			return "", false
		}
		params = append(params, llvmParam)
	}
	return fmt.Sprintf("declare %s @%s(%s)\n", llvmRet, fn.Name, strings.Join(params, ", ")), true
}

func FormatLLVMStdDecls(typeAliases map[string]ast.Type, hasStruct func(string) bool) string {
	var b strings.Builder
	for _, fn := range FunctionSpecs(typeAliases) {
		if !strings.HasPrefix(fn.Name, "__std_") {
			continue
		}
		if decl, ok := FormatLLVMIntrinsicDecl(
			FunctionSpec{
				Name:   fn.Name,
				Params: append([]ast.Type(nil), fn.Params...),
				Ret:    NormalizeLLVMType(fn.Ret),
			},
			func(t ast.Type) (string, bool) {
				return MapLLVMDeclType(t, hasStruct)
			},
			hasStruct,
		); ok {
			b.WriteString(decl)
		}
	}
	b.WriteString("declare void @" + LLVMRuntimeSetArgsFunc + "(i32, ptr)\n")
	return b.String()
}

func llvmSanitizeName(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "T"
	}
	return out
}
