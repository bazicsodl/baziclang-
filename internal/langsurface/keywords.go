package langsurface

type KeywordSpec struct {
	Name           string
	Hover          string
	CompletionKind int
}

var keywordSpecs = []KeywordSpec{
	{Name: "as", Hover: "Import alias binding.", CompletionKind: 14},
	{Name: "const", Hover: "Immutable local or global binding.", CompletionKind: 14},
	{Name: "else", Hover: "Alternate branch for a preceding if.", CompletionKind: 14},
	{Name: "enum", Hover: "Enum declaration.", CompletionKind: 14},
	{Name: "false", Hover: "Boolean literal.", CompletionKind: 12},
	{Name: "fn", Hover: "Function declaration.", CompletionKind: 14},
	{Name: "if", Hover: "Conditional branch.", CompletionKind: 14},
	{Name: "impl", Hover: "Interface implementation.", CompletionKind: 14},
	{Name: "import", Hover: "Import declaration.", CompletionKind: 14},
	{Name: "interface", Hover: "Interface declaration.", CompletionKind: 14},
	{Name: "let", Hover: "Local or global binding.", CompletionKind: 14},
	{Name: "match", Hover: "Exhaustive enum match.", CompletionKind: 14},
	{Name: "nil", Hover: "Nil literal. Bazic currently rejects nil in safe code.", CompletionKind: 12},
	{Name: "package", Hover: "Package declaration.", CompletionKind: 14},
	{Name: "pub", Hover: "Export a top-level declaration from a package.", CompletionKind: 14},
	{Name: "return", Hover: "Return from the current function.", CompletionKind: 14},
	{Name: "struct", Hover: "Struct declaration.", CompletionKind: 14},
	{Name: "true", Hover: "Boolean literal.", CompletionKind: 12},
	{Name: "while", Hover: "While loop.", CompletionKind: 14},
}

func KeywordSpecs() []KeywordSpec {
	return append([]KeywordSpec(nil), keywordSpecs...)
}

func LookupKeyword(name string) (KeywordSpec, bool) {
	for _, spec := range keywordSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return KeywordSpec{}, false
}
