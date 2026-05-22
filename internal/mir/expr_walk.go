package mir

type exprChildrenWalker interface {
	walkExprChildren(func(Expr) bool) bool
}

type exprChildrenRewriter interface {
	rewriteExprChildren(func(Expr) Expr)
}

func (e *UnaryExpr) walkExprChildren(visit func(Expr) bool) bool {
	return walkExpr(e.Right, visit)
}

func (e *UnaryExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Right = RewriteExpr(e.Right, rewrite)
}

func (e *BinaryExpr) walkExprChildren(visit func(Expr) bool) bool {
	return walkExpr(e.Left, visit) && walkExpr(e.Right, visit)
}

func (e *BinaryExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Left = RewriteExpr(e.Left, rewrite)
	e.Right = RewriteExpr(e.Right, rewrite)
}

func (e *CallExpr) walkExprChildren(visit func(Expr) bool) bool {
	for _, arg := range e.Args {
		if !walkExpr(arg, visit) {
			return false
		}
	}
	return true
}

func (e *CallExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Args = mapSlice(e.Args, func(arg Expr) Expr {
		return RewriteExpr(arg, rewrite)
	})
}

func (e *FieldAccessExpr) walkExprChildren(visit func(Expr) bool) bool {
	return walkExpr(e.Object, visit)
}

func (e *FieldAccessExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Object = RewriteExpr(e.Object, rewrite)
}

func (e *StructLitExpr) walkExprChildren(visit func(Expr) bool) bool {
	for _, field := range e.Fields {
		if !walkExpr(field.Value, visit) {
			return false
		}
	}
	return true
}

func (e *StructLitExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Fields = mapSlice(e.Fields, func(field StructLitField) StructLitField {
		field.Value = RewriteExpr(field.Value, rewrite)
		return field
	})
}

func (e *MatchExpr) walkExprChildren(visit func(Expr) bool) bool {
	if !walkExpr(e.Subject, visit) {
		return false
	}
	for _, arm := range e.Arms {
		if !walkExpr(arm.Guard, visit) {
			return false
		}
		if !walkExpr(arm.Value, visit) {
			return false
		}
	}
	return true
}

func (e *MatchExpr) rewriteExprChildren(rewrite func(Expr) Expr) {
	e.Subject = RewriteExpr(e.Subject, rewrite)
	e.Arms = mapSlice(e.Arms, func(arm MatchExprArm) MatchExprArm {
		arm.Guard = RewriteExpr(arm.Guard, rewrite)
		arm.Value = RewriteExpr(arm.Value, rewrite)
		return arm
	})
}

func WalkExpr(e Expr, visit func(Expr)) {
	if visit == nil {
		return
	}
	walkExpr(e, func(expr Expr) bool {
		visit(expr)
		return true
	})
}

func AnyExpr(e Expr, pred func(Expr) bool) bool {
	if pred == nil {
		return false
	}
	found := false
	walkExpr(e, func(expr Expr) bool {
		if pred(expr) {
			found = true
			return false
		}
		return true
	})
	return found
}

func RewriteExpr(e Expr, rewrite func(Expr) Expr) Expr {
	if e == nil || rewrite == nil {
		return e
	}
	if ex, ok := e.(exprChildrenRewriter); ok {
		ex.rewriteExprChildren(rewrite)
	}
	return rewrite(e)
}

func walkExpr(e Expr, visit func(Expr) bool) bool {
	if e == nil {
		return true
	}
	if !visit(e) {
		return false
	}
	if ex, ok := e.(exprChildrenWalker); ok {
		return ex.walkExprChildren(visit)
	}
	return true
}
