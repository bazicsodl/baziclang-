package langsurface

import "testing"

func TestWordAtUsesSharedIdentifierRules(t *testing.T) {
	src := "let answer_value = some(answer_value);\n"
	got := WordAt(src, LineCol{Line: 0, Character: 6})
	if got != "answer_value" {
		t.Fatalf("unexpected word at position: got %q want %q", got, "answer_value")
	}
	if got := WordAt(src, LineCol{Line: 0, Character: len(src)}); got != "" {
		t.Fatalf("expected out-of-range word lookup to be empty, got %q", got)
	}
}

func TestFindWordRangesFindsWholeIdentifiersOnly(t *testing.T) {
	src := "let cat = 1;\nlet catalog = cat + 1;\n"
	got := FindWordRanges(src, "cat")
	if len(got) != 2 {
		t.Fatalf("unexpected word range count: got %d want 2", len(got))
	}
	if got[0][0] >= got[0][1] || got[1][0] >= got[1][1] {
		t.Fatalf("expected valid word ranges, got %#v", got)
	}
}
