package mir

func BoolConstValue(e Expr) (bool, bool) {
	b, ok := e.(*BoolExpr)
	if !ok {
		return false, false
	}
	return b.Value, true
}
