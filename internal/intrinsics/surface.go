package intrinsics

import (
	"sort"
	"strings"

	"baziclang/internal/ast"
)

type SurfaceFunctionSpec struct {
	Name       string
	TypeParams []string
	Params     []ast.Type
	ReturnType ast.Type
	Doc        string
}

func (s SurfaceFunctionSpec) Signature() string {
	var b strings.Builder
	b.WriteString("fn ")
	b.WriteString(s.Name)
	if len(s.TypeParams) > 0 {
		b.WriteString("[")
		b.WriteString(strings.Join(s.TypeParams, ", "))
		b.WriteString("]")
	}
	b.WriteString("(")
	for i, param := range s.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(string(param))
	}
	b.WriteString("): ")
	b.WriteString(string(s.ReturnType))
	return b.String()
}

func (s SurfaceFunctionSpec) Hover() string {
	if strings.TrimSpace(s.Doc) == "" {
		return s.Signature()
	}
	return s.Signature() + "\n" + s.Doc
}

func SurfaceFunctionSpecs() []SurfaceFunctionSpec {
	specs := make([]SurfaceFunctionSpec, 0, 32)
	for _, fn := range FunctionSpecs(nil) {
		if strings.HasPrefix(fn.Name, "__std_") {
			continue
		}
		params := append([]ast.Type(nil), fn.Params...)
		specs = append(specs, SurfaceFunctionSpec{
			Name:       fn.Name,
			Params:     params,
			ReturnType: fn.Ret,
			Doc:        builtinDocs[fn.Name],
		})
	}
	specs = append(specs, preludeSurfaceFunctionSpecs()...)
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	return specs
}

func LookupSurfaceFunction(name string) (SurfaceFunctionSpec, bool) {
	for _, spec := range SurfaceFunctionSpecs() {
		if spec.Name == name {
			return spec, true
		}
	}
	return SurfaceFunctionSpec{}, false
}

var builtinDocs = map[string]string{
	"contains":    "String containment predicate.",
	"ends_with":   "String suffix predicate.",
	"len":         "String length.",
	"parse_float": "Parse a string into a floating-point Result value.",
	"parse_int":   "Parse a string into an integer Result value.",
	"print":       "Builtin output without a trailing newline.",
	"println":     "Builtin output with a trailing newline.",
	"replace":     "Replace all matching substrings.",
	"repeat":      "Repeat a string count times.",
	"starts_with": "String prefix predicate.",
	"str":         "Convert a value to a string.",
	"to_lower":    "Lowercase string transform.",
	"to_upper":    "Uppercase string transform.",
	"trim_space":  "Trim leading and trailing whitespace.",
}

func preludeSurfaceFunctionSpecs() []SurfaceFunctionSpec {
	return []SurfaceFunctionSpec{
		{
			Name:       "assert",
			Params:     []ast.Type{ast.TypeBool},
			ReturnType: ast.TypeVoid,
			Doc:        "Mark the current test/program assertion as failed when the condition is false.",
		},
		{
			Name:       "assert_msg",
			Params:     []ast.Type{ast.TypeBool, ast.TypeString},
			ReturnType: ast.TypeVoid,
			Doc:        "Assertion helper with a custom failure message.",
		},
		{
			Name:       "err",
			TypeParams: []string{"T", "E"},
			Params:     []ast.Type{ast.Type("T"), ast.Type("E")},
			ReturnType: ast.Type("Result[T,E]"),
			Doc:        "Construct an error Result value.",
		},
		{
			Name:       "none",
			TypeParams: []string{"T"},
			Params:     []ast.Type{ast.Type("T")},
			ReturnType: ast.Type("Option[T]"),
			Doc:        "Construct an empty Option value with a fallback payload slot.",
		},
		{
			Name:       "ok",
			TypeParams: []string{"T", "E"},
			Params:     []ast.Type{ast.Type("T"), ast.Type("E")},
			ReturnType: ast.Type("Result[T,E]"),
			Doc:        "Construct a successful Result value.",
		},
		{
			Name:       "result_or",
			TypeParams: []string{"T", "E"},
			Params:     []ast.Type{ast.Type("Result[T,E]"), ast.Type("T")},
			ReturnType: ast.Type("T"),
			Doc:        "Return the successful Result value or a fallback.",
		},
		{
			Name:       "some",
			TypeParams: []string{"T"},
			Params:     []ast.Type{ast.Type("T")},
			ReturnType: ast.Type("Option[T]"),
			Doc:        "Construct a populated Option value.",
		},
		{
			Name:       "unwrap_or",
			TypeParams: []string{"T"},
			Params:     []ast.Type{ast.Type("Option[T]"), ast.Type("T")},
			ReturnType: ast.Type("T"),
			Doc:        "Return the Option value or a fallback.",
		},
	}
}
