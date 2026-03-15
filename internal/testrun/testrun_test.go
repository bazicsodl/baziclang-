package testrun

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesAllTests(t *testing.T) {
	dir := t.TempDir()
	src := `fn test_true(): bool { return true; }
fn test_math(): bool { return (1 + 1) == 2; }
fn test_assertion_style(): void { assert(true); }`
	file := filepath.Join(dir, "sample_test.bz")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(file)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if res.Total != 3 || res.Passed != 3 {
		t.Fatalf("expected 3/3 passed, got %d/%d", res.Passed, res.Total)
	}
}

func TestRunFailsWhenATestFails(t *testing.T) {
	dir := t.TempDir()
	src := `fn test_true(): bool { return true; }
fn test_fail(): bool { return false; }`
	file := filepath.Join(dir, "sample_test.bz")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(file)
	if err == nil {
		t.Fatalf("expected failure error")
	}
	if !strings.Contains(err.Error(), "bazic tests failed") {
		t.Fatalf("expected failure summary error, got: %v", err)
	}
	if res.Total != 2 || res.Passed != 1 {
		t.Fatalf("expected 1/2 passed, got %d/%d", res.Passed, res.Total)
	}
}

func TestRunRejectsInvalidTestSignature(t *testing.T) {
	dir := t.TempDir()
	src := `fn test_bad(x: int): bool { return true; }`
	file := filepath.Join(dir, "sample_test.bz")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(file)
	if err == nil {
		t.Fatalf("expected invalid signature error")
	}
	if !strings.Contains(err.Error(), "invalid test function signature") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFailsVoidAssertionStyleTest(t *testing.T) {
	dir := t.TempDir()
	src := `fn test_assertion_style(): void {
    assert(false);
}`
	file := filepath.Join(dir, "sample_test.bz")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(file)
	if err == nil {
		t.Fatalf("expected failure for assertion-style test")
	}
	if res.Total != 1 || res.Passed != 0 {
		t.Fatalf("expected 0/1 passed, got %d/%d", res.Passed, res.Total)
	}
}

func TestRunReportsAssertionMessage(t *testing.T) {
	dir := t.TempDir()
	src := `fn test_assertion_style(): void {
    assert_msg(false, "expected admin label");
}`
	file := filepath.Join(dir, "sample_test.bz")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(file)
	if err == nil {
		t.Fatalf("expected failure for assertion message test")
	}
	if !strings.Contains(res.Files[0].Output, "expected admin label") {
		t.Fatalf("expected assertion message in output, got:\n%s", res.Files[0].Output)
	}
}

func TestConformanceSuiteGoBackend(t *testing.T) {
	if os.Getenv("BAZIC_RUN_CONFORMANCE") == "" {
		t.Skip("BAZIC_RUN_CONFORMANCE not set; skipping conformance suite")
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	target := filepath.Join(root, "conformance")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("conformance suite missing: %v", err)
	}
	if _, err := RunWithOptions(target, Options{Backend: "go"}); err != nil {
		t.Fatalf("conformance suite failed: %v", err)
	}
}

func TestConformanceSuiteLLVMBackend(t *testing.T) {
	if os.Getenv("BAZIC_RUN_CONFORMANCE") == "" {
		t.Skip("BAZIC_RUN_CONFORMANCE not set; skipping conformance suite")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not found; skipping llvm conformance")
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	target := filepath.Join(root, "conformance")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("conformance suite missing: %v", err)
	}
	if _, err := RunWithOptions(target, Options{Backend: "llvm"}); err != nil {
		t.Fatalf("llvm conformance failed: %v", err)
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	curr := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			return curr, nil
		}
		next := filepath.Dir(curr)
		if next == curr {
			break
		}
		curr = next
	}
	return "", os.ErrNotExist
}
