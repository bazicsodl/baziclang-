package main

import (
	"strings"
	"testing"
)

func TestCompletionItemsForSurfaceTracksLanguageAndPrelude(t *testing.T) {
	items := completionItemsForSurface()
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.Label] = true
	}
	for _, want := range []string{"package", "pub", "as", "const", "assert_msg", "some", "parse_int"} {
		if !seen[want] {
			t.Fatalf("expected completion item %q", want)
		}
	}
	for _, unwanted := range []string{"__std_read_file", "http_serve_app", "ui_element"} {
		if seen[unwanted] {
			t.Fatalf("did not expect stale global completion item %q", unwanted)
		}
	}
}

func TestHoverForUsesIntrinsicSurfaceRegistry(t *testing.T) {
	if got := hoverFor("package"); !strings.Contains(got, "Package declaration") {
		t.Fatalf("expected package hover, got %q", got)
	}
	if got := hoverFor("if"); !strings.Contains(got, "Conditional branch") {
		t.Fatalf("expected if hover, got %q", got)
	}
	if got := hoverFor("assert_msg"); !strings.Contains(got, "fn assert_msg(bool, string): void") {
		t.Fatalf("expected intrinsic hover signature, got %q", got)
	}
	if got := hoverFor("parse_int"); !strings.Contains(got, "Result[int,Error]") {
		t.Fatalf("expected parse_int hover to include return type, got %q", got)
	}
}

func TestIndexDocSymbolsUsesSharedDeclarationSurface(t *testing.T) {
	src := "fn add(a: int, b: int): int { return a + b; }\nstruct User { name: string; }\n"
	got := indexDocSymbols(src)
	if len(got) != 2 {
		t.Fatalf("unexpected document symbol count: got %d want 2", len(got))
	}
	if got[0].Name != "add" || got[0].Kind != 12 {
		t.Fatalf("unexpected function symbol: %#v", got[0])
	}
	if got[1].Name != "User" || got[1].Kind != 23 {
		t.Fatalf("unexpected struct symbol: %#v", got[1])
	}
}
