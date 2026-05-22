package intrinsics

import "baziclang/internal/ast"

type LoweredBuiltinCategory string

const (
	LoweredBuiltinLen                   LoweredBuiltinCategory = "len"
	LoweredBuiltinParseString           LoweredBuiltinCategory = "parse_string"
	LoweredBuiltinStringify             LoweredBuiltinCategory = "stringify"
	LoweredBuiltinStringBinaryPredicate LoweredBuiltinCategory = "string_binary_predicate"
	LoweredBuiltinStringRepeat          LoweredBuiltinCategory = "string_repeat"
	LoweredBuiltinStringTernary         LoweredBuiltinCategory = "string_ternary"
	LoweredBuiltinStringUnary           LoweredBuiltinCategory = "string_unary"
)

type LoweredBuiltinSpec struct {
	Name       string
	Arity      int
	Category   LoweredBuiltinCategory
	GoTarget   string
	LLVMTarget string
	ParseKind  string
	ReturnType ast.Type
}

var loweredBuiltinSpecs = map[string]LoweredBuiltinSpec{
	"contains": {
		Name:       "contains",
		Arity:      2,
		Category:   LoweredBuiltinStringBinaryPredicate,
		LLVMTarget: LLVMRuntimeContainsFunc,
		ReturnType: ast.TypeBool,
	},
	"ends_with": {
		Name:       "ends_with",
		Arity:      2,
		Category:   LoweredBuiltinStringBinaryPredicate,
		LLVMTarget: LLVMRuntimeEndsWithFunc,
		ReturnType: ast.TypeBool,
	},
	"len": {
		Name:       "len",
		Arity:      1,
		Category:   LoweredBuiltinLen,
		GoTarget:   LLVMRuntimeLenFunc,
		LLVMTarget: LLVMRuntimeLenFunc,
		ReturnType: ast.TypeInt,
	},
	"parse_float": {
		Name:       "parse_float",
		Arity:      1,
		Category:   LoweredBuiltinParseString,
		LLVMTarget: LLVMRuntimeParseFloatFunc,
		ParseKind:  "float",
	},
	"parse_int": {
		Name:       "parse_int",
		Arity:      1,
		Category:   LoweredBuiltinParseString,
		LLVMTarget: LLVMRuntimeParseIntFunc,
		ParseKind:  "int",
	},
	"repeat": {
		Name:       "repeat",
		Arity:      2,
		Category:   LoweredBuiltinStringRepeat,
		LLVMTarget: LLVMRuntimeRepeatFunc,
		ReturnType: ast.TypeString,
	},
	"str": {
		Name:       "str",
		Arity:      1,
		Category:   LoweredBuiltinStringify,
		ReturnType: ast.TypeString,
	},
	"replace": {
		Name:       "replace",
		Arity:      3,
		Category:   LoweredBuiltinStringTernary,
		LLVMTarget: LLVMRuntimeReplaceFunc,
		ReturnType: ast.TypeString,
	},
	"starts_with": {
		Name:       "starts_with",
		Arity:      2,
		Category:   LoweredBuiltinStringBinaryPredicate,
		LLVMTarget: LLVMRuntimeStartsWithFunc,
		ReturnType: ast.TypeBool,
	},
	"to_lower": {
		Name:       "to_lower",
		Arity:      1,
		Category:   LoweredBuiltinStringUnary,
		LLVMTarget: LLVMRuntimeToLowerFunc,
		ReturnType: ast.TypeString,
	},
	"to_upper": {
		Name:       "to_upper",
		Arity:      1,
		Category:   LoweredBuiltinStringUnary,
		LLVMTarget: LLVMRuntimeToUpperFunc,
		ReturnType: ast.TypeString,
	},
	"trim_space": {
		Name:       "trim_space",
		Arity:      1,
		Category:   LoweredBuiltinStringUnary,
		LLVMTarget: LLVMRuntimeTrimSpaceFunc,
		ReturnType: ast.TypeString,
	},
}

var builtinVoidCalls = map[string]struct{}{
	"print":   {},
	"println": {},
}

func LookupLoweredBuiltin(name string) (LoweredBuiltinSpec, bool) {
	spec, ok := loweredBuiltinSpecs[name]
	return spec, ok
}

func IsBuiltinVoidCall(name string) bool {
	_, ok := builtinVoidCalls[name]
	return ok
}

func LLVMPrintfFormat(isPrintln bool, argType ast.Type, isEnum bool) (string, bool) {
	switch {
	case argType == ast.TypeInt || isEnum:
		if isPrintln {
			return "%ld\n", true
		}
		return "%ld", true
	case argType == ast.TypeFloat:
		if isPrintln {
			return "%g\n", true
		}
		return "%g", true
	case argType == ast.TypeBool || argType == ast.TypeString || argType == ast.TypeAny:
		if isPrintln {
			return "%s\n", true
		}
		return "%s", true
	default:
		return "", false
	}
}
