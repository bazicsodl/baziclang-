package mir

import (
	"baziclang/internal/ast"
	"baziclang/internal/source"
	baztypes "baziclang/internal/types"
)

func materializeValueOps(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.Body == nil {
		return
	}
	materializeValueOpsBlock(newFuncTypeContext(index, fn), fn.Body)
}

func materializeCFGValueOps(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.CFG == nil {
		return
	}
	ctx := newFuncTypeContext(index, fn)
	for _, block := range fn.CFG.Blocks {
		if block == nil {
			continue
		}
		block.Instrs = materializeValueOpsStmts(ctx, block.Instrs)
	}
}

func materializeValueOpsBlock(ctx *typeContext, b *Block) {
	if b == nil {
		return
	}
	RewriteBlockStmts(b, func(stmt Stmt) []Stmt {
		return []Stmt{materializeValueOpsStmt(ctx, stmt)}
	})
}

func materializeValueOpsStmts(ctx *typeContext, stmts []Stmt) []Stmt {
	return RewriteStmtSlice(stmts, func(stmt Stmt) []Stmt {
		return []Stmt{materializeValueOpsStmt(ctx, stmt)}
	})
}

func materializeValueOpsStmt(ctx *typeContext, stmt Stmt) Stmt {
	if info, ok := StmtControlInfo(stmt); ok {
		MapStmtControl[struct{}](info,
			func(block *Block) struct{} {
				materializeValueOpsBlock(ctx, block)
				return struct{}{}
			},
			func(cond Expr, then, els *Block) struct{} {
				WalkStmtChildBlocks(stmt, func(child *Block) {
					materializeValueOpsBlock(ctx, child)
				})
				return struct{}{}
			},
			func(cond Expr, body *Block) struct{} {
				WalkStmtChildBlocks(stmt, func(child *Block) {
					materializeValueOpsBlock(ctx, child)
				})
				return struct{}{}
			},
			func(subject Expr, arms []MatchArm) struct{} {
				WalkStmtChildBlocks(stmt, func(child *Block) {
					materializeValueOpsBlock(ctx, child)
				})
				return struct{}{}
			},
			func(value Expr) struct{} {
				return struct{}{}
			},
		)
		return stmt
	}
	if st, ok := stmt.(*LetStmt); ok {
		if lowered, ok := materializeBoundValueStmt(st.NodeInfo, st.Name, st.Type, st.Init, st.IsConst); ok {
			return lowered
		}
		return st
	}
	if info, ok := LinearStmtInfo(stmt); ok && !IsValueStmt(stmt) {
		if out, ok := MapLinearStmt[Stmt](info,
			nil,
			func(value Expr) Stmt {
				if lowered, ok := materializeDiscardExprInfo(ctx, stmt.Span(), value); ok {
					return lowered
				}
				return stmt
			},
		); ok {
			return out
		}
	}
	return stmt
}

func materializeDiscardExprInfo(ctx *typeContext, span source.Span, expr Expr) (Stmt, bool) {
	if expr == nil {
		return nil, false
	}
	exprType, ok := inferDiscardExprType(ctx, expr)
	if !ok {
		return nil, false
	}
	return buildBoundValueStmt(NodeInfo{Range: span}, "_", exprType, expr, false), true
}

func inferDiscardExprType(ctx *typeContext, expr Expr) (baztypes.Type, bool) {
	if expr == nil {
		return baztypes.Type{}, false
	}
	if ctx != nil {
		if t, ok := ctx.inferExprType(expr); ok {
			return t, true
		}
	}
	if call, ok := expr.(*CallExpr); ok {
		if call.Func == "print" || call.Func == "println" {
			return baztypes.MustParse(ast.TypeVoid), true
		}
	}
	return baztypes.Type{}, false
}
