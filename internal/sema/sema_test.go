package sema

import (
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
)

func TestCheckPassPipelineCollectsFunctionsBeforeGlobals(t *testing.T) {
	src := `let answer = meaning();

fn meaning(): int { return 42; }

fn main(): void {
    println(answer);
}`
	prog := parseProgramForSema(t, src)
	if err := New().Check(prog); err != nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestCheckPassPipelineRequiresMain(t *testing.T) {
	src := `fn helper(): int { return 1; }`
	prog := parseProgramForSema(t, src)
	if err := New().Check(prog); err == nil {
		t.Fatalf("expected missing main error")
	}
}

func TestCheckPassPipelineRestoresFunctionTypeParamState(t *testing.T) {
	src := `fn id[T](value: T): T { return value; }

fn main(): void {
    let leaked: T = 1;
    println(leaked);
}`
	prog := parseProgramForSema(t, src)
	if err := New().Check(prog); err == nil {
		t.Fatalf("expected unknown type error for leaked function type param")
	}
}

func TestTypeRegistryResolvesCurrentPackageStructBeforeAmbiguousImports(t *testing.T) {
	c := New()
	if err := c.registerStruct("Thing", StructSig{PackageID: "pkg:one", Public: true}); err != nil {
		t.Fatalf("register pkg:one struct: %v", err)
	}
	if err := c.registerStruct("Thing", StructSig{PackageID: "pkg:two", Public: true}); err != nil {
		t.Fatalf("register pkg:two struct: %v", err)
	}
	c.currentPackage = "pkg:one"
	sig, status := c.resolveStruct("Thing")
	if status != typeLookupFound {
		t.Fatalf("expected current-package struct resolution, got status %v", status)
	}
	if sig.PackageID != "pkg:one" {
		t.Fatalf("expected pkg:one struct, got %q", sig.PackageID)
	}
}

func TestTypeRegistryMarksImportedStructsAmbiguousWhenMultiplePackagesExportSameName(t *testing.T) {
	c := New()
	if err := c.registerStruct("Thing", StructSig{PackageID: "pkg:one", Public: true}); err != nil {
		t.Fatalf("register pkg:one struct: %v", err)
	}
	if err := c.registerStruct("Thing", StructSig{PackageID: "pkg:two", Public: true}); err != nil {
		t.Fatalf("register pkg:two struct: %v", err)
	}
	c.currentPackage = "main"
	c.bareImports["main"] = map[string]bool{"pkg:one": true, "pkg:two": true}
	_, status := c.resolveStruct("Thing")
	if status != typeLookupAmbiguous {
		t.Fatalf("expected ambiguous imported struct resolution, got status %v", status)
	}
}

func TestTypeRegistryIgnoresPrivateImportedInterfaceWhenResolvingBounds(t *testing.T) {
	c := New()
	if err := c.registerInterface("Named", InterfaceSig{PackageID: "pkg:one", Public: false}); err != nil {
		t.Fatalf("register private interface: %v", err)
	}
	if err := c.registerInterface("Named", InterfaceSig{PackageID: "pkg:two", Public: true}); err != nil {
		t.Fatalf("register public interface: %v", err)
	}
	c.currentPackage = "main"
	c.bareImports["main"] = map[string]bool{"pkg:one": true, "pkg:two": true}
	sig, status := c.resolveInterface("Named")
	if status != typeLookupFound {
		t.Fatalf("expected visible imported interface resolution, got status %v", status)
	}
	if sig.PackageID != "pkg:two" {
		t.Fatalf("expected public pkg:two interface, got %q", sig.PackageID)
	}
}

func TestTypeRegistryResolvesSamePackageEnumBeforeImportedDuplicate(t *testing.T) {
	c := New()
	if err := c.registerEnum("Role", EnumSig{PackageID: "pkg:main", Public: false, Variants: map[string]bool{"Guest": true}}); err != nil {
		t.Fatalf("register same-package enum: %v", err)
	}
	if err := c.registerEnum("Role", EnumSig{PackageID: "pkg:util", Public: true, Variants: map[string]bool{"Admin": true}}); err != nil {
		t.Fatalf("register imported enum: %v", err)
	}
	c.currentPackage = "pkg:main"
	sig, status := c.resolveEnum("Role")
	if status != typeLookupFound {
		t.Fatalf("expected same-package enum resolution, got status %v", status)
	}
	if !sig.Variants["Guest"] || sig.PackageID != "pkg:main" {
		t.Fatalf("expected same-package enum variants, got %+v", sig)
	}
}

func TestValueRegistryResolvesCurrentPackageFunctionBeforeAmbiguousImports(t *testing.T) {
	c := New()
	if err := c.registerFunction("helper", FuncSig{PackageID: "pkg:one", Public: true}); err != nil {
		t.Fatalf("register pkg:one function: %v", err)
	}
	if err := c.registerFunction("helper", FuncSig{PackageID: "pkg:two", Public: true}); err != nil {
		t.Fatalf("register pkg:two function: %v", err)
	}
	c.currentPackage = "pkg:one"
	sig, status := c.resolveFunction("helper")
	if status != typeLookupFound {
		t.Fatalf("expected current-package function resolution, got status %v", status)
	}
	if sig.PackageID != "pkg:one" {
		t.Fatalf("expected pkg:one function, got %q", sig.PackageID)
	}
}

func TestValueRegistryIgnoresPrivateImportedGlobalWhenSingleVisibleCandidateExists(t *testing.T) {
	c := New()
	if err := c.registerGlobal("answer", GlobalSymbol{PackageID: "pkg:one", Public: false, Type: ast.TypeInt}); err != nil {
		t.Fatalf("register private global: %v", err)
	}
	if err := c.registerGlobal("answer", GlobalSymbol{PackageID: "pkg:two", Public: true, Type: ast.TypeInt}); err != nil {
		t.Fatalf("register public global: %v", err)
	}
	c.currentPackage = "main"
	c.bareImports["main"] = map[string]bool{"pkg:one": true, "pkg:two": true}
	g, status := c.resolveGlobalSymbol("answer")
	if status != typeLookupFound {
		t.Fatalf("expected visible imported global resolution, got status %v", status)
	}
	if g.PackageID != "pkg:two" {
		t.Fatalf("expected public pkg:two global, got %q", g.PackageID)
	}
}

func TestTypeRegistryDoesNotResolveImportedStructWithoutBareImportBinding(t *testing.T) {
	c := New()
	if err := c.registerStruct("Thing", StructSig{PackageID: "pkg:util", Public: true}); err != nil {
		t.Fatalf("register imported struct: %v", err)
	}
	c.currentPackage = "main"
	_, status := c.resolveStruct("Thing")
	if status != typeLookupMissing {
		t.Fatalf("expected missing imported struct without bare binding, got status %v", status)
	}
}

func TestValueRegistryDoesNotResolveImportedFunctionWithoutBareImportBinding(t *testing.T) {
	c := New()
	if err := c.registerFunction("helper", FuncSig{PackageID: "pkg:util", Public: true}); err != nil {
		t.Fatalf("register imported function: %v", err)
	}
	c.currentPackage = "main"
	_, status := c.resolveFunction("helper")
	if status != typeLookupMissing {
		t.Fatalf("expected missing imported function without bare binding, got status %v", status)
	}
}

func parseProgramForSema(t *testing.T, src string) *ast.Program {
	t.Helper()
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return prog
}
