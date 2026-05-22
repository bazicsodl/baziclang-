package types

import "baziclang/internal/ast"

func SplitGenericTypeStrings(t string) (string, []string, bool) {
	base, ok := Base(ast.Type(t))
	if !ok {
		return "", nil, false
	}
	args, ok := Args(ast.Type(t))
	if !ok {
		return "", nil, false
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, string(arg))
	}
	return base, out, true
}
