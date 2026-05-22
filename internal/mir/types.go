package mir

import (
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	baztypes "baziclang/internal/types"
)

type typeIndex struct {
	funcs        map[string]baztypes.Type
	funcDecls    map[string]*FuncDecl
	globals      map[string]baztypes.Type
	structs      map[string]*StructDecl
	enumVariants map[string]baztypes.Type
}

type typeContext struct {
	index  *typeIndex
	locals map[string]baztypes.Type
}

func newTypeIndex(p *Program) *typeIndex {
	idx := &typeIndex{
		funcs:        map[string]baztypes.Type{},
		funcDecls:    map[string]*FuncDecl{},
		globals:      map[string]baztypes.Type{},
		structs:      map[string]*StructDecl{},
		enumVariants: map[string]baztypes.Type{},
	}
	for _, spec := range intrinsics.FunctionSpecs(nil) {
		idx.funcs[spec.Name] = baztypes.MustParse(spec.Ret)
	}
	if p == nil {
		return idx
	}
	for _, decl := range p.Decls {
		switch d := decl.(type) {
		case *FuncDecl:
			idx.funcs[d.Name] = d.ReturnType
			idx.funcDecls[d.Name] = d
		case *GlobalLetDecl:
			idx.globals[d.Name] = d.Type
		case *StructDecl:
			idx.structs[d.Name] = d
		case *EnumDecl:
			enumType := baztypes.MustParse(ast.Type(d.Name))
			for _, variant := range d.Variants {
				idx.enumVariants[variant] = enumType
			}
		}
	}
	return idx
}

func newTypeContext(index *typeIndex) *typeContext {
	if index == nil {
		index = newTypeIndex(nil)
	}
	return &typeContext{
		index:  index,
		locals: map[string]baztypes.Type{},
	}
}

func newFuncTypeContext(index *typeIndex, fn *FuncDecl) *typeContext {
	ctx := newTypeContext(index)
	if fn == nil {
		return ctx
	}
	for _, p := range fn.Params {
		ctx.locals[p.Name] = p.Type
	}
	collectBlockLocals(ctx.locals, fn.Body)
	if fn.CFG != nil {
		for _, block := range fn.CFG.Blocks {
			if block == nil {
				continue
			}
			for _, instr := range block.Instrs {
				if name, typ, ok := NamedValueStmtBinding(instr); ok {
					ctx.locals[name] = typ
				}
			}
		}
	}
	return ctx
}

func collectBlockLocals(out map[string]baztypes.Type, b *Block) {
	if b == nil {
		return
	}
	for _, stmt := range b.Stmts {
		if name, typ, ok := NamedValueStmtBinding(stmt); ok {
			out[name] = typ
		}
		WalkStmtChildBlocks(stmt, func(child *Block) {
			collectBlockLocals(out, child)
		})
	}
}

func (ctx *typeContext) inferExprType(e Expr) (baztypes.Type, bool) {
	if ctx == nil || e == nil {
		return baztypes.Type{}, false
	}
	switch ex := e.(type) {
	case *IdentExpr:
		if t, ok := ctx.locals[ex.Name]; ok {
			return t, true
		}
		if t, ok := ctx.index.globals[ex.Name]; ok {
			return t, true
		}
		if t, ok := ctx.index.enumVariants[ex.Name]; ok {
			return t, true
		}
		return baztypes.Type{}, false
	case *IntExpr:
		return baztypes.MustParse(ast.TypeInt), true
	case *FloatExpr:
		return baztypes.MustParse(ast.TypeFloat), true
	case *BoolExpr:
		return baztypes.MustParse(ast.TypeBool), true
	case *StringExpr:
		return baztypes.MustParse(ast.TypeString), true
	case *NilExpr:
		return baztypes.Type{}, false
	case *UnaryExpr:
		right, ok := ctx.inferExprType(ex.Right)
		if !ok {
			return baztypes.Type{}, false
		}
		switch ex.Op {
		case "!":
			return baztypes.MustParse(ast.TypeBool), true
		case "-":
			if right.String() == string(ast.TypeInt) || right.String() == string(ast.TypeFloat) {
				return right, true
			}
		}
		return baztypes.Type{}, false
	case *BinaryExpr:
		left, okLeft := ctx.inferExprType(ex.Left)
		right, okRight := ctx.inferExprType(ex.Right)
		switch ex.Op {
		case "&&", "||", "==", "!=", "<", "<=", ">", ">=":
			if okLeft || okRight {
				return baztypes.MustParse(ast.TypeBool), true
			}
		case "+", "-", "*", "/", "%":
			if okLeft && okRight && typesCompatible(left, right) {
				return left, true
			}
		}
		return baztypes.Type{}, false
	case *CallExpr:
		if fn, ok := ctx.index.funcDecls[ex.Func]; ok && len(fn.TypeParams) > 0 {
			return baztypes.Type{}, false
		}
		if t, ok := ctx.index.funcs[ex.Func]; ok {
			return t, true
		}
		return baztypes.Type{}, false
	case *FieldAccessExpr:
		objectType, ok := ctx.inferExprType(ex.Object)
		if !ok {
			return baztypes.Type{}, false
		}
		return ctx.structFieldType(objectType, ex.Field)
	case *StructLitExpr:
		return baztypes.MustParse(ast.Type(ex.TypeName)), true
	case *MatchExpr:
		if ex.Type.Kind == baztypes.KindInvalid && ex.Type.Name == "" && len(ex.Type.Args) == 0 {
			return baztypes.Type{}, false
		}
		return ex.Type, true
	default:
		return baztypes.Type{}, false
	}
}

func (ctx *typeContext) structFieldType(structType baztypes.Type, field string) (baztypes.Type, bool) {
	if ctx == nil || field == "" {
		return baztypes.Type{}, false
	}
	decl, mapping, ok := ctx.resolveStructDecl(structType)
	if !ok {
		return baztypes.Type{}, false
	}
	for _, f := range decl.Fields {
		if f.Name != field {
			continue
		}
		return substituteMIRTypeParams(f.Type, mapping), true
	}
	return baztypes.Type{}, false
}

func (ctx *typeContext) resolveStructDecl(structType baztypes.Type) (*StructDecl, map[string]baztypes.Type, bool) {
	if ctx == nil || structType.Kind != baztypes.KindNamed || structType.Name == "" {
		return nil, nil, false
	}
	decl, ok := ctx.index.structs[structType.Name]
	if ok {
		return decl, bindStructTypeArgs(decl, structType), true
	}
	base, ok := baztypes.Base(baztypes.ToAST(structType))
	if !ok {
		return nil, nil, false
	}
	decl, ok = ctx.index.structs[base]
	if !ok {
		return nil, nil, false
	}
	return decl, bindStructTypeArgs(decl, structType), true
}

func bindStructTypeArgs(decl *StructDecl, actual baztypes.Type) map[string]baztypes.Type {
	out := map[string]baztypes.Type{}
	if decl == nil || len(decl.TypeParams) == 0 || len(actual.Args) == 0 {
		return out
	}
	for i, tp := range decl.TypeParams {
		if i >= len(actual.Args) {
			break
		}
		out[tp] = actual.Args[i]
	}
	return out
}

func substituteMIRTypeParams(t baztypes.Type, mapping map[string]baztypes.Type) baztypes.Type {
	if len(mapping) == 0 {
		return t
	}
	if repl, ok := mapping[t.Name]; ok && len(t.Args) == 0 {
		return repl
	}
	if len(t.Args) == 0 {
		return t
	}
	out := baztypes.Type{Kind: t.Kind, Name: t.Name, Args: make([]baztypes.Type, 0, len(t.Args))}
	for _, arg := range t.Args {
		out.Args = append(out.Args, substituteMIRTypeParams(arg, mapping))
	}
	return out
}

func typesCompatible(expected, actual baztypes.Type) bool {
	if expected.Kind == baztypes.KindInvalid || expected.Name == "" {
		return true
	}
	if actual.Kind == baztypes.KindInvalid || actual.Name == "" {
		return true
	}
	if expected.String() == string(ast.TypeAny) || actual.String() == string(ast.TypeAny) {
		return true
	}
	if expected.String() == actual.String() {
		return true
	}
	return canonicalTypeForCompat(expected).String() == canonicalTypeForCompat(actual).String()
}

func canonicalTypeForCompat(t baztypes.Type) baztypes.Type {
	if decoded, ok := decodeLLVMResultStructType(t.Name); ok {
		return decoded
	}
	if t.Name == "" {
		return t
	}
	out := baztypes.Type{Kind: t.Kind, Name: canonicalTypeName(t.Name)}
	if len(t.Args) == 0 {
		return out
	}
	out.Args = make([]baztypes.Type, 0, len(t.Args))
	for _, arg := range t.Args {
		out.Args = append(out.Args, canonicalTypeForCompat(arg))
	}
	return out
}

func decodeLLVMResultStructType(name string) (baztypes.Type, bool) {
	if !strings.HasPrefix(name, "Result__") || !strings.HasSuffix(name, "__Error") {
		return baztypes.Type{}, false
	}
	okName := strings.TrimSuffix(strings.TrimPrefix(name, "Result__"), "__Error")
	okType := decodeLLVMEncodedTypeName(okName)
	return baztypes.Type{
		Kind: baztypes.KindNamed,
		Name: "Result",
		Args: []baztypes.Type{
			okType,
			baztypes.MustParse(ast.Type("Error")),
		},
	}, true
}

func decodeLLVMEncodedTypeName(name string) baztypes.Type {
	switch name {
	case "int":
		return baztypes.MustParse(ast.TypeInt)
	case "string":
		return baztypes.MustParse(ast.TypeString)
	case "float":
		return baztypes.MustParse(ast.TypeFloat)
	case "bool":
		return baztypes.MustParse(ast.TypeBool)
	case "Error":
		return baztypes.MustParse(ast.Type("Error"))
	}
	if strings.HasPrefix(name, "pkg_") {
		if idx := strings.LastIndex(name, "_"); idx >= 0 && idx+1 < len(name) {
			name = name[idx+1:]
		}
	}
	return baztypes.Type{Kind: baztypes.KindNamed, Name: name}
}

func canonicalTypeName(name string) string {
	if !strings.HasPrefix(name, "__pkg_") {
		return name
	}
	rest := strings.TrimPrefix(name, "__pkg_")
	if idx := strings.Index(rest, "__"); idx >= 0 && idx+2 < len(rest) {
		return rest[idx+2:]
	}
	return name
}
