package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/codegen"
	"baziclang/internal/codegenllvm"
	"baziclang/internal/diag"
	"baziclang/internal/hir"
	"baziclang/internal/intrinsics"
	"baziclang/internal/lexer"
	"baziclang/internal/mir"
	"baziclang/internal/parser"
	"baziclang/internal/pkgm"
	"baziclang/internal/sema"
	"baziclang/internal/source"
)

type BuildOptions struct {
	Input     string
	Out       string
	Target    string
	KeepGoSrc bool
	Backend   string
}

func CompileToGo(src string) (string, error) {
	prog, err := parseSource(src)
	if err != nil {
		return "", err
	}
	if err := sema.New().Check(prog); err != nil {
		return "", err
	}
	hp, err := hir.Lower(prog)
	if err != nil {
		return "", err
	}
	if _, err := mir.Lower(hp); err != nil {
		return "", err
	}
	return codegen.GenerateGo(prog)
}

func CompileEntryToGo(entry string) (string, error) {
	merged, err := loadEntryProgram(entry)
	if err != nil {
		return "", err
	}
	return codegen.GenerateGo(merged)
}

func CheckEntry(entry string) error {
	_, err := loadEntryProgram(entry)
	return err
}

func CompileEntryToLLVM(entry string) (string, error) {
	merged, err := loadEntryProgram(entry)
	if err != nil {
		return "", err
	}
	return codegenllvm.GenerateLLVMIR(merged)
}

func loadEntryProgram(entry string) (*ast.Program, error) {
	entryAbs, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	entryDir := filepath.Dir(entryAbs)
	foundRoot := true
	root, err := pkgm.FindProjectRoot(entryDir)
	if err != nil {
		root = entryDir
		foundRoot = false
	}
	if foundRoot {
		if err := pkgm.Verify(root); err != nil {
			return nil, fmt.Errorf("package integrity check failed: %w", err)
		}
	}
	merged := &ast.Program{}
	visited := map[string]visitState{}
	if err := loadFileRecursive(root, entryAbs, merged, visited, nil, true, "main", "main", map[string]bool{}); err != nil {
		return nil, err
	}
	if err := validateMergedTopLevelSymbols(merged); err != nil {
		return nil, err
	}
	injectSafetyPrelude(merged)
	if err := sema.New().Check(merged); err != nil {
		return nil, renderProgramError(err)
	}
	hp, err := hir.Lower(merged)
	if err != nil {
		return nil, renderProgramError(err)
	}
	if _, err := mir.Lower(hp); err != nil {
		return nil, renderProgramError(err)
	}
	return merged, nil
}

func Build(opts BuildOptions) error {
	if opts.Backend == "" {
		opts.Backend = "go"
	}
	if opts.Target == "" {
		opts.Target = "native"
	}
	switch strings.ToLower(opts.Backend) {
	case "go":
		return buildGo(opts)
	case "llvm":
		if opts.Target == "wasm" {
			return fmt.Errorf("llvm backend does not support wasm target yet")
		}
		return buildLLVM(opts)
	default:
		return fmt.Errorf("unknown backend '%s' (expected go or llvm)", opts.Backend)
	}
}

func buildGo(opts BuildOptions) error {
	prevTarget := os.Getenv("BAZIC_TARGET")
	_ = os.Setenv("BAZIC_TARGET", opts.Target)
	defer func() {
		if prevTarget == "" {
			_ = os.Unsetenv("BAZIC_TARGET")
		} else {
			_ = os.Setenv("BAZIC_TARGET", prevTarget)
		}
	}()
	goCode, err := CompileEntryToGo(opts.Input)
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "bazic-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		return fmt.Errorf("write generated go: %w", err)
	}
	if opts.KeepGoSrc {
		_ = os.WriteFile(filepath.Join(filepath.Dir(opts.Out), "generated_from_bazic.go"), []byte(goCode), 0644)
	}

	args := buildArgs(opts.Out, goFile)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if opts.Target == "wasm" {
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	return nil
}

func buildLLVM(opts BuildOptions) error {
	ir, err := CompileEntryToLLVM(opts.Input)
	if err != nil {
		return err
	}
	if err := rejectUnsupportedLLVM(ir); err != nil {
		return err
	}
	clangPath, err := findTool("clang", "BAZIC_CLANG")
	if err != nil {
		return err
	}
	if err := ensureClangVersion(clangPath); err != nil {
		return err
	}
	if triple, err := clangTargetTriple(clangPath); err == nil && triple != "" {
		ir = injectTargetTriple(ir, triple)
	}
	tmpDir, err := os.MkdirTemp("", "bazic-llvm-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	irFile := filepath.Join(tmpDir, "main.ll")
	if err := os.WriteFile(irFile, []byte(ir), 0644); err != nil {
		return fmt.Errorf("write generated llvm: %w", err)
	}
	if runtime.GOOS == "windows" {
		if err := ensureClangStdHeaders(clangPath); err != nil {
			return err
		}
	}
	runtimeFile, err := resolveRuntimeFile(opts.Input)
	if err != nil {
		return err
	}
	args := []string{"-O2", "-std=c11", "-Wno-override-module", "-o", opts.Out}
	if v := strings.TrimSpace(os.Getenv("BAZIC_CLANG_FLAGS")); v != "" {
		args = append(args, strings.Fields(v)...)
	}
	if strings.TrimSpace(os.Getenv("BAZIC_SQLITE")) != "" {
		args = append(args, "-DBAZIC_SQLITE")
	}
	args = append(args, irFile, runtimeFile)
	if runtime.GOOS == "windows" {
		args = append(args, "-lwinhttp", "-lws2_32", "-lbcrypt")
	} else {
		args = append(args, "-lcurl")
	}
	if strings.TrimSpace(os.Getenv("BAZIC_SQLITE")) != "" {
		args = append(args, "-lsqlite3")
	}
	cmd := exec.Command(clangPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang failed: %w", err)
	}
	return nil
}

func rejectUnsupportedLLVM(ir string) error {
	if strings.Contains(ir, "; skipping") || strings.Contains(ir, "; unsupported") {
		return fmt.Errorf("llvm backend: unsupported features detected; remove or use go backend")
	}
	return nil
}

func MaybeInjectTargetTriple(ir string) string {
	clangPath, err := findTool("clang", "BAZIC_CLANG")
	if err != nil {
		return ir
	}
	if triple, err := clangTargetTriple(clangPath); err == nil && triple != "" {
		return injectTargetTriple(ir, triple)
	}
	return ir
}

func injectTargetTriple(ir string, triple string) string {
	if strings.Contains(ir, "target triple") {
		return ir
	}
	line := fmt.Sprintf("target triple = \"%s\"\n", triple)
	idx := strings.Index(ir, "source_filename")
	if idx == -1 {
		return line + ir
	}
	end := strings.Index(ir[idx:], "\n")
	if end == -1 {
		return ir + "\n" + line
	}
	pos := idx + end + 1
	return ir[:pos] + line + ir[pos:]
}

func clangTargetTriple(path string) (string, error) {
	out, err := exec.Command(path, "-print-target-triple").CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RejectUnsupportedLLVM is used by CLI emit-llvm --check to surface unsupported features.
func RejectUnsupportedLLVM(ir string) error {
	return rejectUnsupportedLLVM(ir)
}

func projectRootFor(entry string) string {
	entryAbs, err := filepath.Abs(entry)
	if err != nil {
		return "."
	}
	entryDir := filepath.Dir(entryAbs)
	root, err := pkgm.FindProjectRoot(entryDir)
	if err == nil {
		return root
	}
	return entryDir
}

func resolveRuntimeFile(entry string) (string, error) {
	candidates := []string{
		filepath.Join(projectRootFor(entry), "runtime", "bazic_runtime.c"),
	}
	if v := strings.TrimSpace(os.Getenv("BAZIC_RUNTIME")); v != "" {
		candidates = append(candidates, v)
	}
	if v := strings.TrimSpace(os.Getenv("BAZIC_HOME")); v != "" {
		candidates = append(candidates, filepath.Join(v, "runtime", "bazic_runtime.c"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "runtime", "bazic_runtime.c"),
			filepath.Join(exeDir, "..", "runtime", "bazic_runtime.c"),
		)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("runtime not found: expected runtime/bazic_runtime.c (set BAZIC_HOME or BAZIC_RUNTIME)")
}

func findTool(name string, env string) (string, error) {
	if env != "" {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v, nil
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		if env != "" {
			return "", fmt.Errorf("%s not found; install LLVM/clang or set %s to the compiler path", name, env)
		}
		return "", fmt.Errorf("%s not found; install it and ensure it is on PATH", name)
	}
	return path, nil
}

func ensureClangVersion(path string) error {
	major, err := clangMajorVersion(path)
	if err != nil {
		return nil
	}
	if major > 0 && major < 15 {
		return fmt.Errorf("clang %d detected; LLVM backend requires clang 15+ (opaque pointers)", major)
	}
	return nil
}

func clangMajorVersion(path string) (int, error) {
	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return 0, err
	}
	line := string(out)
	re := regexp.MustCompile(`(?m)(?:clang version|Apple clang version)\s+(\d+)`)
	m := re.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0, fmt.Errorf("clang version parse failed")
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, err
	}
	return v, nil
}

func ensureClangStdHeaders(path string) error {
	cmd := exec.Command(path, "-E", "-xc", "-")
	cmd.Stdin = strings.NewReader("#include <stdio.h>\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = err.Error()
	}
	return fmt.Errorf("clang toolchain missing C headers; install Visual Studio Build Tools (C++ workload). clang output: %s", msg)
}

func buildArgs(outPath, goFile string) []string {
	return []string{
		"build",
		"-trimpath",
		"-ldflags", "-buildid=",
		"-o", outPath,
		goFile,
	}
}

func Run(input string) error {
	return RunWithOptions(RunOptions{Input: input, Backend: "go"})
}

type RunOptions struct {
	Input   string
	Backend string
	Args    []string
}

func RunWithOptions(opts RunOptions) error {
	tmpDir, err := os.MkdirTemp("", "bazic-run-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	tmpExe := filepath.Join(tmpDir, "bazic-run.exe")
	if err := Build(BuildOptions{Input: opts.Input, Out: tmpExe, Target: "native", Backend: opts.Backend}); err != nil {
		return err
	}
	cmd := exec.Command(tmpExe, opts.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}
	return nil
}

type visitState int

const (
	visitUnseen visitState = iota
	visitActive
	visitDone
)

func loadFileRecursive(root, file string, merged *ast.Program, visited map[string]visitState, stack []string, isEntry bool, expectedPackage string, packageID string, packageVisibility map[string]bool) error {
	clean := filepath.Clean(file)
	switch visited[clean] {
	case visitDone:
		return nil
	case visitActive:
		return fmt.Errorf("import cycle detected: %s", formatImportCycle(stack, clean))
	}
	visited[clean] = visitActive
	stack = append(stack, clean)
	data, err := os.ReadFile(clean)
	if err != nil {
		return fmt.Errorf("read %s: %w", clean, err)
	}
	prog, err := parseSourceWithName(string(data), clean)
	if err != nil {
		return fmt.Errorf("parse %s: %w", clean, err)
	}
	filePackage, err := validateProgramPackage(prog, isEntry, expectedPackage)
	if err != nil {
		return err
	}
	explicitVisibility := programHasExplicitPublicDecls(prog)
	if packageID != "" {
		packageVisibility[packageID] = packageVisibility[packageID] || explicitVisibility
	}
	annotateProgramPackageID(prog, packageID)
	assignInternalSymbolNames(prog, packageID, expectedPackage == "")
	applyLegacyPackageVisibility(prog, isEntry, expectedPackage)
	for _, d := range prog.Decls {
		imp, ok := d.(*ast.ImportDecl)
		if !ok {
			continue
		}
		resolved, err := pkgm.ResolveImport(root, filepath.Dir(clean), imp.Path)
		if err != nil {
			return fmt.Errorf("resolve import '%s' in %s: %w", imp.Path, clean, err)
		}
		nextExpectedPackage := ""
		nextPackageID := packageID
		if strings.HasPrefix(imp.Path, ".") {
			if imp.Alias != "" {
				return renderProgramError(diag.New("import error", "relative imports cannot declare an alias; they merge into the current package", imp.Span()))
			}
			nextExpectedPackage = filePackage
		} else {
			nextPackageID = "pkg:" + imp.Path
		}
		if err := loadFileRecursive(root, resolved, merged, visited, stack, false, nextExpectedPackage, nextPackageID, packageVisibility); err != nil {
			return err
		}
		if !strings.HasPrefix(imp.Path, ".") {
			alias := imp.Path
			if imp.Alias != "" {
				alias = imp.Alias
			}
			if err := recordImportRef(merged, ast.ImportRef{
				OwnerPackageID:  packageID,
				Alias:           alias,
				Path:            imp.Path,
				TargetPackageID: nextPackageID,
				ExplicitAlias:   imp.ExplicitAlias,
				BareAllowed:     !imp.ExplicitAlias && !packageVisibility[nextPackageID],
			}); err != nil {
				return renderProgramError(diag.New("import error", err.Error(), imp.Span()))
			}
		}
	}
	for _, d := range prog.Decls {
		if _, isImport := d.(*ast.ImportDecl); isImport {
			continue
		}
		merged.Decls = append(merged.Decls, d)
	}
	visited[clean] = visitDone
	return nil
}

func validateProgramPackage(prog *ast.Program, isEntry bool, expectedPackage string) (string, error) {
	if prog == nil {
		return expectedPackage, nil
	}
	declaredPackage := ""
	if prog.Package != nil {
		declaredPackage = strings.TrimSpace(prog.Package.Name)
	}
	if isEntry {
		if declaredPackage == "" {
			declaredPackage = "main"
		}
		if declaredPackage != "main" {
			return "", renderProgramError(diag.New("package error", "entry file must declare 'package main'", prog.Package.Span()))
		}
		return declaredPackage, nil
	}
	if expectedPackage != "" && declaredPackage != "" && declaredPackage != expectedPackage {
		return "", renderProgramError(diag.New("package error", fmt.Sprintf("relative import package mismatch: expected package '%s' but found '%s'", expectedPackage, declaredPackage), prog.Package.Span()))
	}
	if expectedPackage == "" && declaredPackage == "main" {
		return "", renderProgramError(diag.New("package error", "imported packages must not declare 'package main'; reserve 'package main' for the entry file", prog.Package.Span()))
	}
	if expectedPackage != "" {
		declaredPackage = expectedPackage
	}
	for _, d := range prog.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name != "main" {
			continue
		}
		return "", renderProgramError(diag.New("import error", "imported files must not declare 'main'; only the entry file may define the program entrypoint", fn.Span()))
	}
	return declaredPackage, nil
}

type topLevelSymbol struct {
	kind              string
	span              source.Span
	packageID         string
	allowCrossPackage bool
}

func validateMergedTopLevelSymbols(prog *ast.Program) error {
	if prog == nil {
		return nil
	}
	seen := map[string]topLevelSymbol{}
	for _, d := range prog.Decls {
		symbols := topLevelSymbolsForDecl(d)
		for _, sym := range symbols {
			prev, exists := seen[sym.name]
			if exists {
				if sym.allowCrossPackage && prev.allowCrossPackage && sym.packageID != "" && prev.packageID != "" && sym.packageID != prev.packageID {
					continue
				}
				msg := fmt.Sprintf(
					"duplicate top-level symbol '%s'; previously declared as %s in %s:%d:%d",
					sym.name,
					prev.kind,
					displaySourceFile(prev.span.Start.File),
					prev.span.Start.Line,
					prev.span.Start.Col,
				)
				return renderProgramError(diag.New("import error", msg, sym.span))
			}
			seen[sym.name] = topLevelSymbol{kind: sym.kind, span: sym.span, packageID: sym.packageID, allowCrossPackage: sym.allowCrossPackage}
		}
	}
	return nil
}

type namedTopLevelSymbol struct {
	name              string
	kind              string
	span              source.Span
	packageID         string
	allowCrossPackage bool
}

func topLevelSymbolsForDecl(d ast.Decl) []namedTopLevelSymbol {
	switch decl := d.(type) {
	case *ast.StructDecl:
		return []namedTopLevelSymbol{{name: decl.Name, kind: "struct", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true}}
	case *ast.InterfaceDecl:
		return []namedTopLevelSymbol{{name: decl.Name, kind: "interface", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true}}
	case *ast.EnumDecl:
		out := []namedTopLevelSymbol{{name: decl.Name, kind: "enum", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true}}
		for _, variant := range decl.Variants {
			out = append(out, namedTopLevelSymbol{name: variant, kind: "enum variant", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true})
		}
		return out
	case *ast.FuncDecl:
		return []namedTopLevelSymbol{{name: decl.Name, kind: "function", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true}}
	case *ast.GlobalLetDecl:
		return []namedTopLevelSymbol{{name: decl.Name, kind: "global", span: decl.Span(), packageID: decl.PackageID, allowCrossPackage: true}}
	default:
		return nil
	}
}

func displaySourceFile(path string) string {
	if strings.TrimSpace(path) == "" {
		return "<unknown>"
	}
	return path
}

func recordImportRef(prog *ast.Program, ref ast.ImportRef) error {
	if prog == nil || ref.Alias == "" || ref.TargetPackageID == "" {
		return nil
	}
	for _, existing := range prog.Imports {
		if existing.OwnerPackageID == ref.OwnerPackageID && existing.Alias == ref.Alias {
			if existing.TargetPackageID == ref.TargetPackageID && existing.Path == ref.Path {
				return nil
			}
			return fmt.Errorf("duplicate import alias '%s'", ref.Alias)
		}
	}
	prog.Imports = append(prog.Imports, ref)
	return nil
}

func assignInternalSymbolNames(prog *ast.Program, packageID string, aliasImported bool) {
	if prog == nil {
		return
	}
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			if aliasImported {
				d.InternalName = manglePackageTypeName(packageID, d.Name)
			} else if d.InternalName == "" {
				d.InternalName = d.Name
			}
		case *ast.InterfaceDecl:
			if aliasImported {
				d.InternalName = manglePackageTypeName(packageID, d.Name)
			} else if d.InternalName == "" {
				d.InternalName = d.Name
			}
		case *ast.EnumDecl:
			if aliasImported {
				d.InternalName = manglePackageTypeName(packageID, d.Name)
			} else if d.InternalName == "" {
				d.InternalName = d.Name
			}
		case *ast.FuncDecl:
			if aliasImported {
				d.InternalName = manglePackageSymbol(packageID, d.Name)
			} else if d.InternalName == "" {
				d.InternalName = d.Name
			}
		case *ast.GlobalLetDecl:
			if aliasImported {
				d.InternalName = manglePackageSymbol(packageID, d.Name)
			} else if d.InternalName == "" {
				d.InternalName = d.Name
			}
		}
	}
}

func manglePackageTypeName(packageID, name string) string {
	return manglePackageSymbol(packageID, name)
}

func manglePackageSymbol(packageID, name string) string {
	if packageID == "" {
		return name
	}
	clean := packageID
	clean = strings.ReplaceAll(clean, "pkg:", "pkg_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	return "__" + clean + "__" + name
}

func formatImportCycle(stack []string, repeated string) string {
	start := 0
	for i, f := range stack {
		if f == repeated {
			start = i
			break
		}
	}
	parts := append([]string{}, stack[start:]...)
	parts = append(parts, repeated)
	for i := range parts {
		parts[i] = filepath.Base(parts[i])
	}
	return strings.Join(parts, " -> ")
}

func parseSource(src string) (*ast.Program, error) {
	return parseSourceWithName(src, "<input>")
}

func parseSourceWithName(src, sourceName string) (*ast.Program, error) {
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		return nil, decorateSourceError(err, src, sourceName)
	}
	p, err := parser.New(tokens).ParseProgram()
	if err != nil {
		return nil, decorateSourceError(err, src, sourceName)
	}
	annotateProgramFile(p, sourceName)
	return p, nil
}

func decorateSourceError(err error, src, sourceName string) error {
	derr, ok := diag.Extract(err)
	if ok && derr.Span.Start.File != "" && derr.Span.Start.File != sourceName {
		if data, readErr := os.ReadFile(derr.Span.Start.File); readErr == nil {
			return diag.RenderWithSource(err, string(data), derr.Span.Start.File)
		}
	}
	return diag.RenderWithSource(err, src, sourceName)
}

func renderProgramError(err error) error {
	derr, ok := diag.Extract(err)
	if !ok {
		return err
	}
	file := strings.TrimSpace(derr.Span.Start.File)
	if file == "" {
		return err
	}
	data, readErr := os.ReadFile(file)
	if readErr != nil {
		return err
	}
	return diag.RenderWithSource(err, string(data), file)
}

func injectSafetyPrelude(p *ast.Program) {
	hasStruct := map[string]bool{}
	hasFunc := map[string]bool{}
	hasGlobal := map[string]bool{}
	for _, d := range p.Decls {
		if s, ok := d.(*ast.StructDecl); ok {
			hasStruct[s.Name] = true
		}
		if fn, ok := d.(*ast.FuncDecl); ok {
			hasFunc[fn.Name] = true
		}
		if g, ok := d.(*ast.GlobalLetDecl); ok {
			hasGlobal[g.Name] = true
		}
	}
	prelude := intrinsics.SafetyPreludeDecls(hasStruct, hasFunc, hasGlobal)
	if len(prelude) == 0 {
		return
	}
	p.Decls = append(prelude, p.Decls...)
}

func annotateProgramFile(p *ast.Program, file string) {
	if p == nil || file == "" {
		return
	}
	p.NodeInfo.Range = p.NodeInfo.Range.WithFile(file)
	if p.Package != nil {
		p.Package.NodeInfo.Range = p.Package.NodeInfo.Range.WithFile(file)
	}
	for _, decl := range p.Decls {
		annotateDeclFile(decl, file)
	}
}

func annotateProgramPackageID(p *ast.Program, packageID string) {
	if p == nil {
		return
	}
	for _, decl := range p.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			d.PackageID = packageID
		case *ast.InterfaceDecl:
			d.PackageID = packageID
		case *ast.ImplDecl:
			d.PackageID = packageID
		case *ast.EnumDecl:
			d.PackageID = packageID
		case *ast.FuncDecl:
			d.PackageID = packageID
		case *ast.GlobalLetDecl:
			d.PackageID = packageID
		}
	}
}

func applyLegacyPackageVisibility(p *ast.Program, isEntry bool, expectedPackage string) {
	if p == nil || isEntry || expectedPackage != "" || programHasExplicitPublicDecls(p) {
		return
	}
	for _, decl := range p.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			d.Public = true
		case *ast.InterfaceDecl:
			d.Public = true
		case *ast.EnumDecl:
			d.Public = true
		case *ast.FuncDecl:
			d.Public = true
		case *ast.GlobalLetDecl:
			d.Public = true
		}
	}
}

func programHasExplicitPublicDecls(p *ast.Program) bool {
	if p == nil {
		return false
	}
	for _, decl := range p.Decls {
		switch d := decl.(type) {
		case *ast.StructDecl:
			if d.Public {
				return true
			}
		case *ast.InterfaceDecl:
			if d.Public {
				return true
			}
		case *ast.EnumDecl:
			if d.Public {
				return true
			}
		case *ast.FuncDecl:
			if d.Public {
				return true
			}
		case *ast.GlobalLetDecl:
			if d.Public {
				return true
			}
		}
	}
	return false
}

func annotateDeclFile(d ast.Decl, file string) {
	switch decl := d.(type) {
	case *ast.ImportDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
	case *ast.StructDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
		for i := range decl.Fields {
			decl.Fields[i].Range = decl.Fields[i].Range.WithFile(file)
		}
	case *ast.InterfaceDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
		for i := range decl.Methods {
			decl.Methods[i].Range = decl.Methods[i].Range.WithFile(file)
			for j := range decl.Methods[i].Params {
				decl.Methods[i].Params[j].Range = decl.Methods[i].Params[j].Range.WithFile(file)
			}
		}
	case *ast.ImplDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
	case *ast.EnumDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
	case *ast.FuncDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
		for i := range decl.Params {
			decl.Params[i].Range = decl.Params[i].Range.WithFile(file)
		}
		annotateBlockFile(decl.Body, file)
	case *ast.GlobalLetDecl:
		decl.NodeInfo.Range = decl.NodeInfo.Range.WithFile(file)
		annotateExprFile(decl.Init, file)
	}
}

func annotateBlockFile(b *ast.BlockStmt, file string) {
	if b == nil {
		return
	}
	b.NodeInfo.Range = b.NodeInfo.Range.WithFile(file)
	for _, stmt := range b.Stmts {
		annotateStmtFile(stmt, file)
	}
}

func annotateStmtFile(s ast.Stmt, file string) {
	switch st := s.(type) {
	case *ast.BlockStmt:
		annotateBlockFile(st, file)
	case *ast.LetStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Init, file)
	case *ast.AssignStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Target, file)
		annotateExprFile(st.Value, file)
	case *ast.IfStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Cond, file)
		annotateBlockFile(st.Then, file)
		annotateBlockFile(st.Else, file)
	case *ast.WhileStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Cond, file)
		annotateBlockFile(st.Body, file)
	case *ast.MatchStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Subject, file)
		for i := range st.Arms {
			st.Arms[i].Range = st.Arms[i].Range.WithFile(file)
			annotateExprFile(st.Arms[i].Guard, file)
			annotateBlockFile(st.Arms[i].Body, file)
		}
	case *ast.ReturnStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Value, file)
	case *ast.ExprStmt:
		st.NodeInfo.Range = st.NodeInfo.Range.WithFile(file)
		annotateExprFile(st.Expr, file)
	}
}

func annotateExprFile(e ast.Expr, file string) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.IntExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.FloatExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.BoolExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.StringExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.NilExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
	case *ast.UnaryExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		annotateExprFile(ex.Right, file)
	case *ast.BinaryExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		annotateExprFile(ex.Left, file)
		annotateExprFile(ex.Right, file)
	case *ast.CallExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		annotateExprFile(ex.Receiver, file)
		for _, arg := range ex.Args {
			annotateExprFile(arg, file)
		}
	case *ast.FieldAccessExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		annotateExprFile(ex.Object, file)
	case *ast.StructLitExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		for i := range ex.Fields {
			ex.Fields[i].Range = ex.Fields[i].Range.WithFile(file)
			annotateExprFile(ex.Fields[i].Value, file)
		}
	case *ast.MatchExpr:
		ex.NodeInfo.Range = ex.NodeInfo.Range.WithFile(file)
		annotateExprFile(ex.Subject, file)
		for i := range ex.Arms {
			ex.Arms[i].Range = ex.Arms[i].Range.WithFile(file)
			annotateExprFile(ex.Arms[i].Guard, file)
			annotateExprFile(ex.Arms[i].Value, file)
		}
	}
}
