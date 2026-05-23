package langsurface

import "testing"

func TestMatchingQuickFixRules(t *testing.T) {
	cases := []struct {
		msg   string
		title string
		op    QuickFixOp
	}{
		{msg: "expected ';' before newline", title: "Bazic: Insert ';'", op: QuickFixInsertLineEnd},
		{msg: "unterminated string literal", title: "Bazic: Close string literal", op: QuickFixInsertLineEnd},
		{msg: "expected '&' after '&'", title: "Bazic: Replace '&' with '&&'", op: QuickFixDoubleOperator},
		{msg: "invalid escape \\q", title: "Bazic: Escape backslashes in string", op: QuickFixEscapeBackslashes},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			rules := MatchingQuickFixRules(tc.msg)
			if len(rules) == 0 {
				t.Fatalf("expected quick-fix rule for %q", tc.msg)
			}
			if rules[0].Title != tc.title {
				t.Fatalf("unexpected quick-fix title: got %q want %q", rules[0].Title, tc.title)
			}
			if rules[0].Op != tc.op {
				t.Fatalf("unexpected quick-fix op: got %q want %q", rules[0].Op, tc.op)
			}
		})
	}
}

func TestQuickFixRulesHaveUniqueMatchStrings(t *testing.T) {
	seen := map[string]bool{}
	for _, rule := range QuickFixRules() {
		if seen[rule.Match] {
			t.Fatalf("duplicate quick-fix match string %q", rule.Match)
		}
		seen[rule.Match] = true
		if rule.Title == "" {
			t.Fatalf("expected title for quick-fix rule %q", rule.Match)
		}
		if rule.Op == "" {
			t.Fatalf("expected op for quick-fix rule %q", rule.Match)
		}
	}
}
