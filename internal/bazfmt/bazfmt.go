package bazfmt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
)

func Format(src string) (string, error) {
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		return "", err
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, d := range prog.Decls {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if err := writeDecl(&b, d, 0); err != nil {
			return "", err
		}
	}
	out := b.String()
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

func FormatFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	formatted, err := Format(string(data))
	if err != nil {
		return false, err
	}
	if string(data) == formatted {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func CollectBZFiles(target string) ([]string, error) {
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
			return nil, fmt.Errorf("fmt target must be a .bz file or directory")
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
			if name == ".git" || name == ".bazic" {
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

func writeDecl(b *strings.Builder, d ast.Decl, indent int) error {
	switch decl := d.(type) {
	case *ast.ImportDecl:
		indentWrite(b, indent, "import "+quote(decl.Path))
	case *ast.StructDecl:
		if len(decl.TypeParams) > 0 {
			indentWrite(b, indent, fmt.Sprintf("struct %s[%s] {", decl.Name, strings.Join(decl.TypeParams, ", ")))
		} else {
			indentWrite(b, indent, fmt.Sprintf("struct %s {", decl.Name))
		}
		for _, f := range decl.Fields {
			indentWrite(b, indent+1, fmt.Sprintf("%s: %s", f.Name, f.Type))
		}
		indentWrite(b, indent, "}")
	case *ast.InterfaceDecl:
		indentWrite(b, indent, fmt.Sprintf("interface %s {", decl.Name))
		for _, m := range decl.Methods {
			params := formatParams(m.Params)
			if m.Return == ast.TypeVoid {
				indentWrite(b, indent+1, fmt.Sprintf("fn %s(%s)", m.Name, params))
			} else {
				indentWrite(b, indent+1, fmt.Sprintf("fn %s(%s): %s", m.Name, params, m.Return))
			}
		}
		indentWrite(b, indent, "}")
	case *ast.ImplDecl:
		indentWrite(b, indent, fmt.Sprintf("impl %s: %s", decl.StructType, decl.InterfaceName))
	case *ast.EnumDecl:
		indentWrite(b, indent, fmt.Sprintf("enum %s { %s }", decl.Name, strings.Join(decl.Variants, ", ")))
	case *ast.FuncDecl:
		head := "fn " + decl.Name
		if len(decl.TypeParams) > 0 {
			head += "[" + strings.Join(decl.TypeParams, ", ") + "]"
		}
		head += "(" + formatParams(decl.Params) + ")"
		head += ": " + string(decl.ReturnType)
		indentWrite(b, indent, head+" {")
		for _, st := range decl.Body.Stmts {
			if err := writeStmt(b, st, indent+1); err != nil {
				return err
			}
		}
		indentWrite(b, indent, "}")
	case *ast.GlobalLetDecl:
		rhs, err := formatExpr(decl.Init)
		if err != nil {
			return err
		}
		kw := "let"
		if decl.IsConst {
			kw = "const"
		}
		if decl.Type == ast.TypeInvalid {
			indentWrite(b, indent, fmt.Sprintf("%s %s = %s", kw, decl.Name, rhs))
		} else {
			indentWrite(b, indent, fmt.Sprintf("%s %s: %s = %s", kw, decl.Name, decl.Type, rhs))
		}
	default:
		return fmt.Errorf("formatter: unsupported declaration")
	}
	return nil
}

func writeStmt(b *strings.Builder, s ast.Stmt, indent int) error {
	switch st := s.(type) {
	case *ast.LetStmt:
		rhs, err := formatExpr(st.Init)
		if err != nil {
			return err
		}
		kw := "let"
		if st.IsConst {
			kw = "const"
		}
		if st.Type == ast.TypeInvalid {
			indentWrite(b, indent, fmt.Sprintf("%s %s = %s", kw, st.Name, rhs))
		} else {
			indentWrite(b, indent, fmt.Sprintf("%s %s: %s = %s", kw, st.Name, st.Type, rhs))
		}
	case *ast.AssignStmt:
		rhs, err := formatExpr(st.Value)
		if err != nil {
			return err
		}
		target, err := formatAssignTarget(st.Target)
		if err != nil {
			return err
		}
		indentWrite(b, indent, fmt.Sprintf("%s = %s", target, rhs))
	case *ast.ExprStmt:
		e, err := formatExpr(st.Expr)
		if err != nil {
			return err
		}
		indentWrite(b, indent, e)
	case *ast.ReturnStmt:
		if st.Value == nil {
			indentWrite(b, indent, "return")
		} else {
			e, err := formatExpr(st.Value)
			if err != nil {
				return err
			}
			indentWrite(b, indent, "return "+e)
		}
	case *ast.IfStmt:
		cond, err := formatExpr(st.Cond)
		if err != nil {
			return err
		}
		indentWrite(b, indent, "if "+cond+" {")
		for _, x := range st.Then.Stmts {
			if err := writeStmt(b, x, indent+1); err != nil {
				return err
			}
		}
		if st.Else == nil {
			indentWrite(b, indent, "}")
			return nil
		}
		indentWrite(b, indent, "} else {")
		for _, x := range st.Else.Stmts {
			if err := writeStmt(b, x, indent+1); err != nil {
				return err
			}
		}
		indentWrite(b, indent, "}")
	case *ast.WhileStmt:
		cond, err := formatExpr(st.Cond)
		if err != nil {
			return err
		}
		indentWrite(b, indent, "while "+cond+" {")
		for _, x := range st.Body.Stmts {
			if err := writeStmt(b, x, indent+1); err != nil {
				return err
			}
		}
		indentWrite(b, indent, "}")
	case *ast.MatchStmt:
		subject, err := formatExpr(st.Subject)
		if err != nil {
			return err
		}
		indentWrite(b, indent, "match "+subject+" {")
		for _, arm := range st.Arms {
			head := arm.Variant
			if arm.Guard != nil {
				g, err := formatExpr(arm.Guard)
				if err != nil {
					return err
				}
				head += " if " + g
			}
			indentWrite(b, indent+1, head+": {")
			for _, x := range arm.Body.Stmts {
				if err := writeStmt(b, x, indent+2); err != nil {
					return err
				}
			}
			indentWrite(b, indent+1, "}")
		}
		indentWrite(b, indent, "}")
	default:
		return fmt.Errorf("formatter: unsupported statement")
	}
	return nil
}

func formatExpr(e ast.Expr) (string, error) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return ex.Name, nil
	case *ast.IntExpr:
		return fmt.Sprintf("%d", ex.Value), nil
	case *ast.FloatExpr:
		return fmt.Sprintf("%g", ex.Value), nil
	case *ast.BoolExpr:
		if ex.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.StringExpr:
		return quote(ex.Value), nil
	case *ast.NilExpr:
		return "nil", nil
	case *ast.UnaryExpr:
		r, err := formatExpr(ex.Right)
		if err != nil {
			return "", err
		}
		return ex.Op + r, nil
	case *ast.BinaryExpr:
		l, err := formatExpr(ex.Left)
		if err != nil {
			return "", err
		}
		r, err := formatExpr(ex.Right)
		if err != nil {
			return "", err
		}
		return "(" + l + " " + ex.Op + " " + r + ")", nil
	case *ast.CallExpr:
		args := make([]string, 0, len(ex.Args))
		for _, a := range ex.Args {
			s, err := formatExpr(a)
			if err != nil {
				return "", err
			}
			args = append(args, s)
		}
		if ex.Callee != "" {
			return ex.Callee + "(" + strings.Join(args, ", ") + ")", nil
		}
		if ex.Receiver != nil {
			recv, err := formatExpr(ex.Receiver)
			if err != nil {
				return "", err
			}
			return recv + "." + ex.Method + "(" + strings.Join(args, ", ") + ")", nil
		}
		return "", fmt.Errorf("formatter: unresolved call expression")
	case *ast.FieldAccessExpr:
		base, err := formatExpr(ex.Object)
		if err != nil {
			return "", err
		}
		return base + "." + ex.Field, nil
	case *ast.StructLitExpr:
		fields := append([]ast.StructLitField{}, ex.Fields...)
		sort.SliceStable(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
		parts := make([]string, 0, len(fields))
		for _, f := range fields {
			v, err := formatExpr(f.Value)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%s: %s", f.Name, v))
		}
		return fmt.Sprintf("%s { %s }", ex.TypeName, strings.Join(parts, ", ")), nil
	case *ast.MatchExpr:
		subject, err := formatExpr(ex.Subject)
		if err != nil {
			return "", err
		}
		parts := make([]string, 0, len(ex.Arms))
		for _, arm := range ex.Arms {
			head := arm.Variant
			if arm.Guard != nil {
				g, err := formatExpr(arm.Guard)
				if err != nil {
					return "", err
				}
				head += " if " + g
			}
			v, err := formatExpr(arm.Value)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%s: %s", head, v))
		}
		return fmt.Sprintf("match %s { %s }", subject, strings.Join(parts, ", ")), nil
	default:
		return "", fmt.Errorf("formatter: unsupported expression")
	}
}

func formatAssignTarget(e ast.Expr) (string, error) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return ex.Name, nil
	case *ast.FieldAccessExpr:
		return formatExpr(ex)
	default:
		return "", fmt.Errorf("formatter: invalid assignment target")
	}
}

func formatParams(params []ast.Param) string {
	out := make([]string, 0, len(params))
	for _, p := range params {
		out = append(out, fmt.Sprintf("%s: %s", p.Name, p.Type))
	}
	return strings.Join(out, ", ")
}

func indentWrite(b *strings.Builder, indent int, s string) {
	for i := 0; i < indent; i++ {
		b.WriteString("    ")
	}
	b.WriteString(s)
	b.WriteString("\n")
}

func quote(s string) string {
	var b strings.Builder
	b.WriteString(`"`)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString(`"`)
	return b.String()
}
