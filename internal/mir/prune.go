package mir

func pruneSyntheticTemps(fn *FuncDecl) {
	if fn == nil || fn.Body == nil {
		return
	}
	pruneBlockTemps(fn.Body)
}

func pruneDeadCFGInstructions(fn *FuncDecl) {
	pruneDeadCFGInstructionsWithAnalysis(fn, AnalyzeCFGLiveness(fn))
}

func pruneDeadCFGInstructionsWithAnalysis(fn *FuncDecl, analysis *CFGLiveness) {
	if fn == nil || fn.CFG == nil {
		return
	}
	deadByBlock := map[string]map[int]bool{}
	if analysis != nil {
		deadByBlock = analysis.DeadByBlock
	}
	for _, block := range fn.CFG.Blocks {
		if block == nil || len(block.Instrs) == 0 {
			continue
		}
		dead := deadByBlock[block.Name]
		if len(dead) == 0 {
			continue
		}
		kept := make([]Stmt, 0, len(block.Instrs))
		for i, stmt := range block.Instrs {
			if dead[i] {
				continue
			}
			kept = append(kept, stmt)
		}
		block.Instrs = kept
	}
}

func pruneBlockTemps(b *Block) map[string]struct{} {
	live := map[string]struct{}{}
	if b == nil {
		return live
	}
	kept := make([]Stmt, 0, len(b.Stmts))
	for i := len(b.Stmts) - 1; i >= 0; i-- {
		stmt := b.Stmts[i]
		if IsValueStmt(stmt) {
			info, _ := ValueStmtInfo(stmt)
			if info.Name == "_" && !ValueStmtMayHaveSideEffects(stmt) {
				continue
			}
			nameLive := containsLive(live, info.Name)
			if nameLive {
				delete(live, info.Name)
			}
			CollectValueStmtUses(live, stmt)
			if info.Name != "_" && !nameLive {
				if ValueStmtMayHaveSideEffects(stmt) {
					SetValueStmtBindingName(stmt, "_")
					kept = append(kept, stmt)
				}
				continue
			}
			kept = append(kept, stmt)
			continue
		}
		if CollectStmtUses(live, stmt) {
			if _, ok := stmt.(*ExprStmt); ok && !StmtMayHaveSideEffects(stmt) {
				continue
			}
			kept = append(kept, stmt)
			continue
		}
		WalkStmtChildBlocks(stmt, func(child *Block) {
			mergeLiveSets(live, pruneBlockTemps(child))
		})
		WalkStmtExprs(stmt, func(expr Expr) {
			collectExprUses(live, expr)
		})
		kept = append(kept, stmt)
	}
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	b.Stmts = kept
	return live
}

func mergeLiveSets(dst, src map[string]struct{}) {
	for name := range src {
		dst[name] = struct{}{}
	}
}

func containsLive(values map[string]struct{}, name string) bool {
	_, ok := values[name]
	return ok
}

func collectAssignTargetUses(live map[string]struct{}, e Expr) {
	switch ex := e.(type) {
	case *IdentExpr:
		live[ex.Name] = struct{}{}
	case *FieldAccessExpr:
		collectExprUses(live, ex.Object)
	}
}

func collectExprUses(live map[string]struct{}, e Expr) {
	WalkExpr(e, func(expr Expr) {
		if ident, ok := expr.(*IdentExpr); ok {
			live[ident.Name] = struct{}{}
		}
	})
}

func exprMayHaveSideEffects(e Expr) bool {
	return AnyExpr(e, func(expr Expr) bool {
		_, ok := expr.(*CallExpr)
		return ok
	})
}
