package mir

type linearStmtInfo struct {
	Kind   string
	Target Expr
	Value  Expr
}

type linearStmtMeta interface {
	linearStmtInfo() linearStmtInfo
}

func (s *AssignStmt) linearStmtInfo() linearStmtInfo {
	return linearStmtInfo{Kind: "assign", Target: s.Target, Value: s.Value}
}

func (s *ExprStmt) linearStmtInfo() linearStmtInfo {
	return linearStmtInfo{Kind: "expr", Value: s.Expr}
}

func LinearStmtInfo(stmt Stmt) (linearStmtInfo, bool) {
	if info, ok := ValueStmtInfo(stmt); ok {
		return linearStmtInfo{Kind: info.Kind, Value: info.Expr}, true
	}
	if st, ok := stmt.(linearStmtMeta); ok {
		return st.linearStmtInfo(), true
	}
	return linearStmtInfo{}, false
}

func MapLinearStmt[T any](info linearStmtInfo, onAssign func(Expr, Expr) T, onExpr func(Expr) T) (T, bool) {
	if info.Target != nil {
		if onAssign == nil {
			var zero T
			return zero, false
		}
		return onAssign(info.Target, info.Value), true
	}
	if onExpr == nil {
		var zero T
		return zero, false
	}
	return onExpr(info.Value), true
}

func IsLinearStmt(stmt Stmt) bool {
	_, ok := LinearStmtInfo(stmt)
	return ok
}

func CollectStmtUses(live map[string]struct{}, stmt Stmt) bool {
	info, ok := LinearStmtInfo(stmt)
	if !ok {
		return false
	}
	MapLinearStmt[struct{}](info,
		func(target Expr, value Expr) struct{} {
			collectAssignTargetUses(live, target)
			collectExprUses(live, value)
			return struct{}{}
		},
		func(value Expr) struct{} {
			collectExprUses(live, value)
			return struct{}{}
		},
	)
	return true
}

func StmtMayHaveSideEffects(stmt Stmt) bool {
	info, ok := LinearStmtInfo(stmt)
	if !ok {
		return false
	}
	if IsValueStmt(stmt) {
		return ValueStmtMayHaveSideEffects(stmt)
	}
	out, _ := MapLinearStmt[bool](info,
		func(target Expr, value Expr) bool {
			return true
		},
		func(value Expr) bool {
			return exprMayHaveSideEffects(value)
		},
	)
	return out
}
