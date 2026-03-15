package pkgm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAddSyncResolve(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}

	resolved, err := ResolveImport(root, root, "util")
	if err != nil {
		t.Fatalf("resolve import: %v", err)
	}
	if filepath.Base(resolved) != "main.bz" {
		t.Fatalf("expected main.bz target, got %s", resolved)
	}
	if _, err := os.Stat(filepath.Join(root, ".bazic", "pkg", "util", "main.bz")); err != nil {
		t.Fatalf("synced package missing: %v", err)
	}
	lock, err := LoadLockfile(root)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if lock.Version != 2 {
		t.Fatalf("expected lockfile version 2, got %d", lock.Version)
	}
	if lock.SigningPublicKey == "" {
		t.Fatalf("expected signing public key in lockfile")
	}
	locked, ok := lock.Deps["util"]
	if !ok {
		t.Fatalf("expected util entry in lockfile")
	}
	if locked.Integrity.Checksum == "" {
		t.Fatalf("expected checksum for util in lockfile")
	}
	if locked.Signature == "" {
		t.Fatalf("expected signature for util in lockfile")
	}
}

func TestResolveImportRejectsAbsoluteAndMissingLock(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if _, err := ResolveImport(root, root, filepath.Join(depRoot, "main.bz")); err == nil {
		t.Fatalf("expected absolute import rejection")
	}
	if _, err := ResolveImport(root, root, "util"); err == nil {
		t.Fatalf("expected lockfile error before sync")
	}
}

func TestResolveImportDetectsTampering(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	cachedMain := filepath.Join(root, ".bazic", "pkg", "util", "main.bz")
	if err := os.WriteFile(cachedMain, []byte(`fn helper(): int { return 2; }`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveImport(root, root, "util")
	if err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch message, got: %v", err)
	}
}

func TestAddDepRejectsBadAlias(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "../bad", depRoot); err == nil {
		t.Fatalf("expected bad alias error")
	}
}

func TestVerifySuccessAndDetectsDrift(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := Verify(root); err != nil {
		t.Fatalf("verify should pass after sync: %v", err)
	}
	m, err := LoadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	delete(m.Deps, "util")
	if err := SaveManifest(root, m); err != nil {
		t.Fatal(err)
	}
	err = Verify(root)
	if err == nil {
		t.Fatalf("expected verify to fail on stale lock dep")
	}
	if !strings.Contains(err.Error(), "stale dependency") {
		t.Fatalf("expected stale dependency error, got: %v", err)
	}
}

func TestVerifyDetectsSourcePathMismatch(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	otherDepRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDepRoot, "main.bz"), []byte(`fn helper(): int { return 9; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := AddDep(root, "util", otherDepRoot); err != nil {
		t.Fatalf("update dep source: %v", err)
	}
	err := Verify(root)
	if err == nil {
		t.Fatalf("expected verify to fail on source path mismatch")
	}
	if !strings.Contains(err.Error(), "source path mismatch") {
		t.Fatalf("expected source path mismatch error, got: %v", err)
	}
}

func TestVerifyDetectsSourceDriftWithoutSync(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 9; }`), 0644); err != nil {
		t.Fatal(err)
	}
	err := Verify(root)
	if err == nil {
		t.Fatalf("expected source drift error")
	}
	if !strings.Contains(err.Error(), "source drift for package 'util'") {
		t.Fatalf("expected source drift message, got: %v", err)
	}
}

func TestWriteSBOM(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(`fn helper(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	outPath := filepath.Join(root, "bazic.sbom.json")
	if err := WriteSBOM(root, outPath); err != nil {
		t.Fatalf("write sbom: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read sbom: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "\"deps\"") || !strings.Contains(got, "\"util\"") {
		t.Fatalf("expected sbom to include util dep, got:\n%s", got)
	}
}
