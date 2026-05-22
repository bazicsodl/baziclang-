package mir

type stmtControlInfo struct {
	Kind    string
	Block   *Block
	Cond    Expr
	Subject Expr
	Value   Expr
	Then    *Block
	Else    *Block
	Body    *Block
	Arms    []MatchArm
}

type stmtControlMeta interface {
	stmtControlInfo() stmtControlInfo
	setStmtControlInfo(stmtControlInfo) bool
}

type stmtReturnsMeta interface {
	stmtAlwaysReturns() bool
}

func (s *Block) stmtControlInfo() stmtControlInfo {
	return stmtControlInfo{Kind: "block", Block: s}
}

func (s *Block) setStmtControlInfo(info stmtControlInfo) bool {
	if info.Kind != "block" {
		return false
	}
	if info.Block == nil {
		s.Stmts = nil
		return true
	}
	s.NodeInfo = info.Block.NodeInfo
	s.Stmts = info.Block.Stmts
	return true
}

func (s *IfStmt) stmtControlInfo() stmtControlInfo {
	return stmtControlInfo{Kind: "if", Cond: s.Cond, Then: s.Then, Else: s.Else}
}

func (s *IfStmt) setStmtControlInfo(info stmtControlInfo) bool {
	if info.Kind != "if" {
		return false
	}
	s.Cond = info.Cond
	s.Then = info.Then
	s.Else = info.Else
	return true
}

func (s *WhileStmt) stmtControlInfo() stmtControlInfo {
	return stmtControlInfo{Kind: "while", Cond: s.Cond, Body: s.Body}
}

func (s *WhileStmt) setStmtControlInfo(info stmtControlInfo) bool {
	if info.Kind != "while" {
		return false
	}
	s.Cond = info.Cond
	s.Body = info.Body
	return true
}

func (s *MatchStmt) stmtControlInfo() stmtControlInfo {
	return stmtControlInfo{Kind: "match", Subject: s.Subject, Arms: s.Arms}
}

func (s *MatchStmt) setStmtControlInfo(info stmtControlInfo) bool {
	if info.Kind != "match" {
		return false
	}
	s.Subject = info.Subject
	s.Arms = info.Arms
	return true
}

func (s *ReturnStmt) stmtControlInfo() stmtControlInfo {
	return stmtControlInfo{Kind: "return", Value: s.Value}
}

func (s *ReturnStmt) setStmtControlInfo(info stmtControlInfo) bool {
	if info.Kind != "return" {
		return false
	}
	s.Value = info.Value
	return true
}

func (s *Block) stmtAlwaysReturns() bool {
	return blockAlwaysReturns(s)
}

func (s *ReturnStmt) stmtAlwaysReturns() bool {
	return true
}

func (s *IfStmt) stmtAlwaysReturns() bool {
	if s.Else == nil {
		return false
	}
	return blockAlwaysReturns(s.Then) && blockAlwaysReturns(s.Else)
}

func (s *MatchStmt) stmtAlwaysReturns() bool {
	if len(s.Arms) == 0 {
		return false
	}
	for _, arm := range s.Arms {
		if !blockAlwaysReturns(arm.Body) {
			return false
		}
	}
	return true
}

func StmtAlwaysReturns(s Stmt) bool {
	st, ok := s.(stmtReturnsMeta)
	return ok && st.stmtAlwaysReturns()
}

func StmtControlInfo(s Stmt) (stmtControlInfo, bool) {
	st, ok := s.(stmtControlMeta)
	if !ok {
		return stmtControlInfo{}, false
	}
	return st.stmtControlInfo(), true
}

func SetStmtControlInfo(s Stmt, info stmtControlInfo) bool {
	st, ok := s.(stmtControlMeta)
	if !ok {
		return false
	}
	return st.setStmtControlInfo(info)
}

func MapStmtControl[T any](
	info stmtControlInfo,
	onBlock func(*Block) T,
	onIf func(Expr, *Block, *Block) T,
	onWhile func(Expr, *Block) T,
	onMatch func(Expr, []MatchArm) T,
	onReturn func(Expr) T,
) (T, bool) {
	switch info.Kind {
	case "block":
		if onBlock == nil {
			var zero T
			return zero, false
		}
		return onBlock(info.Block), true
	case "if":
		if onIf == nil {
			var zero T
			return zero, false
		}
		return onIf(info.Cond, info.Then, info.Else), true
	case "while":
		if onWhile == nil {
			var zero T
			return zero, false
		}
		return onWhile(info.Cond, info.Body), true
	case "match":
		if onMatch == nil {
			var zero T
			return zero, false
		}
		return onMatch(info.Subject, info.Arms), true
	case "return":
		if onReturn == nil {
			var zero T
			return zero, false
		}
		return onReturn(info.Value), true
	default:
		var zero T
		return zero, false
	}
}
