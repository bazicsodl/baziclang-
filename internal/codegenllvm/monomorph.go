package codegenllvm

import (
	"fmt"
	"strings"

	"baziclang/internal/ast"
)

type monoCtx struct {
	prog           *ast.Program
	genericFuncs   map[string]*ast.FuncDecl
	genericStructs map[string]*ast.StructDecl
	structs        map[string]*ast.StructDecl
	funcs          map[string]*ast.FuncDecl
	enums          map[string]map[string]bool
	outDecls       []ast.Decl
	emittedFuncs   map[string]bool
	emittedStructs map[string]bool
}

func monomorphizeProgram(p *ast.Program) *ast.Program {
	m := &monoCtx{
		prog:           p,
		genericFuncs:   map[string]*ast.FuncDecl{},
		genericStructs: map[string]*ast.StructDecl{},
		structs:        map[string]*ast.StructDecl{},
		funcs:          map[string]*ast.FuncDecl{},
		enums:          map[string]map[string]bool{},
		outDecls:       []ast.Decl{},
		emittedFuncs:   map[string]bool{},
		emittedStructs: map[string]bool{},
	}
	m.indexDecls()
	m.emitNonGenericDecls()
	m.processFunctions()
	return &ast.Program{Decls: m.outDecls}
}

func (m *monoCtx) indexDecls() {
	for _, d := range m.prog.Decls {
		switch decl := d.(type) {
		case *ast.StructDecl:
			if len(decl.TypeParams) > 0 {
				m.genericStructs[decl.Name] = decl
			} else {
				m.structs[decl.Name] = decl
			}
		case *ast.FuncDecl:
			if len(decl.TypeParams) > 0 {
				m.genericFuncs[decl.Name] = decl
			} else {
				m.funcs[decl.Name] = decl
			}
		case *ast.EnumDecl:
			vars := map[string]bool{}
			for _, v := range decl.Variants {
				vars[v] = true
			}
			m.enums[decl.Name] = vars
		}
	}
}

func (m *monoCtx) emitNonGenericDecls() {
	for _, d := range m.prog.Decls {
		switch decl := d.(type) {
		case *ast.EnumDecl, *ast.InterfaceDecl, *ast.ImplDecl:
			m.outDecls = append(m.outDecls, d)
		case *ast.StructDecl:
			if len(decl.TypeParams) == 0 {
				m.outDecls = append(m.outDecls, d)
				m.emittedStructs[decl.Name] = true
			}
		case *ast.GlobalLetDecl:
			newDecl, _ := m.rewriteGlobal(decl, nil)
			m.outDecls = append(m.outDecls, newDecl)
		case *ast.FuncDecl:
			if len(decl.TypeParams) == 0 {
				newFn := m.rewriteFunc(decl, nil)
				m.outDecls = append(m.outDecls, newFn)
				m.emittedFuncs[newFn.Name] = true
			}
		}
	}
}

func (m *monoCtx) processFunctions() {
	// Iterate until no new specialized funcs are added.
	changed := true
	for changed {
		changed = false
		for name, fn := range m.funcs {
			if m.emittedFuncs[name] {
				continue
			}
			newFn := m.rewriteFunc(fn, nil)
			m.outDecls = append(m.outDecls, newFn)
			m.emittedFuncs[name] = true
			changed = true
		}
		for name, fn := range m.genericFuncs {
			_ = name
			_ = fn
		}
	}
}

func (m *monoCtx) rewriteGlobal(g *ast.GlobalLetDecl, mapping map[string]ast.Type) (*ast.GlobalLetDecl, ast.Type) {
	expr, typ := m.rewriteExpr(g.Init, mapping, map[string]ast.Type{})
	rawType := g.Type
	if rawType == ast.TypeInvalid {
		rawType = typ
	} else {
		rawType = substituteType(rawType, mapping)
	}
	normType := m.normalizeType(rawType)
	return &ast.GlobalLetDecl{
		Name: g.Name,
		Type: normType,
		Init: expr,
		IsConst: g.IsConst,
	}, normType
}

func (m *monoCtx) rewriteFunc(fn *ast.FuncDecl, mapping map[string]ast.Type) *ast.FuncDecl {
	env := map[string]ast.Type{}
	params := make([]ast.Param, 0, len(fn.Params))
	for _, p := range fn.Params {
		raw := substituteType(p.Type, mapping)
		norm := m.normalizeType(raw)
		env[p.Name] = raw
		params = append(params, ast.Param{Name: p.Name, Type: norm})
	}
	rawRet := substituteType(fn.ReturnType, mapping)
	normRet := m.normalizeType(rawRet)
	body := m.rewriteBlock(fn.Body, mapping, env)
	return &ast.FuncDecl{
		Name:       fn.Name,
		TypeParams: nil,
		Params:     params,
		ReturnType: normRet,
		Body:       body,
	}
}

func (m *monoCtx) rewriteBlock(b *ast.BlockStmt, mapping map[string]ast.Type, env map[string]ast.Type) *ast.BlockStmt {
	if b == nil {
		return nil
	}
	local := cloneEnv(env)
	out := &ast.BlockStmt{Stmts: []ast.Stmt{}}
	for _, st := range b.Stmts {
		nst := m.rewriteStmt(st, mapping, local)
		if nst != nil {
			out.Stmts = append(out.Stmts, nst)
		}
	}
	return out
}

func (m *monoCtx) rewriteStmt(s ast.Stmt, mapping map[string]ast.Type, env map[string]ast.Type) ast.Stmt {
	switch st := s.(type) {
	case *ast.LetStmt:
		expr, typ := m.rewriteExpr(st.Init, mapping, env)
		raw := st.Type
		if raw == ast.TypeInvalid {
			raw = typ
		} else {
			raw = substituteType(raw, mapping)
		}
		norm := m.normalizeType(raw)
		env[st.Name] = raw
		return &ast.LetStmt{Name: st.Name, Type: norm, Init: expr, IsConst: st.IsConst}
	case *ast.AssignStmt:
		target, _ := m.rewriteExpr(st.Target, mapping, env)
		expr, _ := m.rewriteExpr(st.Value, mapping, env)
		return &ast.AssignStmt{Target: target, Value: expr}
	case *ast.ExprStmt:
		expr, _ := m.rewriteExpr(st.Expr, mapping, env)
		return &ast.ExprStmt{Expr: expr}
	case *ast.ReturnStmt:
		if st.Value == nil {
			return &ast.ReturnStmt{Value: nil}
		}
		expr, _ := m.rewriteExpr(st.Value, mapping, env)
		return &ast.ReturnStmt{Value: expr}
	case *ast.IfStmt:
		cond, _ := m.rewriteExpr(st.Cond, mapping, env)
		thenBlk := m.rewriteBlock(st.Then, mapping, env)
		var elseBlk *ast.BlockStmt
		if st.Else != nil {
			elseBlk = m.rewriteBlock(st.Else, mapping, env)
		}
		return &ast.IfStmt{Cond: cond, Then: thenBlk, Else: elseBlk}
	case *ast.WhileStmt:
		cond, _ := m.rewriteExpr(st.Cond, mapping, env)
		body := m.rewriteBlock(st.Body, mapping, env)
		return &ast.WhileStmt{Cond: cond, Body: body}
	case *ast.MatchStmt:
		subj, _ := m.rewriteExpr(st.Subject, mapping, env)
		arms := make([]ast.MatchArm, 0, len(st.Arms))
		for _, arm := range st.Arms {
			var guard ast.Expr
			if arm.Guard != nil {
				guard, _ = m.rewriteExpr(arm.Guard, mapping, env)
			}
			body := m.rewriteBlock(arm.Body, mapping, env)
			arms = append(arms, ast.MatchArm{Variant: arm.Variant, Guard: guard, Body: body})
		}
		return &ast.MatchStmt{Subject: subj, Arms: arms}
	default:
		return nil
	}
}

func (m *monoCtx) rewriteExpr(e ast.Expr, mapping map[string]ast.Type, env map[string]ast.Type) (ast.Expr, ast.Type) {
	switch ex := e.(type) {
	case *ast.IntExpr:
		return ex, ast.TypeInt
	case *ast.FloatExpr:
		return ex, ast.TypeFloat
	case *ast.BoolExpr:
		return ex, ast.TypeBool
	case *ast.StringExpr:
		return ex, ast.TypeString
	case *ast.IdentExpr:
		if t, ok := env[ex.Name]; ok {
			return ex, t
		}
		for enumName, vars := range m.enums {
			if vars[ex.Name] {
				return ex, ast.Type(enumName)
			}
		}
		return ex, ast.TypeInvalid
	case *ast.UnaryExpr:
		right, rt := m.rewriteExpr(ex.Right, mapping, env)
		return &ast.UnaryExpr{Op: ex.Op, Right: right}, rt
	case *ast.BinaryExpr:
		left, lt := m.rewriteExpr(ex.Left, mapping, env)
		right, rt := m.rewriteExpr(ex.Right, mapping, env)
		_ = rt
		return &ast.BinaryExpr{Left: left, Op: ex.Op, Right: right}, lt
	case *ast.StructLitExpr:
		rawType := substituteType(ast.Type(ex.TypeName), mapping)
		normType := m.normalizeType(rawType)
		fields := make([]ast.StructLitField, 0, len(ex.Fields))
		for _, f := range ex.Fields {
			val, _ := m.rewriteExpr(f.Value, mapping, env)
			fields = append(fields, ast.StructLitField{Name: f.Name, Value: val})
		}
		return &ast.StructLitExpr{TypeName: string(normType), Fields: fields}, rawType
	case *ast.FieldAccessExpr:
		obj, objType := m.rewriteExpr(ex.Object, mapping, env)
		fieldType := m.structFieldType(objType, ex.Field)
		return &ast.FieldAccessExpr{Object: obj, Field: ex.Field}, fieldType
	case *ast.CallExpr:
		args := make([]ast.Expr, 0, len(ex.Args))
		argTypes := make([]ast.Type, 0, len(ex.Args))
		for _, a := range ex.Args {
			ra, rt := m.rewriteExpr(a, mapping, env)
			args = append(args, ra)
			argTypes = append(argTypes, rt)
		}
		name := ex.Callee
		if gfn, ok := m.genericFuncs[name]; ok {
			mapping := m.inferTypeArgs(gfn, argTypes)
			specName := m.specializeFunc(gfn, mapping)
			retRaw := substituteType(gfn.ReturnType, mapping)
			return &ast.CallExpr{Callee: specName, Args: args}, retRaw
		}
		if sig, ok := m.funcs[name]; ok {
			return &ast.CallExpr{Callee: name, Args: args}, sig.ReturnType
		}
		return &ast.CallExpr{Callee: name, Args: args}, ast.TypeInvalid
	case *ast.MatchExpr:
		subj, _ := m.rewriteExpr(ex.Subject, mapping, env)
		arms := make([]ast.MatchExprArm, 0, len(ex.Arms))
		for _, arm := range ex.Arms {
			var guard ast.Expr
			if arm.Guard != nil {
				guard, _ = m.rewriteExpr(arm.Guard, mapping, env)
			}
			val, _ := m.rewriteExpr(arm.Value, mapping, env)
			arms = append(arms, ast.MatchExprArm{Variant: arm.Variant, Guard: guard, Value: val})
		}
		raw := substituteType(ex.ResolvedType, mapping)
		norm := m.normalizeType(raw)
		return &ast.MatchExpr{Subject: subj, Arms: arms, ResolvedType: norm}, raw
	default:
		return ex, ast.TypeInvalid
	}
}

func (m *monoCtx) structFieldType(rawStruct ast.Type, field string) ast.Type {
	if rawStruct == ast.TypeInvalid {
		return ast.TypeInvalid
	}
	base, args, ok := splitGenericType(string(rawStruct))
	if ok {
		generic := m.genericStructs[base]
		if generic == nil {
			return ast.TypeInvalid
		}
		bind := map[string]ast.Type{}
		for i, tp := range generic.TypeParams {
			bind[tp] = ast.Type(args[i])
		}
		for _, f := range generic.Fields {
			if f.Name == field {
				return substituteType(f.Type, bind)
			}
		}
		return ast.TypeInvalid
	}
	if s, ok := m.structs[string(rawStruct)]; ok {
		for _, f := range s.Fields {
			if f.Name == field {
				return f.Type
			}
		}
	}
	return ast.TypeInvalid
}

func (m *monoCtx) inferTypeArgs(fn *ast.FuncDecl, argTypes []ast.Type) map[string]ast.Type {
	bind := map[string]ast.Type{}
	for i, p := range fn.Params {
		if i >= len(argTypes) {
			break
		}
		unifyType(p.Type, argTypes[i], bind)
	}
	return bind
}

func (m *monoCtx) specializeFunc(fn *ast.FuncDecl, mapping map[string]ast.Type) string {
	keyParts := []string{fn.Name}
	for _, tp := range fn.TypeParams {
		keyParts = append(keyParts, encodeTypeForName(mapping[tp]))
	}
	specName := strings.Join(keyParts, "__")
	if m.emittedFuncs[specName] {
		return specName
	}
	clone := m.rewriteFuncWithName(fn, mapping, specName)
	m.outDecls = append(m.outDecls, clone)
	m.emittedFuncs[specName] = true
	m.funcs[specName] = clone
	return specName
}

func (m *monoCtx) rewriteFuncWithName(fn *ast.FuncDecl, mapping map[string]ast.Type, name string) *ast.FuncDecl {
	env := map[string]ast.Type{}
	params := make([]ast.Param, 0, len(fn.Params))
	for _, p := range fn.Params {
		raw := substituteType(p.Type, mapping)
		norm := m.normalizeType(raw)
		env[p.Name] = raw
		params = append(params, ast.Param{Name: p.Name, Type: norm})
	}
	rawRet := substituteType(fn.ReturnType, mapping)
	normRet := m.normalizeType(rawRet)
	body := m.rewriteBlock(fn.Body, mapping, env)
	return &ast.FuncDecl{
		Name:       name,
		TypeParams: nil,
		Params:     params,
		ReturnType: normRet,
		Body:       body,
	}
}

func (m *monoCtx) normalizeType(t ast.Type) ast.Type {
	if t == ast.TypeInvalid || t == ast.TypeVoid || t == ast.TypeInt || t == ast.TypeFloat || t == ast.TypeBool || t == ast.TypeString || t == ast.TypeAny {
		return t
	}
	if _, ok := m.enums[string(t)]; ok {
		return t
	}
	base, args, ok := splitGenericType(string(t))
	if ok {
		for i := range args {
			args[i] = string(m.normalizeType(ast.Type(args[i])))
		}
		return ast.Type(m.ensureStruct(base, args))
	}
	return t
}

func (m *monoCtx) ensureStruct(base string, args []string) string {
	key := fmt.Sprintf("%s[%s]", base, strings.Join(args, ","))
	name := fmt.Sprintf("%s__%s", base, strings.Join(encodedArgs(args), "__"))
	if m.emittedStructs[name] {
		return name
	}
	gen := m.genericStructs[base]
	if gen == nil {
		return base
	}
	bind := map[string]ast.Type{}
	for i, tp := range gen.TypeParams {
		bind[tp] = ast.Type(args[i])
	}
	fields := make([]ast.StructField, 0, len(gen.Fields))
	for _, f := range gen.Fields {
		raw := substituteType(f.Type, bind)
		norm := m.normalizeType(raw)
		fields = append(fields, ast.StructField{Name: f.Name, Type: norm})
	}
	_ = key
	decl := &ast.StructDecl{Name: name, TypeParams: nil, Fields: fields}
	m.outDecls = append(m.outDecls, decl)
	m.emittedStructs[name] = true
	m.structs[name] = decl
	return name
}

func encodedArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, encodeTypeForName(ast.Type(a)))
	}
	return out
}

func encodeTypeForName(t ast.Type) string {
	if t == "" {
		return "unknown"
	}
	base, args, ok := splitGenericType(string(t))
	if ok {
		parts := []string{base}
		for _, a := range args {
			parts = append(parts, encodeTypeForName(ast.Type(a)))
		}
		return strings.Join(parts, "__")
	}
	return sanitizeName(string(t))
}

func sanitizeName(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteRune('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "type"
	}
	return out
}

func substituteType(t ast.Type, mapping map[string]ast.Type) ast.Type {
	if mapping == nil {
		return t
	}
	if v, ok := mapping[string(t)]; ok {
		return v
	}
	base, args, ok := splitGenericType(string(t))
	if !ok {
		return t
	}
	outArgs := make([]string, 0, len(args))
	for _, a := range args {
		outArgs = append(outArgs, string(substituteType(ast.Type(a), mapping)))
	}
	return ast.Type(fmt.Sprintf("%s[%s]", base, strings.Join(outArgs, ",")))
}

func splitGenericType(t string) (string, []string, bool) {
	open := strings.IndexRune(t, '[')
	close := strings.LastIndex(t, "]")
	if open <= 0 || close <= open {
		return "", nil, false
	}
	base := t[:open]
	inner := t[open+1 : close]
	parts := splitTopLevel(inner)
	if len(parts) == 0 {
		return "", nil, false
	}
	return base, parts, true
}

func splitTopLevel(s string) []string {
	depth := 0
	parts := []string{}
	start := 0
	for i, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func unifyType(expected ast.Type, actual ast.Type, mapping map[string]ast.Type) {
	if mapping == nil {
		return
	}
	if _, ok := mapping[string(expected)]; ok {
		return
	}
	base, args, ok := splitGenericType(string(expected))
	if ok {
		abase, aargs, aok := splitGenericType(string(actual))
		if !aok || abase != base || len(args) != len(aargs) {
			return
		}
		for i := range args {
			unifyType(ast.Type(args[i]), ast.Type(aargs[i]), mapping)
		}
		return
	}
	mapping[string(expected)] = actual
}

func cloneEnv(env map[string]ast.Type) map[string]ast.Type {
	out := map[string]ast.Type{}
	for k, v := range env {
		out[k] = v
	}
	return out
}
