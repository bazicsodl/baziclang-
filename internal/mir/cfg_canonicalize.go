package mir

import (
	"fmt"

	baztypes "baziclang/internal/types"
)

func canonicalizeCFGOperands(index *typeIndex, fn *FuncDecl) {
	if fn == nil || fn.CFG == nil {
		return
	}
	ctx := newFuncTypeContext(index, fn)
	namer := newCFGTempNamer(fn)
	for _, block := range fn.CFG.Blocks {
		if block == nil || block.Term == nil {
			continue
		}
		info, ok := TerminatorInfo(block.Term)
		if !ok {
			continue
		}
		changed := false
		MapTerminator[struct{}](info,
			func(value Expr) struct{} {
				if value == nil || isAtomicExpr(value) {
					return struct{}{}
				}
				tmpType := fn.ReturnType
				if tmpType.Kind == baztypes.KindInvalid || tmpType.Name == "" {
					if inferred, ok := ctx.inferExprType(value); ok {
						tmpType = inferred
					} else {
						return struct{}{}
					}
				}
				name := namer.fresh("ret")
				node := NodeInfo{Range: block.Term.Span()}
				block.Instrs = append(block.Instrs, buildBoundValueStmt(node, name, tmpType, value, true))
				ctx.locals[name] = tmpType
				info.Value = &IdentExpr{NodeInfo: node, Name: name}
				changed = true
				return struct{}{}
			},
			nil,
			func(cond Expr, thenTarget, elseTarget string) struct{} {
				if isAtomicExpr(cond) {
					return struct{}{}
				}
				name := namer.fresh("cond")
				condType := baztypes.MustParse("bool")
				node := NodeInfo{Range: block.Term.Span()}
				block.Instrs = append(block.Instrs, buildBoundValueStmt(node, name, condType, cond, true))
				ctx.locals[name] = condType
				info.Cond = &IdentExpr{NodeInfo: node, Name: name}
				changed = true
				return struct{}{}
			},
			func(subject Expr, defaultTarget string, arms []MatchTerminatorArm) struct{} {
				if isAtomicExpr(subject) {
					return struct{}{}
				}
				subjectType, ok := ctx.inferExprType(subject)
				if !ok {
					return struct{}{}
				}
				name := namer.fresh("subject")
				node := NodeInfo{Range: block.Term.Span()}
				block.Instrs = append(block.Instrs, buildBoundValueStmt(node, name, subjectType, subject, true))
				ctx.locals[name] = subjectType
				info.Subject = &IdentExpr{NodeInfo: node, Name: name}
				changed = true
				return struct{}{}
			},
		)
		if changed {
			SetTerminatorInfo(block.Term, info)
		}
	}
}

type cfgTempNamer struct {
	used    map[string]struct{}
	counter int
}

func newCFGTempNamer(fn *FuncDecl) *cfgTempNamer {
	out := &cfgTempNamer{used: map[string]struct{}{}}
	if fn == nil {
		return out
	}
	for _, p := range fn.Params {
		out.used[p.Name] = struct{}{}
	}
	for _, block := range fn.CFG.Blocks {
		if block == nil {
			continue
		}
		for _, instr := range block.Instrs {
			if name, ok := ValueStmtBindingName(instr); ok {
				out.used[name] = struct{}{}
			}
		}
	}
	return out
}

func (n *cfgTempNamer) fresh(prefix string) string {
	if prefix == "" {
		prefix = "tmp"
	}
	for {
		n.counter++
		name := fmt.Sprintf("%s__mircfg%d", prefix, n.counter)
		if _, ok := n.used[name]; ok {
			continue
		}
		n.used[name] = struct{}{}
		return name
	}
}
