package langsurface

import "testing"

func TestKeywordSpecsAreUniqueAndLookupBacked(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range KeywordSpecs() {
		if seen[spec.Name] {
			t.Fatalf("duplicate keyword spec %q", spec.Name)
		}
		seen[spec.Name] = true
		got, ok := LookupKeyword(spec.Name)
		if !ok {
			t.Fatalf("lookup missing keyword %q", spec.Name)
		}
		if got != spec {
			t.Fatalf("lookup mismatch for %q: got %#v want %#v", spec.Name, got, spec)
		}
	}
}

func TestKeywordSpecsCoverEditorFacingSurface(t *testing.T) {
	for _, name := range []string{"package", "pub", "if", "else", "match", "while", "true", "false"} {
		spec, ok := LookupKeyword(name)
		if !ok {
			t.Fatalf("missing keyword %q", name)
		}
		if spec.Hover == "" {
			t.Fatalf("expected hover for %q", name)
		}
		if spec.CompletionKind == 0 {
			t.Fatalf("expected completion kind for %q", name)
		}
	}
}
