package bazlint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintRulesTrigger(t *testing.T) {
	dir := t.TempDir()
	src := `fn BadName(): void {
    print("hi");
    if true { println("x"); }
    while false { println("y"); }
    __std_read_file("x");
}`
	path := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	issues, err := Lint(dir)
	if err != nil {
		t.Fatalf("lint error: %v", err)
	}
	rules := map[string]int{}
	for _, iss := range issues {
		rules[iss.Rule]++
	}
	for _, rule := range []string{"BL001", "BL002", "BL003", "BL004", "BL005"} {
		if rules[rule] == 0 {
			t.Fatalf("expected rule %s to trigger; got: %+v", rule, rules)
		}
	}
}

func TestLintSkipsStdInternalCalls(t *testing.T) {
	dir := t.TempDir()
	stdDir := filepath.Join(dir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatal(err)
	}
	src := `fn helper(): void { __std_read_file("x"); }`
	path := filepath.Join(stdDir, "io.bz")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	issues, err := Lint(dir)
	if err != nil {
		t.Fatalf("lint error: %v", err)
	}
	for _, iss := range issues {
		if iss.Rule == "BL002" && strings.Contains(iss.File, string(filepath.Separator)+"std"+string(filepath.Separator)) {
			t.Fatalf("BL002 should be skipped in std path, got: %+v", iss)
		}
	}
}
