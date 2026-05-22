package mir

func ProgramUsesCall(p *Program, pred func(string) bool) bool {
	if p == nil || pred == nil {
		return false
	}
	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *FuncDecl:
			if blockUsesCall(decl.Body, pred) || cfgUsesCall(decl.CFG, pred) {
				return true
			}
		case *GlobalLetDecl:
			if exprUsesCall(decl.Init, pred) {
				return true
			}
		}
	}
	return false
}

func cfgUsesCall(cfg *CFG, pred func(string) bool) bool {
	if cfg == nil {
		return false
	}
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		for _, instr := range block.Instrs {
			if stmtUsesCall(instr, pred) {
				return true
			}
		}
		if terminatorUsesCall(block.Term, pred) {
			return true
		}
	}
	return false
}

func blockUsesCall(b *Block, pred func(string) bool) bool {
	if b == nil {
		return false
	}
	for _, st := range b.Stmts {
		if stmtUsesCall(st, pred) {
			return true
		}
	}
	return false
}

func stmtUsesCall(s Stmt, pred func(string) bool) bool {
	found := false
	WalkStmtExprs(s, func(expr Expr) {
		if !found && exprUsesCall(expr, pred) {
			found = true
		}
	})
	if found {
		return true
	}
	WalkStmtChildBlocks(s, func(block *Block) {
		if !found && blockUsesCall(block, pred) {
			found = true
		}
	})
	return found
}

func terminatorUsesCall(term Terminator, pred func(string) bool) bool {
	found := false
	WalkTerminatorExprs(term, func(expr Expr) {
		if !found && exprUsesCall(expr, pred) {
			found = true
		}
	})
	return found
}

func exprUsesCall(e Expr, pred func(string) bool) bool {
	return AnyExpr(e, func(expr Expr) bool {
		call, ok := expr.(*CallExpr)
		return ok && pred(call.Func)
	})
}
