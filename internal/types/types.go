package types

import (
	"fmt"
	"strings"

	"baziclang/internal/ast"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindNamed
)

type Type struct {
	Kind Kind
	Name string
	Args []Type
}

func Parse(t ast.Type) (Type, error) {
	return parseString(string(t))
}

func MustParse(t ast.Type) Type {
	parsed, err := Parse(t)
	if err != nil {
		return Type{Kind: KindInvalid, Name: string(t)}
	}
	return parsed
}

func (t Type) String() string {
	if t.Kind != KindNamed || t.Name == "" {
		return string(ast.TypeInvalid)
	}
	if len(t.Args) == 0 {
		return t.Name
	}
	parts := make([]string, 0, len(t.Args))
	for _, arg := range t.Args {
		parts = append(parts, arg.String())
	}
	return fmt.Sprintf("%s[%s]", t.Name, strings.Join(parts, ","))
}

func ToAST(t Type) ast.Type {
	return ast.Type(t.String())
}

func IsGeneric(t ast.Type) bool {
	parsed, err := Parse(t)
	return err == nil && len(parsed.Args) > 0
}

func Base(t ast.Type) (string, bool) {
	parsed, err := Parse(t)
	if err != nil || len(parsed.Args) == 0 {
		return "", false
	}
	return parsed.Name, true
}

func Args(t ast.Type) ([]ast.Type, bool) {
	parsed, err := Parse(t)
	if err != nil || len(parsed.Args) == 0 {
		return nil, false
	}
	out := make([]ast.Type, 0, len(parsed.Args))
	for _, arg := range parsed.Args {
		out = append(out, ToAST(arg))
	}
	return out, true
}

func BindTypeParams(typeParams []string, actual []ast.Type) (map[string]ast.Type, error) {
	if len(typeParams) == 0 {
		if len(actual) > 0 {
			return nil, fmt.Errorf("non-generic type used with type arguments")
		}
		return map[string]ast.Type{}, nil
	}
	if len(actual) == 0 {
		return nil, fmt.Errorf("generic type requires %d type arguments", len(typeParams))
	}
	if len(typeParams) != len(actual) {
		return nil, fmt.Errorf("generic type expected %d type arguments, got %d", len(typeParams), len(actual))
	}
	mapping := make(map[string]ast.Type, len(typeParams))
	for i, tp := range typeParams {
		mapping[tp] = actual[i]
	}
	return mapping, nil
}

func Substitute(t ast.Type, mapping map[string]ast.Type) ast.Type {
	if v, ok := mapping[string(t)]; ok {
		return v
	}
	parsed, err := Parse(t)
	if err != nil || len(parsed.Args) == 0 {
		return t
	}
	out := Type{Kind: KindNamed, Name: parsed.Name, Args: make([]Type, 0, len(parsed.Args))}
	for _, arg := range parsed.Args {
		out.Args = append(out.Args, MustParse(Substitute(ToAST(arg), mapping)))
	}
	return ToAST(out)
}

func Unify(expected ast.Type, actual ast.Type, mapping map[string]ast.Type, params []string) bool {
	if expected == ast.TypeAny {
		return true
	}
	if contains(params, string(expected)) {
		if bound, ok := mapping[string(expected)]; ok {
			return bound == actual
		}
		mapping[string(expected)] = actual
		return true
	}
	exp, errExp := Parse(expected)
	act, errAct := Parse(actual)
	if errExp == nil && errAct == nil && len(exp.Args) > 0 {
		if act.Name != exp.Name || len(act.Args) != len(exp.Args) {
			return false
		}
		for i := range exp.Args {
			if !Unify(ToAST(exp.Args[i]), ToAST(act.Args[i]), mapping, params) {
				return false
			}
		}
		return true
	}
	return expected == actual
}

func ContainsParam(t ast.Type, param string) bool {
	if string(t) == param {
		return true
	}
	parsed, err := Parse(t)
	if err != nil || len(parsed.Args) == 0 {
		return false
	}
	for _, arg := range parsed.Args {
		if ContainsParam(ToAST(arg), param) {
			return true
		}
	}
	return false
}

func parseString(s string) (Type, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Type{}, fmt.Errorf("empty type")
	}
	open := strings.IndexRune(s, '[')
	close := strings.LastIndex(s, "]")
	if open == -1 && close == -1 {
		return Type{Kind: KindNamed, Name: s}, nil
	}
	if open <= 0 || close <= open || close != len(s)-1 {
		return Type{}, fmt.Errorf("invalid generic type %q", s)
	}
	base := strings.TrimSpace(s[:open])
	if base == "" {
		return Type{}, fmt.Errorf("missing generic base type")
	}
	inner := s[open+1 : close]
	parts, err := splitTopLevel(inner)
	if err != nil {
		return Type{}, err
	}
	if len(parts) == 0 {
		return Type{}, fmt.Errorf("generic type %q has no arguments", s)
	}
	args := make([]Type, 0, len(parts))
	for _, part := range parts {
		arg, err := parseString(part)
		if err != nil {
			return Type{}, err
		}
		args = append(args, arg)
	}
	return Type{Kind: KindNamed, Name: base, Args: args}, nil
}

func splitTopLevel(s string) ([]string, error) {
	depth := 0
	start := 0
	parts := []string{}
	for i, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced type brackets")
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(s[start:i])
				if part == "" {
					return nil, fmt.Errorf("empty generic type argument")
				}
				parts = append(parts, part)
				start = i + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced type brackets")
	}
	part := strings.TrimSpace(s[start:])
	if part == "" {
		return nil, fmt.Errorf("empty generic type argument")
	}
	parts = append(parts, part)
	return parts, nil
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
