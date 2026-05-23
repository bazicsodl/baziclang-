package perfcontract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBenchManifestValid(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "bench", "manifest.json")
	manifest, err := LoadBenchManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadBenchManifest: %v", err)
	}
	if manifest.BaselineThresholdPercent <= 0 {
		t.Fatalf("expected positive baseline threshold, got %d", manifest.BaselineThresholdPercent)
	}
	if len(manifest.Benchmarks) == 0 {
		t.Fatal("expected at least one benchmark entry")
	}

	seen := map[string]struct{}{}
	for _, entry := range manifest.Benchmarks {
		if entry.Name == "" {
			t.Fatal("benchmark entry missing name")
		}
		if _, ok := seen[entry.Name]; ok {
			t.Fatalf("duplicate benchmark name %q", entry.Name)
		}
		seen[entry.Name] = struct{}{}

		if entry.Path == "" {
			t.Fatalf("benchmark %q missing path", entry.Name)
		}
		fullPath := filepath.Join("..", "..", entry.Path)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("benchmark %q path %q: %v", entry.Name, fullPath, err)
		}
		if entry.LLVMRatioTarget <= 0 {
			t.Fatalf("benchmark %q has non-positive llvm_ratio_target %v", entry.Name, entry.LLVMRatioTarget)
		}
	}
}
