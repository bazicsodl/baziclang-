package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"baziclang/internal/pkgm"
)

func TestCompileEntryToGoWithImport(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	lib := `fn inc(x: int): int { return x + 1; }`
	main := `import "./lib/main.bz";
fn main(): void {
    println(inc(41));
}`
	if err := os.WriteFile(filepath.Join(libDir, "main.bz"), []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := CompileEntryToGo(mainPath)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(out, "func inc(") || !strings.Contains(out, "func main(") {
		t.Fatalf("generated code missing expected functions:\n%s", out)
	}
}

func TestCheckEntryGenericStructEnumInterfaceImpl(t *testing.T) {
	dir := t.TempDir()
	src := `struct User {
    name: string;
    age: int;
}

struct Box[T] {
    value: T;
}

interface Named {
    fn label(self: User): string;
}

impl User: Named;

enum Role { Guest, Admin }

fn identity[T](x: T): T { return x; }
fn User_label(self: User): string { return self.name; }

fn main(): void {
    let u = User { name: "A", age: 1 };
    println(u.name);
    let b = Box[int] { value: 7 };
    println(b.value);
    let n = identity(7);
    println(n);
    println(User_label(u));
    let r: Role = Admin;
    println(r);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestCompileEntryToLLVM(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void { println("ok"); }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "define i32 @main(") {
		t.Fatalf("expected main definition in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMEmitsNonMainFunction(t *testing.T) {
	dir := t.TempDir()
	src := `fn addOne(x: int): int { return x + 1; }
fn main(): void { println(addOne(2)); }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "define i64 @addOne(i64 %x)") {
		t.Fatalf("expected addOne signature in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "add i64") {
		t.Fatalf("expected add instruction in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "ret i64") {
		t.Fatalf("expected typed return in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMEmitsComparisonAndLogicalOps(t *testing.T) {
	dir := t.TempDir()
	src := `fn cmp(a: int, b: int): bool { return (a > b) && (a != 0); }
fn main(): void { println(cmp(2, 1)); }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "icmp sgt i64") {
		t.Fatalf("expected signed int comparison in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "icmp ne i64") {
		t.Fatalf("expected int inequality compare in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "and i8") && !strings.Contains(out, "and i1") {
		t.Fatalf("expected logical and lowering in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMEmitsFunctionCall(t *testing.T) {
	dir := t.TempDir()
	src := `fn inc(x: int): int { return x + 1; }
fn useInc(v: int): int { return inc(v); }
fn main(): void { println(useInc(3)); }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "call i64 @inc(i64") {
		t.Fatalf("expected function call lowering in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMEmitsControlFlowAndMatch(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn score(r: Role): int {
    let s: int = 0;
    if r == Admin {
        s = 2;
    } else {
        s = 1;
    }
    let i: int = 0;
    while i < 2 {
        i = i + 1;
    }
    match r {
        Guest: { s = s + 1; }
        Admin: { s = s + 2; }
    }
    return s;
}

fn main(): void { return; }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "alloca i64") {
		t.Fatalf("expected local allocas in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "br i1") {
		t.Fatalf("expected conditional branch in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "icmp eq i64") {
		t.Fatalf("expected enum equality compare in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "switch i64") {
		t.Fatalf("expected match switch in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMLowersPrintln(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    println("hi");
    print(2);
    println(true);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "@printf") {
		t.Fatalf("expected printf calls in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "getelementptr inbounds") {
		t.Fatalf("expected string literal gep in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "true") || !strings.Contains(out, "false") {
		t.Fatalf("expected bool string globals in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMLowersStructFieldAccess(t *testing.T) {
	dir := t.TempDir()
	src := `struct User { name: string; age: int; }

fn main(): void {
    let u = User { name: "A", age: 2 };
    println(u.name);
    println(u.age);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "%User = type { ptr, i64 }") {
		t.Fatalf("expected struct type definition in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "extractvalue %User") {
		t.Fatalf("expected field extractvalue in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMMonomorphizesGenericStructAndFunc(t *testing.T) {
	dir := t.TempDir()
	src := `struct Box[T] { value: T; }

fn identity[T](x: T): T { return x; }

fn main(): void {
    let b = Box[int] { value: 7 };
    let v = identity(3);
    println(b.value);
    println(v);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "%Box__int = type { i64 }") {
		t.Fatalf("expected monomorphized Box__int struct in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "define i64 @identity__int") {
		t.Fatalf("expected monomorphized identity__int function in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMLowersStringOps(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let a = "hi";
    let b = "there";
    let c = a + b;
    println(c);
    println(a == b);
    println(a < b);
    println(len(a));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "@bazic_str_concat") {
		t.Fatalf("expected string concat runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "@bazic_str_cmp") {
		t.Fatalf("expected string compare runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "call i64 @strlen") {
		t.Fatalf("expected strlen call in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMLowersStringBuiltins(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    println(contains("bazic", "zi"));
    println(starts_with("bazic", "ba"));
    println(ends_with("bazic", "ic"));
    println(to_upper("BaZiC"));
    println(to_lower("BaZiC"));
    println(trim_space("  bazic  "));
    println(replace("bazic", "zi", "za"));
    println(repeat("ba", 3));
    println(str(123));
    println(str(true));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "@bazic_contains") || !strings.Contains(out, "@bazic_starts_with") || !strings.Contains(out, "@bazic_ends_with") {
		t.Fatalf("expected string predicate runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "@bazic_to_upper") || !strings.Contains(out, "@bazic_to_lower") {
		t.Fatalf("expected case runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "@bazic_trim_space") || !strings.Contains(out, "@bazic_replace") || !strings.Contains(out, "@bazic_repeat") {
		t.Fatalf("expected string helper runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "@bazic_int_to_str") {
		t.Fatalf("expected int->str runtime in llvm output, got:\n%s", out)
	}
}

func TestCompileEntryToLLVMLowersParseBuiltins(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let a = parse_int("12");
    let b = parse_float("3.5");
    println(a.is_ok);
    println(b.is_ok);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "@bazic_parse_int") || !strings.Contains(out, "@bazic_parse_float") {
		t.Fatalf("expected parse runtime in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "%Result__int__Error") || !strings.Contains(out, "%Result__float__Error") {
		t.Fatalf("expected Result monomorph types in llvm output, got:\n%s", out)
	}
}

func TestMatchGuardsAreAllowedAndExhaustive(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn main(): void {
    let r: Role = Admin;
    match r {
        Admin if true: { println("admin"); }
        Admin: { println("admin fallback"); }
        Guest: { println("guest"); }
    }
    let label = match r {
        Admin if false: "x",
        Admin: "admin",
        Guest: "guest",
    };
    println(label);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestGenericConstraintsEnforced(t *testing.T) {
	dir := t.TempDir()
	src := `struct User { name: string; }
struct Item { id: int; }

interface Named { fn label(self: User): string; }
impl User: Named;

fn User_label(self: User): string { return self.name; }

fn pick[T: Named](x: T): T { return x; }

fn main(): void {
    let u = User { name: "ok" };
    let _ = pick(u);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("expected constraints to pass, got: %v", err)
	}
	bad := `struct User { name: string; }
struct Item { id: int; }

interface Named { fn label(self: User): string; }
impl User: Named;

fn User_label(self: User): string { return self.name; }

fn pick[T: Named](x: T): T { return x; }

fn main(): void {
    let i = Item { id: 1 };
    let _ = pick(i);
}`
	badPath := filepath.Join(dir, "bad.bz")
	if err := os.WriteFile(badPath, []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(badPath); err == nil {
		t.Fatalf("expected constraint failure, got nil")
	}
}

func TestCompileEntryToLLVMLowersInterfaceType(t *testing.T) {
	dir := t.TempDir()
	src := `struct User { name: string; }

interface Named { fn label(self: User): string; }

impl User: Named;

fn User_label(self: User): string { return self.name; }

fn pass(n: Named): Named { return n; }

fn main(): void { return; }`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := CompileEntryToLLVM(entry)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(out, "%Named = type { ptr, ptr }") {
		t.Fatalf("expected interface type lowering in llvm output, got:\n%s", out)
	}
	if !strings.Contains(out, "define %Named @pass(%Named %n)") {
		t.Fatalf("expected interface param/return in llvm output, got:\n%s", out)
	}
}

func TestCheckEntryReportsTypeError(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let x: int = "bad";
    println(x);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err == nil {
		t.Fatalf("expected type error, got nil")
	}
}

func TestMethodCallSyntaxResolvesToStructMethodFunction(t *testing.T) {
	dir := t.TempDir()
	src := `struct User {
    name: string;
}

interface Named {
    fn label(self: User): string;
}

impl User: Named;

fn User_label(self: User): string {
    return self.name;
}

fn main(): void {
    let u = User { name: "Bazic" };
    println(u.label());
    println(User_label(u));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	goOut, err := CompileEntryToGo(entry)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(goOut, "println(User_label(u))") {
		t.Fatalf("expected method call to compile to function call, got:\n%s", goOut)
	}
}

func TestMethodCallSyntaxReportsUnknownMethod(t *testing.T) {
	dir := t.TempDir()
	src := `struct User {
    name: string;
}

fn main(): void {
    let u = User { name: "Bazic" };
    println(u.missing());
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected type error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown method 'missing'") {
		t.Fatalf("expected unknown method error, got: %v", err)
	}
}

func TestSafetyPreludeOptionResultErrorAvailable(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let o = some(7);
    let missing = none(0);
    let r = err(0, Error { message: "boom" });
    let okv = ok(9, Error { message: "" });
    println(o.is_some);
    println(missing.is_some);
    println(r.err.message);
    println(okv.value);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	goOut, err := CompileEntryToGo(entry)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(goOut, "type Option[T any] struct") {
		t.Fatalf("expected prelude Option type in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "type Result[T any, E any] struct") {
		t.Fatalf("expected prelude Result type in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "type Error struct") {
		t.Fatalf("expected prelude Error type in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "func some[T any](value T) Option[T]") {
		t.Fatalf("expected prelude some() helper in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "func none[T any](fallback T) Option[T]") {
		t.Fatalf("expected prelude none() helper in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "func ok[T any, E any](value T, fallback_err E) Result[T, E]") {
		t.Fatalf("expected prelude ok() helper in generated code, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "func err[T any, E any](fallback_value T, err_value E) Result[T, E]") {
		t.Fatalf("expected prelude err() helper in generated code, got:\n%s", goOut)
	}
}

func TestNilLiteralPolicyRejectsNil(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let bad = nil;
    println(bad);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected nil policy error, got nil")
	}
	if !strings.Contains(err.Error(), "'nil' is not a value in Bazic") {
		t.Fatalf("expected nil policy guidance, got: %v", err)
	}
}

func TestMatchEnumExhaustiveCheckAndCodegen(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn main(): void {
    let r: Role = Admin;
    match r {
        Guest: { println("guest"); }
        Admin: { println("admin"); }
    }
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	goOut, err := CompileEntryToGo(entry)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(goOut, "switch r {") {
		t.Fatalf("expected switch generation for match, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "case Guest:") || !strings.Contains(goOut, "case Admin:") {
		t.Fatalf("expected match arms in generated switch, got:\n%s", goOut)
	}
}

func TestMatchEnumRequiresExhaustiveCoverage(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn main(): void {
    let r: Role = Admin;
    match r {
        Guest: { println("guest"); }
    }
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected non-exhaustive match error, got nil")
	}
	if !strings.Contains(err.Error(), "non-exhaustive match for enum 'Role'") {
		t.Fatalf("expected non-exhaustive error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Admin") {
		t.Fatalf("expected missing variant in error, got: %v", err)
	}
}

func TestMatchExpressionTypeAndCodegen(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn main(): void {
    let role: Role = Admin;
    let label = match role {
        Guest: "guest",
        Admin: "admin",
    };
    println(label);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	goOut, err := CompileEntryToGo(entry)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if !strings.Contains(goOut, "var label string = func() string {") {
		t.Fatalf("expected typed match expression codegen, got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "case Guest:") || !strings.Contains(goOut, "return \"guest\"") {
		t.Fatalf("expected match expression arms in generated code, got:\n%s", goOut)
	}
}

func TestMatchExpressionArmTypeMismatch(t *testing.T) {
	dir := t.TempDir()
	src := `enum Role { Guest, Admin }

fn main(): void {
    let role: Role = Admin;
    let x = match role {
        Guest: "guest",
        Admin: 1,
    };
    println(x);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected match expression arm type mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "match expression arm type mismatch") {
		t.Fatalf("expected arm type mismatch error, got: %v", err)
	}
}

func TestCheckEntryParseErrorIncludesSourceSnippet(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let x = ;
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "-->") || !strings.Contains(msg, "^") {
		t.Fatalf("expected source pointer diagnostics, got: %v", err)
	}
	if !strings.Contains(msg, "let x = ;") {
		t.Fatalf("expected failing source line in diagnostics, got: %v", err)
	}
}

func TestCheckEntryLexErrorIncludesSourceSnippet(t *testing.T) {
	dir := t.TempDir()
	src := "fn main(): void { let x = @; }\n"
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected lex error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "-->") || !strings.Contains(msg, "^") {
		t.Fatalf("expected source pointer diagnostics, got: %v", err)
	}
	if !strings.Contains(msg, "let x = @;") {
		t.Fatalf("expected failing source line in diagnostics, got: %v", err)
	}
}

func TestCheckEntryRejectsUnusedLocalVariable(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let x = 1;
    println("ok");
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected unused variable error")
	}
	if !strings.Contains(err.Error(), "unused variable 'x'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsUnusedParameter(t *testing.T) {
	dir := t.TempDir()
	src := `fn helper(x: int): int {
    return 1;
}
fn main(): void {
    println(helper(1));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected unused parameter error")
	}
	if !strings.Contains(err.Error(), "unused variable 'x'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAllowsDiscardUnderscoreBinding(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let _ = 1;
    println("ok");
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("expected underscore discard to pass, got: %v", err)
	}
}

func TestCheckEntryRejectsMissingReturnPath(t *testing.T) {
	dir := t.TempDir()
	src := `fn maybe(flag: bool): int {
    if flag {
        return 1;
    }
}
fn main(): void {
    println(maybe(true));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected missing-return error")
	}
	if !strings.Contains(err.Error(), "missing return on some control paths") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAcceptsAllPathsReturning(t *testing.T) {
	dir := t.TempDir()
	src := `fn decide(flag: bool): int {
    if flag {
        return 1;
    } else {
        return 2;
    }
}
fn main(): void {
    println(decide(true));
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("expected full-return function to pass, got: %v", err)
	}
}

func TestBuildArgsAreDeterministic(t *testing.T) {
	args := buildArgs("out.exe", "tmp/main.go")
	expected := []string{"build", "-trimpath", "-ldflags", "-buildid=", "-o", "out.exe", "tmp/main.go"}
	if len(args) != len(expected) {
		t.Fatalf("unexpected args length: got %d want %d (%v)", len(args), len(expected), args)
	}
	for i := range expected {
		if args[i] != expected[i] {
			t.Fatalf("arg %d mismatch: got %q want %q (full args: %v)", i, args[i], expected[i], args)
		}
	}
}

func TestCheckEntryFailsOnPackageIntegrityMismatch(t *testing.T) {
	root := t.TempDir()
	depA := t.TempDir()
	depB := t.TempDir()
	if err := os.WriteFile(filepath.Join(depA, "main.bz"), []byte(`fn util(): int { return 1; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depB, "main.bz"), []byte(`fn util(): int { return 2; }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depA); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	// Change manifest dependency source without syncing lock/cache.
	if err := pkgm.AddDep(root, "util", depB); err != nil {
		t.Fatalf("update dep: %v", err)
	}
	mainSrc := `fn main(): void { println("ok"); }`
	entry := filepath.Join(root, "main.bz")
	if err := os.WriteFile(entry, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected package integrity failure")
	}
	if !strings.Contains(err.Error(), "package integrity check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "source path mismatch") {
		t.Fatalf("expected source path mismatch detail, got: %v", err)
	}
}

func TestCheckEntryRejectsMainWithParams(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(x: int): void {
    println(x);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected main signature error")
	}
	if !strings.Contains(err.Error(), "'main' must not take parameters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsMainNonVoid(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): int {
    return 0;
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected main signature error")
	}
	if !strings.Contains(err.Error(), "'main' must return void") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsGenericMain(t *testing.T) {
	dir := t.TempDir()
	src := `fn main[T](): void {
    return;
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected main signature error")
	}
	if !strings.Contains(err.Error(), "'main' cannot be generic") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryDetectsImportCycle(t *testing.T) {
	root := t.TempDir()
	aSrc := `import "./b.bz";
fn helperA(): int { return helperB(); }`
	bSrc := `import "./a.bz";
fn helperB(): int { return helperA(); }`
	mainSrc := `import "./a.bz";
fn main(): void { println("ok"); }`
	aPath := filepath.Join(root, "a.bz")
	bPath := filepath.Join(root, "b.bz")
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(aPath, []byte(aSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(bSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	if !strings.Contains(err.Error(), "import cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "a.bz -> b.bz -> a.bz") {
		t.Fatalf("expected cycle chain in error, got: %v", err)
	}
}

func TestCheckEntrySuggestsUnknownIdentifier(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
    let count = 1;
    println(coutn);
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected unknown identifier error")
	}
	if !strings.Contains(err.Error(), "did you mean 'count'") {
		t.Fatalf("expected suggestion for count, got: %v", err)
	}
}

func TestCheckEntrySuggestsUnknownFunction(t *testing.T) {
	dir := t.TempDir()
	src := `fn helper(): int { return 1; }
fn main(): void {
    println(helpr());
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected unknown function error")
	}
	if !strings.Contains(err.Error(), "did you mean 'helper'") {
		t.Fatalf("expected suggestion for helper, got: %v", err)
	}
}
