package mir

import (
	"fmt"
	"strings"

	"baziclang/internal/source"
	baztypes "baziclang/internal/types"
)

type localNormalizer struct {
	counter int
	used    map[string]struct{}
	scopes  []map[string]string
}

func normalizeFuncLocals(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.Body == nil {
		return
	}
	n := &localNormalizer{
		used:   map[string]struct{}{},
		scopes: []map[string]string{{}},
	}
	for i := range fn.Params {
		name := fn.Params[i].Name
		n.used[name] = struct{}{}
		n.scopes[0][name] = name
	}
	n.normalizeBlockInPlace(fn.Body, false)
	ctx := newFuncTypeContext(index, fn)
	n.anormalizeBlockInPlace(ctx, fn.Body, fn.ReturnType, false)
}

func (n *localNormalizer) pushScope() {
	n.scopes = append(n.scopes, map[string]string{})
}

func (n *localNormalizer) popScope() {
	if len(n.scopes) == 0 {
		return
	}
	n.scopes = n.scopes[:len(n.scopes)-1]
}

func (n *localNormalizer) declare(sourceName string) string {
	if sourceName == "_" {
		return "_"
	}
	current := n.scopes[len(n.scopes)-1]
	canonical := sourceName
	if _, exists := n.used[canonical]; exists {
		canonical = n.freshName(sourceName)
	}
	current[sourceName] = canonical
	n.used[canonical] = struct{}{}
	return canonical
}

func (n *localNormalizer) freshName(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "tmp"
	}
	for {
		n.counter++
		candidate := fmt.Sprintf("%s__mir%d", base, n.counter)
		if _, exists := n.used[candidate]; !exists {
			return candidate
		}
	}
}

func (n *localNormalizer) resolve(name string) string {
	for i := len(n.scopes) - 1; i >= 0; i-- {
		if resolved, ok := n.scopes[i][name]; ok {
			return resolved
		}
	}
	return name
}

func (n *localNormalizer) normalizeBlockInPlace(b *Block, push bool) {
	if b == nil {
		return
	}
	if push {
		n.pushScope()
		defer n.popScope()
	}
	for _, stmt := range b.Stmts {
		n.normalizeStmtInPlace(stmt)
	}
}

func (n *localNormalizer) normalizeStmtInPlace(s Stmt) {
	if st, ok := s.(*LetStmt); ok {
		RewriteStmtExprs(st, n.normalizeExprInPlace)
		st.Name = n.declare(st.Name)
		return
	}
	if info, ok := StmtControlInfo(s); ok {
		RewriteStmtExprs(s, n.normalizeExprInPlace)
		MapStmtControl[struct{}](info,
			func(block *Block) struct{} {
				n.normalizeBlockInPlace(block, true)
				return struct{}{}
			},
			func(cond Expr, then, els *Block) struct{} {
				WalkStmtChildBlocks(s, func(child *Block) {
					n.normalizeBlockInPlace(child, true)
				})
				return struct{}{}
			},
			func(cond Expr, body *Block) struct{} {
				WalkStmtChildBlocks(s, func(child *Block) {
					n.normalizeBlockInPlace(child, true)
				})
				return struct{}{}
			},
			func(subject Expr, arms []MatchArm) struct{} {
				WalkStmtChildBlocks(s, func(child *Block) {
					n.normalizeBlockInPlace(child, true)
				})
				return struct{}{}
			},
			func(value Expr) struct{} {
				return struct{}{}
			},
		)
		return
	}
	if _, ok := LinearStmtInfo(s); ok {
		RewriteStmtExprs(s, n.normalizeExprInPlace)
	}
}

func (n *localNormalizer) normalizeExprInPlace(e Expr) Expr {
	return RewriteExpr(e, func(expr Expr) Expr {
		if ident, ok := expr.(*IdentExpr); ok {
			ident.Name = n.resolve(ident.Name)
		}
		return expr
	})
}

func (n *localNormalizer) anormalizeBlockInPlace(ctx *typeContext, b *Block, retType baztypes.Type, push bool) {
	if b == nil {
		return
	}
	if push {
		n.pushScope()
		defer n.popScope()
	}
	RewriteBlockStmts(b, func(stmt Stmt) []Stmt {
		return n.anormalizeStmtInPlace(ctx, stmt, retType)
	})
}

func (n *localNormalizer) anormalizeStmtInPlace(ctx *typeContext, s Stmt, retType baztypes.Type) []Stmt {
	if st, ok := s.(*LetStmt); ok {
		prefix, init := n.anormalizeExprInPlace(ctx, st.Init)
		st.Init = init
		if st.Name == "_" {
			return append(prefix, st)
		}
		if isAtomicExpr(st.Init) {
			return append(prefix, st)
		}
		tmp, ident := n.newTempBinding(ctx, "let", st.Type, st.Init, st.Span())
		st.Init = ident
		return append(prefix, tmp, st)
	}
	if st, ok := s.(*AssignStmt); ok {
		prefix, target := n.anormalizeAssignTargetInPlace(ctx, st.Target, st.Span())
		st.Target = target
		valuePrefix, value := n.anormalizeExprInPlace(ctx, st.Value)
		st.Value = value
		prefix = append(prefix, valuePrefix...)
		if st.Value != nil && !isAtomicExpr(st.Value) {
			if tmpType, ok := inferNormalizedExprType(ctx, st.Value); ok {
				tmp, ident := n.newTempBinding(ctx, "assign", tmpType, st.Value, st.Span())
				prefix = append(prefix, tmp)
				st.Value = ident
			}
		}
		return append(prefix, st)
	}
	if info, ok := StmtControlInfo(s); ok {
		if out, ok := MapStmtControl[[]Stmt](info,
			func(block *Block) []Stmt {
				n.anormalizeBlockInPlace(ctx, block, retType, true)
				return []Stmt{s}
			},
			func(cond Expr, then, els *Block) []Stmt {
				prefix, cond := n.anormalizeExprInPlace(ctx, cond)
				info.Cond = cond
				if !isAtomicExpr(info.Cond) {
					tmp, ident := n.newTempBinding(ctx, "cond", baztypes.MustParse("bool"), info.Cond, s.Span())
					prefix = append(prefix, tmp)
					info.Cond = ident
				}
				SetStmtControlInfo(s, info)
				WalkStmtChildBlocks(s, func(child *Block) {
					n.anormalizeBlockInPlace(ctx, child, retType, true)
				})
				return append(prefix, s)
			},
			func(cond Expr, body *Block) []Stmt {
				WalkStmtChildBlocks(s, func(child *Block) {
					n.anormalizeBlockInPlace(ctx, child, retType, true)
				})
				return []Stmt{s}
			},
			func(subject Expr, arms []MatchArm) []Stmt {
				prefix, subject := n.anormalizeExprInPlace(ctx, subject)
				info.Subject = subject
				if info.Subject != nil && !isAtomicExpr(info.Subject) {
					if tmpType, ok := inferNormalizedExprType(ctx, info.Subject); ok {
						tmp, ident := n.newTempBinding(ctx, "match", tmpType, info.Subject, s.Span())
						prefix = append(prefix, tmp)
						info.Subject = ident
					}
				}
				SetStmtControlInfo(s, info)
				WalkStmtChildBlocks(s, func(child *Block) {
					n.anormalizeBlockInPlace(ctx, child, retType, true)
				})
				return append(prefix, s)
			},
			func(value Expr) []Stmt {
				prefix, value := n.anormalizeExprInPlace(ctx, value)
				info.Value = value
				if info.Value == nil || isAtomicExpr(info.Value) {
					SetStmtControlInfo(s, info)
					return append(prefix, s)
				}
				tmp, ident := n.newTempBinding(ctx, "ret", retType, info.Value, s.Span())
				info.Value = ident
				SetStmtControlInfo(s, info)
				prefix = append(prefix, tmp)
				return append(prefix, s)
			},
		); ok {
			return out
		}
	}
	if st, ok := s.(*ExprStmt); ok {
		prefix, expr := n.anormalizeExprInPlace(ctx, st.Expr)
		st.Expr = expr
		return append(prefix, st)
	}
	return []Stmt{s}
}

func inferNormalizedExprType(ctx *typeContext, e Expr) (baztypes.Type, bool) {
	if ctx == nil || e == nil {
		return baztypes.Type{}, false
	}
	return ctx.inferExprType(e)
}

func (n *localNormalizer) anormalizeAssignTargetInPlace(ctx *typeContext, target Expr, span source.Span) ([]Stmt, Expr) {
	switch ex := target.(type) {
	case *FieldAccessExpr:
		prefix, object := n.anormalizeExprInPlace(ctx, ex.Object)
		ex.Object = object
		if ex.Object != nil && !isAtomicExpr(ex.Object) {
			if tmpType, ok := inferNormalizedExprType(ctx, ex.Object); ok {
				tmp, ident := n.newTempBinding(ctx, "target", tmpType, ex.Object, span)
				prefix = append(prefix, tmp)
				ex.Object = ident
			}
		}
		return prefix, ex
	default:
		return nil, target
	}
}

func (n *localNormalizer) newTempStmt(prefix string, typ baztypes.Type, init Expr, span source.Span) Stmt {
	name := n.declare(n.freshName(prefix))
	return buildBoundValueStmt(NodeInfo{Range: span}, name, typ, init, true)
}

func (n *localNormalizer) newTempBinding(ctx *typeContext, prefix string, typ baztypes.Type, expr Expr, span source.Span) (Stmt, *IdentExpr) {
	tmp := n.newTempStmt(prefix, typ, expr, span)
	info, _ := ValueStmtInfo(tmp)
	if ctx != nil {
		ctx.locals[info.Name] = info.Type
	}
	return tmp, &IdentExpr{NodeInfo: NodeInfo{Range: expr.Span()}, Name: info.Name}
}

func isAtomicExpr(e Expr) bool {
	switch e.(type) {
	case *IdentExpr, *IntExpr, *FloatExpr, *BoolExpr, *StringExpr, *NilExpr:
		return true
	default:
		return false
	}
}

func (n *localNormalizer) anormalizeExprInPlace(ctx *typeContext, e Expr) ([]Stmt, Expr) {
	if e == nil {
		return nil, nil
	}
	switch ex := e.(type) {
	case *UnaryExpr:
		prefix, right := n.anormalizeExprInPlace(ctx, ex.Right)
		ex.Right = right
		if isAtomicExpr(ex.Right) {
			return prefix, ex
		}
		if rightType, ok := ctx.inferExprType(ex.Right); ok {
			tmp, ident := n.newTempBinding(ctx, "unary", rightType, ex.Right, ex.Right.Span())
			ex.Right = ident
			return append(prefix, tmp), ex
		}
		return prefix, ex
	case *BinaryExpr:
		if ex.Op == "&&" || ex.Op == "||" {
			return nil, ex
		}
		prefix := []Stmt{}
		leftPrefix, left := n.anormalizeExprInPlace(ctx, ex.Left)
		prefix = append(prefix, leftPrefix...)
		ex.Left = left
		if !isAtomicExpr(ex.Left) {
			if leftType, ok := ctx.inferExprType(ex.Left); ok {
				tmp, ident := n.newTempBinding(ctx, "lhs", leftType, ex.Left, ex.Left.Span())
				prefix = append(prefix, tmp)
				ex.Left = ident
			}
		}
		rightPrefix, right := n.anormalizeExprInPlace(ctx, ex.Right)
		prefix = append(prefix, rightPrefix...)
		ex.Right = right
		if !isAtomicExpr(ex.Right) {
			if rightType, ok := ctx.inferExprType(ex.Right); ok {
				tmp, ident := n.newTempBinding(ctx, "rhs", rightType, ex.Right, ex.Right.Span())
				prefix = append(prefix, tmp)
				ex.Right = ident
			}
		}
		return prefix, ex
	case *CallExpr:
		prefix := []Stmt{}
		for i := range ex.Args {
			argPrefix, arg := n.anormalizeExprInPlace(ctx, ex.Args[i])
			prefix = append(prefix, argPrefix...)
			ex.Args[i] = arg
			if isAtomicExpr(ex.Args[i]) {
				continue
			}
			if argType, ok := ctx.inferExprType(ex.Args[i]); ok {
				tmp, ident := n.newTempBinding(ctx, "arg", argType, ex.Args[i], ex.Args[i].Span())
				prefix = append(prefix, tmp)
				ex.Args[i] = ident
			}
		}
		return prefix, ex
	case *FieldAccessExpr:
		prefix, object := n.anormalizeExprInPlace(ctx, ex.Object)
		ex.Object = object
		if isAtomicExpr(ex.Object) {
			return prefix, ex
		}
		if objectType, ok := ctx.inferExprType(ex.Object); ok {
			tmp, ident := n.newTempBinding(ctx, "obj", objectType, ex.Object, ex.Object.Span())
			ex.Object = ident
			return append(prefix, tmp), ex
		}
		return prefix, ex
	case *StructLitExpr:
		prefix := []Stmt{}
		for i := range ex.Fields {
			fieldPrefix, value := n.anormalizeExprInPlace(ctx, ex.Fields[i].Value)
			prefix = append(prefix, fieldPrefix...)
			ex.Fields[i].Value = value
			if isAtomicExpr(ex.Fields[i].Value) {
				continue
			}
			if valueType, ok := ctx.inferExprType(ex.Fields[i].Value); ok {
				tmp, ident := n.newTempBinding(ctx, "field", valueType, ex.Fields[i].Value, ex.Fields[i].Value.Span())
				prefix = append(prefix, tmp)
				ex.Fields[i].Value = ident
			}
		}
		return prefix, ex
	case *MatchExpr:
		prefix, subject := n.anormalizeExprInPlace(ctx, ex.Subject)
		ex.Subject = subject
		if isAtomicExpr(ex.Subject) {
			return prefix, ex
		}
		if subjectType, ok := ctx.inferExprType(ex.Subject); ok {
			tmp, ident := n.newTempBinding(ctx, "subject", subjectType, ex.Subject, ex.Subject.Span())
			ex.Subject = ident
			return append(prefix, tmp), ex
		}
		return prefix, ex
	default:
		return nil, e
	}
}
