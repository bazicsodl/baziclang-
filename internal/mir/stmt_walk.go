package mir

type stmtExprMeta interface {
	walkStmtExprs(func(Expr))
	rewriteStmtExprs(func(Expr) Expr)
}

type terminatorExprMeta interface {
	walkTerminatorExprs(func(Expr))
	rewriteTerminatorExprs(func(Expr) Expr)
}

type stmtChildBlocksMeta interface {
	walkStmtChildBlocks(func(*Block))
}

func (t *ReturnTerminator) walkTerminatorExprs(visit func(Expr)) {
	if t.Value != nil {
		visit(t.Value)
	}
}

func (t *ReturnTerminator) rewriteTerminatorExprs(rewrite func(Expr) Expr) {
	t.Value = rewrite(t.Value)
}

func (t *CondTerminator) walkTerminatorExprs(visit func(Expr)) {
	visit(t.Cond)
}

func (t *CondTerminator) rewriteTerminatorExprs(rewrite func(Expr) Expr) {
	t.Cond = rewrite(t.Cond)
}

func (t *MatchTerminator) walkTerminatorExprs(visit func(Expr)) {
	visit(t.Subject)
	for _, arm := range t.Arms {
		if arm.Guard != nil {
			visit(arm.Guard)
		}
	}
}

func (t *MatchTerminator) rewriteTerminatorExprs(rewrite func(Expr) Expr) {
	t.Subject = rewrite(t.Subject)
	t.Arms = mapSlice(t.Arms, func(arm MatchTerminatorArm) MatchTerminatorArm {
		arm.Guard = rewrite(arm.Guard)
		return arm
	})
}

func (s *Block) walkStmtChildBlocks(visit func(*Block)) {
	visit(s)
}

func (s *IfStmt) walkStmtChildBlocks(visit func(*Block)) {
	visit(s.Then)
	visit(s.Else)
}

func (s *WhileStmt) walkStmtChildBlocks(visit func(*Block)) {
	visit(s.Body)
}

func (s *MatchStmt) walkStmtChildBlocks(visit func(*Block)) {
	for _, arm := range s.Arms {
		visit(arm.Body)
	}
}

func (s *AssignStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Target)
	visit(s.Value)
}

func (s *AssignStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Target = rewrite(s.Target)
	s.Value = rewrite(s.Value)
}

func (s *ExprStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Expr)
}

func (s *ExprStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Expr = rewrite(s.Expr)
}

func (s *LetStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Init)
}

func (s *LetStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Init = rewrite(s.Init)
}

func (s *ConstStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Value)
}

func (s *ConstStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Value = rewrite(s.Value)
}

func (s *CopyStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *CopyStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Source = rewrite(s.Source).(*IdentExpr)
}

func (s *UnaryOpStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *UnaryOpStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Right = rewrite(s.Right)
}

func (s *BinaryOpStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *BinaryOpStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Left = rewrite(s.Left)
	s.Right = rewrite(s.Right)
}

func (s *CallStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *CallStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Args = mapSlice(s.Args, rewrite)
}

func (s *FieldAccessStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *FieldAccessStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Object = rewrite(s.Object)
}

func (s *StructLitStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *StructLitStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Fields = mapSlice(s.Fields, func(field StructLitField) StructLitField {
		field.Value = rewrite(field.Value)
		return field
	})
}

func (s *MatchValueStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.valueStmtExpr())
}

func (s *MatchValueStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Subject = rewrite(s.Subject)
	s.Arms = mapSlice(s.Arms, func(arm MatchExprArm) MatchExprArm {
		arm.Guard = rewrite(arm.Guard)
		arm.Value = rewrite(arm.Value)
		return arm
	})
}

func (s *IfStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Cond)
}

func (s *IfStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Cond = rewrite(s.Cond)
}

func (s *WhileStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Cond)
}

func (s *WhileStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Cond = rewrite(s.Cond)
}

func (s *MatchStmt) walkStmtExprs(visit func(Expr)) {
	visit(s.Subject)
	for _, arm := range s.Arms {
		if arm.Guard != nil {
			visit(arm.Guard)
		}
	}
}

func (s *MatchStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Subject = rewrite(s.Subject)
	s.Arms = mapSlice(s.Arms, func(arm MatchArm) MatchArm {
		arm.Guard = rewrite(arm.Guard)
		return arm
	})
}

func (s *ReturnStmt) walkStmtExprs(visit func(Expr)) {
	if s.Value != nil {
		visit(s.Value)
	}
}

func (s *ReturnStmt) rewriteStmtExprs(rewrite func(Expr) Expr) {
	s.Value = rewrite(s.Value)
}

func RewriteStmtSlice(stmts []Stmt, rewrite func(Stmt) []Stmt) []Stmt {
	if len(stmts) == 0 || rewrite == nil {
		return stmts
	}
	out := make([]Stmt, 0, len(stmts))
	for _, stmt := range stmts {
		out = append(out, rewrite(stmt)...)
	}
	return out
}

func RewriteBlockStmts(block *Block, rewrite func(Stmt) []Stmt) {
	if block == nil {
		return
	}
	block.Stmts = RewriteStmtSlice(block.Stmts, rewrite)
}

func WalkStmtExprs(stmt Stmt, visit func(Expr)) {
	if stmt == nil || visit == nil {
		return
	}
	if st, ok := stmt.(stmtExprMeta); ok {
		st.walkStmtExprs(visit)
	}
}

func WalkStmtChildBlocks(stmt Stmt, visit func(*Block)) {
	if stmt == nil || visit == nil {
		return
	}
	if st, ok := stmt.(stmtChildBlocksMeta); ok {
		st.walkStmtChildBlocks(visit)
	}
}

func WalkTerminatorExprs(term Terminator, visit func(Expr)) {
	if term == nil || visit == nil {
		return
	}
	if t, ok := term.(terminatorExprMeta); ok {
		t.walkTerminatorExprs(visit)
	}
}

func RewriteStmtExprs(stmt Stmt, rewrite func(Expr) Expr) bool {
	if stmt == nil || rewrite == nil {
		return false
	}
	if st, ok := stmt.(stmtExprMeta); ok {
		st.rewriteStmtExprs(rewrite)
		return true
	}
	return false
}

func RewriteTerminatorExprs(term Terminator, rewrite func(Expr) Expr) bool {
	if term == nil || rewrite == nil {
		return false
	}
	if t, ok := term.(terminatorExprMeta); ok {
		t.rewriteTerminatorExprs(rewrite)
		return true
	}
	return false
}
