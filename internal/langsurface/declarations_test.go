package langsurface

import "testing"

func TestFindDeclSymbolsTracksTopLevelDeclarationKinds(t *testing.T) {
	src := `
fn add(a: int, b: int): int { return a + b; }
struct User { name: string; }
enum Role { Guest, Admin }
interface Named { fn label(self: User): string; }
let answer = 42;
`
	got := FindDeclSymbols(src)
	want := []struct {
		name string
		kind DeclKind
	}{
		{name: "add", kind: DeclKindFunction},
		{name: "User", kind: DeclKindStruct},
		{name: "Role", kind: DeclKindEnum},
		{name: "Named", kind: DeclKindInterface},
		{name: "label", kind: DeclKindFunction},
		{name: "answer", kind: DeclKindLet},
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected decl symbol count: got %d want %d", len(got), len(want))
	}
	for i, sym := range got {
		if sym.Name != want[i].name || sym.Kind != want[i].kind {
			t.Fatalf("decl symbol %d mismatch: got %#v want name=%q kind=%q", i, sym, want[i].name, want[i].kind)
		}
		if sym.Start >= sym.End {
			t.Fatalf("expected valid symbol span for %#v", sym)
		}
	}
}

func TestDeclKindLSPCompletionOrSymbolKind(t *testing.T) {
	cases := map[DeclKind]int{
		DeclKindFunction:  12,
		DeclKindStruct:    23,
		DeclKindEnum:      10,
		DeclKindInterface: 11,
		DeclKindLet:       13,
	}
	for kind, want := range cases {
		if got := kind.LSPCompletionOrSymbolKind(); got != want {
			t.Fatalf("unexpected LSP kind for %q: got %d want %d", kind, got, want)
		}
	}
}
