package mir

import (
	"fmt"

	"baziclang/internal/ast"
	"baziclang/internal/hir"
	baztypes "baziclang/internal/types"
)

func ToHIRGlobalLetDecl(g *GlobalLetDecl) *hir.GlobalLetDecl {
	if g == nil {
		return nil
	}
	return &hir.GlobalLetDecl{
		NodeInfo: hir.NodeInfo{Range: g.Span()},
		Name:     g.Name,
		Type:     g.Type,
		Init:     toHIRExpr(g.Init),
		IsConst:  g.IsConst,
	}
}

func ToHIRFuncDecl(fn *FuncDecl) *hir.FuncDecl {
	if fn == nil {
		return nil
	}
	params := mapSlice(fn.Params, func(p Param) hir.Param {
		return hir.Param{Range: p.Range, Name: p.Name, Type: p.Type}
	})
	return &hir.FuncDecl{
		NodeInfo:        hir.NodeInfo{Range: fn.Span()},
		Name:            fn.Name,
		TypeParams:      append([]string{}, fn.TypeParams...),
		TypeParamBounds: cloneTypeMap(fn.TypeParamBounds),
		Params:          params,
		ReturnType:      fn.ReturnType,
		Body:            toHIRBlock(fn.Body),
	}
}

func ToASTFuncDecl(fn *FuncDecl) *ast.FuncDecl {
	if fn == nil {
		return nil
	}
	params := mapSlice(fn.Params, func(p Param) ast.Param {
		return ast.Param{Range: p.Range, Name: p.Name, Type: baztypes.ToAST(p.Type)}
	})
	return &ast.FuncDecl{
		NodeInfo:        ast.NodeInfo{Range: fn.Span()},
		Name:            fn.Name,
		TypeParams:      append([]string{}, fn.TypeParams...),
		TypeParamBounds: toASTTypeMap(fn.TypeParamBounds),
		Params:          params,
		ReturnType:      baztypes.ToAST(fn.ReturnType),
		Body:            toASTBlock(fn.Body),
	}
}

func ToASTBlock(b *Block) *ast.BlockStmt {
	return toASTBlock(b)
}

func ToASTExpr(e Expr) ast.Expr {
	return toASTExpr(e)
}

func toHIRBlock(b *Block) *hir.BlockStmt {
	if b == nil {
		return nil
	}
	stmts := mapSlice(b.Stmts, toHIRStmt)
	return &hir.BlockStmt{NodeInfo: hir.NodeInfo{Range: b.Span()}, Stmts: stmts}
}

func toHIRStmt(s Stmt) hir.Stmt {
	if info, ok := ValueStmtInfo(s); ok {
		if info.Expr == nil {
			panic(fmt.Sprintf("mir: value stmt binding without expression %T", s))
		}
		return &hir.LetStmt{
			NodeInfo: hir.NodeInfo{Range: s.Span()},
			Name:     info.Name,
			Type:     info.Type,
			Init:     toHIRExpr(info.Expr),
			IsConst:  info.IsConst,
		}
	}
	if info, ok := LinearStmtInfo(s); ok {
		if out, ok := MapLinearStmt[hirstmtMarker](info,
			func(target Expr, value Expr) hirstmtMarker {
				return hirstmtMarker{stmt: &hir.AssignStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Target: toHIRExpr(target), Value: toHIRExpr(value)}}
			},
			func(value Expr) hirstmtMarker {
				return hirstmtMarker{stmt: &hir.ExprStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Expr: toHIRExpr(value)}}
			},
		); ok {
			return out.stmt
		}
	}
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[hirstmtMarker](info,
			func(block *Block) hirstmtMarker {
				return hirstmtMarker{stmt: toHIRBlock(block)}
			},
			func(cond Expr, then, els *Block) hirstmtMarker {
				return hirstmtMarker{stmt: &hir.IfStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Cond: toHIRExpr(cond), Then: toHIRBlock(then), Else: toHIRBlock(els)}}
			},
			func(cond Expr, body *Block) hirstmtMarker {
				return hirstmtMarker{stmt: &hir.WhileStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Cond: toHIRExpr(cond), Body: toHIRBlock(body)}}
			},
			func(subject Expr, armsIn []MatchArm) hirstmtMarker {
				arms := mapSlice(armsIn, func(arm MatchArm) hir.MatchArm {
					return hir.MatchArm{
						Range:   arm.Range,
						Variant: arm.Variant,
						Guard:   mapOptional(arm.Guard, Expr(nil), toHIRExpr),
						Body:    toHIRBlock(arm.Body),
					}
				})
				return hirstmtMarker{stmt: &hir.MatchStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Subject: toHIRExpr(subject), Arms: arms}}
			},
			func(value Expr) hirstmtMarker {
				return hirstmtMarker{stmt: &hir.ReturnStmt{NodeInfo: hir.NodeInfo{Range: s.Span()}, Value: mapOptional(value, Expr(nil), toHIRExpr)}}
			},
		); ok {
			return out.stmt
		}
	}
	panic(fmt.Sprintf("mir: unsupported HIR adapter statement %T", s))
}

func toHIRExpr(e Expr) hir.Expr {
	switch ex := e.(type) {
	case *IdentExpr:
		return &hir.IdentExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Name: ex.Name}
	case *IntExpr:
		return &hir.IntExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *FloatExpr:
		return &hir.FloatExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *BoolExpr:
		return &hir.BoolExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *StringExpr:
		return &hir.StringExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *NilExpr:
		return &hir.NilExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}}
	case *UnaryExpr:
		return &hir.UnaryExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Op: ex.Op, Right: toHIRExpr(ex.Right)}
	case *BinaryExpr:
		return &hir.BinaryExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Left: toHIRExpr(ex.Left), Op: ex.Op, Right: toHIRExpr(ex.Right)}
	case *CallExpr:
		args := mapSlice(ex.Args, toHIRExpr)
		return &hir.CallExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Func: ex.Func, Args: args}
	case *FieldAccessExpr:
		return &hir.FieldAccessExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Object: toHIRExpr(ex.Object), Field: ex.Field}
	case *StructLitExpr:
		fields := mapSlice(ex.Fields, func(f StructLitField) hir.StructLitField {
			return hir.StructLitField{Range: f.Range, Name: f.Name, Value: toHIRExpr(f.Value)}
		})
		return &hir.StructLitExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, TypeName: ex.TypeName, Fields: fields}
	case *MatchExpr:
		arms := mapSlice(ex.Arms, func(arm MatchExprArm) hir.MatchExprArm {
			return hir.MatchExprArm{
				Range:   arm.Range,
				Variant: arm.Variant,
				Guard:   mapOptional(arm.Guard, Expr(nil), toHIRExpr),
				Value:   toHIRExpr(arm.Value),
			}
		})
		return &hir.MatchExpr{NodeInfo: hir.NodeInfo{Range: ex.Span()}, Subject: toHIRExpr(ex.Subject), Arms: arms, Type: ex.Type}
	default:
		panic(fmt.Sprintf("mir: unsupported HIR adapter expression %T", e))
	}
}

func toASTBlock(b *Block) *ast.BlockStmt {
	if b == nil {
		return nil
	}
	stmts := mapSlice(b.Stmts, toASTStmt)
	return &ast.BlockStmt{NodeInfo: ast.NodeInfo{Range: b.Span()}, Stmts: stmts}
}

func toASTStmt(s Stmt) ast.Stmt {
	if info, ok := ValueStmtInfo(s); ok {
		if info.Expr == nil {
			panic(fmt.Sprintf("mir: value stmt binding without expression %T", s))
		}
		return &ast.LetStmt{
			NodeInfo: ast.NodeInfo{Range: s.Span()},
			Name:     info.Name,
			Type:     baztypes.ToAST(info.Type),
			Init:     toASTExpr(info.Expr),
			IsConst:  info.IsConst,
		}
	}
	if info, ok := LinearStmtInfo(s); ok {
		if out, ok := MapLinearStmt[aststmtMarker](info,
			func(target Expr, value Expr) aststmtMarker {
				return aststmtMarker{stmt: &ast.AssignStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Target: toASTExpr(target), Value: toASTExpr(value)}}
			},
			func(value Expr) aststmtMarker {
				return aststmtMarker{stmt: &ast.ExprStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Expr: toASTExpr(value)}}
			},
		); ok {
			return out.stmt
		}
	}
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[aststmtMarker](info,
			func(block *Block) aststmtMarker {
				return aststmtMarker{stmt: toASTBlock(block)}
			},
			func(cond Expr, then, els *Block) aststmtMarker {
				return aststmtMarker{stmt: &ast.IfStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Cond: toASTExpr(cond), Then: toASTBlock(then), Else: toASTBlock(els)}}
			},
			func(cond Expr, body *Block) aststmtMarker {
				return aststmtMarker{stmt: &ast.WhileStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Cond: toASTExpr(cond), Body: toASTBlock(body)}}
			},
			func(subject Expr, armsIn []MatchArm) aststmtMarker {
				arms := mapSlice(armsIn, func(arm MatchArm) ast.MatchArm {
					return ast.MatchArm{
						Range:   arm.Range,
						Variant: arm.Variant,
						Guard:   mapOptional(arm.Guard, Expr(nil), toASTExpr),
						Body:    toASTBlock(arm.Body),
					}
				})
				return aststmtMarker{stmt: &ast.MatchStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Subject: toASTExpr(subject), Arms: arms}}
			},
			func(value Expr) aststmtMarker {
				return aststmtMarker{stmt: &ast.ReturnStmt{NodeInfo: ast.NodeInfo{Range: s.Span()}, Value: mapOptional(value, Expr(nil), toASTExpr)}}
			},
		); ok {
			return out.stmt
		}
	}
	panic(fmt.Sprintf("mir: unsupported AST adapter statement %T", s))
}

func toASTExpr(e Expr) ast.Expr {
	switch ex := e.(type) {
	case *IdentExpr:
		return &ast.IdentExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Name: ex.Name}
	case *IntExpr:
		return &ast.IntExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *FloatExpr:
		return &ast.FloatExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *BoolExpr:
		return &ast.BoolExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *StringExpr:
		return &ast.StringExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Value: ex.Value}
	case *NilExpr:
		return &ast.NilExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}}
	case *UnaryExpr:
		return &ast.UnaryExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Op: ex.Op, Right: toASTExpr(ex.Right)}
	case *BinaryExpr:
		return &ast.BinaryExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Left: toASTExpr(ex.Left), Op: ex.Op, Right: toASTExpr(ex.Right)}
	case *CallExpr:
		args := mapSlice(ex.Args, toASTExpr)
		return &ast.CallExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Callee: ex.Func, Args: args}
	case *FieldAccessExpr:
		return &ast.FieldAccessExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, Object: toASTExpr(ex.Object), Field: ex.Field}
	case *StructLitExpr:
		fields := mapSlice(ex.Fields, func(f StructLitField) ast.StructLitField {
			return ast.StructLitField{Range: f.Range, Name: f.Name, Value: toASTExpr(f.Value)}
		})
		return &ast.StructLitExpr{NodeInfo: ast.NodeInfo{Range: ex.Span()}, TypeName: ex.TypeName, Fields: fields}
	case *MatchExpr:
		arms := mapSlice(ex.Arms, func(arm MatchExprArm) ast.MatchExprArm {
			return ast.MatchExprArm{
				Range:   arm.Range,
				Variant: arm.Variant,
				Guard:   mapOptional(arm.Guard, Expr(nil), toASTExpr),
				Value:   toASTExpr(arm.Value),
			}
		})
		return &ast.MatchExpr{
			NodeInfo:     ast.NodeInfo{Range: ex.Span()},
			Subject:      toASTExpr(ex.Subject),
			Arms:         arms,
			ResolvedType: baztypes.ToAST(ex.Type),
		}
	default:
		panic(fmt.Sprintf("mir: unsupported AST adapter expression %T", e))
	}
}

type hirstmtMarker struct {
	stmt hir.Stmt
}

type aststmtMarker struct {
	stmt ast.Stmt
}

func toASTTypeMap(in map[string]baztypes.Type) map[string]ast.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]ast.Type, len(in))
	for k, v := range in {
		out[k] = baztypes.ToAST(v)
	}
	return out
}
