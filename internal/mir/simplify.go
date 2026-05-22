package mir

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	baztypes "baziclang/internal/types"
)

func simplifyFunc(index *typeIndex, fn *FuncDecl, globals map[string]Expr) {
	if fn == nil || fn.Body == nil {
		return
	}
	ctx := newFuncTypeContext(index, fn)
	consts := cloneConstExprMap(globals)
	for _, param := range fn.Params {
		delete(consts, param.Name)
	}
	simplifyBlock(ctx, fn.Body, consts)
}

func simplifyCFGFunc(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.CFG == nil {
		return
	}
	ctx := newFuncTypeContext(index, fn)
	for _, block := range fn.CFG.Blocks {
		simplifyCFGBlock(ctx, block)
	}
}

func simplifyCFGBlock(ctx *typeContext, block *BasicBlock) {
	if block == nil {
		return
	}
	consts := map[string]Expr{}
	for i, stmt := range block.Instrs {
		block.Instrs[i] = simplifyStmt(ctx, stmt, consts)
	}
	RewriteTerminatorExprs(block.Term, func(expr Expr) Expr {
		return simplifyExprWithConsts(ctx, expr, consts)
	})
	if info, ok := TerminatorInfo(block.Term); ok && info.Kind == "match" {
		info.Arms = simplifyMatchTerminatorArms(info.Arms)
		SetTerminatorInfo(block.Term, info)
	}
}

func simplifyBlock(ctx *typeContext, b *Block, seed map[string]Expr) {
	if b == nil {
		return
	}
	consts := cloneConstExprMap(seed)
	RewriteBlockStmts(b, func(stmt Stmt) []Stmt {
		return []Stmt{simplifyStmt(ctx, stmt, consts)}
	})
}

func simplifyStmt(ctx *typeContext, s Stmt, consts map[string]Expr) Stmt {
	if IsValueStmt(s) {
		return simplifyValueStmt(ctx, s, consts)
	}
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[Stmt](info,
			func(block *Block) Stmt {
				simplifyBlock(ctx, block, consts)
				return s
			},
			func(cond Expr, then, els *Block) Stmt {
				RewriteStmtExprs(s, func(expr Expr) Expr {
					return simplifyExprWithConsts(ctx, expr, consts)
				})
				WalkStmtChildBlocks(s, func(child *Block) {
					simplifyBlock(ctx, child, consts)
				})
				clearConstExprMap(consts)
				return s
			},
			func(cond Expr, body *Block) Stmt {
				RewriteStmtExprs(s, func(expr Expr) Expr {
					return simplifyExprWithConsts(ctx, expr, consts)
				})
				WalkStmtChildBlocks(s, func(child *Block) {
					simplifyBlock(ctx, child, consts)
				})
				clearConstExprMap(consts)
				return s
			},
			func(subject Expr, arms []MatchArm) Stmt {
				RewriteStmtExprs(s, func(expr Expr) Expr {
					return simplifyExprWithConsts(ctx, expr, consts)
				})
				WalkStmtChildBlocks(s, func(child *Block) {
					simplifyBlock(ctx, child, consts)
				})
				info, _ = StmtControlInfo(s)
				info.Arms = simplifyMatchArms(info.Arms)
				SetStmtControlInfo(s, info)
				clearConstExprMap(consts)
				return s
			},
			func(value Expr) Stmt {
				RewriteStmtExprs(s, func(expr Expr) Expr {
					return simplifyExprWithConsts(ctx, expr, consts)
				})
				return s
			},
		); ok {
			return out
		}
	}
	switch st := s.(type) {
	case *AssignStmt:
		RewriteStmtExprs(st, func(expr Expr) Expr {
			return simplifyExprWithConsts(ctx, expr, consts)
		})
		invalidateAssignTargetConsts(consts, st.Target)
		return st
	case *ExprStmt:
		RewriteStmtExprs(st, func(expr Expr) Expr {
			return simplifyExprWithConsts(ctx, expr, consts)
		})
		return st
	}
	return s
}

func simplifyValueStmt(ctx *typeContext, s Stmt, consts map[string]Expr) Stmt {
	info, ok := ValueStmtInfo(s)
	if !ok {
		return s
	}
	simplified := simplifyExprWithConsts(ctx, info.Expr, consts)
	if info.Name != "_" && (isSyntheticTempName(info.Name) || info.IsConst) && isConstLikeExpr(ctx, simplified) {
		consts[info.Name] = cloneConstLikeExpr(simplified)
	} else {
		delete(consts, info.Name)
	}
	return rebuildValueStmt(s, info.Type, simplified)
}

func invalidateAssignTargetConsts(consts map[string]Expr, target Expr) {
	if consts == nil || target == nil {
		return
	}
	switch ex := target.(type) {
	case *IdentExpr:
		delete(consts, ex.Name)
	case *FieldAccessExpr:
		invalidateAssignTargetConsts(consts, ex.Object)
	}
}

func rebuildValueStmt(s Stmt, typ baztypes.Type, expr Expr) Stmt {
	info, ok := ValueStmtInfo(s)
	if !ok {
		return s
	}
	return buildBoundValueStmt(NodeInfo{Range: s.Span()}, info.Name, typ, expr, info.IsConst)
}

func simplifyExpr(ctx *typeContext, e Expr) Expr {
	return simplifyExprWithConsts(ctx, e, nil)
}

func simplifyExprWithConsts(ctx *typeContext, e Expr, consts map[string]Expr) Expr {
	return RewriteExpr(e, func(expr Expr) Expr {
		switch ex := expr.(type) {
		case *IdentExpr:
			if consts != nil {
				if constant, ok := consts[ex.Name]; ok {
					return cloneConstLikeExpr(constant)
				}
			}
			return ex
		case *UnaryExpr:
			if out, ok := foldUnaryExpr(ex); ok {
				return out
			}
			return ex
		case *BinaryExpr:
			if out, ok := foldBinaryExpr(ctx, ex); ok {
				return out
			}
			return ex
		case *CallExpr:
			if out, ok := foldCallExpr(ex); ok {
				return out
			}
			return ex
		case *FieldAccessExpr:
			if out, ok := foldFieldAccessExpr(ctx, ex); ok {
				return out
			}
			return ex
		case *MatchExpr:
			ex.Arms = simplifyMatchExprArms(ex.Arms)
			if out, ok := foldMatchExpr(ctx, ex); ok {
				return out
			}
			return ex
		default:
			return expr
		}
	})
}

func foldUnaryExpr(ex *UnaryExpr) (Expr, bool) {
	switch ex.Op {
	case "!":
		if right, ok := ex.Right.(*BoolExpr); ok {
			return &BoolExpr{NodeInfo: ex.NodeInfo, Value: !right.Value}, true
		}
	case "-":
		if right, ok := ex.Right.(*IntExpr); ok {
			return &IntExpr{NodeInfo: ex.NodeInfo, Value: -right.Value}, true
		}
		if right, ok := ex.Right.(*FloatExpr); ok {
			return &FloatExpr{NodeInfo: ex.NodeInfo, Value: -right.Value}, true
		}
	}
	return nil, false
}

func foldBinaryExpr(ctx *typeContext, ex *BinaryExpr) (Expr, bool) {
	switch left := ex.Left.(type) {
	case *IntExpr:
		if right, ok := ex.Right.(*IntExpr); ok {
			return foldIntBinary(ex.NodeInfo, left.Value, ex.Op, right.Value)
		}
	case *FloatExpr:
		if right, ok := ex.Right.(*FloatExpr); ok {
			return foldFloatBinary(ex.NodeInfo, left.Value, ex.Op, right.Value)
		}
	case *BoolExpr:
		if right, ok := ex.Right.(*BoolExpr); ok {
			return foldBoolBinary(ex.NodeInfo, left.Value, ex.Op, right.Value)
		}
	case *StringExpr:
		if right, ok := ex.Right.(*StringExpr); ok {
			return foldStringBinary(ex.NodeInfo, left.Value, ex.Op, right.Value)
		}
	}
	if ctx != nil {
		if t, ok := ctx.inferExprType(ex); ok && t.String() == string(ast.TypeBool) {
			if left, lok := ex.Left.(*BoolExpr); lok && ex.Op == "&&" {
				if left.Value {
					return ex.Right, true
				}
				return &BoolExpr{NodeInfo: ex.NodeInfo, Value: false}, true
			}
			if left, lok := ex.Left.(*BoolExpr); lok && ex.Op == "||" {
				if left.Value {
					return &BoolExpr{NodeInfo: ex.NodeInfo, Value: true}, true
				}
				return ex.Right, true
			}
			if right, rok := ex.Right.(*BoolExpr); rok && ex.Op == "&&" {
				if right.Value {
					return ex.Left, true
				}
				return &BoolExpr{NodeInfo: ex.NodeInfo, Value: false}, true
			}
			if right, rok := ex.Right.(*BoolExpr); rok && ex.Op == "||" {
				if right.Value {
					return &BoolExpr{NodeInfo: ex.NodeInfo, Value: true}, true
				}
				return ex.Left, true
			}
		}
	}
	return nil, false
}

func foldIntBinary(info NodeInfo, left int64, op string, right int64) (Expr, bool) {
	switch op {
	case "+":
		return &IntExpr{NodeInfo: info, Value: left + right}, true
	case "-":
		return &IntExpr{NodeInfo: info, Value: left - right}, true
	case "*":
		return &IntExpr{NodeInfo: info, Value: left * right}, true
	case "/":
		if right == 0 {
			return nil, false
		}
		return &IntExpr{NodeInfo: info, Value: left / right}, true
	case "%":
		if right == 0 {
			return nil, false
		}
		return &IntExpr{NodeInfo: info, Value: left % right}, true
	case "==":
		return &BoolExpr{NodeInfo: info, Value: left == right}, true
	case "!=":
		return &BoolExpr{NodeInfo: info, Value: left != right}, true
	case "<":
		return &BoolExpr{NodeInfo: info, Value: left < right}, true
	case "<=":
		return &BoolExpr{NodeInfo: info, Value: left <= right}, true
	case ">":
		return &BoolExpr{NodeInfo: info, Value: left > right}, true
	case ">=":
		return &BoolExpr{NodeInfo: info, Value: left >= right}, true
	default:
		return nil, false
	}
}

func foldFloatBinary(info NodeInfo, left float64, op string, right float64) (Expr, bool) {
	switch op {
	case "+":
		return &FloatExpr{NodeInfo: info, Value: left + right}, true
	case "-":
		return &FloatExpr{NodeInfo: info, Value: left - right}, true
	case "*":
		return &FloatExpr{NodeInfo: info, Value: left * right}, true
	case "/":
		if right == 0 {
			return nil, false
		}
		return &FloatExpr{NodeInfo: info, Value: left / right}, true
	case "==":
		return &BoolExpr{NodeInfo: info, Value: left == right}, true
	case "!=":
		return &BoolExpr{NodeInfo: info, Value: left != right}, true
	case "<":
		return &BoolExpr{NodeInfo: info, Value: left < right}, true
	case "<=":
		return &BoolExpr{NodeInfo: info, Value: left <= right}, true
	case ">":
		return &BoolExpr{NodeInfo: info, Value: left > right}, true
	case ">=":
		return &BoolExpr{NodeInfo: info, Value: left >= right}, true
	default:
		return nil, false
	}
}

func foldBoolBinary(info NodeInfo, left bool, op string, right bool) (Expr, bool) {
	switch op {
	case "&&":
		return &BoolExpr{NodeInfo: info, Value: left && right}, true
	case "||":
		return &BoolExpr{NodeInfo: info, Value: left || right}, true
	case "==":
		return &BoolExpr{NodeInfo: info, Value: left == right}, true
	case "!=":
		return &BoolExpr{NodeInfo: info, Value: left != right}, true
	default:
		return nil, false
	}
}

func foldStringBinary(info NodeInfo, left string, op string, right string) (Expr, bool) {
	switch op {
	case "+":
		return &StringExpr{NodeInfo: info, Value: left + right}, true
	case "==":
		return &BoolExpr{NodeInfo: info, Value: left == right}, true
	case "!=":
		return &BoolExpr{NodeInfo: info, Value: left != right}, true
	case "<":
		return &BoolExpr{NodeInfo: info, Value: left < right}, true
	case "<=":
		return &BoolExpr{NodeInfo: info, Value: left <= right}, true
	case ">":
		return &BoolExpr{NodeInfo: info, Value: left > right}, true
	case ">=":
		return &BoolExpr{NodeInfo: info, Value: left >= right}, true
	default:
		return nil, false
	}
}

func foldCallExpr(ex *CallExpr) (Expr, bool) {
	spec, ok := intrinsics.LookupLoweredBuiltin(ex.Func)
	if !ok {
		return nil, false
	}
	switch spec.Name {
	case "len":
		if len(ex.Args) == 1 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				return &IntExpr{NodeInfo: ex.NodeInfo, Value: int64(utf8.RuneCountInString(s.Value))}, true
			}
		}
	case "contains":
		if len(ex.Args) == 2 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				if sub, ok := ex.Args[1].(*StringExpr); ok {
					return &BoolExpr{NodeInfo: ex.NodeInfo, Value: strings.Contains(s.Value, sub.Value)}, true
				}
			}
		}
	case "starts_with":
		if len(ex.Args) == 2 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				if prefix, ok := ex.Args[1].(*StringExpr); ok {
					return &BoolExpr{NodeInfo: ex.NodeInfo, Value: strings.HasPrefix(s.Value, prefix.Value)}, true
				}
			}
		}
	case "ends_with":
		if len(ex.Args) == 2 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				if suffix, ok := ex.Args[1].(*StringExpr); ok {
					return &BoolExpr{NodeInfo: ex.NodeInfo, Value: strings.HasSuffix(s.Value, suffix.Value)}, true
				}
			}
		}
	case "to_upper":
		if len(ex.Args) == 1 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strings.ToUpper(s.Value)}, true
			}
		}
	case "to_lower":
		if len(ex.Args) == 1 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strings.ToLower(s.Value)}, true
			}
		}
	case "trim_space":
		if len(ex.Args) == 1 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strings.TrimSpace(s.Value)}, true
			}
		}
	case "replace":
		if len(ex.Args) == 3 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				if old, ok := ex.Args[1].(*StringExpr); ok {
					if newv, ok := ex.Args[2].(*StringExpr); ok {
						return &StringExpr{NodeInfo: ex.NodeInfo, Value: strings.ReplaceAll(s.Value, old.Value, newv.Value)}, true
					}
				}
			}
		}
	case "repeat":
		if len(ex.Args) == 2 {
			if s, ok := ex.Args[0].(*StringExpr); ok {
				if count, ok := ex.Args[1].(*IntExpr); ok && count.Value >= 0 {
					return &StringExpr{NodeInfo: ex.NodeInfo, Value: strings.Repeat(s.Value, int(count.Value))}, true
				}
			}
		}
	case "str":
		if len(ex.Args) == 1 {
			switch v := ex.Args[0].(type) {
			case *StringExpr:
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: v.Value}, true
			case *IntExpr:
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strconv.FormatInt(v.Value, 10)}, true
			case *FloatExpr:
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strconv.FormatFloat(v.Value, 'g', -1, 64)}, true
			case *BoolExpr:
				return &StringExpr{NodeInfo: ex.NodeInfo, Value: strconv.FormatBool(v.Value)}, true
			}
		}
	}
	return nil, false
}

func nearlyEqualFloat(a, b float64) bool {
	diff := math.Abs(a - b)
	scale := math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
	return diff <= 1e-12*scale
}

func isConstantExpr(e Expr) bool {
	switch e.(type) {
	case *IntExpr, *FloatExpr, *BoolExpr, *StringExpr, *NilExpr:
		return true
	default:
		return false
	}
}

func isConstLikeExpr(ctx *typeContext, e Expr) bool {
	if isConstantExpr(e) {
		return true
	}
	if lit, ok := e.(*StructLitExpr); ok {
		for _, field := range lit.Fields {
			if !isConstLikeExpr(ctx, field.Value) {
				return false
			}
		}
		return true
	}
	if ctx == nil {
		return false
	}
	_, ok := constantEnumVariantName(ctx, e)
	return ok
}

func cloneConstLikeExpr(e Expr) Expr {
	if out := cloneConstantExpr(e); out != nil {
		return out
	}
	if lit, ok := e.(*StructLitExpr); ok {
		fields := make([]StructLitField, 0, len(lit.Fields))
		for _, field := range lit.Fields {
			fields = append(fields, StructLitField{
				Range: field.Range,
				Name:  field.Name,
				Value: cloneConstLikeExpr(field.Value),
			})
		}
		return &StructLitExpr{
			NodeInfo: lit.NodeInfo,
			TypeName: lit.TypeName,
			Fields:   fields,
		}
	}
	if ident, ok := e.(*IdentExpr); ok {
		return &IdentExpr{NodeInfo: ident.NodeInfo, Name: ident.Name}
	}
	return e
}

func cloneConstExprMap(values map[string]Expr) map[string]Expr {
	if len(values) == 0 {
		return map[string]Expr{}
	}
	out := make(map[string]Expr, len(values))
	for name, value := range values {
		out[name] = cloneConstLikeExpr(value)
	}
	return out
}

func cloneConstantExpr(e Expr) Expr {
	switch ex := e.(type) {
	case *IntExpr:
		return &IntExpr{NodeInfo: ex.NodeInfo, Value: ex.Value}
	case *FloatExpr:
		return &FloatExpr{NodeInfo: ex.NodeInfo, Value: ex.Value}
	case *BoolExpr:
		return &BoolExpr{NodeInfo: ex.NodeInfo, Value: ex.Value}
	case *StringExpr:
		return &StringExpr{NodeInfo: ex.NodeInfo, Value: ex.Value}
	case *NilExpr:
		return &NilExpr{NodeInfo: ex.NodeInfo}
	default:
		return nil
	}
}

func clearConstExprMap(values map[string]Expr) {
	for name := range values {
		delete(values, name)
	}
}

func isSyntheticTempName(name string) bool {
	return strings.HasPrefix(name, "let__mir") ||
		strings.HasPrefix(name, "lhs__mir") ||
		strings.HasPrefix(name, "rhs__mir") ||
		strings.HasPrefix(name, "arg__mir") ||
		strings.HasPrefix(name, "cond__mir") ||
		strings.HasPrefix(name, "ret__mir") ||
		strings.HasPrefix(name, "obj__mir") ||
		strings.HasPrefix(name, "field__mir") ||
		strings.HasPrefix(name, "subject__mir") ||
		strings.HasPrefix(name, "unary__mir")
}

func foldMatchExpr(ctx *typeContext, ex *MatchExpr) (Expr, bool) {
	arm, ok := SelectConstantMatchArm(ctx, ex.Subject, ex.Arms)
	if !ok {
		return nil, false
	}
	return arm.Value, true
}

func foldFieldAccessExpr(ctx *typeContext, ex *FieldAccessExpr) (Expr, bool) {
	lit, ok := ex.Object.(*StructLitExpr)
	if !ok {
		return nil, false
	}
	if !isConstLikeExpr(ctx, lit) {
		return nil, false
	}
	for _, field := range lit.Fields {
		if field.Name == ex.Field {
			return cloneConstLikeExpr(field.Value), true
		}
	}
	return nil, false
}

func constantEnumVariantName(ctx *typeContext, e Expr) (string, bool) {
	if ctx == nil || e == nil {
		return "", false
	}
	ident, ok := e.(*IdentExpr)
	if !ok {
		return "", false
	}
	_, ok = ctx.index.enumVariants[ident.Name]
	return ident.Name, ok
}
