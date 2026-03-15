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
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
	"baziclang/internal/pkgm"
	"baziclang/internal/sema"
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
	if err := loadFileRecursive(root, entryAbs, merged, visited, nil); err != nil {
		return nil, err
	}
	injectSafetyPrelude(merged)
	if err := sema.New().Check(merged); err != nil {
		return nil, err
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
	args := []string{"-O2", "-std=c11", "-o", opts.Out}
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

func loadFileRecursive(root, file string, merged *ast.Program, visited map[string]visitState, stack []string) error {
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
	for _, d := range prog.Decls {
		imp, ok := d.(*ast.ImportDecl)
		if !ok {
			continue
		}
		resolved, err := pkgm.ResolveImport(root, filepath.Dir(clean), imp.Path)
		if err != nil {
			return fmt.Errorf("resolve import '%s' in %s: %w", imp.Path, clean, err)
		}
		if err := loadFileRecursive(root, resolved, merged, visited, stack); err != nil {
			return err
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
	return p, nil
}

var atLineColPattern = regexp.MustCompile(`at (\d+):(\d+)`)

func decorateSourceError(err error, src, sourceName string) error {
	msg := err.Error()
	m := atLineColPattern.FindStringSubmatch(msg)
	if len(m) != 3 {
		return err
	}
	line, errLine := strconv.Atoi(m[1])
	col, errCol := strconv.Atoi(m[2])
	if errLine != nil || errCol != nil || line <= 0 || col <= 0 {
		return err
	}
	lines := strings.Split(src, "\n")
	if line > len(lines) {
		return err
	}
	text := lines[line-1]
	caretCol := col
	if caretCol > len([]rune(text))+1 {
		caretCol = len([]rune(text)) + 1
	}
	spaces := strings.Repeat(" ", max(0, caretCol-1))
	return fmt.Errorf("%s\n --> %s:%d:%d\n  |\n%3d | %s\n  | %s^", msg, sourceName, line, col, line, text, spaces)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	prelude := make([]ast.Decl, 0, 9)
	if !hasGlobal["__bazic_assert_failed"] {
		prelude = append(prelude, &ast.GlobalLetDecl{
			Name: "__bazic_assert_failed",
			Type: ast.TypeBool,
			Init: &ast.BoolExpr{Value: false},
		})
	}
	if !hasGlobal["__bazic_assert_message"] {
		prelude = append(prelude, &ast.GlobalLetDecl{
			Name: "__bazic_assert_message",
			Type: ast.TypeString,
			Init: &ast.StringExpr{Value: ""},
		})
	}
	if !hasStruct["Error"] {
		prelude = append(prelude, &ast.StructDecl{
			Name: "Error",
			Fields: []ast.StructField{
				{Name: "message", Type: ast.TypeString},
			},
		})
	}
	if !hasStruct["Option"] {
		prelude = append(prelude, &ast.StructDecl{
			Name:       "Option",
			TypeParams: []string{"T"},
			Fields: []ast.StructField{
				{Name: "is_some", Type: ast.TypeBool},
				{Name: "value", Type: ast.Type("T")},
			},
		})
	}
	if !hasStruct["Result"] {
		prelude = append(prelude, &ast.StructDecl{
			Name:       "Result",
			TypeParams: []string{"T", "E"},
			Fields: []ast.StructField{
				{Name: "is_ok", Type: ast.TypeBool},
				{Name: "value", Type: ast.Type("T")},
				{Name: "err", Type: ast.Type("E")},
			},
		})
	}
	if !hasFunc["some"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "some",
			TypeParams: []string{"T"},
			Params:     []ast.Param{{Name: "value", Type: ast.Type("T")}},
			ReturnType: ast.Type("Option[T]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Option[T]",
					Fields: []ast.StructLitField{
						{Name: "is_some", Value: &ast.BoolExpr{Value: true}},
						{Name: "value", Value: &ast.IdentExpr{Name: "value"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["none"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "none",
			TypeParams: []string{"T"},
			Params:     []ast.Param{{Name: "fallback", Type: ast.Type("T")}},
			ReturnType: ast.Type("Option[T]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Option[T]",
					Fields: []ast.StructLitField{
						{Name: "is_some", Value: &ast.BoolExpr{Value: false}},
						{Name: "value", Value: &ast.IdentExpr{Name: "fallback"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["ok"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "ok",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "value", Type: ast.Type("T")},
				{Name: "fallback_err", Type: ast.Type("E")},
			},
			ReturnType: ast.Type("Result[T,E]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Result[T,E]",
					Fields: []ast.StructLitField{
						{Name: "is_ok", Value: &ast.BoolExpr{Value: true}},
						{Name: "value", Value: &ast.IdentExpr{Name: "value"}},
						{Name: "err", Value: &ast.IdentExpr{Name: "fallback_err"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["err"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "err",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "fallback_value", Type: ast.Type("T")},
				{Name: "err_value", Type: ast.Type("E")},
			},
			ReturnType: ast.Type("Result[T,E]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Result[T,E]",
					Fields: []ast.StructLitField{
						{Name: "is_ok", Value: &ast.BoolExpr{Value: false}},
						{Name: "value", Value: &ast.IdentExpr{Name: "fallback_value"}},
						{Name: "err", Value: &ast.IdentExpr{Name: "err_value"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["assert"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "assert",
			Params:     []ast.Param{{Name: "cond", Type: ast.TypeBool}},
			ReturnType: ast.TypeVoid,
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.UnaryExpr{Op: "!", Right: &ast.IdentExpr{Name: "cond"}},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: "__bazic_assert_failed"}, Value: &ast.BoolExpr{Value: true}},
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: "__bazic_assert_message"}, Value: &ast.StringExpr{Value: "assertion failed"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["assert_msg"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name: "assert_msg",
			Params: []ast.Param{
				{Name: "cond", Type: ast.TypeBool},
				{Name: "msg", Type: ast.TypeString},
			},
			ReturnType: ast.TypeVoid,
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.UnaryExpr{Op: "!", Right: &ast.IdentExpr{Name: "cond"}},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: "__bazic_assert_failed"}, Value: &ast.BoolExpr{Value: true}},
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: "__bazic_assert_message"}, Value: &ast.IdentExpr{Name: "msg"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["unwrap_or"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "unwrap_or",
			TypeParams: []string{"T"},
			Params: []ast.Param{
				{Name: "opt", Type: ast.Type("Option[T]")},
				{Name: "fallback", Type: ast.Type("T")},
			},
			ReturnType: ast.Type("T"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "opt"}, Field: "is_some"},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "opt"}, Field: "value"}},
					}},
					Else: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.IdentExpr{Name: "fallback"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["result_or"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "result_or",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "res", Type: ast.Type("Result[T,E]")},
				{Name: "fallback", Type: ast.Type("T")},
			},
			ReturnType: ast.Type("T"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "res"}, Field: "is_ok"},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "res"}, Field: "value"}},
					}},
					Else: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.IdentExpr{Name: "fallback"}},
					}},
				},
			}},
		})
	}
	if len(prelude) == 0 {
		return
	}
	p.Decls = append(prelude, p.Decls...)
}
