package langsurface

import "testing"

func TestLookupSurfaceSymbolCoversKeywordsAndIntrinsics(t *testing.T) {
	cases := []struct {
		name string
		kind int
	}{
		{name: "package", kind: 14},
		{name: "if", kind: 14},
		{name: "true", kind: 12},
		{name: "assert_msg", kind: 3},
		{name: "parse_int", kind: 3},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			spec, ok := LookupSurfaceSymbol(tc.name)
			if !ok {
				t.Fatalf("missing surface symbol %q", tc.name)
			}
			if spec.CompletionKind != tc.kind {
				t.Fatalf("unexpected completion kind for %q: got %d want %d", tc.name, spec.CompletionKind, tc.kind)
			}
			if spec.Hover == "" {
				t.Fatalf("expected hover for %q", tc.name)
			}
		})
	}
}

func TestSurfaceSymbolsExcludeInternalIntrinsics(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range SurfaceSymbols() {
		if seen[spec.Name] {
			t.Fatalf("duplicate surface symbol %q", spec.Name)
		}
		seen[spec.Name] = true
	}
	for _, unwanted := range []string{"__std_read_file", "http_serve_app", "ui_element"} {
		if seen[unwanted] {
			t.Fatalf("did not expect internal or stdlib symbol %q in surface", unwanted)
		}
	}
}
