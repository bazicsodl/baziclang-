package bazlint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Issue struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type Options struct {
	Filter string
}

func Lint(target string) ([]Issue, error) {
	files, err := collectBZFiles(target)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .bz files found in %s", target)
	}
	out := []Issue{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		src := string(data)
		out = append(out, lintFile(file, src)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Column < out[j].Column
	})
	return out, nil
}

func lintFile(path, src string) []Issue {
	issues := []Issue{}
	lints := []struct {
		Rule    string
		Pattern *regexp.Regexp
		Message string
		Skip    func(string, int) bool
	}{
		{
			Rule:    "BL001",
			Pattern: regexp.MustCompile(`\bprint\s*\(`),
			Message: "avoid print; use println or structured output",
		},
		{
			Rule:    "BL002",
			Pattern: regexp.MustCompile(`\b__std_[A-Za-z0-9_]+\b`),
			Message: "do not call internal __std_* functions directly; use std wrappers",
			Skip: func(p string, _ int) bool {
				return isStdPath(p)
			},
		},
		{
			Rule:    "BL003",
			Pattern: regexp.MustCompile(`\bfn\s+([A-Za-z_][A-Za-z0-9_]*)`),
			Message: "function names should be snake_case (prefer lowercase with underscores)",
			Skip: func(_ string, idx int) bool {
				_ = idx
				return false
			},
		},
		{
			Rule:    "BL004",
			Pattern: regexp.MustCompile(`\bif\s+(true|false)\b\s*\{`),
			Message: "constant if condition; consider removing dead branch",
			Skip: func(p string, _ int) bool {
				return isConformancePath(p)
			},
		},
		{
			Rule:    "BL005",
			Pattern: regexp.MustCompile(`\bwhile\s+false\b`),
			Message: "while false never executes",
			Skip: func(p string, _ int) bool {
				return isConformancePath(p)
			},
		},
		{
			Rule:    "BL006",
			Pattern: regexp.MustCompile(`:\s*any\b`),
			Message: "avoid 'any' in user code; prefer concrete types or generics",
			Skip: func(p string, _ int) bool {
				return isStdPath(p) || isConformancePath(p)
			},
		},
	}
	for _, lint := range lints {
		matches := lint.Pattern.FindAllStringSubmatchIndex(src, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			idx := m[0]
			if lint.Rule == "BL003" && len(m) >= 4 {
				name := src[m[2]:m[3]]
				if name != strings.ToLower(name) && !strings.Contains(name, "_") {
					line, col := lineColAt(src, m[2])
					issues = append(issues, Issue{
						File:    path,
						Line:    line,
						Column:  col,
						Rule:    lint.Rule,
						Message: lint.Message,
					})
				}
				continue
			}
			if lint.Skip != nil && lint.Skip(path, idx) {
				continue
			}
			line, col := lineColAt(src, idx)
			issues = append(issues, Issue{
				File:    path,
				Line:    line,
				Column:  col,
				Rule:    lint.Rule,
				Message: lint.Message,
			})
		}
	}
	return issues
}

func lineColAt(src string, idx int) (int, int) {
	line, col := 1, 1
	for i, r := range src {
		if i >= idx {
			break
		}
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func isStdPath(path string) bool {
	clean := filepath.Clean(path)
	sep := string(filepath.Separator)
	if strings.Contains(clean, sep+"std"+sep) {
		return true
	}
	if strings.Contains(clean, sep+".bazic"+sep+"pkg"+sep+"std"+sep) {
		return true
	}
	return false
}

func isConformancePath(path string) bool {
	clean := filepath.Clean(path)
	sep := string(filepath.Separator)
	return strings.Contains(clean, sep+"conformance"+sep)
}

func collectBZFiles(target string) ([]string, error) {
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
			return nil, fmt.Errorf("lint target must be a .bz file or directory")
		}
		return []string{abs}, nil
	}
	out := []string{}
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".bazic" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".bazic_") {
			return nil
		}
		if filepath.Ext(d.Name()) == ".bz" {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
