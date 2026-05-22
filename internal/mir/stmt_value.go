package mir

import baztypes "baziclang/internal/types"

type valueStmtMeta interface {
	Stmt
	valueStmtBinding() (string, baztypes.Type)
	valueStmtIsConst() bool
	setValueStmtBindingName(string)
	valueStmtKind() string
	valueStmtExpr() Expr
}

type boundValueExpr interface {
	Expr
	materializeBoundValueStmt(NodeInfo, string, baztypes.Type, bool) Stmt
}

type valueStmtInfo struct {
	Name    string
	Type    baztypes.Type
	Expr    Expr
	IsConst bool
	Kind    string
}

func (s *LetStmt) valueStmtBinding() (string, baztypes.Type)         { return s.Name, s.Type }
func (s *ConstStmt) valueStmtBinding() (string, baztypes.Type)       { return s.Name, s.Type }
func (s *CopyStmt) valueStmtBinding() (string, baztypes.Type)        { return s.Name, s.Type }
func (s *UnaryOpStmt) valueStmtBinding() (string, baztypes.Type)     { return s.Name, s.Type }
func (s *BinaryOpStmt) valueStmtBinding() (string, baztypes.Type)    { return s.Name, s.Type }
func (s *CallStmt) valueStmtBinding() (string, baztypes.Type)        { return s.Name, s.Type }
func (s *FieldAccessStmt) valueStmtBinding() (string, baztypes.Type) { return s.Name, s.Type }
func (s *StructLitStmt) valueStmtBinding() (string, baztypes.Type)   { return s.Name, s.Type }
func (s *MatchValueStmt) valueStmtBinding() (string, baztypes.Type)  { return s.Name, s.Type }

func (s *LetStmt) valueStmtIsConst() bool         { return s.IsConst }
func (s *ConstStmt) valueStmtIsConst() bool       { return s.IsConst }
func (s *CopyStmt) valueStmtIsConst() bool        { return s.IsConst }
func (s *UnaryOpStmt) valueStmtIsConst() bool     { return s.IsConst }
func (s *BinaryOpStmt) valueStmtIsConst() bool    { return s.IsConst }
func (s *CallStmt) valueStmtIsConst() bool        { return s.IsConst }
func (s *FieldAccessStmt) valueStmtIsConst() bool { return s.IsConst }
func (s *StructLitStmt) valueStmtIsConst() bool   { return s.IsConst }
func (s *MatchValueStmt) valueStmtIsConst() bool  { return s.IsConst }

func (s *LetStmt) setValueStmtBindingName(name string)         { s.Name = name }
func (s *ConstStmt) setValueStmtBindingName(name string)       { s.Name = name }
func (s *CopyStmt) setValueStmtBindingName(name string)        { s.Name = name }
func (s *UnaryOpStmt) setValueStmtBindingName(name string)     { s.Name = name }
func (s *BinaryOpStmt) setValueStmtBindingName(name string)    { s.Name = name }
func (s *CallStmt) setValueStmtBindingName(name string)        { s.Name = name }
func (s *FieldAccessStmt) setValueStmtBindingName(name string) { s.Name = name }
func (s *StructLitStmt) setValueStmtBindingName(name string)   { s.Name = name }
func (s *MatchValueStmt) setValueStmtBindingName(name string)  { s.Name = name }

func (s *LetStmt) valueStmtKind() string         { return "let" }
func (s *ConstStmt) valueStmtKind() string       { return "const" }
func (s *CopyStmt) valueStmtKind() string        { return "copy" }
func (s *UnaryOpStmt) valueStmtKind() string     { return "unary op" }
func (s *BinaryOpStmt) valueStmtKind() string    { return "binary op" }
func (s *CallStmt) valueStmtKind() string        { return "call" }
func (s *FieldAccessStmt) valueStmtKind() string { return "field access" }
func (s *StructLitStmt) valueStmtKind() string   { return "struct literal" }
func (s *MatchValueStmt) valueStmtKind() string  { return "match value" }

func (s *LetStmt) valueStmtExpr() Expr {
	return s.Init
}

func (s *ConstStmt) valueStmtExpr() Expr {
	return s.Value
}

func (s *CopyStmt) valueStmtExpr() Expr {
	return s.Source
}

func (s *UnaryOpStmt) valueStmtExpr() Expr {
	return &UnaryExpr{NodeInfo: s.NodeInfo, Op: s.Op, Right: s.Right}
}

func (s *BinaryOpStmt) valueStmtExpr() Expr {
	return &BinaryExpr{NodeInfo: s.NodeInfo, Left: s.Left, Op: s.Op, Right: s.Right}
}

func (s *CallStmt) valueStmtExpr() Expr {
	return &CallExpr{NodeInfo: s.NodeInfo, Func: s.Func, Args: s.Args}
}

func (s *FieldAccessStmt) valueStmtExpr() Expr {
	return &FieldAccessExpr{NodeInfo: s.NodeInfo, Object: s.Object, Field: s.Field}
}

func (s *StructLitStmt) valueStmtExpr() Expr {
	return &StructLitExpr{NodeInfo: s.NodeInfo, TypeName: s.TypeName, Fields: s.Fields}
}

func (s *MatchValueStmt) valueStmtExpr() Expr {
	return &MatchExpr{NodeInfo: s.NodeInfo, Subject: s.Subject, Arms: s.Arms, Type: s.Type}
}

func (e *IdentExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &CopyStmt{NodeInfo: info, Name: name, Type: typ, Source: e, IsConst: isConst}
}

func (e *IntExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &ConstStmt{NodeInfo: info, Name: name, Type: typ, Value: e, IsConst: isConst}
}

func (e *FloatExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &ConstStmt{NodeInfo: info, Name: name, Type: typ, Value: e, IsConst: isConst}
}

func (e *BoolExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &ConstStmt{NodeInfo: info, Name: name, Type: typ, Value: e, IsConst: isConst}
}

func (e *StringExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &ConstStmt{NodeInfo: info, Name: name, Type: typ, Value: e, IsConst: isConst}
}

func (e *NilExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &ConstStmt{NodeInfo: info, Name: name, Type: typ, Value: e, IsConst: isConst}
}

func (e *UnaryExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &UnaryOpStmt{NodeInfo: info, Name: name, Type: typ, Op: e.Op, Right: e.Right, IsConst: isConst}
}

func (e *BinaryExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &BinaryOpStmt{NodeInfo: info, Name: name, Type: typ, Left: e.Left, Op: e.Op, Right: e.Right, IsConst: isConst}
}

func (e *CallExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &CallStmt{NodeInfo: info, Name: name, Type: typ, Func: e.Func, Args: e.Args, IsConst: isConst}
}

func (e *FieldAccessExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &FieldAccessStmt{NodeInfo: info, Name: name, Type: typ, Object: e.Object, Field: e.Field, IsConst: isConst}
}

func (e *StructLitExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &StructLitStmt{NodeInfo: info, Name: name, Type: typ, TypeName: e.TypeName, Fields: e.Fields, IsConst: isConst}
}

func (e *MatchExpr) materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, isConst bool) Stmt {
	return &MatchValueStmt{NodeInfo: info, Name: name, Type: typ, Subject: e.Subject, Arms: e.Arms, IsConst: isConst}
}

func ValueStmtKind(s Stmt) string {
	if info, ok := ValueStmtInfo(s); ok {
		return info.Kind
	}
	return "value"
}

func ValueStmtExpr(s Stmt) (Expr, bool) {
	if info, ok := ValueStmtInfo(s); ok {
		return info.Expr, true
	}
	return nil, false
}

func materializeBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, expr Expr, isConst bool) (Stmt, bool) {
	if ex, ok := expr.(boundValueExpr); ok {
		return ex.materializeBoundValueStmt(info, name, typ, isConst), true
	}
	return nil, false
}

func buildBoundValueStmt(info NodeInfo, name string, typ baztypes.Type, expr Expr, isConst bool) Stmt {
	if lowered, ok := materializeBoundValueStmt(info, name, typ, expr, isConst); ok {
		return lowered
	}
	return &LetStmt{
		NodeInfo: info,
		Name:     name,
		Type:     typ,
		Init:     expr,
		IsConst:  isConst,
	}
}

func ValueStmtInfo(s Stmt) (valueStmtInfo, bool) {
	st, ok := s.(valueStmtMeta)
	if !ok {
		return valueStmtInfo{}, false
	}
	name, typ := st.valueStmtBinding()
	return valueStmtInfo{
		Name:    name,
		Type:    typ,
		Expr:    st.valueStmtExpr(),
		IsConst: st.valueStmtIsConst(),
		Kind:    st.valueStmtKind(),
	}, true
}

func MapValueStmt[T any](s Stmt, fn func(string, baztypes.Type, Expr, bool, string) T) (T, bool) {
	info, ok := ValueStmtInfo(s)
	if !ok || fn == nil {
		var zero T
		return zero, false
	}
	return fn(info.Name, info.Type, info.Expr, info.IsConst, info.Kind), true
}

func ValueStmtBinding(s Stmt) (string, baztypes.Type, bool) {
	if info, ok := ValueStmtInfo(s); ok {
		return info.Name, info.Type, true
	}
	return "", baztypes.Type{}, false
}

func ValueStmtBindingName(s Stmt) (string, bool) {
	name, _, ok := ValueStmtBinding(s)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

func NamedValueStmtBinding(s Stmt) (string, baztypes.Type, bool) {
	name, typ, ok := ValueStmtBinding(s)
	if !ok || name == "" || name == "_" {
		return "", baztypes.Type{}, false
	}
	return name, typ, true
}

func CollectValueStmtUses(live map[string]struct{}, s Stmt) bool {
	info, ok := ValueStmtInfo(s)
	if !ok {
		return false
	}
	collectExprUses(live, info.Expr)
	return true
}

func ValueStmtMayHaveSideEffects(s Stmt) bool {
	info, ok := ValueStmtInfo(s)
	return ok && exprMayHaveSideEffects(info.Expr)
}

func IsValueStmt(s Stmt) bool {
	_, ok := ValueStmtInfo(s)
	return ok
}

func ValueStmtIsConst(s Stmt) bool {
	if info, ok := ValueStmtInfo(s); ok {
		return info.IsConst
	}
	return false
}

func SetValueStmtBindingName(stmt Stmt, name string) bool {
	if st, ok := stmt.(valueStmtMeta); ok {
		st.setValueStmtBindingName(name)
		return true
	}
	return false
}

func IsSyntheticTempName(name string) bool {
	return isSyntheticTempName(name)
}

func ExprMayHaveSideEffects(e Expr) bool {
	return exprMayHaveSideEffects(e)
}
