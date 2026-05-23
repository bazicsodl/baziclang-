package langsurface

import "strings"

type QuickFixOp string

const (
	QuickFixInsertLineEnd     QuickFixOp = "insert_line_end"
	QuickFixDoubleOperator    QuickFixOp = "double_operator"
	QuickFixEscapeBackslashes QuickFixOp = "escape_backslashes"
)

type QuickFixRule struct {
	Match     string
	Title     string
	Op        QuickFixOp
	Arg       string
	Preferred bool
}

var quickFixRules = []QuickFixRule{
	{Match: "expected ';'", Title: "Bazic: Insert ';'", Op: QuickFixInsertLineEnd, Arg: ";", Preferred: true},
	{Match: "expected '}'", Title: "Bazic: Insert '}'", Op: QuickFixInsertLineEnd, Arg: "}"},
	{Match: "expected ')'", Title: "Bazic: Insert ')'", Op: QuickFixInsertLineEnd, Arg: ")"},
	{Match: "expected ']'", Title: "Bazic: Insert ']'", Op: QuickFixInsertLineEnd, Arg: "]"},
	{Match: "unterminated string literal", Title: "Bazic: Close string literal", Op: QuickFixInsertLineEnd, Arg: "\""},
	{Match: "unterminated block comment", Title: "Bazic: Close block comment", Op: QuickFixInsertLineEnd, Arg: "*/"},
	{Match: "expected '&' after '&'", Title: "Bazic: Replace '&' with '&&'", Op: QuickFixDoubleOperator, Arg: "&", Preferred: true},
	{Match: "expected '|' after '|'", Title: "Bazic: Replace '|' with '||'", Op: QuickFixDoubleOperator, Arg: "|", Preferred: true},
	{Match: "invalid escape", Title: "Bazic: Escape backslashes in string", Op: QuickFixEscapeBackslashes},
}

func QuickFixRules() []QuickFixRule {
	return append([]QuickFixRule(nil), quickFixRules...)
}

func MatchingQuickFixRules(msg string) []QuickFixRule {
	out := make([]QuickFixRule, 0, len(quickFixRules))
	for _, rule := range quickFixRules {
		if strings.Contains(msg, rule.Match) {
			out = append(out, rule)
		}
	}
	return out
}
