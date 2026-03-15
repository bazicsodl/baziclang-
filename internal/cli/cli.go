package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"baziclang/internal/bazfmt"
	"baziclang/internal/bazlint"
	"baziclang/internal/compiler"
	"baziclang/internal/pkgm"
	"baziclang/internal/testrun"
)

type Options struct {
	BinaryName     string
	DefaultBackend string
	Version        string
}

func Run(args []string, opts Options) int {
	name := strings.TrimSpace(opts.BinaryName)
	if name == "" {
		name = "bazic"
	}
	defaultBackend := resolveDefaultBackend(opts.DefaultBackend)
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "v0.2.0"
	}
	if len(args) < 2 {
		usage(name)
		return 1
	}
	switch args[1] {
	case "build":
		return buildCmd(name, defaultBackend, args[2:])
	case "run":
		return runCmd(name, defaultBackend, args[2:])
	case "transpile":
		return transpileCmd(name, args[2:])
	case "new":
		return newCmd(name, args[2:])
	case "init":
		return initCmd(name, args[2:])
	case "doctor":
		return doctorCmd(name, defaultBackend, args[2:])
	case "clean":
		return cleanCmd(name, args[2:])
	case "repl":
		return replCmd(name, args[2:])
	case "fmt":
		return fmtCmd(name, args[2:])
	case "test":
		return testCmd(name, defaultBackend, args[2:])
	case "lint":
		return lintCmd(name, args[2:])
	case "check":
		return checkCmd(name, args[2:])
	case "emit-llvm":
		return emitLLVMCmd(name, args[2:])
	case "pkg":
		return pkgCmd(name, args[2:])
	case "migrate":
		return migrateCmd(name, args[2:])
	case "model":
		return modelCmd(name, args[2:])
	case "openapi":
		return openapiCmd(name, args[2:])
	case "api":
		return apiCmd(name, args[2:])
	case "web":
		return webCmd(name, args[2:])
	case "ui":
		return uiCmd(name, args[2:])
	case "version":
		fmt.Printf("%s %s\n", name, version)
		return 0
	default:
		usage(name)
		return 1
	}
}

func resolveDefaultBackend(defaultBackend string) string {
	if v := strings.TrimSpace(os.Getenv("BAZIC_BACKEND")); v != "" {
		return v
	}
	defaultBackend = strings.TrimSpace(defaultBackend)
	if defaultBackend == "" {
		defaultBackend = "go"
	}
	if strings.EqualFold(defaultBackend, "llvm") {
		if _, err := exec.LookPath("clang"); err != nil {
			return "go"
		}
	}
	return defaultBackend
}

func buildCmd(binaryName string, defaultBackend string, args []string) int {
	backendProvided := hasBackendArg(args)
	targetProvided := hasTargetArg(args)
	args = normalizeArgs(args, map[string]bool{
		"-o":        true,
		"--target":  true,
		"--backend": true,
		"--keep-go": false,
	})
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	out := fs.String("o", "", "output binary path")
	target := fs.String("target", "native", "build target: native|wasm")
	backend := fs.String("backend", defaultBackend, "backend: go|llvm")
	keepGo := fs.Bool("keep-go", false, "also write generated Go source")
	_ = fs.Parse(args)
	input, err := defaultInputIfMissing(fs.NArg(), fs.Arg(0))
	if err != nil {
		return die(fmt.Sprintf("usage: %s build [--backend go|llvm] [--target native|wasm] [-o output] [--keep-go] <file.bz|dir>", binaryName))
	}
	if *target == "wasm" && !backendProvided {
		*backend = "go"
	}
	if !backendProvided && *backend == "llvm" && !runtimeAvailableForInput(input) {
		*backend = "go"
	}
	output := *out
	if output == "" {
		if !targetProvided && looksLikeWebTarget(input) {
			*target = "wasm"
			if !backendProvided {
				*backend = "go"
			}
		}
		output = defaultOutputPath(input, *target)
	}
	if err := compiler.Build(compiler.BuildOptions{Input: input, Out: output, Target: *target, KeepGoSrc: *keepGo, Backend: *backend}); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Built %s\n", output)
	return 0
}

func runCmd(binaryName string, defaultBackend string, args []string) int {
	runArgs, progArgs := splitRunArgs(args)
	backendProvided := hasBackendArg(runArgs)
	runArgs = normalizeArgs(runArgs, map[string]bool{"--backend": true})
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	backend := fs.String("backend", defaultBackend, "backend: go|llvm")
	_ = fs.Parse(runArgs)
	input, err := defaultInputIfMissing(fs.NArg(), fs.Arg(0))
	if err != nil {
		return die(fmt.Sprintf("usage: %s run [--backend go|llvm] <file.bz|dir> [-- args...]", binaryName))
	}
	if !backendProvided && *backend == "llvm" && !runtimeAvailableForInput(input) {
		*backend = "go"
	}
	if err := compiler.RunWithOptions(compiler.RunOptions{Input: input, Backend: *backend, Args: progArgs}); err != nil {
		return die(err.Error())
	}
	return 0
}

func transpileCmd(binaryName string, args []string) int {
	args = normalizeArgs(args, map[string]bool{"-o": true})
	fs := flag.NewFlagSet("transpile", flag.ExitOnError)
	out := fs.String("o", "", "output Go file path")
	_ = fs.Parse(args)
	input, err := defaultInputIfMissing(fs.NArg(), fs.Arg(0))
	if err != nil {
		return die(fmt.Sprintf("usage: %s transpile [-o out.go] <file.bz|dir>", binaryName))
	}
	goSrc, err := compiler.CompileEntryToGo(input)
	if err != nil {
		return die(err.Error())
	}
	if *out == "" {
		fmt.Println(goSrc)
		return 0
	}
	if err := os.WriteFile(*out, []byte(goSrc), 0644); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Wrote %s\n", *out)
	return 0
}

func fmtCmd(binaryName string, args []string) int {
	args = normalizeArgs(args, map[string]bool{"--check": false, "--stdout": false})
	fs := flag.NewFlagSet("fmt", flag.ExitOnError)
	check := fs.Bool("check", false, "check if file is already formatted")
	stdout := fs.Bool("stdout", false, "write formatted output to stdout (file only)")
	_ = fs.Parse(args)
	target := "."
	if fs.NArg() > 1 {
		return die(fmt.Sprintf("usage: %s fmt [--check] [--stdout] [path]", binaryName))
	}
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}
	if *stdout {
		if fs.NArg() != 1 {
			return die(fmt.Sprintf("usage: %s fmt --stdout <file>", binaryName))
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return die(err.Error())
		}
		formatted, err := bazfmt.Format(string(data))
		if err != nil {
			return die(fmt.Sprintf("%s: %v", target, err))
		}
		fmt.Println(formatted)
		return 0
	}
	files, err := bazfmt.CollectBZFiles(target)
	if err != nil {
		return die(err.Error())
	}
	if len(files) == 0 {
		return die(fmt.Sprintf("no .bz files found in %s", target))
	}
	if *check {
		notFormatted := []string{}
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				return die(err.Error())
			}
			formatted, err := bazfmt.Format(string(data))
			if err != nil {
				return die(fmt.Sprintf("%s: %v", path, err))
			}
			if string(data) != formatted {
				notFormatted = append(notFormatted, path)
			}
		}
		if len(notFormatted) > 0 {
			return die(fmt.Sprintf("%d files are not formatted:\n%s", len(notFormatted), strings.Join(notFormatted, "\n")))
		}
		fmt.Printf("Formatting OK (%d files)\n", len(files))
		return 0
	}
	changed := 0
	for _, path := range files {
		updated, err := bazfmt.FormatFile(path)
		if err != nil {
			return die(fmt.Sprintf("%s: %v", path, err))
		}
		if updated {
			changed++
			fmt.Printf("Formatted %s\n", path)
		}
	}
	if changed == 0 {
		fmt.Printf("Already formatted (%d files)\n", len(files))
		return 0
	}
	fmt.Printf("Formatted %d/%d files\n", changed, len(files))
	return 0
}

func testCmd(binaryName string, defaultBackend string, args []string) int {
	args = normalizeArgs(args, map[string]bool{"--filter": true, "--json": false, "--backend": true})
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	filter := fs.String("filter", "", "run only tests matching substring")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON output")
	backend := fs.String("backend", defaultBackend, "backend: go|llvm|all")
	_ = fs.Parse(args)
	target := "."
	if fs.NArg() > 1 {
		return die(fmt.Sprintf("usage: %s test [--backend go|llvm|all] [--filter name] [--json] [path]", binaryName))
	}
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}
	if strings.ToLower(*backend) == "all" {
		resGo, errGo := testrun.RunWithOptions(target, testrun.Options{Filter: *filter, Backend: "go"})
		resLLVM, errLLVM := testrun.RunWithOptions(target, testrun.Options{Filter: *filter, Backend: "llvm"})
		if *jsonOut {
			payload := struct {
				Go      testrun.Result `json:"go"`
				LLVM    testrun.Result `json:"llvm"`
				ErrGo   string         `json:"error_go,omitempty"`
				ErrLLVM string         `json:"error_llvm,omitempty"`
			}{
				Go:   resGo,
				LLVM: resLLVM,
			}
			if errGo != nil {
				payload.ErrGo = errGo.Error()
			}
			if errLLVM != nil {
				payload.ErrLLVM = errLLVM.Error()
			}
			data, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Println(string(data))
			if errGo != nil || errLLVM != nil {
				return 1
			}
			return 0
		}
		fmt.Printf("[go] %s: %d/%d passed\n", target, resGo.Passed, resGo.Total)
		fmt.Printf("[llvm] %s: %d/%d passed\n", target, resLLVM.Passed, resLLVM.Total)
		if errGo != nil {
			fmt.Printf("[go] error: %s\n", errGo.Error())
		}
		if errLLVM != nil {
			fmt.Printf("[llvm] error: %s\n", errLLVM.Error())
		}
		if errGo != nil || errLLVM != nil {
			return 1
		}
		return 0
	}
	res, err := testrun.RunWithOptions(target, testrun.Options{Filter: *filter, Backend: *backend})
	if *jsonOut {
		payload := struct {
			testrun.Result
			Error string `json:"error,omitempty"`
		}{
			Result: res,
		}
		if err != nil {
			payload.Error = err.Error()
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		if err != nil {
			return 1
		}
		return 0
	}
	for _, fr := range res.Files {
		fmt.Printf("%s: %d/%d passed\n", fr.File, fr.Passed, fr.Total)
	}
	fmt.Printf("bazic test summary: %d/%d passed\n", res.Passed, res.Total)
	if err != nil {
		return die(err.Error())
	}
	return 0
}

func lintCmd(binaryName string, args []string) int {
	args = normalizeArgs(args, map[string]bool{"--json": false})
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON output")
	_ = fs.Parse(args)
	target := "."
	if fs.NArg() > 1 {
		return die(fmt.Sprintf("usage: %s lint [--json] [path]", binaryName))
	}
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}
	issues, err := bazlint.Lint(target)
	if *jsonOut {
		payload := struct {
			Issues []bazlint.Issue `json:"issues"`
			Count  int             `json:"count"`
			Error  string          `json:"error,omitempty"`
		}{
			Issues: issues,
			Count:  len(issues),
		}
		if err != nil {
			payload.Error = err.Error()
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		if err != nil || len(issues) > 0 {
			return 1
		}
		return 0
	}
	if err != nil {
		return die(err.Error())
	}
	for _, issue := range issues {
		fmt.Printf("%s:%d:%d: [%s] %s\n", issue.File, issue.Line, issue.Column, issue.Rule, issue.Message)
	}
	if len(issues) > 0 {
		return die(fmt.Sprintf("lint found %d issues", len(issues)))
	}
	fmt.Println("Lint OK")
	return 0
}

func pkgCmd(binaryName string, args []string) int {
	if len(args) == 0 {
		return die(fmt.Sprintf("usage: %s pkg <init|add|sync|verify|sbom> ...", binaryName))
	}
	switch args[0] {
	case "init":
		if len(args) < 2 {
			return die(fmt.Sprintf("usage: %s pkg init <project-name>", binaryName))
		}
		wd, _ := os.Getwd()
		if err := pkgm.Init(wd, args[1]); err != nil {
			return die(err.Error())
		}
		fmt.Println("Initialized bazic.mod.json")
	case "add":
		if len(args) < 3 {
			return die(fmt.Sprintf("usage: %s pkg add <alias> <path>", binaryName))
		}
		wd, _ := os.Getwd()
		root, err := pkgm.FindProjectRoot(wd)
		if err != nil {
			return die(fmt.Sprintf("run '%s pkg init <name>' first", binaryName))
		}
		if err := pkgm.AddDep(root, args[1], args[2]); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Added dependency '%s'\n", args[1])
	case "sync":
		wd, _ := os.Getwd()
		root, err := pkgm.FindProjectRoot(wd)
		if err != nil {
			return die(fmt.Sprintf("run '%s pkg init <name>' first", binaryName))
		}
		if err := pkgm.Sync(root); err != nil {
			return die(err.Error())
		}
		fmt.Println("Synced dependencies into .bazic/pkg")
	case "verify":
		wd, _ := os.Getwd()
		root, err := pkgm.FindProjectRoot(wd)
		if err != nil {
			return die(fmt.Sprintf("run '%s pkg init <name>' first", binaryName))
		}
		if err := pkgm.Verify(root); err != nil {
			return die(err.Error())
		}
		fmt.Println("Package integrity verified")
	case "sbom":
		args = normalizeArgs(args[1:], map[string]bool{"-o": true})
		fs := flag.NewFlagSet("sbom", flag.ExitOnError)
		out := fs.String("o", "", "output SBOM file path")
		_ = fs.Parse(args)
		if fs.NArg() > 0 {
			return die(fmt.Sprintf("usage: %s pkg sbom [-o out.json]", binaryName))
		}
		wd, _ := os.Getwd()
		root, err := pkgm.FindProjectRoot(wd)
		if err != nil {
			return die(fmt.Sprintf("run '%s pkg init <name>' first", binaryName))
		}
		path := *out
		if path == "" {
			path = filepath.Join(root, "bazic.sbom.json")
		}
		if err := pkgm.WriteSBOM(root, path); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Wrote %s\n", path)
	default:
		return die(fmt.Sprintf("usage: %s pkg <init|add|sync|verify|sbom> ...", binaryName))
	}
	return 0
}

func checkCmd(binaryName string, args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	compat := fs.String("compat", "", "compatibility target (e.g. v1.0)")
	_ = fs.Parse(args)
	input, err := defaultInputIfMissing(fs.NArg(), fs.Arg(0))
	if err != nil {
		return die(fmt.Sprintf("usage: %s check <file.bz|dir>", binaryName))
	}
	target := strings.TrimSpace(*compat)
	if target == "" {
		target = strings.TrimSpace(os.Getenv("BAZIC_COMPAT_TARGET"))
	}
	if target != "" {
		if err := compatCheck(target); err != nil {
			return die(err.Error())
		}
	}
	if err := compiler.CheckEntry(input); err != nil {
		return die(err.Error())
	}
	fmt.Println("Check passed")
	return 0
}

func emitLLVMCmd(binaryName string, args []string) int {
	args = normalizeArgs(args, map[string]bool{"-o": true})
	fs := flag.NewFlagSet("emit-llvm", flag.ExitOnError)
	out := fs.String("o", "", "output LLVM IR file path")
	check := fs.Bool("check", false, "fail if unsupported LLVM features are detected")
	_ = fs.Parse(args)
	input, err := defaultInputIfMissing(fs.NArg(), fs.Arg(0))
	if err != nil {
		return die(fmt.Sprintf("usage: %s emit-llvm [-o out.ll] [--check] <file.bz|dir>", binaryName))
	}
	ir, err := compiler.CompileEntryToLLVM(input)
	if err != nil {
		return die(err.Error())
	}
	ir = compiler.MaybeInjectTargetTriple(ir)
	if *check {
		if err := compiler.RejectUnsupportedLLVM(ir); err != nil {
			return die(err.Error())
		}
	}
	if *out == "" {
		fmt.Print(ir)
		return 0
	}
	if err := os.WriteFile(*out, []byte(ir), 0644); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Wrote %s\n", *out)
	return 0
}

func usage(binaryName string) {
	display := binaryName
	if len(display) > 0 {
		display = strings.ToUpper(display[:1]) + display[1:]
	}
	fmt.Printf("%s compiler (AOT)\n", display)
	fmt.Println("Commands:")
	fmt.Printf("  %s build [--backend go|llvm] [--target native|wasm] [-o output] [--keep-go] <file.bz|dir>\n", binaryName)
	fmt.Printf("  %s run [--backend go|llvm] <file.bz|dir> [-- args...]\n", binaryName)
	fmt.Printf("  %s transpile [-o out.go] <file.bz|dir>\n", binaryName)
	fmt.Printf("  %s new <project-name>\n", binaryName)
	fmt.Printf("  %s init <project-name>\n", binaryName)
	fmt.Printf("  %s doctor\n", binaryName)
	fmt.Printf("  %s clean [--all]\n", binaryName)
	fmt.Printf("  %s repl\n", binaryName)
	fmt.Printf("  %s fmt [--check] [--stdout] [path]\n", binaryName)
	fmt.Printf("  %s test [--backend go|llvm|all] [path]\n", binaryName)
	fmt.Printf("  %s lint [path]\n", binaryName)
	fmt.Printf("  %s check [--compat v1.0] <file.bz|dir>\n", binaryName)
	fmt.Printf("  %s emit-llvm [-o out.ll] <file.bz|dir>\n", binaryName)
	fmt.Printf("  %s pkg init <project-name>\n", binaryName)
	fmt.Printf("  %s pkg add <alias> <path>\n", binaryName)
	fmt.Printf("  %s pkg sync\n", binaryName)
	fmt.Printf("  %s pkg verify\n", binaryName)
	fmt.Printf("  %s pkg sbom [-o out.json]\n", binaryName)
	fmt.Printf("  %s migrate <create|apply|rollback|status> [options]\n", binaryName)
	fmt.Printf("  %s model <init|auth|generate|migrate> [options]\n", binaryName)
	fmt.Printf("  %s openapi --routes <file.bz> --models <models.bz> --out <openapi.json>\n", binaryName)
	fmt.Printf("  %s api --routes <file.bz> --models <models.bz> --out <handlers.bz>\n", binaryName)
	fmt.Printf("  %s web <init|build|dev> [--dir path] [--port 8080]\n", binaryName)
	fmt.Printf("  %s ui <init|page|component|layout|routes|migrate-layout|build|dev> [--dir path] [--template name] [--port 8080] <name>\n", binaryName)
	fmt.Printf("  %s version\n", binaryName)
}

func die(msg string) int {
	fmt.Fprintln(os.Stderr, "error:", msg)
	return 1
}

func trimExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}

func normalizeArgs(args []string, flags map[string]bool) []string {
	opts := make([]string, 0, len(args))
	positionals := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 0 && a[0] == '-' {
			opts = append(opts, a)
			if strings.Contains(a, "=") {
				continue
			}
			if takesVal, ok := flags[a]; ok && takesVal && i+1 < len(args) {
				opts = append(opts, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return append(opts, positionals...)
}

func hasBackendArg(args []string) bool {
	for _, a := range args {
		if a == "-backend" || a == "--backend" || strings.HasPrefix(a, "-backend=") || strings.HasPrefix(a, "--backend=") {
			return true
		}
	}
	return false
}

func hasTargetArg(args []string) bool {
	for _, a := range args {
		if a == "-target" || a == "--target" || strings.HasPrefix(a, "-target=") || strings.HasPrefix(a, "--target=") {
			return true
		}
	}
	return false
}

func splitRunArgs(args []string) ([]string, []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func defaultOutputPath(input string, target string) string {
	root, name, ok := projectNameFor(input)
	if ok && filepath.Base(input) == "main.bz" {
		binDir := filepath.Join(root, "bin")
		_ = os.MkdirAll(binDir, 0755)
		ext := ""
		if target == "wasm" {
			ext = ".wasm"
		} else if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		return filepath.Join(binDir, name+ext)
	}
	ext := ""
	if target == "wasm" {
		ext = ".wasm"
	} else if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(filepath.Dir(input), trimExt(filepath.Base(input))+ext)
}

func projectNameFor(input string) (string, string, bool) {
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", "", false
	}
	root, err := pkgm.FindProjectRoot(filepath.Dir(abs))
	if err != nil {
		return "", "", false
	}
	m, err := pkgm.LoadManifest(root)
	if err != nil {
		return root, filepath.Base(root), true
	}
	name := strings.TrimSpace(m.Name)
	if name == "" {
		name = filepath.Base(root)
	}
	return root, name, true
}

func runtimeAvailableForInput(input string) bool {
	abs, err := filepath.Abs(input)
	if err != nil {
		return false
	}
	root, err := pkgm.FindProjectRoot(filepath.Dir(abs))
	if err != nil {
		root = filepath.Dir(abs)
	}
	candidates := []string{
		filepath.Join(root, "runtime", "bazic_runtime.c"),
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
			return true
		}
	}
	return false
}

func compatCheck(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	wd, _ := os.Getwd()
	root, err := pkgm.FindProjectRoot(wd)
	if err != nil {
		root = wd
	}
	specPath := filepath.Join(root, "LANGUAGE.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("compat check failed: LANGUAGE.md not found at %s", specPath)
	}
	if strings.EqualFold(target, "v1.0") && strings.Contains(string(data), "Draft") {
		return fmt.Errorf("compat check failed: LANGUAGE.md still marked Draft")
	}
	return nil
}

func looksLikeWebTarget(input string) bool {
	abs, err := filepath.Abs(input)
	if err != nil {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(abs))
	return strings.Contains(dir, "/examples/web") || strings.HasSuffix(dir, "/web")
}

func defaultInputIfMissing(narg int, arg0 string) (string, error) {
	if narg == 1 {
		return resolveInputPath(arg0)
	}
	if narg != 0 {
		return "", fmt.Errorf("expected single input")
	}
	if _, err := os.Stat("main.bz"); err == nil {
		return "main.bz", nil
	}
	return "", fmt.Errorf("missing input")
}

func resolveInputPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		mainPath := filepath.Join(path, "main.bz")
		if _, err := os.Stat(mainPath); err == nil {
			return mainPath, nil
		}
		return "", fmt.Errorf("directory missing main.bz: %s", path)
	}
	if err == nil {
		return path, nil
	}
	return "", err
}

func newCmd(binaryName string, args []string) int {
	if len(args) != 1 {
		return die(fmt.Sprintf("usage: %s new <project-name>", binaryName))
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return die("project name is required")
	}
	if err := os.MkdirAll(name, 0755); err != nil {
		return die(err.Error())
	}
	mainPath := filepath.Join(name, "main.bz")
	if _, err := os.Stat(mainPath); err == nil {
		return die("main.bz already exists in target directory")
	}
	if err := os.WriteFile(mainPath, []byte(defaultMainTemplate()), 0644); err != nil {
		return die(err.Error())
	}
	testPath := filepath.Join(name, "main_test.bz")
	if _, err := os.Stat(testPath); err == nil {
		return die("main_test.bz already exists in target directory")
	}
	if err := os.WriteFile(testPath, []byte(defaultTestTemplate()), 0644); err != nil {
		return die(err.Error())
	}
	readmePath := filepath.Join(name, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		_ = os.WriteFile(readmePath, []byte(defaultReadmeTemplate(name)), 0644)
	}
	wd, _ := os.Getwd()
	root := filepath.Join(wd, name)
	if err := pkgm.Init(root, name); err != nil {
		return die(err.Error())
	}
	if err := maybeAddStdlib(root); err != nil {
		fmt.Printf("warning: stdlib sync failed: %s\n", err.Error())
	}
	if err := ensureGitignore(filepath.Join(name, ".gitignore")); err != nil {
		return die(err.Error())
	}
	fmt.Printf("Created %s\n", name)
	fmt.Printf("Next: cd %s && %s run\n", name, binaryName)
	return 0
}

func defaultMainTemplate() string {
	return "fn main(): void {\n    println(\"hello from bazic\");\n}\n"
}

func defaultTestTemplate() string {
	return "fn test_hello(): bool {\n    return true;\n}\n"
}

func defaultReadmeTemplate(name string) string {
	var b strings.Builder
	b.WriteString("# ")
	if name == "" {
		b.WriteString("Bazic App")
	} else {
		b.WriteString(name)
	}
	b.WriteString("\n\n")
	b.WriteString("Run:\n")
	b.WriteString("```powershell\n")
	b.WriteString(".\\bin\\bazic.exe run\n")
	b.WriteString("```\n\n")
	b.WriteString("Test:\n")
	b.WriteString("```powershell\n")
	b.WriteString(".\\bin\\bazic.exe test .\n")
	b.WriteString("```\n")
	return b.String()
}

func initCmd(binaryName string, args []string) int {
	if len(args) != 1 {
		return die(fmt.Sprintf("usage: %s init <project-name>", binaryName))
	}
	name := strings.TrimSpace(args[0])
	if name == "" {
		return die("project name is required")
	}
	wd, _ := os.Getwd()
	if err := pkgm.Init(wd, name); err != nil {
		return die(err.Error())
	}
	if err := maybeAddStdlib(wd); err != nil {
		fmt.Printf("warning: stdlib sync failed: %s\n", err.Error())
	}
	if err := ensureGitignore(filepath.Join(wd, ".gitignore")); err != nil {
		return die(err.Error())
	}
	fmt.Println("Initialized bazic.mod.json")
	return 0
}

func maybeAddStdlib(root string) error {
	stdPath, ok := pkgm.DetectStdlibPath()
	if !ok {
		return nil
	}
	if err := pkgm.AddDep(root, "std", stdPath); err != nil {
		return err
	}
	return pkgm.Sync(root)
}

func ensureGitignore(path string) error {
	entries := []string{".bazic/", "*.exe", "*.wasm", "bin/"}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existing := string(data)
	var b strings.Builder
	b.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		b.WriteString("\n")
	}
	added := false
	for _, e := range entries {
		if strings.Contains(existing, e) {
			continue
		}
		b.WriteString(e)
		b.WriteString("\n")
		added = true
	}
	if !added && err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func doctorCmd(binaryName string, defaultBackend string, args []string) int {
	if len(args) != 0 {
		return die(fmt.Sprintf("usage: %s doctor", binaryName))
	}
	fail := false
	if defaultBackend == "llvm" {
		if _, err := exec.LookPath("clang"); err != nil {
			fmt.Println("clang: missing (required for LLVM backend)")
			fail = true
		} else {
			fmt.Println("clang: ok")
		}
	} else {
		fmt.Println("clang: optional")
	}
	if v := strings.TrimSpace(os.Getenv("BAZIC_CLANG")); v != "" {
		fmt.Printf("BAZIC_CLANG: %s\n", v)
	}
	if v := strings.TrimSpace(os.Getenv("BAZIC_CLANG_FLAGS")); v != "" {
		fmt.Printf("BAZIC_CLANG_FLAGS: %s\n", v)
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Println("go: missing (required for Go backend / wasm builds)")
	} else {
		fmt.Println("go: ok")
	}
	if stdPath, ok := pkgm.DetectStdlibPath(); ok {
		fmt.Printf("stdlib: %s\n", stdPath)
	} else {
		fmt.Println("stdlib: not found (set BAZIC_STDLIB or install with std next to bazic)")
	}
	wd, _ := os.Getwd()
	if root, err := pkgm.FindProjectRoot(wd); err == nil {
		fmt.Printf("project: %s\n", root)
		if err := pkgm.Verify(root); err != nil {
			fmt.Printf("pkg verify: failed (%s)\n", err.Error())
			fail = true
		} else {
			fmt.Println("pkg verify: ok")
		}
	} else {
		fmt.Println("project: none")
	}
	if fail {
		return 1
	}
	return 0
}

func cleanCmd(binaryName string, args []string) int {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	all := fs.Bool("all", false, "also remove lockfile and sbom")
	_ = fs.Parse(args)
	if fs.NArg() != 0 {
		return die(fmt.Sprintf("usage: %s clean [--all]", binaryName))
	}
	wd, _ := os.Getwd()
	root, err := pkgm.FindProjectRoot(wd)
	if err != nil {
		return die("clean must be run inside a bazic project (missing bazic.mod.json)")
	}
	dirs := []string{filepath.Join(root, ".bazic"), filepath.Join(root, "bin")}
	for _, d := range dirs {
		if err := os.RemoveAll(d); err != nil {
			return die(err.Error())
		}
	}
	if *all {
		_ = os.Remove(filepath.Join(root, pkgm.LockfileFile))
		_ = os.Remove(filepath.Join(root, "bazic.sbom.json"))
	}
	fmt.Println("Cleaned build artifacts")
	return 0
}

func replCmd(binaryName string, args []string) int {
	if len(args) != 0 {
		return die(fmt.Sprintf("usage: %s repl", binaryName))
	}
	fmt.Println("Bazic REPL (type :quit to exit)")
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">> ")
		line, err := in.ReadString('\n')
		if err != nil && strings.TrimSpace(line) == "" {
			return 0
		}
		src := strings.TrimSpace(line)
		if src == "" {
			continue
		}
		if src == ":quit" || src == ":q" || src == ":exit" {
			return 0
		}
		if !strings.HasSuffix(src, ";") {
			src += ";"
		}
		prog := "fn main(): void { println(" + src + "); }"
		out, err := compiler.CompileToGo(prog)
		if err != nil {
			fmt.Println("error:", err.Error())
			continue
		}
		tmpDir, err := os.MkdirTemp("", "bazic-repl-*")
		if err != nil {
			fmt.Println("error:", err.Error())
			continue
		}
		goFile := filepath.Join(tmpDir, "main.go")
		exe := filepath.Join(tmpDir, "bazic-repl.exe")
		if err := os.WriteFile(goFile, []byte(out), 0644); err != nil {
			fmt.Println("error:", err.Error())
			continue
		}
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-buildid=", "-o", exe, goFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("error: go build failed")
			continue
		}
		run := exec.Command(exe)
		run.Stdout = os.Stdout
		run.Stderr = os.Stderr
		_ = run.Run()
		_ = os.RemoveAll(tmpDir)
	}
}

func webCmd(binaryName string, args []string) int {
	if len(args) == 0 {
		return die(fmt.Sprintf("usage: %s web <init|build|dev> [--dir path] [--port 8080]", binaryName))
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	dir := fs.String("dir", filepath.Join("examples", "web"), "web app directory")
	port := fs.Int("port", 8080, "dev server port")
	_ = fs.Parse(args[1:])

	switch sub {
	case "init":
		target := *dir
		if fs.NArg() > 0 {
			target = fs.Arg(0)
		}
		if err := initWeb(target); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created web app in %s\n", target)
		return 0
	case "build":
		if err := buildWeb(*dir); err != nil {
			return die(err.Error())
		}
		fmt.Println("Web build complete")
		return 0
	case "dev":
		if err := buildWeb(*dir); err != nil {
			return die(err.Error())
		}
		return serveWeb(*dir, *port)
	default:
		return die(fmt.Sprintf("usage: %s web <init|build|dev> [--dir path] [--port 8080]", binaryName))
	}
}

func uiCmd(binaryName string, args []string) int {
	if len(args) == 0 {
		return die(fmt.Sprintf("usage: %s ui <init|page|component|layout|routes|migrate-layout|build|dev> [--dir path] [--template name] [--port 8080] <name>", binaryName))
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	dir := fs.String("dir", filepath.Join("examples", "web"), "ui app directory")
	template := fs.String("template", "default", "ui template: default|react|react-ts")
	port := fs.Int("port", 8080, "dev server port")
	_ = fs.Parse(args[1:])

	switch sub {
	case "init":
		target := *dir
		if fs.NArg() > 0 {
			target = fs.Arg(0)
		}
		if err := initUI(target, *template); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created Bazic UI app in %s\n", target)
		return 0
	case "build":
		if err := buildWeb(*dir); err != nil {
			return die(err.Error())
		}
		fmt.Println("UI build complete")
		return 0
	case "dev":
		if err := buildWeb(*dir); err != nil {
			return die(err.Error())
		}
		return serveWeb(*dir, *port)
	case "page":
		if fs.NArg() == 0 {
			return die(fmt.Sprintf("usage: %s ui page <name> [--dir path]", binaryName))
		}
		name := strings.TrimSpace(fs.Arg(0))
		if name == "" {
			return die("page name is required")
		}
		if err := createUIPage(*dir, name); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created page %s in %s\n", name, *dir)
		return 0
	case "component":
		if fs.NArg() == 0 {
			return die(fmt.Sprintf("usage: %s ui component <name> [--dir path]", binaryName))
		}
		name := strings.TrimSpace(fs.Arg(0))
		if name == "" {
			return die("component name is required")
		}
		if err := createUIComponent(*dir, name); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created component %s in %s\n", name, *dir)
		return 0
	case "layout":
		if err := createUILayout(*dir); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Created layout in %s\n", *dir)
		return 0
	case "routes":
		routes, err := listUIRoutes(*dir)
		if err != nil {
			return die(err.Error())
		}
		for _, r := range routes {
			fmt.Println(r)
		}
		return 0
	case "migrate-layout":
		if err := migrateUILayout(*dir); err != nil {
			return die(err.Error())
		}
		fmt.Printf("Migrated layout in %s\n", *dir)
		return 0
	default:
		return die(fmt.Sprintf("usage: %s ui <init|page|component|layout|routes|migrate-layout|build|dev> [--dir path] [--template name] [--port 8080] <name>", binaryName))
	}
}

func buildWeb(dir string) error {
	appPath := filepath.Join(dir, "app.bz")
	if _, err := os.Stat(appPath); err != nil {
		return fmt.Errorf("web build failed: %s not found", appPath)
	}
	outWasm := filepath.Join(dir, "app.wasm")
	if err := compiler.Build(compiler.BuildOptions{
		Input:   appPath,
		Out:     outWasm,
		Target:  "wasm",
		Backend: "go",
	}); err != nil {
		return err
	}
	return ensureWasmExec(dir)
}

func initWeb(dir string) error {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	writeIfMissing := func(name, data string) error {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(data), 0644)
	}
	if err := writeIfMissing("app.bz", webAppTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing(filepath.Join("pages", "home.bz"), webPageHomeTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing(filepath.Join("pages", "about.bz"), webPageAboutTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing("app.js", webAppJsTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing("index.html", webIndexTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing("theme.css", webThemeTemplate()); err != nil {
		return err
	}
	if err := writeIfMissing("style.css", webStyleTemplate()); err != nil {
		return err
	}
	return nil
}

func initUI(dir string, template string) error {
	if err := initWeb(dir); err != nil {
		return err
	}
	_ = createUILayout(dir)
	tpl := strings.ToLower(strings.TrimSpace(template))
	switch tpl {
	case "", "default":
		return nil
	case "react":
		return initUIReact(dir)
	case "react-ts", "reactts", "react_typescript":
		return initUIReactTS(dir)
	default:
		return fmt.Errorf("unknown ui template: %s", template)
	}
}

func createUILayout(dir string) error {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	layoutPath := filepath.Join(dir, "layout.bz")
	if _, err := os.Stat(layoutPath); err == nil {
		return fmt.Errorf("layout already exists: %s", layoutPath)
	}
	if err := os.WriteFile(layoutPath, []byte(uiLayoutTemplate()), 0644); err != nil {
		return err
	}
	return nil
}

func webAppTemplate() string {
	return `import "std";
// BAZIC_UI_PAGES_IMPORTS_START
import "./pages/home.bz";
import "./pages/about.bz";
// BAZIC_UI_PAGES_IMPORTS_END

fn render_page(page: string, clicks: int, name: string): string {
    if page == "about" {
        return page_about(clicks, name);
    }
    // BAZIC_UI_PAGES_ROUTER
    return page_home(clicks, name);
}

fn render_ui(page: string, clicks: int, name: string): void {
    let nav = ui_element("nav", ui_props("nav", "nav"),
        ui_children_two(
            ui_nav_link("Home", "nav-home", page == "home"),
            ui_nav_link("About", "nav-about", page == "about")
        )
    );
    let hero = ui_element("div", ui_props("", "hero"),
        ui_children_three(
            ui_button("Click Me", "btn", "ghost"),
            ui_bind_input("name", "Type your name", "name"),
            ui_p("Clicks: " + str(clicks))
        )
    );
    let content = ui_element("div", ui_props("", "content"), ui_children_two(hero, render_page(page, clicks, name)));
    let root = ui_layout("Bazic UI (WASM)", nav, content);
    let _ = ui_render(root);
}

fn main(): void {
    let clicks = ui_state_int_get("clicks", 0);
    let page = ui_route_get("home");
    let name = ui_state_get("name", "");
    render_ui(page, clicks, name);

    let i = 0;
    while i < 24 {
        let ev = ui_event_type();
        let target = ui_event_target();
        let val = ui_event_value();
        if ev == "click" && target == "btn" {
            clicks = clicks + 1;
            ui_event_clear();
            let _ = ui_state_int_set("clicks", clicks);
        }
        if ev == "input" && target == "name" {
            ui_event_clear();
            name = val;
            let _ = ui_state_set("name", name);
        }
        if ev == "nav" {
            ui_event_clear();
            if val == "nav-about" { page = "about"; }
            if val == "nav-home" { page = "home"; }
            let _ = ui_state_set("page", page);
            let _ = ui_route_set(page);
        }
        if ev == "route" && val != "" {
            ui_event_clear();
            page = val;
            let _ = ui_state_set("page", page);
        }
        render_ui(page, clicks, name);
        i = i + 1;
    }
}
`
}
func initUIReact(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte(webReactIndexTemplate()), 0644); err != nil {
		return err
	}
	adapterPath := filepath.Join(dir, "react_adapter.js")
	if err := os.WriteFile(adapterPath, []byte(webReactAdapterTemplate()), 0644); err != nil {
		return err
	}
	return nil
}

func initUIReactTS(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte(webReactIndexTemplate()), 0644); err != nil {
		return err
	}
	adapterPath := filepath.Join(dir, "react_adapter.js")
	if err := os.WriteFile(adapterPath, []byte(webReactAdapterTemplate()), 0644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "react_adapter.ts"), []byte(webReactAdapterTSTemplate()), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(webReactTSPackageTemplate()), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(webReactTSTsconfigTemplate()), 0644); err != nil {
		return err
	}
	return nil
}

func webAppJsTemplate() string {
	return `const output = document.getElementById("output");
const statusEl = document.getElementById("status");
const runBtn = document.getElementById("run");
const clearBtn = document.getElementById("clear");
const uiRoot = document.getElementById("ui");

function setStatus(msg) {
  statusEl.textContent = msg;
}

function append(line) {
  output.textContent += line + "\n";
  output.scrollTop = output.scrollHeight;
}

function applyProps(el, props) {
  if (!props) return;
  if (props.id) el.id = String(props.id);
  if (props.class) el.className = String(props.class);
  if (props.key) el.dataset.key = String(props.key);
  for (const [key, value] of Object.entries(props)) {
    if (key === "id" || key === "class" || key === "key" || key === "value" || key === "checked") continue;
    if (value === undefined || value === null || value === false) {
      el.removeAttribute(key);
    } else if (value === true) {
      el.setAttribute(key, "");
    } else {
      el.setAttribute(key, String(value));
    }
  }
}

function renderNode(node) {
  if (!node) return document.createTextNode("");
  if (node.t === "text") {
    return document.createTextNode(String(node.v || ""));
  }
  if (node.t === "elem") {
    const el = document.createElement(node.tag || "div");
    const props = node.props || {};
    applyProps(el, props);
    if (node.tag === "a") {
      el.href = "#";
    }
    if (node.tag === "input") {
      if (props.type) el.type = String(props.type);
      if (props.type === "checkbox" && props.checked !== undefined) {
        el.checked = !!props.checked;
      }
      if (props.value !== undefined) {
        const next = String(props.value);
        if (el.value !== next) {
          el.value = next;
        }
      }
    }
    if (node.tag === "select" && props.value !== undefined) {
      const next = String(props.value);
      if (el.value !== next) {
        el.value = next;
      }
    }
    const children = Array.isArray(node.children) ? node.children : [];
    for (const child of children) {
      el.appendChild(renderNode(child));
    }
    return el;
  }
  return document.createTextNode("");
}

function patchNode(el, node) {
  if (!node) {
    return null;
  }
  if (node.t === "text") {
    const text = String(node.v || "");
    if (el && el.nodeType === Node.TEXT_NODE) {
      if (el.nodeValue !== text) el.nodeValue = text;
      return el;
    }
    return document.createTextNode(text);
  }
  if (node.t === "elem") {
    const tag = node.tag || "div";
    if (!el || el.nodeType !== Node.ELEMENT_NODE || el.tagName.toLowerCase() !== tag) {
      return renderNode(node);
    }
    const props = node.props || {};
    applyProps(el, props);
    if (tag === "a") {
      el.href = "#";
    }
    if (tag === "input") {
      if (props.type) el.type = String(props.type);
      if (props.type === "checkbox" && props.checked !== undefined) {
        el.checked = !!props.checked;
      }
      if (props.value !== undefined) {
        const next = String(props.value);
        if (el.value !== next) el.value = next;
      }
    }
    if (tag === "select" && props.value !== undefined) {
      const next = String(props.value);
      if (el.value !== next) el.value = next;
    }
    const nextChildren = Array.isArray(node.children) ? node.children : [];
    const keyed = new Map();
    for (const child of el.childNodes) {
      if (child.nodeType === Node.ELEMENT_NODE && child.dataset && child.dataset.key) {
        keyed.set(child.dataset.key, child);
      }
    }
    const max = Math.max(el.childNodes.length, nextChildren.length);
    for (let i = 0; i < max; i++) {
      const next = nextChildren[i];
      const key = next && next.props && next.props.key ? String(next.props.key) : "";
      const child = key && keyed.has(key) ? keyed.get(key) : el.childNodes[i];
      if (!next) {
        if (child) el.removeChild(child);
        continue;
      }
      const patched = patchNode(child, next);
      if (!child) {
        el.appendChild(patched);
      } else if (patched !== child) {
        el.replaceChild(patched, child);
      }
    }
    return el;
  }
  return el || document.createTextNode("");
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  if (!uiRoot) return;
  if (globalThis.BAZIC_UI_LAST === jsonText) {
    return;
  }
  globalThis.BAZIC_UI_LAST = jsonText;
  try {
    const tree = JSON.parse(jsonText);
    const current = uiRoot.firstChild;
    const patched = patchNode(current, tree);
    if (!current && patched) {
      uiRoot.appendChild(patched);
    } else if (patched && patched !== current) {
      uiRoot.replaceChild(patched, current);
    }
  } catch (e) {
    uiRoot.textContent = "UI JSON parse error";
  }
};

const originalLog = console.log;
console.log = (...args) => {
  append(args.join(" "));
  originalLog(...args);
};

async function runBazic() {
  output.textContent = "";
  setStatus("loading wasm...");
  const go = new Go();
  const now = new Date().toISOString();
  go.argv = ["bazic", "web", now];
  go.env = Object.assign({}, go.env || {}, {
    BAZIC_MESSAGE: "hello from js",
    BAZIC_TS: now
  });
  const routeFromHash = () => {
    const hash = window.location.hash || "";
    let route = hash.replace(/^#/, "");
    if (route.startsWith("/")) {
      route = route.slice(1);
    }
    if (route === "" || route === "/") return "home";
    return route;
  };
  globalThis.BAZIC_WEB = {
    _data: new Map(),
    get(key) {
      const k = String(key);
      if (this._data.has(k)) return this._data.get(k);
      try {
        const v = localStorage.getItem(k);
        if (v !== null) return v;
      } catch (e) {}
      return undefined;
    },
    set(key, value) {
      const k = String(key);
      const v = String(value);
      this._data.set(k, v);
      try {
        localStorage.setItem(k, v);
      } catch (e) {}
      if (k === "focus") {
        const el = document.getElementById(v);
        if (el && typeof el.focus === "function") {
          el.focus();
        }
      }
      if (k === "route") {
        const next = v === "home" ? "#/" : "#/" + v;
        if (window.location.hash !== next) {
          window.location.hash = next;
        }
      }
      if (String(key) === "ui" && typeof globalThis.BAZIC_UI_RENDER === "function") {
        try {
          globalThis.BAZIC_UI_RENDER(String(value));
        } catch (e) {
          append("UI render error: " + String(e));
        }
      }
      return true;
    }
  };
  globalThis.BAZIC_WEB.set("route", routeFromHash());
  window.addEventListener("hashchange", () => {
    const route = routeFromHash();
    globalThis.BAZIC_WEB.set("route", route);
    globalThis.BAZIC_WEB.set("event", "route");
    globalThis.BAZIC_WEB.set("event_type", "route");
    globalThis.BAZIC_WEB.set("event_target", "hash");
    globalThis.BAZIC_WEB.set("event_value", route);
  });
  globalThis.BAZIC_WEB.set("payload", JSON.stringify({ ok: true, ts: now, name: "bazic" }));
  if (uiRoot) {
    const emitEvent = (type, targetId, value, action) => {
      globalThis.BAZIC_WEB.set("event", type);
      globalThis.BAZIC_WEB.set("event_type", type);
      globalThis.BAZIC_WEB.set("event_target", String(targetId || ""));
      globalThis.BAZIC_WEB.set("event_value", String(value || ""));
      if (action !== undefined) {
        globalThis.BAZIC_WEB.set("event_action", String(action || ""));
      }
    };
    uiRoot.addEventListener("click", (e) => {
      const target = e.target;
      if (!target) return;
      const actionEl = target.closest ? target.closest("[data-action]") : null;
      if (actionEl) {
        emitEvent("action", actionEl.getAttribute("data-action"), actionEl.id || "", actionEl.getAttribute("data-action"));
        if (actionEl.getAttribute("data-stop") === "1") {
          e.stopPropagation();
        }
        return;
      }
      if (!target.id) return;
      if (target.id === "btn") {
        emitEvent("click", "btn", "");
      }
      if (target.id === "focus-note") {
        emitEvent("click", "focus-note", "");
      }
      if (target.id === "form-submit") {
        emitEvent("click", "form-submit", "");
      }
      if (target.id === "toast-btn") {
        emitEvent("click", "toast-btn", "");
      }
      if (target.id === "toast-clear") {
        emitEvent("click", "toast-clear", "");
      }
      if (target.id === "tab-overview") {
        emitEvent("click", "tab-overview", "");
      }
      if (target.id === "tab-stats") {
        emitEvent("click", "tab-stats", "");
      }
      if (target.id === "tab-alerts") {
        emitEvent("click", "tab-alerts", "");
      }
      if (target.id === "nav-home" || target.id === "nav-about" || target.id === "nav-components" || target.id === "nav-dashboard" || target.id === "nav-form") {
        emitEvent("nav", target.id, target.id);
        return;
      }
      if (target.id === "table-empty" || target.id === "table-sort-reset" || target.id === "sort-latency" || target.id === "sort-status") {
        emitEvent("click", target.id, "");
        return;
      }
      if (target.id === "menu-toggle" || target.id === "menu-starter" || target.id === "menu-team" || target.id === "menu-enterprise") {
        emitEvent("click", target.id, "");
        return;
      }
      emitEvent("click", target.id, "");
    });
    const emitInput = (target) => {
      if (!target || !target.id) return;
      const type = target.type || "";
      const value = type === "checkbox" ? String(!!target.checked) : String(target.value || "");
      let handled = false;
      if (target.id === "name") {
        emitEvent("input", "name", value);
        handled = true;
      }
      if (target.id === "note") {
        emitEvent("input", "note", value);
        handled = true;
      }
      if (target.id === "form-email") {
        emitEvent("input", "form-email", value);
        handled = true;
      }
      if (target.id === "form-company") {
        emitEvent("input", "form-company", value);
        handled = true;
      }
      if (target.id === "plan") {
        emitEvent("input", "plan", value);
        handled = true;
      }
      if (target.id === "opt-in") {
        emitEvent("input", "opt-in", value);
        handled = true;
      }
      if (target.id === "dark-mode") {
        emitEvent("input", "dark-mode", value);
        handled = true;
      }
      if (target.id === "volume") {
        emitEvent("input", "volume", value);
        handled = true;
      }
      if (!handled) {
        emitEvent("input", target.id, value);
      }
    };
    uiRoot.addEventListener("input", (e) => {
      emitInput(e.target);
    });
    uiRoot.addEventListener("change", (e) => {
      emitInput(e.target);
    });
  }
  let resp = await fetch("app.wasm");
  if (!resp.ok) {
    resp = await fetch("../../app.wasm");
  }
  const buffer = await resp.arrayBuffer();
  const { instance } = await WebAssembly.instantiate(buffer, go.importObject);
  setStatus("running");
  try {
    await go.run(instance);
  } finally {
    setStatus("done");
  }
}

runBtn.addEventListener("click", () => {
  runBazic().catch(err => {
    setStatus("error");
    append(String(err));
  });
});

clearBtn.addEventListener("click", () => {
  output.textContent = "";
  setStatus("idle");
});
`
}

func webIndexTemplate() string {
	return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Bazic Web</title>
    <link rel="stylesheet" href="theme.css" />
    <link rel="stylesheet" href="style.css" />
  </head>
  <body>
    <main class="wrap">
      <header>
        <div class="badge">WASM</div>
        <h1>Bazic Web</h1>
        <p>Straightforward Bazic UI demo: run, click, render.</p>
      </header>
      <section class="panel">
        <div class="row">
          <button id="run">Run Bazic</button>
          <button id="clear" class="ghost">Clear</button>
          <span class="status" id="status">idle</span>
        </div>
        <pre id="output"></pre>
      </section>
      <section class="panel">
        <h2>Bazic UI</h2>
        <div id="ui" class="ui-root"></div>
      </section>
    </main>
    <script src="wasm_exec.js"></script>
    <script src="app.js"></script>
  </body>
</html>
`
}
func webStyleTemplate() string {
	return `:root {
  color-scheme: light dark;
  font-family: "Space Grotesk", "IBM Plex Sans", "Segoe UI", sans-serif;
  background: var(--ui-bg);
  color: var(--ui-ink);
}

body {
  margin: 0;
  padding: 0;
  background:
    radial-gradient(circle at 15% 15%, rgba(45, 122, 107, 0.12), transparent 45%),
    radial-gradient(circle at 85% 0%, rgba(15, 42, 39, 0.08), transparent 40%),
    var(--ui-bg);
}

.wrap {
  max-width: 980px;
  margin: 0 auto;
  padding: 56px 24px;
}

header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 24px;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--ui-accent);
  color: var(--ui-accent-ink);
  width: fit-content;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

header h1 {
  margin: 0;
  font-size: 40px;
}

header p {
  margin: 0;
  color: var(--ui-muted);
  font-size: 16px;
}

.panel {
  background: linear-gradient(160deg, var(--ui-panel) 0%, rgba(255, 255, 255, 0.7) 100%);
  border: 1px solid var(--ui-panel-border);
  border-radius: 20px;
  padding: 22px;
  box-shadow: 0 12px 30px var(--ui-shadow);
}

.row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

button {
  background: var(--ui-nav);
  color: var(--ui-nav-ink);
  border: none;
  padding: 10px 18px;
  border-radius: 10px;
  font-size: 14px;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

button:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 18px rgba(15, 26, 23, 0.24);
}

button.ghost {
  background: transparent;
  color: var(--ui-ink);
  border: 1px solid var(--ui-panel-border);
}

.status {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--ui-muted);
}

pre {
  background: #0f1a17;
  color: #e9f3ef;
  padding: 18px;
  border-radius: 14px;
  min-height: 160px;
  overflow: auto;
}

.ui-root {
  min-height: 120px;
  border: 1px dashed var(--ui-panel-border);
  padding: 12px;
  border-radius: 12px;
  background: var(--ui-panel);
}

.ui-root :focus-visible {
  outline: 2px solid var(--ui-focus);
  outline-offset: 2px;
}

.card {
  padding: 12px;
  border-radius: 12px;
  background: var(--ui-card);
  box-shadow: 0 6px 18px var(--ui-shadow);
}

.card-title {
  margin: 0 0 8px 0;
  font-size: 18px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.title {
  margin: 0 0 8px 0;
  font-size: 20px;
}

.muted {
  margin: 0;
  color: var(--ui-muted);
}

.nav {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
}

.link {
  text-decoration: none;
  color: var(--ui-ink);
  font-weight: 600;
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid transparent;
}

.link.active {
  background: var(--ui-nav);
  color: var(--ui-nav-ink);
  border-color: var(--ui-nav);
}

.hero {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.layout-header {
  margin-bottom: 8px;
}

.layout-main {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section {
  padding: 12px;
  border-radius: 12px;
  background: var(--ui-panel);
  border: 1px solid var(--ui-panel-border);
}

.section-title {
  margin: 0 0 8px 0;
  font-size: 18px;
}

.pages {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 10px;
}

@media (max-width: 640px) {
  header h1 {
    font-size: 32px;
  }
}
`
}
func webThemeTemplate() string {
	return `:root {
  --ui-bg: #f3f6f4;
  --ui-panel: #ffffff;
  --ui-panel-border: #d7e0dc;
  --ui-ink: #0f1a17;
  --ui-muted: #4e645f;
  --ui-accent: #2d7a6b;
  --ui-accent-ink: #f7fff9;
  --ui-card: #ffffff;
  --ui-shadow: rgba(15, 26, 23, 0.12);
  --ui-nav: #0f2a27;
  --ui-nav-ink: #f7fff9;
  --ui-focus: #2d7a6b;
  --ui-radius: 14px;
}

@media (prefers-color-scheme: dark) {
  :root {
    --ui-bg: #0e1412;
    --ui-panel: #141d1a;
    --ui-panel-border: #2a3a35;
    --ui-ink: #e6f1ed;
    --ui-muted: #9fb2ac;
    --ui-accent: #53c2a7;
    --ui-accent-ink: #0f1a17;
    --ui-card: #111815;
    --ui-shadow: rgba(0, 0, 0, 0.45);
    --ui-nav: #53c2a7;
    --ui-nav-ink: #0f1a17;
    --ui-focus: #53c2a7;
  }
}
`
}
func webReactIndexTemplate() string {
	return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Bazic UI (React)</title>
    <link rel="stylesheet" href="theme.css" />
    <link rel="stylesheet" href="style.css" />
  </head>
  <body>
    <main class="wrap">
      <header>
        <div class="badge">WASM</div>
        <h1>Bazic UI (React)</h1>
        <p>React adapter rendering the Bazic UI tree.</p>
      </header>
      <section class="panel">
        <div class="row">
          <button id="run">Run Bazic</button>
          <button id="clear" class="ghost">Clear</button>
          <span class="status" id="status">idle</span>
        </div>
        <pre id="output"></pre>
      </section>
      <section class="panel">
        <h2>Bazic UI</h2>
        <div id="ui" class="ui-root"></div>
      </section>
    </main>
    <script crossorigin src="https://unpkg.com/react@18/umd/react.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"></script>
    <script src="react_adapter.js"></script>
    <script src="wasm_exec.js"></script>
    <script src="app.js"></script>
  </body>
</html>
`
}

func webReactAdapterTemplate() string {
	return `if (typeof globalThis.BAZIC_UI_RENDER !== "function") {
  globalThis.BAZIC_UI_RENDER = () => {};
}

function toReact(node) {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  if (node.t === "elem") {
    const props = node.props || {};
    const children = Array.isArray(node.children) ? node.children.map(toReact) : [];
    return React.createElement(node.tag || "div", props, ...children);
  }
  return null;
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  if (!globalThis.React || !globalThis.ReactDOM) return;
  const root = document.getElementById("ui");
  if (!root) return;
  const tree = JSON.parse(jsonText);
  const element = toReact(tree);
  ReactDOM.render(element, root);
};
`
}

func webReactAdapterTSTemplate() string {
	return `declare const React: any;
declare const ReactDOM: any;

type UINode = {
  t: string;
  v?: string;
  tag?: string;
  props?: Record<string, unknown>;
  children?: UINode[];
};

if (typeof (globalThis as any).BAZIC_UI_RENDER !== "function") {
  (globalThis as any).BAZIC_UI_RENDER = () => {};
}

function toReact(node: UINode | null): any {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  if (node.t === "elem") {
    const props = node.props || {};
    const children = Array.isArray(node.children) ? node.children.map(toReact) : [];
    return React.createElement(node.tag || "div", props, ...children);
  }
  return null;
}

(globalThis as any).BAZIC_UI_RENDER = (jsonText: string) => {
  if (!React || !ReactDOM) return;
  const root = document.getElementById("ui");
  if (!root) return;
  const tree = JSON.parse(jsonText) as UINode;
  const element = toReact(tree);
  ReactDOM.render(element, root);
};
`
}

func webReactTSPackageTemplate() string {
	return `{
  "name": "bazic-ui-react-ts",
  "private": true,
  "devDependencies": {
    "esbuild": "^0.21.5"
  },
  "scripts": {
    "build:ui": "esbuild src/react_adapter.ts --bundle --format=iife --outfile=react_adapter.js",
    "watch:ui": "esbuild src/react_adapter.ts --bundle --format=iife --outfile=react_adapter.js --watch"
  }
}
`
}

func webReactTSTsconfigTemplate() string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "strict": false,
    "skipLibCheck": true
  }
}
`
}

func uiLayoutTemplate() string {
	return `import "std";

fn ui_layout_shell(title: string, nav_json: string, body_json: string): string {
    let header = ui_element("header", ui_props("", "layout-header"), ui_children_one(ui_h1(title)));
    let main = ui_element("main", ui_props("", "layout-main"), ui_children_two(nav_json, body_json));
    return ui_app_root(ui_component("App", "{}", ui_children_two(header, main)));
}
`
}

func webPageHomeTemplate() string {
	return `import "std";

fn page_home(clicks: int, name: string): string {
    let greet = "Welcome to Bazic UI.";
    if name != "" {
        greet = "Welcome, " + name + ".";
    }
    let summary = "Clicks so far: " + str(clicks);
    let body = ui_element("div", ui_props("", "stack"), ui_children_two(ui_p(greet), ui_p(summary)));
    return ui_section("Home", body);
}
`
}
func webPageAboutTemplate() string {
	return `import "std";

fn page_about(clicks: int, name: string): string {
    let intro = "Bazic UI is a small JSON UI DSL rendered in the browser.";
    let meta = "State is explicit. Events are explicit.";
    let who = "Hello.";
    if name != "" {
        who = "Hello, " + name + ".";
    }
    let stats = "Clicks so far: " + str(clicks);
    let body = ui_element("div", ui_props("", "stack"), ui_children_four(ui_p(intro), ui_p(meta), ui_p(who), ui_p(stats)));
    return ui_section("About", body);
}
`
}
func webPageComponentsTemplate() string {
	return `import "std";

fn page_components(clicks: int, name: string): string {
    let card = ui_card("Example", ui_p("Cards, lists, and inputs are plain Bazic functions."));
    let list = ui_list(ui_children_three(
        ui_list_item("Small"),
        ui_list_item("Fast"),
        ui_list_item("Predictable")
    ));
    let stats = "Clicks so far: " + str(clicks);
    let who = "Name: " + name;
    if name == "" {
        who = "Name: (not set)";
    }
    let body = ui_element("div", ui_props("", "stack"), ui_children_four(card, list, ui_p(stats), ui_p(who)));
    return ui_section("Components", body);
}
`
}
func createUIPage(dir string, name string) error {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	routeName := normalizePageRoute(name)
	if routeName == "" {
		return fmt.Errorf("invalid page name: %s", name)
	}
	appPath := filepath.Join(dir, "app.bz")
	var appData []byte
	if data, err := os.ReadFile(appPath); err == nil {
		appData = data
	}
	pagePath := pageFilePath(dir, routeName)
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(pagePath); err == nil {
		return fmt.Errorf("page already exists: %s", pagePath)
	}
	funcName := pageFuncName(routeName)
	useLayout := layoutExists(dir)
	if useLayout && len(appData) > 0 && !strings.Contains(string(appData), "ui_layout_shell(") {
		useLayout = false
	}
	if err := os.WriteFile(pagePath, []byte(webPageTemplate(routeName, funcName, useLayout)), 0644); err != nil {
		return err
	}
	if len(appData) > 0 {
		updated, err := updateUIAppForPage(string(appData), routeName, funcName)
		if err != nil {
			return err
		}
		if updated != string(appData) {
			if err := os.WriteFile(appPath, []byte(updated), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func createUIComponent(dir string, name string) error {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	if err := os.MkdirAll(filepath.Join(dir, "components"), 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "components", name+".bz")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("component already exists: %s", path)
	}
	return os.WriteFile(path, []byte(webComponentTemplate(name)), 0644)
}

func updateUIAppForPage(src string, routeName string, funcName string) (string, error) {
	importLine := fmt.Sprintf("import \"./%s\";\n", pageImportPath(routeName))
	routeLine := fmt.Sprintf("    if page == \"%s\" {\n        return %s(clicks, name);\n    }\n", routeName, funcName)
	if strings.Contains(src, importLine) {
		return src, nil
	}
	const importsEnd = "// BAZIC_UI_PAGES_IMPORTS_END"
	if strings.Contains(src, importsEnd) {
		src = strings.Replace(src, importsEnd, importLine+importsEnd, 1)
	}
	const routerMarker = "// BAZIC_UI_PAGES_ROUTER"
	if strings.Contains(src, routerMarker) && !strings.Contains(src, routeLine) {
		src = strings.Replace(src, routerMarker, routeLine+routerMarker, 1)
	}
	return src, nil
}

func webPageTemplate(routeName string, funcName string, useLayout bool) string {
	layout := ""
	if useLayout {
		layout = "ui_layout_shell(title, \"\", body)"
	} else {
		layout = "ui_section(title, body)"
	}
	return `import "std";

fn ` + funcName + `(clicks: int, name: string): string {
    let title = "` + strings.Title(routeName) + `";
    let line1 = "This is the ` + routeName + ` page.";
    let line2 = "Clicks so far: " + str(clicks);
    let body = ui_element("div", ui_props("", "stack"), ui_children_two(ui_p(line1), ui_p(line2)));
    return ` + layout + `;
}
`
}
func webComponentTemplate(name string) string {
	return fmt.Sprintf(`import "std";

fn component_%s(title: string, subtitle: string): string {
    let t = ui_element("h3", ui_props("", "section-title"), ui_children_one(ui_text(title)));
    let s = ui_p(subtitle);
    let body = ui_element("div", ui_props("", "stack"), ui_children_two(t, s));
    return ui_element("div", ui_props("", "card"), ui_children_one(body));
}
`, name)
}

func layoutExists(dir string) bool {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	_, err := os.Stat(filepath.Join(dir, "layout.bz"))
	return err == nil
}

func capitalize(name string) string {
	if name == "" {
		return name
	}
	if len(name) == 1 {
		return strings.ToUpper(name)
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func normalizePageRoute(name string) string {
	raw := strings.TrimSpace(name)
	raw = strings.Trim(raw, "/")
	raw = strings.ReplaceAll(raw, "\\", "/")
	for strings.Contains(raw, "//") {
		raw = strings.ReplaceAll(raw, "//", "/")
	}
	return raw
}

func pageFilePath(dir string, routeName string) string {
	segments := strings.Split(routeName, "/")
	parts := append([]string{dir, "pages"}, segments...)
	path := filepath.Join(parts...)
	return path + ".bz"
}

func pageImportPath(routeName string) string {
	return "pages/" + routeName + ".bz"
}

func pageFuncName(routeName string) string {
	return "page_" + sanitizeIdent(routeName)
}

func sanitizeIdent(input string) string {
	out := make([]rune, 0, len(input))
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "page"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = append([]rune{'p', '_'}, out...)
	}
	return string(out)
}

func titleFromRoute(routeName string) string {
	parts := strings.Split(routeName, "/")
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, "-", " ")
	last = strings.ReplaceAll(last, "_", " ")
	if last == "" {
		return "Page"
	}
	return capitalize(last)
}

func listUIRoutes(dir string) ([]string, error) {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	pagesDir := filepath.Join(dir, "pages")
	if _, err := os.Stat(pagesDir); err != nil {
		return nil, fmt.Errorf("pages directory not found: %s", pagesDir)
	}
	var routes []string
	err := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".bz" {
			return nil
		}
		rel, err := filepath.Rel(pagesDir, path)
		if err != nil {
			return err
		}
		route := filepath.ToSlash(strings.TrimSuffix(rel, ".bz"))
		if route == "home" {
			route = "/"
		} else {
			route = "/" + route
		}
		routes = append(routes, route)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return routes, nil
}

func migrateUILayout(dir string) error {
	if dir == "" {
		dir = filepath.Join("examples", "web")
	}
	if !layoutExists(dir) {
		if err := createUILayout(dir); err != nil {
			return err
		}
	}
	pagesDir := filepath.Join(dir, "pages")
	if _, err := os.Stat(pagesDir); err != nil {
		return fmt.Errorf("pages directory not found: %s", pagesDir)
	}
	return filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".bz" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(data)
		if strings.Contains(src, "ui_layout_shell(") {
			return nil
		}
		if !strings.Contains(src, "import \"./layout.bz\";") {
			src = "import \"./layout.bz\";\n" + src
		}
		if !strings.Contains(src, "ui_layout_shell(") && strings.Contains(src, "return ui_section(") {
			src = strings.Replace(src, "return ui_section(", "return ui_layout_shell(\"Page\", nav, ui_section(", 1)
		}
		if !strings.Contains(src, "let nav =") {
			insert := "    let nav = ui_element(\"nav\", ui_props(\"nav\", \"nav\"),\n" +
				"        ui_children_two(\n" +
				"            ui_nav_link(\"Home\", \"nav-home\", page == \"home\"),\n" +
				"            ui_nav_link(\"About\", \"nav-about\", page == \"about\")\n" +
				"        )\n" +
				"    );\n"
			lines := strings.Split(src, "\n")
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "let title") {
					lines = append(lines[:i], append([]string{insert}, lines[i:]...)...)
					break
				}
			}
			src = strings.Join(lines, "\n")
		}
		return os.WriteFile(path, []byte(src), 0644)
	})
}

func ensureWasmExec(dir string) error {
	dst := filepath.Join(dir, "wasm_exec.js")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return fmt.Errorf("failed to locate GOROOT for wasm_exec.js: %w", err)
	}
	goroot := strings.TrimSpace(string(out))
	src := filepath.Join(goroot, "misc", "wasm", "wasm_exec.js")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read wasm_exec.js from %s", src)
	}
	return os.WriteFile(dst, data, 0644)
}

func serveWeb(dir string, port int) int {
	var reloadVersion int64
	go watchWeb(dir, &reloadVersion)

	fs := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.HandleFunc("/__bazic_reload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		fmt.Fprint(w, bazicReloadScript())
	})
	mux.HandleFunc("/__bazic_reload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		ctx := r.Context()
		last := int64(-1)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			current := atomic.LoadInt64(&reloadVersion)
			if current != last {
				fmt.Fprintf(w, "data: %d\n\n", current)
				flusher.Flush()
				last = current
			}
			time.Sleep(300 * time.Millisecond)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		if strings.HasSuffix(path, ".html") {
			full := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(path, "/")))
			data, err := os.ReadFile(full)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			body := string(data)
			if !strings.Contains(body, "/__bazic_reload.js") {
				inject := "    <script src=\"/__bazic_reload.js\"></script>\n"
				if strings.Contains(body, "</body>") {
					body = strings.Replace(body, "</body>", inject+"</body>", 1)
				} else {
					body += "\n" + inject
				}
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(body))
			return
		}
		fs.ServeHTTP(w, r)
	})
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Serving %s at http://localhost%s\n", dir, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		return die(err.Error())
	}
	return 0
}

func bazicReloadScript() string {
	return `(function () {
  if (window.__bazicReload) return;
  window.__bazicReload = true;
  function connect() {
    try {
      var es = new EventSource("/__bazic_reload");
      es.onmessage = function () {
        try {
          es.close();
        } catch (e) {}
        window.location.reload();
      };
      es.onerror = function () {
        try {
          es.close();
        } catch (e) {}
        setTimeout(connect, 500);
      };
    } catch (e) {
      setTimeout(connect, 500);
    }
  }
  connect();
})();`
}

func watchWeb(dir string, reloadVersion *int64) {
	prev, err := scanWebSnapshot(dir)
	if err != nil {
		fmt.Printf("web watch error: %v\n", err)
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		next, err := scanWebSnapshot(dir)
		if err != nil {
			continue
		}
		changed, needsBuild := diffWebSnapshot(prev, next)
		if changed {
			if needsBuild {
				if err := buildWeb(dir); err != nil {
					fmt.Printf("web rebuild failed: %v\n", err)
				} else {
					fmt.Println("Web rebuild complete")
				}
			}
			atomic.AddInt64(reloadVersion, 1)
		}
		prev = next
	}
}

func scanWebSnapshot(dir string) (map[string]time.Time, error) {
	files := make(map[string]time.Time)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".bazic" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".bz" && ext != ".css" && ext != ".js" && ext != ".html" && ext != ".json" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files[path] = info.ModTime()
		return nil
	})
	return files, err
}

func diffWebSnapshot(prev map[string]time.Time, next map[string]time.Time) (bool, bool) {
	changed := false
	needsBuild := false
	for path, nextTime := range next {
		prevTime, ok := prev[path]
		if !ok || !prevTime.Equal(nextTime) {
			changed = true
			if strings.ToLower(filepath.Ext(path)) == ".bz" {
				needsBuild = true
			}
		}
	}
	for path := range prev {
		if _, ok := next[path]; !ok {
			changed = true
			if strings.ToLower(filepath.Ext(path)) == ".bz" {
				needsBuild = true
			}
		}
	}
	return changed, needsBuild
}
