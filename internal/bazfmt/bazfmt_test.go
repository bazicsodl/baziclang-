package bazfmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatCanonicalizesProgram(t *testing.T) {
	src := `struct User{name:string;age:int;}
fn main():void{let u=User{name:"Ipeh",age:27};let s=match Admin{Guest:"guest",Admin:"admin"};println(u.name);println(s);}`
	out, err := Format(src)
	if err != nil {
		t.Fatalf("format failed: %v", err)
	}
	if !strings.Contains(out, "struct User {") {
		t.Fatalf("expected canonical struct formatting, got:\n%s", out)
	}
	if !strings.Contains(out, "fn main(): void {") {
		t.Fatalf("expected canonical fn signature formatting, got:\n%s", out)
	}
	if !strings.Contains(out, "let s = match Admin { Guest: \"guest\", Admin: \"admin\" }") {
		t.Fatalf("expected canonical match expression formatting, got:\n%s", out)
	}
}

func TestFormatIsIdempotent(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn main(): void {
    let role: Role = Admin
    match role {
        Guest: { println("guest") }
        Admin: { println("admin") }
    }
}`
	once, err := Format(src)
	if err != nil {
		t.Fatalf("first format failed: %v", err)
	}
	twice, err := Format(once)
	if err != nil {
		t.Fatalf("second format failed: %v", err)
	}
	if once != twice {
		t.Fatalf("formatter is not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestFormatConstAndFieldAssign(t *testing.T) {
	src := `struct User{name:string;}
fn main():void{const u=User{name:"a"};u.name="b"}`
	out, err := Format(src)
	if err != nil {
		t.Fatalf("format failed: %v", err)
	}
	if !strings.Contains(out, "const u = User {") && !strings.Contains(out, "const u = User{") {
		t.Fatalf("expected const formatting, got:\n%s", out)
	}
	if !strings.Contains(out, "u.name = \"b\"") {
		t.Fatalf("expected field assignment formatting, got:\n%s", out)
	}
}

func TestCollectBZFilesRecursiveAndSkipsInternalDirs(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(path, body string) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(root, "a.bz"), "fn main(): void {}")
	mustWrite(filepath.Join(root, "sub", "b.bz"), "fn main(): void {}")
	mustWrite(filepath.Join(root, ".bazic", "pkg", "ignored.bz"), "fn main(): void {}")
	mustWrite(filepath.Join(root, ".git", "ignored2.bz"), "fn main(): void {}")
	mustWrite(filepath.Join(root, ".bazic_test_harness.bz"), "fn main(): void {}")
	mustWrite(filepath.Join(root, "note.txt"), "x")

	files, err := CollectBZFiles(root)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if strings.Contains(f, ".bazic") || strings.Contains(f, ".git") || strings.Contains(f, ".bazic_test_harness") {
			t.Fatalf("unexpected internal file in result: %s", f)
		}
	}
}
