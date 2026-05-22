package mir

func simplifyControlFlow(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.Body == nil {
		return
	}
	simplifyControlFlowBlock(newFuncTypeContext(index, fn), fn.Body)
}

func simplifyControlFlowBlock(ctx *typeContext, b *Block) {
	if b == nil {
		return
	}
	RewriteBlockStmts(b, func(stmt Stmt) []Stmt {
		if info, ok := StmtControlInfo(stmt); ok {
			if out, ok := MapStmtControl[[]Stmt](info,
				func(block *Block) []Stmt {
					simplifyControlFlowBlock(ctx, block)
					return []Stmt{stmt}
				},
				func(cond Expr, then, els *Block) []Stmt {
					WalkStmtChildBlocks(stmt, func(child *Block) {
						simplifyControlFlowBlock(ctx, child)
					})
					if cond, ok := BoolConstValue(cond); ok {
						if cond {
							return cloneBlockStmts(then)
						}
						return cloneBlockStmts(els)
					}
					return []Stmt{stmt}
				},
				func(cond Expr, body *Block) []Stmt {
					WalkStmtChildBlocks(stmt, func(child *Block) {
						simplifyControlFlowBlock(ctx, child)
					})
					if cond, ok := BoolConstValue(cond); ok && !cond {
						return nil
					}
					return []Stmt{stmt}
				},
				func(subject Expr, arms []MatchArm) []Stmt {
					WalkStmtChildBlocks(stmt, func(child *Block) {
						simplifyControlFlowBlock(ctx, child)
					})
					info.Arms = simplifyMatchArms(arms)
					SetStmtControlInfo(stmt, info)
					if body, ok := simplifyConstantMatchStmt(ctx, subject, info.Arms); ok {
						return cloneBlockStmts(body)
					}
					return []Stmt{stmt}
				},
				nil,
			); ok {
				return out
			}
		}
		return []Stmt{stmt}
	})
}

func cloneBlockStmts(b *Block) []Stmt {
	if b == nil || len(b.Stmts) == 0 {
		return nil
	}
	out := make([]Stmt, 0, len(b.Stmts))
	out = append(out, b.Stmts...)
	return out
}

func simplifyConstantMatchStmt(ctx *typeContext, subject Expr, arms []MatchArm) (*Block, bool) {
	arm, ok := SelectConstantMatchArm(ctx, subject, arms)
	if !ok {
		return nil, false
	}
	return arm.Body, true
}
