package langsurface

import "regexp"

type DeclKind string

const (
	DeclKindFunction  DeclKind = "fn"
	DeclKindStruct    DeclKind = "struct"
	DeclKindEnum      DeclKind = "enum"
	DeclKindInterface DeclKind = "interface"
	DeclKindLet       DeclKind = "let"
)

type DeclSymbol struct {
	Kind  DeclKind
	Name  string
	Start int
	End   int
}

var declSymbolPattern = regexp.MustCompile(`\b(fn|struct|enum|interface|let)\s+([A-Za-z_][A-Za-z0-9_]*)`)

func FindDeclSymbols(text string) []DeclSymbol {
	matches := declSymbolPattern.FindAllStringSubmatchIndex(text, -1)
	out := make([]DeclSymbol, 0, len(matches))
	for _, m := range matches {
		if len(m) < 6 {
			continue
		}
		out = append(out, DeclSymbol{
			Kind:  DeclKind(text[m[2]:m[3]]),
			Name:  text[m[4]:m[5]],
			Start: m[4],
			End:   m[5],
		})
	}
	return out
}

func (k DeclKind) LSPCompletionOrSymbolKind() int {
	switch k {
	case DeclKindFunction:
		return 12
	case DeclKindStruct:
		return 23
	case DeclKindEnum:
		return 10
	case DeclKindInterface:
		return 11
	case DeclKindLet:
		return 13
	default:
		return 13
	}
}
