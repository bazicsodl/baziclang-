package testrun

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/compiler"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
)

type FileResult struct {
	File   string
	Passed int
	Total  int
	Output string
}

type Result struct {
	Files  []FileResult
	Passed int
	Total  int
}

type Options struct {
	Filter  string
	Backend string
}

type testKind int

const (
	testBool testKind = iota
	testVoid
)

type testSpec struct {
	Name string
	Kind testKind
}

func Run(target string) (Result, error) {
	return RunWithOptions(target, Options{})
}

func RunWithOptions(target string, opts Options) (Result, error) {
	files, err := discoverTestFiles(target)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{}, fmt.Errorf("no *_test.bz files found in %s", target)
	}
	res := Result{Files: make([]FileResult, 0, len(files))}
	for _, file := range files {
		fr, err := runFile(file, opts)
		if err != nil {
			return res, err
		}
		res.Files = append(res.Files, fr)
		res.Passed += fr.Passed
		res.Total += fr.Total
	}
	if res.Passed != res.Total {
		return res, fmt.Errorf("bazic tests failed: %d/%d passed", res.Passed, res.Total)
	}
	return res, nil
}

func discoverTestFiles(target string) ([]string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if filepath.Ext(abs) != ".bz" {
			return nil, fmt.Errorf("test target must be a .bz file or directory")
		}
		return []string{abs}, nil
	}
	out := []string{}
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.bz") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func runFile(file string, opts Options) (FileResult, error) {
	tests, err := collectTestFunctions(file, opts.Filter)
	if err != nil {
		return FileResult{}, err
	}
	if len(tests) == 0 {
		return FileResult{}, fmt.Errorf("no test functions found in %s (expected fn test_*(): bool or fn test_*(): void)", file)
	}
	harnessPath, err := writeHarness(file, tests)
	if err != nil {
		return FileResult{}, err
	}
	defer os.Remove(harnessPath)

	tmpDir, err := os.MkdirTemp("", "bazic-test-*")
	if err != nil {
		return FileResult{}, err
	}
	defer os.RemoveAll(tmpDir)
	exePath := filepath.Join(tmpDir, "bazic-test.exe")
	if err := compiler.Build(compiler.BuildOptions{
		Input:   harnessPath,
		Out:     exePath,
		Target:  "native",
		Backend: opts.Backend,
	}); err != nil {
		return FileResult{}, fmt.Errorf("build tests for %s: %w", file, err)
	}
	cmd := exec.Command(exePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return FileResult{}, fmt.Errorf("run tests for %s: %w\n%s", file, err, stderr.String())
	}
	passed, total, err := parseSummary(stdout.String())
	if err != nil {
		return FileResult{}, fmt.Errorf("parse test output for %s: %w\n%s", file, err, stdout.String())
	}
	return FileResult{
		File:   file,
		Passed: passed,
		Total:  total,
		Output: stdout.String(),
	}, nil
}

func collectTestFunctions(file string, filter string) ([]testSpec, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.New(string(data)).Tokenize()
	if err != nil {
		return nil, err
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		return nil, err
	}
	out := []testSpec{}
	for _, d := range prog.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !strings.HasPrefix(fn.Name, "test_") {
			continue
		}
		if filter != "" && !strings.Contains(fn.Name, filter) {
			continue
		}
		if len(fn.Params) != 0 {
			return nil, fmt.Errorf("invalid test function signature in %s: %s must not take params", file, fn.Name)
		}
		if fn.ReturnType == ast.TypeBool {
			out = append(out, testSpec{Name: fn.Name, Kind: testBool})
			continue
		}
		if fn.ReturnType == ast.TypeVoid {
			out = append(out, testSpec{Name: fn.Name, Kind: testVoid})
			continue
		}
		return nil, fmt.Errorf("invalid test function signature in %s: %s must return bool or void", file, fn.Name)
	}
	if filter != "" && len(out) == 0 {
		return nil, fmt.Errorf("no tests matched filter '%s' in %s", filter, file)
	}
	return out, nil
}

func writeHarness(testFile string, funcs []testSpec) (string, error) {
	testDir := filepath.Dir(testFile)
	testBase := filepath.Base(testFile)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("import \"./%s\";\n\n", testBase))
	b.WriteString("fn main(): void {\n")
	b.WriteString(fmt.Sprintf("    let total: int = %d;\n", len(funcs)))
	b.WriteString("    let passed: int = 0;\n")
	for _, fn := range funcs {
		if fn.Kind == testBool {
			b.WriteString(fmt.Sprintf("    if %s() {\n", fn.Name))
			b.WriteString(fmt.Sprintf("        println(\"PASS %s\");\n", fn.Name))
			b.WriteString("        passed = passed + 1;\n")
			b.WriteString("    } else {\n")
			b.WriteString(fmt.Sprintf("        println(\"FAIL %s\");\n", fn.Name))
			b.WriteString("    }\n")
			continue
		}
		b.WriteString("    __bazic_assert_failed = false;\n")
		b.WriteString("    __bazic_assert_message = \"\";\n")
		b.WriteString(fmt.Sprintf("    %s();\n", fn.Name))
		b.WriteString("    if __bazic_assert_failed {\n")
		b.WriteString(fmt.Sprintf("        println(\"FAIL %s\");\n", fn.Name))
		b.WriteString("        if __bazic_assert_message != \"\" {\n")
		b.WriteString("            println(__bazic_assert_message);\n")
		b.WriteString("        }\n")
		b.WriteString("    } else {\n")
		b.WriteString(fmt.Sprintf("        println(\"PASS %s\");\n", fn.Name))
		b.WriteString("        passed = passed + 1;\n")
		b.WriteString("    }\n")
	}
	b.WriteString("    println(\"__BAZIC_TEST_SUMMARY__\");\n")
	b.WriteString("    println(passed);\n")
	b.WriteString("    println(total);\n")
	b.WriteString("}\n")
	harnessPath := filepath.Join(testDir, ".bazic_test_harness.bz")
	if err := os.WriteFile(harnessPath, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return harnessPath, nil
}

func parseSummary(output string) (int, int, error) {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "__BAZIC_TEST_SUMMARY__" {
			continue
		}
		if i+2 >= len(lines) {
			return 0, 0, fmt.Errorf("incomplete summary output")
		}
		passed, err := strconv.Atoi(strings.TrimSpace(lines[i+1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid passed count: %w", err)
		}
		total, err := strconv.Atoi(strings.TrimSpace(lines[i+2]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid total count: %w", err)
		}
		return passed, total, nil
	}
	return 0, 0, fmt.Errorf("missing test summary marker")
}
