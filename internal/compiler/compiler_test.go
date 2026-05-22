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
	if !strings.Contains(goOut, "User_label(u)") {
		t.Fatalf("expected method call to lower through User_label(u), got:\n%s", goOut)
	}
	if !strings.Contains(goOut, "println(arg__mir") && !strings.Contains(goOut, "println(User_label(u))") {
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

func TestCompileToGoEmitsHttpServeAppHelperOnlyOnce(t *testing.T) {
	src := `struct ServerRequest {
    method: string;
    path: string;
    query: string;
    headers: string;
    cookies: string;
    body: string;
}

struct ServerResponse {
    status: int;
    headers: string;
    body: string;
}

fn GET_root(req: ServerRequest): ServerResponse {
    return ServerResponse { status: 200, headers: "", body: req.path };
}

fn main(): void {
    println("ok");
}`
	goOut, err := CompileToGo(src)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if got := strings.Count(goOut, "func __std_http_serve_app(addr string) Result[bool, Error]"); got != 1 {
		t.Fatalf("expected exactly one serve_app helper, got %d\n%s", got, goOut)
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
	if !strings.Contains(goOut, "var label string = func() string {") && !strings.Contains(goOut, "var let__mir") {
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
	if !strings.Contains(err.Error(), "type error at 2:") {
		t.Fatalf("expected span-aware semantic location, got: %v", err)
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
	if !strings.Contains(err.Error(), "type error at 3:") {
		t.Fatalf("expected span-aware semantic location, got: %v", err)
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

func TestCheckEntryRendersSemanticErrorFromImportedFile(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	lib := `fn helper(): void {
    println(coutn);
}`
	main := `import "./lib/main.bz";
fn main(): void {
    helper();
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected semantic error from imported file")
	}
	msg := err.Error()
	if !strings.Contains(msg, libPath) {
		t.Fatalf("expected imported file path in diagnostic, got: %v", err)
	}
	if !strings.Contains(msg, "println(coutn);") {
		t.Fatalf("expected imported file source line in diagnostic, got: %v", err)
	}
}

func TestCheckEntryRejectsImportedMainFunction(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	lib := `fn main(): void {
    println("bad");
}`
	main := `import "./lib/main.bz";
fn main(): void {
    println("ok");
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected imported main rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "imported files must not declare 'main'") {
		t.Fatalf("expected imported main diagnostic, got: %v", err)
	}
	if !strings.Contains(msg, libPath) {
		t.Fatalf("expected imported file path in diagnostic, got: %v", err)
	}
}

func TestCheckEntryRejectsDuplicateTopLevelSymbolAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	lib := `fn helper(): int { return 1; }`
	main := `import "./lib/main.bz";
fn helper(): int { return 2; }
fn main(): void {
    println(helper());
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected duplicate top-level symbol rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "duplicate top-level symbol 'helper'") {
		t.Fatalf("expected duplicate top-level symbol diagnostic, got: %v", err)
	}
	if !strings.Contains(msg, libPath) {
		t.Fatalf("expected original declaration file in diagnostic, got: %v", err)
	}
}

func TestCheckEntryAcceptsExplicitMainPackage(t *testing.T) {
	dir := t.TempDir()
	src := `package main;
fn main(): void {
    println("ok");
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(entry); err != nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestCheckEntryRejectsNonMainEntryPackage(t *testing.T) {
	dir := t.TempDir()
	src := `package app;
fn main(): void {
    println("ok");
}`
	entry := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(entry, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected entry package rejection")
	}
	if !strings.Contains(err.Error(), "entry file must declare 'package main'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsRelativeImportPackageMismatch(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	lib := `package lib;
fn helper(): int { return 1; }`
	main := `package main;
import "./lib/main.bz";
fn main(): void {
    println(helper());
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(lib), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.bz")
	if err := os.WriteFile(mainPath, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected relative import package mismatch")
	}
	msg := err.Error()
	if !strings.Contains(msg, "relative import package mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, libPath) {
		t.Fatalf("expected imported file path in diagnostic, got: %v", err)
	}
}

func TestCheckEntryRejectsImportedPackageMainDeclaration(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package main;
fn helper(): int { return 1; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	main := `package main;
import "util";
fn main(): void {
    println(helper());
}`
	entry := filepath.Join(root, "main.bz")
	if err := os.WriteFile(entry, []byte(main), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(entry)
	if err == nil {
		t.Fatalf("expected imported package main rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "imported packages must not declare 'package main'") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, filepath.Join(root, ".bazic", "pkg", "util", "main.bz")) {
		t.Fatalf("expected cached imported package path in diagnostic, got: %v", err)
	}
}

func TestCheckEntryEnforcesPubVisibilityForImportedPackageFunctions(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
fn secret(): int { return 7; }
pub fn helper(): int { return secret(); }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	okSrc := `package main;
import "util";
fn main(): void {
    println(util.helper());
}`
	okPath := filepath.Join(root, "main_ok.bz")
	if err := os.WriteFile(okPath, []byte(okSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(okPath); err != nil {
		t.Fatalf("expected qualified public helper to be accessible: %v", err)
	}
	badSrc := `package main;
import "util";
fn main(): void {
    println(util.secret());
}`
	badPath := filepath.Join(root, "main_bad.bz")
	if err := os.WriteFile(badPath, []byte(badSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(badPath)
	if err == nil {
		t.Fatalf("expected private imported function rejection")
	}
	if !strings.Contains(err.Error(), "package 'util' has no public function 'secret'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryEnforcesPubVisibilityForImportedPackageGlobals(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 42;
const hidden = 9;
pub fn reveal(): int { return hidden; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	okSrc := `package main;
import "util";
fn main(): void {
    println(util.answer);
    println(util.reveal());
}`
	okPath := filepath.Join(root, "globals_ok.bz")
	if err := os.WriteFile(okPath, []byte(okSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(okPath); err != nil {
		t.Fatalf("expected qualified public imported global to be accessible: %v", err)
	}
	badSrc := `package main;
import "util";
fn main(): void {
    println(util.hidden);
}`
	badPath := filepath.Join(root, "globals_bad.bz")
	if err := os.WriteFile(badPath, []byte(badSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(badPath)
	if err == nil {
		t.Fatalf("expected private imported global rejection")
	}
	if !strings.Contains(err.Error(), "package 'util' has no public value 'hidden'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAllowsSamePackagePrivateRelativeImports(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	libSrc := `package main;
fn secret(): int { return 5; }`
	mainSrc := `package main;
import "./lib/main.bz";
fn main(): void {
    println(secret());
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(libSrc), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected same-package private helper access: %v", err)
	}
}

func TestCheckEntryEnforcesPubVisibilityForImportedPackageTypes(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
struct Hidden { value: int; }
pub struct Visible { value: int; }
pub fn make_visible(): Visible { return Visible { value: 1 }; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	okSrc := `package main;
import "util";
fn main(): void {
    let v = util.Visible { value: 2 };
    println(v.value);
    println(util.make_visible().value);
}`
	okPath := filepath.Join(root, "types_ok.bz")
	if err := os.WriteFile(okPath, []byte(okSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(okPath); err != nil {
		t.Fatalf("expected qualified public imported type access: %v", err)
	}
	badSrc := `package main;
import "util";
fn main(): void {
    let h = util.Hidden { value: 3 };
    println(h.value);
}`
	badPath := filepath.Join(root, "types_bad.bz")
	if err := os.WriteFile(badPath, []byte(badSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(badPath)
	if err == nil {
		t.Fatalf("expected private imported type rejection")
	}
	if !strings.Contains(err.Error(), "package 'util' has no public type 'Hidden'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAllowsQualifiedImportedPackageTypes(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
struct Hidden { value: int; }
pub struct Visible { value: int; }
pub fn make_visible(): Visible { return Visible { value: 1 }; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    let v: util.Visible = util.Visible { value: 2 };
    println(v.value);
    println(util.make_visible().value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected qualified imported type access: %v", err)
	}
}

func TestCheckEntryRejectsBareAccessFromExplicitPublicImportedPackage(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 42;
pub fn helper(): int { return answer; }
pub struct Visible { value: int; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    println(answer);
    println(helper());
    let v = Visible { value: 1 };
    println(v.value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected bare imported access rejection for explicit public package")
	}
	if !strings.Contains(err.Error(), "unknown identifier 'answer'") && !strings.Contains(err.Error(), "unknown function 'helper'") && !strings.Contains(err.Error(), "unknown type 'Visible'") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "did you mean 'util.answer'") {
		t.Fatalf("expected qualified import suggestion, got: %v", err)
	}
}

func TestCheckEntryRejectsQualifiedPrivateImportedPackageTypes(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
struct Hidden { value: int; }
pub struct Visible { value: int; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    let h = util.Hidden { value: 3 };
    println(h.value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected qualified private imported type rejection")
	}
	if !strings.Contains(err.Error(), "package 'util' has no public type 'Hidden'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAllowsSamePackagePrivateRelativeTypes(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	libSrc := `package main;
struct Hidden { value: int; }`
	mainSrc := `package main;
import "./lib/main.bz";
fn main(): void {
    let h = Hidden { value: 8 };
    println(h.value);
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(libSrc), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected same-package private type access: %v", err)
	}
}

func TestCheckEntryAllowsQualifiedImportedPackageAccess(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 42;
pub fn helper(): int { return answer; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    println(util.answer);
    println(util.helper());
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected qualified imported package access: %v", err)
	}
}

func TestCheckEntryAllowsAliasedImportedPackageAccess(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 42;
pub fn helper(): int { return answer; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util" as tools;
fn main(): void {
    println(tools.answer);
    println(tools.helper());
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected aliased imported package access: %v", err)
	}
}

func TestCheckEntryRejectsBareAccessFromExplicitImportAlias(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 42;
pub fn helper(): int { return answer; }
pub struct Visible { value: int; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util" as tools;
fn main(): void {
    println(answer);
    println(helper());
    let v = Visible { value: 1 };
    println(v.value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected bare imported access rejection for explicit alias import")
	}
	if !strings.Contains(err.Error(), "unknown identifier 'answer'") && !strings.Contains(err.Error(), "unknown function 'helper'") && !strings.Contains(err.Error(), "unknown type 'Visible'") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "did you mean 'tools.answer'") {
		t.Fatalf("expected qualified alias suggestion, got: %v", err)
	}
}

func TestCheckEntryRejectsQualifiedPrivateImportedPackageAccess(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub fn helper(): int { return 1; }
fn secret(): int { return 9; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    println(util.secret());
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected qualified private imported access rejection")
	}
	if !strings.Contains(err.Error(), "package 'util' has no public function 'secret'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsRelativeImportAlias(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	libSrc := `package main;
fn helper(): int { return 1; }`
	mainSrc := `package main;
import "./lib/main.bz" as lib;
fn main(): void {
    println(helper());
}`
	libPath := filepath.Join(libDir, "main.bz")
	if err := os.WriteFile(libPath, []byte(libSrc), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected relative import alias rejection")
	}
	if !strings.Contains(err.Error(), "relative imports cannot declare an alias") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryRejectsDuplicateImportAlias(t *testing.T) {
	root := t.TempDir()
	depOne := t.TempDir()
	depTwo := t.TempDir()
	depOneSrc := `package utilone;
pub const answer = 1;`
	depTwoSrc := `package utiltwo;
pub const answer = 2;`
	if err := os.WriteFile(filepath.Join(depOne, "main.bz"), []byte(depOneSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depTwo, "main.bz"), []byte(depTwoSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "utilone", depOne); err != nil {
		t.Fatalf("add dep utilone: %v", err)
	}
	if err := pkgm.AddDep(root, "utiltwo", depTwo); err != nil {
		t.Fatalf("add dep utiltwo: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "utilone" as tools;
import "utiltwo" as tools;
fn main(): void {
    println("ok");
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected duplicate import alias rejection")
	}
	if !strings.Contains(err.Error(), "duplicate import alias 'tools'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileEntryPreservesQualifiedImportedPackageSymbols(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub const answer = 41;
pub fn helper(): int { return answer + 1; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    println(util.answer);
    println(util.helper());
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}

	goOut, err := CompileEntryToGo(mainPath)
	if err != nil {
		t.Fatalf("go compile failed: %v", err)
	}
	if !strings.Contains(goOut, "__pkg_util__answer") || !strings.Contains(goOut, "__pkg_util__helper") {
		t.Fatalf("expected mangled util symbols in go output, got:\n%s", goOut)
	}

	llvmOut, err := CompileEntryToLLVM(mainPath)
	if err != nil {
		t.Fatalf("llvm compile failed: %v", err)
	}
	if !strings.Contains(llvmOut, "@__pkg_util__answer") || !strings.Contains(llvmOut, "@__pkg_util__helper") {
		t.Fatalf("expected mangled util symbols in llvm output, got:\n%s", llvmOut)
	}
}

func TestCheckEntryAllowsCrossPackageDuplicateTypeNamesWhenUnused(t *testing.T) {
	root := t.TempDir()
	depOne := t.TempDir()
	depTwo := t.TempDir()
	depOneSrc := `package utilone;
pub struct Shared { value: int; }`
	depTwoSrc := `package utiltwo;
pub struct Shared { value: int; }`
	if err := os.WriteFile(filepath.Join(depOne, "main.bz"), []byte(depOneSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depTwo, "main.bz"), []byte(depTwoSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "utilone", depOne); err != nil {
		t.Fatalf("add dep utilone: %v", err)
	}
	if err := pkgm.AddDep(root, "utiltwo", depTwo); err != nil {
		t.Fatalf("add dep utiltwo: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "utilone";
import "utiltwo";
fn main(): void {
    println("ok");
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected duplicate imported type names to be allowed when unused: %v", err)
	}
}

func TestCheckEntryRejectsBareImportedTypeNameUseFromExplicitPackages(t *testing.T) {
	root := t.TempDir()
	depOne := t.TempDir()
	depTwo := t.TempDir()
	depOneSrc := `package utilone;
pub struct Shared { value: int; }`
	depTwoSrc := `package utiltwo;
pub struct Shared { value: int; }`
	if err := os.WriteFile(filepath.Join(depOne, "main.bz"), []byte(depOneSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depTwo, "main.bz"), []byte(depTwoSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "utilone", depOne); err != nil {
		t.Fatalf("add dep utilone: %v", err)
	}
	if err := pkgm.AddDep(root, "utiltwo", depTwo); err != nil {
		t.Fatalf("add dep utiltwo: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "utilone";
import "utiltwo";
fn main(): void {
    let s = Shared { value: 1 };
    println(s.value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	err := CheckEntry(mainPath)
	if err == nil {
		t.Fatalf("expected bare imported type use error")
	}
	if !strings.Contains(err.Error(), "unknown type 'Shared'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckEntryAllowsQualifiedDisambiguationForDuplicateImportedTypeNames(t *testing.T) {
	root := t.TempDir()
	depOne := t.TempDir()
	depTwo := t.TempDir()
	depOneSrc := `package utilone;
pub struct Shared { value: int; }`
	depTwoSrc := `package utiltwo;
pub struct Shared { value: int; }`
	if err := os.WriteFile(filepath.Join(depOne, "main.bz"), []byte(depOneSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depTwo, "main.bz"), []byte(depTwoSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "utilone", depOne); err != nil {
		t.Fatalf("add dep utilone: %v", err)
	}
	if err := pkgm.AddDep(root, "utiltwo", depTwo); err != nil {
		t.Fatalf("add dep utiltwo: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "utilone";
import "utiltwo";
fn main(): void {
    let a: utilone.Shared = utilone.Shared { value: 1 };
    let b: utiltwo.Shared = utiltwo.Shared { value: 2 };
    println(a.value);
    println(b.value);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected qualified imported type disambiguation: %v", err)
	}
}

func TestCheckEntryAllowsQualifiedImportedEnumMatchArms(t *testing.T) {
	root := t.TempDir()
	depRoot := t.TempDir()
	depSrc := `package util;
pub enum Role { Guest, Admin }
pub fn current(): Role { return Admin; }`
	if err := os.WriteFile(filepath.Join(depRoot, "main.bz"), []byte(depSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := pkgm.Init(root, "demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pkgm.AddDep(root, "util", depRoot); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if err := pkgm.Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}
	mainSrc := `package main;
import "util";
fn main(): void {
    let role: util.Role = util.current();
    match role {
        util.Guest: { println("guest"); }
        util.Admin: { println("admin"); }
    }
    let label = match role {
        util.Guest: "guest",
        util.Admin: "admin",
    };
    println(label);
}`
	mainPath := filepath.Join(root, "main.bz")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEntry(mainPath); err != nil {
		t.Fatalf("expected qualified imported enum match arms: %v", err)
	}
	goOut, err := CompileEntryToGo(mainPath)
	if err != nil {
		t.Fatalf("go compile failed: %v", err)
	}
	if !strings.Contains(goOut, "case Guest:") || !strings.Contains(goOut, "case Admin:") {
		t.Fatalf("expected normalized match arm cases in go output, got:\n%s", goOut)
	}
}

func TestCheckEntrySemanticRangeUsesUnderlineWidth(t *testing.T) {
	dir := t.TempDir()
	src := `fn main(): void {
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
	if !strings.Contains(err.Error(), "^~~~~") {
		t.Fatalf("expected range underline in diagnostic, got: %v", err)
	}
}
