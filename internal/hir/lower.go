package hir

import (
	"fmt"

	"baziclang/internal/ast"
	baztypes "baziclang/internal/types"
)

func Lower(p *ast.Program) (*Program, error) {
	if p == nil {
		return nil, fmt.Errorf("hir: nil program")
	}
	out := &Program{
		NodeInfo: NodeInfo{Range: p.Span()},
		Decls:    make([]Decl, 0, len(p.Decls)),
	}
	for _, decl := range p.Decls {
		lowered, err := lowerDecl(decl)
		if err != nil {
			return nil, err
		}
		out.Decls = append(out.Decls, lowered)
	}
	return out, nil
}

func lowerDecl(d ast.Decl) (Decl, error) {
	switch decl := d.(type) {
	case *ast.ImportDecl:
		return &ImportDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Path: decl.Path, Alias: decl.Alias, ExplicitAlias: decl.ExplicitAlias}, nil
	case *ast.StructDecl:
		fields := make([]StructField, 0, len(decl.Fields))
		for _, field := range decl.Fields {
			fields = append(fields, StructField{Range: field.Range, Name: field.Name, Type: baztypes.MustParse(field.Type)})
		}
		return &StructDecl{
			NodeInfo:        NodeInfo{Range: decl.Span()},
			Name:            firstNonEmpty(decl.InternalName, decl.Name),
			TypeParams:      append([]string{}, decl.TypeParams...),
			TypeParamBounds: cloneTypeMap(decl.TypeParamBounds),
			Fields:          fields,
		}, nil
	case *ast.InterfaceDecl:
		methods := make([]InterfaceMethod, 0, len(decl.Methods))
		for _, method := range decl.Methods {
			params := make([]Param, 0, len(method.Params))
			for _, param := range method.Params {
				params = append(params, Param{Range: param.Range, Name: param.Name, Type: baztypes.MustParse(param.Type)})
			}
			methods = append(methods, InterfaceMethod{Range: method.Range, Name: method.Name, Params: params, Return: baztypes.MustParse(method.Return)})
		}
		return &InterfaceDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Name: firstNonEmpty(decl.InternalName, decl.Name), Methods: methods}, nil
	case *ast.ImplDecl:
		return &ImplDecl{
			NodeInfo:      NodeInfo{Range: decl.Span()},
			StructType:    baztypes.MustParse(decl.StructType),
			InterfaceName: decl.InterfaceName,
		}, nil
	case *ast.EnumDecl:
		return &EnumDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Name: firstNonEmpty(decl.InternalName, decl.Name), Variants: append([]string{}, decl.Variants...)}, nil
	case *ast.FuncDecl:
		params := make([]Param, 0, len(decl.Params))
		for _, param := range decl.Params {
			params = append(params, Param{Range: param.Range, Name: param.Name, Type: baztypes.MustParse(param.Type)})
		}
		body, err := lowerBlock(decl.Body)
		if err != nil {
			return nil, err
		}
		return &FuncDecl{
			NodeInfo:        NodeInfo{Range: decl.Span()},
			Name:            firstNonEmpty(decl.InternalName, decl.Name),
			TypeParams:      append([]string{}, decl.TypeParams...),
			TypeParamBounds: cloneTypeMap(decl.TypeParamBounds),
			Params:          params,
			ReturnType:      baztypes.MustParse(decl.ReturnType),
			Body:            body,
		}, nil
	case *ast.GlobalLetDecl:
		init, err := lowerExpr(decl.Init)
		if err != nil {
			return nil, err
		}
		return &GlobalLetDecl{
			NodeInfo: NodeInfo{Range: decl.Span()},
			Name:     firstNonEmpty(decl.InternalName, decl.Name),
			Type:     baztypes.MustParse(decl.Type),
			Init:     init,
			IsConst:  decl.IsConst,
		}, nil
	default:
		return nil, fmt.Errorf("hir: unsupported declaration %T", d)
	}
}

func lowerBlock(b *ast.BlockStmt) (*BlockStmt, error) {
	if b == nil {
		return nil, nil
	}
	out := &BlockStmt{NodeInfo: NodeInfo{Range: b.Span()}, Stmts: make([]Stmt, 0, len(b.Stmts))}
	for _, stmt := range b.Stmts {
		lowered, err := lowerStmt(stmt)
		if err != nil {
			return nil, err
		}
		out.Stmts = append(out.Stmts, lowered)
	}
	return out, nil
}

func lowerStmt(s ast.Stmt) (Stmt, error) {
	switch st := s.(type) {
	case *ast.BlockStmt:
		return lowerBlock(st)
	case *ast.LetStmt:
		init, err := lowerExpr(st.Init)
		if err != nil {
			return nil, err
		}
		return &LetStmt{NodeInfo: NodeInfo{Range: st.Span()}, Name: st.Name, Type: baztypes.MustParse(st.Type), Init: init, IsConst: st.IsConst}, nil
	case *ast.AssignStmt:
		target, err := lowerExpr(st.Target)
		if err != nil {
			return nil, err
		}
		value, err := lowerExpr(st.Value)
		if err != nil {
			return nil, err
		}
		return &AssignStmt{NodeInfo: NodeInfo{Range: st.Span()}, Target: target, Value: value}, nil
	case *ast.IfStmt:
		cond, err := lowerExpr(st.Cond)
		if err != nil {
			return nil, err
		}
		thenBlock, err := lowerBlock(st.Then)
		if err != nil {
			return nil, err
		}
		elseBlock, err := lowerBlock(st.Else)
		if err != nil {
			return nil, err
		}
		return &IfStmt{NodeInfo: NodeInfo{Range: st.Span()}, Cond: cond, Then: thenBlock, Else: elseBlock}, nil
	case *ast.WhileStmt:
		cond, err := lowerExpr(st.Cond)
		if err != nil {
			return nil, err
		}
		body, err := lowerBlock(st.Body)
		if err != nil {
			return nil, err
		}
		return &WhileStmt{NodeInfo: NodeInfo{Range: st.Span()}, Cond: cond, Body: body}, nil
	case *ast.MatchStmt:
		subject, err := lowerExpr(st.Subject)
		if err != nil {
			return nil, err
		}
		arms := make([]MatchArm, 0, len(st.Arms))
		for _, arm := range st.Arms {
			guard, err := lowerOptionalExpr(arm.Guard)
			if err != nil {
				return nil, err
			}
			body, err := lowerBlock(arm.Body)
			if err != nil {
				return nil, err
			}
			arms = append(arms, MatchArm{Range: arm.Range, Variant: arm.Variant, Guard: guard, Body: body})
		}
		return &MatchStmt{NodeInfo: NodeInfo{Range: st.Span()}, Subject: subject, Arms: arms}, nil
	case *ast.ReturnStmt:
		value, err := lowerOptionalExpr(st.Value)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{NodeInfo: NodeInfo{Range: st.Span()}, Value: value}, nil
	case *ast.ExprStmt:
		expr, err := lowerExpr(st.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{NodeInfo: NodeInfo{Range: st.Span()}, Expr: expr}, nil
	default:
		return nil, fmt.Errorf("hir: unsupported statement %T", s)
	}
}

func lowerExpr(e ast.Expr) (Expr, error) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		return &IdentExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Name: firstNonEmpty(ex.Resolved, ex.Name)}, nil
	case *ast.IntExpr:
		return &IntExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *ast.FloatExpr:
		return &FloatExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *ast.BoolExpr:
		return &BoolExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *ast.StringExpr:
		return &StringExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *ast.NilExpr:
		return &NilExpr{NodeInfo: NodeInfo{Range: ex.Span()}}, nil
	case *ast.UnaryExpr:
		right, err := lowerExpr(ex.Right)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Op: ex.Op, Right: right}, nil
	case *ast.BinaryExpr:
		left, err := lowerExpr(ex.Left)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpr(ex.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Left: left, Op: ex.Op, Right: right}, nil
	case *ast.CallExpr:
		if ex.Receiver != nil || ex.Method != "" || ex.Callee == "" {
			return nil, fmt.Errorf("hir: unresolved call expression at %d:%d", ex.Span().Start.Line, ex.Span().Start.Col)
		}
		args := make([]Expr, 0, len(ex.Args))
		for _, arg := range ex.Args {
			lowered, err := lowerExpr(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, lowered)
		}
		return &CallExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Func: ex.Callee, Args: args}, nil
	case *ast.FieldAccessExpr:
		if ex.ResolvedGlobal != "" {
			return &IdentExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Name: ex.ResolvedGlobal}, nil
		}
		object, err := lowerExpr(ex.Object)
		if err != nil {
			return nil, err
		}
		return &FieldAccessExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Object: object, Field: ex.Field}, nil
	case *ast.StructLitExpr:
		fields := make([]StructLitField, 0, len(ex.Fields))
		for _, field := range ex.Fields {
			value, err := lowerExpr(field.Value)
			if err != nil {
				return nil, err
			}
			fields = append(fields, StructLitField{Range: field.Range, Name: field.Name, Value: value})
		}
		return &StructLitExpr{NodeInfo: NodeInfo{Range: ex.Span()}, TypeName: ex.TypeName, Fields: fields}, nil
	case *ast.MatchExpr:
		if ex.ResolvedType == ast.TypeInvalid {
			return nil, fmt.Errorf("hir: unresolved match expression type at %d:%d", ex.Span().Start.Line, ex.Span().Start.Col)
		}
		subject, err := lowerExpr(ex.Subject)
		if err != nil {
			return nil, err
		}
		arms := make([]MatchExprArm, 0, len(ex.Arms))
		for _, arm := range ex.Arms {
			guard, err := lowerOptionalExpr(arm.Guard)
			if err != nil {
				return nil, err
			}
			value, err := lowerExpr(arm.Value)
			if err != nil {
				return nil, err
			}
			arms = append(arms, MatchExprArm{Range: arm.Range, Variant: arm.Variant, Guard: guard, Value: value})
		}
		return &MatchExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Subject: subject, Arms: arms, Type: baztypes.MustParse(ex.ResolvedType)}, nil
	default:
		return nil, fmt.Errorf("hir: unsupported expression %T", e)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func lowerOptionalExpr(e ast.Expr) (Expr, error) {
	if e == nil {
		return nil, nil
	}
	return lowerExpr(e)
}

func cloneTypeMap(in map[string]ast.Type) map[string]baztypes.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]baztypes.Type, len(in))
	for k, v := range in {
		out[k] = baztypes.MustParse(v)
	}
	return out
}
