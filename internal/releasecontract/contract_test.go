package releasecontract

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAlphaStdlibModulesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for tier, modules := range map[string][]string{
		string(StdlibTierStable):       AlphaStableStdModules(),
		string(StdlibTierExperimental): AlphaExperimentalStdModules(),
	} {
		for _, mod := range modules {
			if prev, ok := seen[mod]; ok {
				t.Fatalf("module %q appears in both %s and %s tiers", mod, prev, tier)
			}
			seen[mod] = tier
		}
	}
}

func TestAlphaStdlibManifestMatchesTierHelpers(t *testing.T) {
	stable := 0
	experimental := 0
	for _, mod := range AlphaStdlibModules() {
		switch mod.Tier {
		case StdlibTierStable:
			stable++
		case StdlibTierExperimental:
			experimental++
		default:
			t.Fatalf("unexpected tier for %q: %q", mod.Name, mod.Tier)
		}
	}
	if stable != len(AlphaStableStdModules()) {
		t.Fatalf("stable module count mismatch: manifest=%d helper=%d", stable, len(AlphaStableStdModules()))
	}
	if experimental != len(AlphaExperimentalStdModules()) {
		t.Fatalf("experimental module count mismatch: manifest=%d helper=%d", experimental, len(AlphaExperimentalStdModules()))
	}
}

func TestContractDocsMentionSharedAlphaSurface(t *testing.T) {
	t.Parallel()

	checks := []struct {
		path             string
		wants            []string
		stableMods       bool
		experimentalMods bool
	}{
		{
			path: "ALPHA_SCOPE.md",
			wants: []string{
				"Stable default backend: Go",
				"Experimental backend: LLVM",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "STDLIB_TIERS.md",
			wants: []string{
				"## Tier 1: Alpha Stable Core",
				"## Tier 2: Alpha Experimental",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "RELEASE_SCOPE.md",
			wants: []string{
				"LLVM backend status: experimental",
				"Stable core for v1:",
				"Experimental or lower-confidence areas for v1:",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "README.md",
			wants: []string{
				"See `RUNTIME_CONTRACT.md` for the compiler/runtime boundary and supported implicit surface.",
				"Note: Bazic is currently on an alpha release track.",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "RUNTIME_CONTRACT.md",
			wants: []string{
				"## Compiler-Owned Implicit Surface",
				"## Runtime-Owned Implementation Surface",
				"## Stdlib-Owned Package Surface",
				"## Backend And Stability Policy",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "std/README.md",
			wants: []string{
				"Alpha stable core is the supported Bazic alpha stdlib surface on the Go backend.",
				"Alpha experimental modules exist, but they are not part of the supported alpha core:",
			},
			stableMods:       true,
			experimentalMods: true,
		},
		{
			path: "V1_GUIDE.md",
			wants: []string{
				"**Backend services** on the Go backend release path.",
				"**Web** as an experimental WASM workload via the Go backend.",
				"**Desktop** as an experimental surface, not part of the stable alpha core.",
				"Native (Go backend release path)",
				"Native (LLVM experimental)",
			},
			stableMods:       true,
			experimentalMods: true,
		},
	}

	for _, tc := range checks {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			text := readRepoDoc(t, tc.path)
			for _, want := range tc.wants {
				if !strings.Contains(text, want) {
					t.Fatalf("%s missing %q", tc.path, want)
				}
			}
			if tc.stableMods {
				for _, mod := range AlphaStableStdModules() {
					want := "`std/" + mod + "`"
					if !strings.Contains(text, want) {
						t.Fatalf("%s missing stable module %q", tc.path, want)
					}
				}
			}
			if tc.experimentalMods {
				for _, mod := range AlphaExperimentalStdModules() {
					want := "`std/" + mod + "`"
					if !strings.Contains(text, want) {
						t.Fatalf("%s missing experimental module %q", tc.path, want)
					}
				}
			}
		})
	}
}

func readRepoDoc(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}
