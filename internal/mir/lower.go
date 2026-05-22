package mir

import (
	"fmt"

	"baziclang/internal/ast"
	"baziclang/internal/hir"
	baztypes "baziclang/internal/types"
)

func Lower(p *hir.Program) (*Program, error) {
	if p == nil {
		return nil, fmt.Errorf("mir: nil program")
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
	index := newTypeIndex(out)
	globalCtx := newTypeContext(index)
	globalConsts := map[string]Expr{}
	for _, decl := range out.Decls {
		global, ok := decl.(*GlobalLetDecl)
		if !ok || global.Init == nil {
			continue
		}
		global.Init = simplifyExprWithConsts(globalCtx, global.Init, globalConsts)
		if global.IsConst && global.Name != "" && isConstLikeExpr(globalCtx, global.Init) {
			globalConsts[global.Name] = cloneConstLikeExpr(global.Init)
		}
	}
	for _, decl := range out.Decls {
		fn, ok := decl.(*FuncDecl)
		if !ok {
			continue
		}
		normalizeFuncLocals(index, fn)
		materializeValueOps(index, fn)
		simplifyFunc(index, fn, globalConsts)
		simplifyControlFlow(index, fn)
		pruneSyntheticTemps(fn)
		cfg, err := lowerCFG(fn.Body)
		if err != nil {
			return nil, err
		}
		simplifyCFG(newFuncTypeContext(index, fn), cfg)
		fn.CFG = cfg
		canonicalizeCFGOperands(index, fn)
		materializeCFGValueOps(index, fn)
		simplifyCFGFunc(index, fn)
		simplifyCFG(newFuncTypeContext(index, fn), fn.CFG)
		liveness := AnalyzeCFGLiveness(fn)
		pruneDeadCFGInstructionsWithAnalysis(fn, liveness)
		discardUnusedCFGValueBindingsWithAnalysis(fn, liveness)
		simplifyCFG(newFuncTypeContext(index, fn), fn.CFG)
	}
	if err := Validate(out); err != nil {
		return nil, err
	}
	return out, nil
}

func Validate(p *Program) error {
	if p == nil {
		return fmt.Errorf("mir: nil program")
	}
	index := newTypeIndex(p)
	for _, decl := range p.Decls {
		if err := validateDecl(index, decl); err != nil {
			return err
		}
	}
	return nil
}

func validateDecl(index *typeIndex, d Decl) error {
	switch decl := d.(type) {
	case *ImportDecl:
		if decl.Path == "" {
			return fmt.Errorf("mir: empty import path")
		}
	case *StructDecl:
		for _, field := range decl.Fields {
			if field.Name == "" {
				return fmt.Errorf("mir: unnamed struct field")
			}
		}
	case *InterfaceDecl:
		for _, method := range decl.Methods {
			if method.Name == "" {
				return fmt.Errorf("mir: unnamed interface method")
			}
		}
	case *ImplDecl:
		if decl.InterfaceName == "" {
			return fmt.Errorf("mir: impl missing interface")
		}
	case *EnumDecl:
		if len(decl.Variants) == 0 {
			return fmt.Errorf("mir: enum '%s' has no variants", decl.Name)
		}
	case *FuncDecl:
		if decl.Name == "" {
			return fmt.Errorf("mir: unnamed function")
		}
		if decl.Body == nil {
			return fmt.Errorf("mir: function '%s' missing body", decl.Name)
		}
		ctx := newFuncTypeContext(index, decl)
		if err := validateBlock(ctx, decl.Body, decl.ReturnType); err != nil {
			return err
		}
		if err := validateReturnShapeInBlock(decl.Body, decl.ReturnType); err != nil {
			return fmt.Errorf("mir: function '%s': %w", decl.Name, err)
		}
		if baztypes.ToAST(decl.ReturnType) != ast.TypeVoid && !blockAlwaysReturns(decl.Body) {
			return fmt.Errorf("mir: function '%s' falls through without returning %s", decl.Name, decl.ReturnType)
		}
		if decl.CFG != nil {
			if err := validateCFG(ctx, decl.CFG, decl.ReturnType); err != nil {
				return err
			}
		}
		return nil
	case *GlobalLetDecl:
		if decl.Name == "" {
			return fmt.Errorf("mir: unnamed global")
		}
		if decl.Init == nil {
			return fmt.Errorf("mir: global '%s' missing initializer", decl.Name)
		}
		ctx := newTypeContext(index)
		if err := validateExpr(ctx, decl.Init); err != nil {
			return err
		}
		if initType, ok := ctx.inferExprType(decl.Init); ok && !typesCompatible(decl.Type, initType) {
			return fmt.Errorf("mir: global '%s' initializer has type %s, expected %s", decl.Name, initType, decl.Type)
		}
		return nil
	default:
		return fmt.Errorf("mir: unsupported declaration %T", d)
	}
	return nil
}

func validateBlock(ctx *typeContext, b *Block, ret baztypes.Type) error {
	if b == nil {
		return nil
	}
	terminated := false
	for _, stmt := range b.Stmts {
		if terminated {
			return fmt.Errorf("mir: unreachable statement after terminator")
		}
		if err := validateStmt(ctx, stmt, ret); err != nil {
			return err
		}
		if StmtAlwaysReturns(stmt) {
			terminated = true
		}
	}
	return nil
}

func validateReturnShapeInBlock(b *Block, ret baztypes.Type) error {
	if b == nil {
		return nil
	}
	for _, stmt := range b.Stmts {
		if err := validateReturnShapeInStmt(stmt, ret); err != nil {
			return err
		}
	}
	return nil
}

func validateReturnShapeInStmt(s Stmt, ret baztypes.Type) error {
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[error](info,
			nil,
			nil,
			nil,
			nil,
			func(value Expr) error {
				return validateReturnValue(nil, value, ret,
					"void function cannot return a value",
					"non-void function must return a value of type %s",
					"",
				)
			},
		); ok {
			return out
		}
	}
	var err error
	WalkStmtChildBlocks(s, func(child *Block) {
		if err == nil {
			err = validateReturnShapeInBlock(child, ret)
		}
	})
	return err
}

func validateStmt(ctx *typeContext, s Stmt, ret baztypes.Type) error {
	if IsValueStmt(s) {
		info, ok := ValueStmtInfo(s)
		if !ok {
			return fmt.Errorf("mir: unsupported value statement %T", s)
		}
		if info.Name == "" {
			return fmt.Errorf("mir: %s statement missing name", info.Kind)
		}
		if info.Expr == nil {
			return fmt.Errorf("mir: %s statement missing value expression", info.Kind)
		}
		if err := validateExpr(ctx, info.Expr); err != nil {
			return err
		}
		if valueType, ok := ctx.inferExprType(info.Expr); ok && !typesCompatible(info.Type, valueType) {
			return fmt.Errorf("mir: %s '%s' has type %s, expected %s", info.Kind, info.Name, valueType, info.Type)
		}
		return nil
	}
	if info, ok := LinearStmtInfo(s); ok {
		if out, ok := MapLinearStmt[error](info,
			func(target Expr, value Expr) error {
				if err := validateAssignTarget(target); err != nil {
					return err
				}
				targetType, okTarget := ctx.inferExprType(target)
				valueType, okValue := ctx.inferExprType(value)
				if okTarget && okValue && !typesCompatible(targetType, valueType) {
					return fmt.Errorf("mir: assignment value has type %s, expected %s", valueType, targetType)
				}
				return validateExpr(ctx, value)
			},
			func(value Expr) error {
				return validateExpr(ctx, value)
			},
		); ok {
			return out
		}
	}
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[error](info,
			func(block *Block) error {
				return validateBlock(ctx, block, ret)
			},
			func(cond Expr, then, els *Block) error {
				if err := validateBoolExpr(ctx, cond, "mir: if condition has type %s, expected bool"); err != nil {
					return err
				}
				if err := validateBlock(ctx, then, ret); err != nil {
					return err
				}
				return validateBlock(ctx, els, ret)
			},
			func(cond Expr, body *Block) error {
				if err := validateBoolExpr(ctx, cond, "mir: while condition has type %s, expected bool"); err != nil {
					return err
				}
				return validateBlock(ctx, body, ret)
			},
			func(subject Expr, arms []MatchArm) error {
				if err := validateExpr(ctx, subject); err != nil {
					return err
				}
				return validateMatchArms(ctx, arms,
					"mir: match statement has no arms",
					"mir: match statement arm missing variant",
					"mir: match statement guard has type %s, expected bool",
					func(arm MatchArm, variant string, guard Expr) error {
						return validateBlock(ctx, arm.Body, ret)
					},
				)
			},
			func(value Expr) error {
				return validateReturnValue(ctx, value, ret,
					"void function cannot return a value",
					"non-void function must return a value of type %s",
					"mir: return value has type %s, expected %s",
				)
			},
		); ok {
			return out
		}
	}
	return fmt.Errorf("mir: unsupported statement %T", s)
}

func validateExpr(ctx *typeContext, e Expr) error {
	switch ex := e.(type) {
	case *IdentExpr, *IntExpr, *FloatExpr, *BoolExpr, *StringExpr, *NilExpr:
		return nil
	case *UnaryExpr:
		return validateExpr(ctx, ex.Right)
	case *BinaryExpr:
		if err := validateExpr(ctx, ex.Left); err != nil {
			return err
		}
		return validateExpr(ctx, ex.Right)
	case *CallExpr:
		if ex.Func == "" {
			return fmt.Errorf("mir: unresolved call")
		}
		for _, arg := range ex.Args {
			if err := validateExpr(ctx, arg); err != nil {
				return err
			}
		}
	case *FieldAccessExpr:
		return validateExpr(ctx, ex.Object)
	case *StructLitExpr:
		if ex.TypeName == "" {
			return fmt.Errorf("mir: struct literal missing type")
		}
		for _, field := range ex.Fields {
			if field.Name == "" {
				return fmt.Errorf("mir: struct literal field missing name")
			}
			if err := validateExpr(ctx, field.Value); err != nil {
				return err
			}
		}
	case *MatchExpr:
		if ex.Type.Kind == baztypes.KindInvalid && ex.Type.Name == "" && len(ex.Type.Args) == 0 {
			return fmt.Errorf("mir: match expression missing type")
		}
		if err := validateExpr(ctx, ex.Subject); err != nil {
			return err
		}
		return validateMatchExprArms(ctx, ex)
	default:
		return fmt.Errorf("mir: unsupported expression %T", e)
	}
	return nil
}

func validateAssignTarget(e Expr) error {
	switch ex := e.(type) {
	case *IdentExpr:
		if ex.Name == "" {
			return fmt.Errorf("mir: assignment target identifier missing name")
		}
		return nil
	case *FieldAccessExpr:
		if ex.Field == "" {
			return fmt.Errorf("mir: assignment target field missing name")
		}
		return validateExpr(nil, ex.Object)
	default:
		return fmt.Errorf("mir: invalid assignment target %T", e)
	}
}

func blockAlwaysReturns(b *Block) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if StmtAlwaysReturns(s) {
			return true
		}
	}
	return false
}

func lowerDecl(d hir.Decl) (Decl, error) {
	switch decl := d.(type) {
	case *hir.ImportDecl:
		return &ImportDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Path: decl.Path, Alias: decl.Alias, ExplicitAlias: decl.ExplicitAlias}, nil
	case *hir.StructDecl:
		fields := mapSlice(decl.Fields, func(field hir.StructField) StructField {
			return StructField{Range: field.Range, Name: field.Name, Type: field.Type}
		})
		return &StructDecl{
			NodeInfo:        NodeInfo{Range: decl.Span()},
			Name:            decl.Name,
			TypeParams:      append([]string{}, decl.TypeParams...),
			TypeParamBounds: cloneTypeMap(decl.TypeParamBounds),
			Fields:          fields,
		}, nil
	case *hir.InterfaceDecl:
		methods := mapSlice(decl.Methods, func(method hir.InterfaceMethod) InterfaceMethod {
			params := mapSlice(method.Params, func(param hir.Param) Param {
				return Param{Range: param.Range, Name: param.Name, Type: param.Type}
			})
			return InterfaceMethod{Range: method.Range, Name: method.Name, Params: params, Return: method.Return}
		})
		return &InterfaceDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Name: decl.Name, Methods: methods}, nil
	case *hir.ImplDecl:
		return &ImplDecl{NodeInfo: NodeInfo{Range: decl.Span()}, StructType: decl.StructType, InterfaceName: decl.InterfaceName}, nil
	case *hir.EnumDecl:
		return &EnumDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Name: decl.Name, Variants: append([]string{}, decl.Variants...)}, nil
	case *hir.FuncDecl:
		params := mapSlice(decl.Params, func(param hir.Param) Param {
			return Param{Range: param.Range, Name: param.Name, Type: param.Type}
		})
		body, err := lowerBlock(decl.Body)
		if err != nil {
			return nil, err
		}
		outFn := &FuncDecl{
			NodeInfo:        NodeInfo{Range: decl.Span()},
			Name:            decl.Name,
			TypeParams:      append([]string{}, decl.TypeParams...),
			TypeParamBounds: cloneTypeMap(decl.TypeParamBounds),
			Params:          params,
			ReturnType:      decl.ReturnType,
			Body:            body,
		}
		return outFn, nil
	case *hir.GlobalLetDecl:
		init, err := lowerExpr(decl.Init)
		if err != nil {
			return nil, err
		}
		return &GlobalLetDecl{NodeInfo: NodeInfo{Range: decl.Span()}, Name: decl.Name, Type: decl.Type, Init: init, IsConst: decl.IsConst}, nil
	default:
		return nil, fmt.Errorf("mir: unsupported declaration %T", d)
	}
}

func lowerBlock(b *hir.BlockStmt) (*Block, error) {
	if b == nil {
		return nil, nil
	}
	stmts, err := mapSliceE(b.Stmts, lowerStmt)
	if err != nil {
		return nil, err
	}
	return &Block{NodeInfo: NodeInfo{Range: b.Span()}, Stmts: stmts}, nil
}

func lowerStmt(s hir.Stmt) (Stmt, error) {
	switch st := s.(type) {
	case *hir.BlockStmt:
		return lowerBlock(st)
	case *hir.LetStmt:
		init, err := lowerExpr(st.Init)
		if err != nil {
			return nil, err
		}
		return &LetStmt{NodeInfo: NodeInfo{Range: st.Span()}, Name: st.Name, Type: st.Type, Init: init, IsConst: st.IsConst}, nil
	case *hir.AssignStmt:
		target, err := lowerExpr(st.Target)
		if err != nil {
			return nil, err
		}
		value, err := lowerExpr(st.Value)
		if err != nil {
			return nil, err
		}
		return &AssignStmt{NodeInfo: NodeInfo{Range: st.Span()}, Target: target, Value: value}, nil
	case *hir.IfStmt:
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
	case *hir.WhileStmt:
		cond, err := lowerExpr(st.Cond)
		if err != nil {
			return nil, err
		}
		body, err := lowerBlock(st.Body)
		if err != nil {
			return nil, err
		}
		return &WhileStmt{NodeInfo: NodeInfo{Range: st.Span()}, Cond: cond, Body: body}, nil
	case *hir.MatchStmt:
		subject, err := lowerExpr(st.Subject)
		if err != nil {
			return nil, err
		}
		arms, err := mapSliceE(st.Arms, func(arm hir.MatchArm) (MatchArm, error) {
			guard, err := mapOptionalE(arm.Guard, hir.Expr(nil), lowerExpr)
			if err != nil {
				return MatchArm{}, err
			}
			body, err := lowerBlock(arm.Body)
			if err != nil {
				return MatchArm{}, err
			}
			return MatchArm{Range: arm.Range, Variant: arm.Variant, Guard: guard, Body: body}, nil
		})
		if err != nil {
			return nil, err
		}
		return &MatchStmt{NodeInfo: NodeInfo{Range: st.Span()}, Subject: subject, Arms: arms}, nil
	case *hir.ReturnStmt:
		value, err := mapOptionalE(st.Value, hir.Expr(nil), lowerExpr)
		if err != nil {
			return nil, err
		}
		return &ReturnStmt{NodeInfo: NodeInfo{Range: st.Span()}, Value: value}, nil
	case *hir.ExprStmt:
		expr, err := lowerExpr(st.Expr)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{NodeInfo: NodeInfo{Range: st.Span()}, Expr: expr}, nil
	default:
		return nil, fmt.Errorf("mir: unsupported statement %T", s)
	}
}

func lowerExpr(e hir.Expr) (Expr, error) {
	switch ex := e.(type) {
	case *hir.IdentExpr:
		return &IdentExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Name: ex.Name}, nil
	case *hir.IntExpr:
		return &IntExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *hir.FloatExpr:
		return &FloatExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *hir.BoolExpr:
		return &BoolExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *hir.StringExpr:
		return &StringExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Value: ex.Value}, nil
	case *hir.NilExpr:
		return &NilExpr{NodeInfo: NodeInfo{Range: ex.Span()}}, nil
	case *hir.UnaryExpr:
		right, err := lowerExpr(ex.Right)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Op: ex.Op, Right: right}, nil
	case *hir.BinaryExpr:
		left, err := lowerExpr(ex.Left)
		if err != nil {
			return nil, err
		}
		right, err := lowerExpr(ex.Right)
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Left: left, Op: ex.Op, Right: right}, nil
	case *hir.CallExpr:
		args, err := mapSliceE(ex.Args, lowerExpr)
		if err != nil {
			return nil, err
		}
		return &CallExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Func: ex.Func, Args: args}, nil
	case *hir.FieldAccessExpr:
		object, err := lowerExpr(ex.Object)
		if err != nil {
			return nil, err
		}
		return &FieldAccessExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Object: object, Field: ex.Field}, nil
	case *hir.StructLitExpr:
		fields, err := mapSliceE(ex.Fields, func(field hir.StructLitField) (StructLitField, error) {
			value, err := lowerExpr(field.Value)
			if err != nil {
				return StructLitField{}, err
			}
			return StructLitField{Range: field.Range, Name: field.Name, Value: value}, nil
		})
		if err != nil {
			return nil, err
		}
		return &StructLitExpr{NodeInfo: NodeInfo{Range: ex.Span()}, TypeName: ex.TypeName, Fields: fields}, nil
	case *hir.MatchExpr:
		subject, err := lowerExpr(ex.Subject)
		if err != nil {
			return nil, err
		}
		arms, err := mapSliceE(ex.Arms, func(arm hir.MatchExprArm) (MatchExprArm, error) {
			guard, err := mapOptionalE(arm.Guard, hir.Expr(nil), lowerExpr)
			if err != nil {
				return MatchExprArm{}, err
			}
			value, err := lowerExpr(arm.Value)
			if err != nil {
				return MatchExprArm{}, err
			}
			return MatchExprArm{Range: arm.Range, Variant: arm.Variant, Guard: guard, Value: value}, nil
		})
		if err != nil {
			return nil, err
		}
		return &MatchExpr{NodeInfo: NodeInfo{Range: ex.Span()}, Subject: subject, Arms: arms, Type: ex.Type}, nil
	default:
		return nil, fmt.Errorf("mir: unsupported expression %T", e)
	}
}

func cloneTypeMap(in map[string]baztypes.Type) map[string]baztypes.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]baztypes.Type, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
