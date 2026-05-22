package mir

type CFGLiveness struct {
	Topology    *CFGTopology
	LiveIn      map[string]map[string]struct{}
	LiveOut     map[string]map[string]struct{}
	DeadByBlock map[string]map[int]bool
}

func AnalyzeCFGLivenessFromCFG(cfg *CFG) *CFGLiveness {
	if cfg == nil {
		return nil
	}
	return AnalyzeCFGLiveness(&FuncDecl{CFG: cfg})
}

func DeadSyntheticValueStmtIndexes(block *BasicBlock) map[int]bool {
	dead := map[int]bool{}
	if block == nil {
		return dead
	}
	live := map[string]struct{}{}
	collectTerminatorUses(live, block.Term)
	for i := len(block.Instrs) - 1; i >= 0; i-- {
		stmt := block.Instrs[i]
		if info, ok := ValueStmtInfo(stmt); ok {
			if info.Name == "_" && !ValueStmtMayHaveSideEffects(stmt) {
				dead[i] = true
				continue
			}
			nameLive := containsLive(live, info.Name)
			if info.Name != "_" {
				delete(live, info.Name)
			}
			CollectValueStmtUses(live, stmt)
			if IsSyntheticTempName(info.Name) && !nameLive {
				dead[i] = true
			}
			continue
		}
		if _, ok := LinearStmtInfo(stmt); ok {
			if StmtMayHaveSideEffects(stmt) {
				CollectStmtUses(live, stmt)
			} else {
				dead[i] = true
			}
		}
	}
	return dead
}

func AnalyzeCFGLiveness(fn *FuncDecl) *CFGLiveness {
	if fn == nil || fn.CFG == nil {
		return nil
	}
	topology, err := AnalyzeCFG(fn.CFG)
	if err != nil || topology == nil {
		return nil
	}
	out := &CFGLiveness{
		Topology:    topology,
		LiveIn:      map[string]map[string]struct{}{},
		LiveOut:     map[string]map[string]struct{}{},
		DeadByBlock: map[string]map[int]bool{},
	}
	for _, name := range topology.ReachableBlockNames() {
		out.LiveIn[name] = map[string]struct{}{}
		out.LiveOut[name] = map[string]struct{}{}
	}
	order := topology.ReversePostOrderNames()
	changed := true
	for changed {
		changed = false
		for i := len(order) - 1; i >= 0; i-- {
			name := order[i]
			block := topology.Blocks[name]
			if block == nil {
				continue
			}
			nextOut := map[string]struct{}{}
			for _, succ := range topology.Successors[name] {
				for live := range out.LiveIn[succ] {
					nextOut[live] = struct{}{}
				}
			}
			nextIn := transferCFGLiveness(block, nextOut)
			if !sameLiveSet(out.LiveOut[name], nextOut) {
				out.LiveOut[name] = nextOut
				changed = true
			}
			if !sameLiveSet(out.LiveIn[name], nextIn) {
				out.LiveIn[name] = nextIn
				changed = true
			}
		}
	}
	for _, name := range topology.ReachableBlockNames() {
		block := topology.Blocks[name]
		if block == nil || len(block.Instrs) == 0 {
			continue
		}
		dead := deadCFGInstructionIndexesForBlock(block, out.LiveOut[name])
		if len(dead) != 0 {
			out.DeadByBlock[name] = dead
		}
	}
	return out
}

func DeadCFGInstructionIndexes(fn *FuncDecl) map[string]map[int]bool {
	if analysis := AnalyzeCFGLiveness(fn); analysis != nil {
		return analysis.DeadByBlock
	}
	return map[string]map[int]bool{}
}

func DeadCFGValueNames(fn *FuncDecl) map[string]struct{} {
	analysis := AnalyzeCFGLiveness(fn)
	return deadCFGValueNamesFromAnalysis(fn, analysis)
}

func deadCFGValueNamesFromAnalysis(fn *FuncDecl, analysis *CFGLiveness) map[string]struct{} {
	out := map[string]struct{}{}
	if fn == nil || fn.CFG == nil || analysis == nil {
		return out
	}
	for _, block := range fn.CFG.Blocks {
		if block == nil {
			continue
		}
		dead := analysis.DeadByBlock[block.Name]
		for i, stmt := range block.Instrs {
			if !dead[i] {
				continue
			}
			info, ok := ValueStmtInfo(stmt)
			if !ok || info.Name == "_" {
				continue
			}
			out[info.Name] = struct{}{}
		}
	}
	return out
}

func DeadCFGValueNamesFromAnalysis(fn *FuncDecl, analysis *CFGLiveness) map[string]struct{} {
	return deadCFGValueNamesFromAnalysis(fn, analysis)
}

func DiscardUnusedCFGValueBindings(fn *FuncDecl) {
	discardUnusedCFGValueBindingsWithAnalysis(fn, AnalyzeCFGLiveness(fn))
}

func discardUnusedCFGValueBindingsWithAnalysis(fn *FuncDecl, analysis *CFGLiveness) {
	if fn == nil || fn.CFG == nil || analysis == nil {
		return
	}
	for _, name := range analysis.Topology.ReachableBlockNames() {
		block := analysis.Topology.Blocks[name]
		if block == nil {
			continue
		}
		discardUnusedCFGBlockBindings(block, analysis.LiveOut[name])
	}
}

func transferCFGLiveness(block *BasicBlock, liveOut map[string]struct{}) map[string]struct{} {
	live := cloneLiveSet(liveOut)
	if block == nil {
		return live
	}
	collectTerminatorUses(live, block.Term)
	for i := len(block.Instrs) - 1; i >= 0; i-- {
		transferCFGStmtLiveness(live, block.Instrs[i], false)
	}
	return live
}

func deadCFGInstructionIndexesForBlock(block *BasicBlock, liveOut map[string]struct{}) map[int]bool {
	dead := map[int]bool{}
	if block == nil {
		return dead
	}
	live := cloneLiveSet(liveOut)
	collectTerminatorUses(live, block.Term)
	for i := len(block.Instrs) - 1; i >= 0; i-- {
		if !transferCFGStmtLiveness(live, block.Instrs[i], true) {
			dead[i] = true
		}
	}
	return dead
}

func discardUnusedCFGBlockBindings(block *BasicBlock, liveOut map[string]struct{}) {
	if block == nil {
		return
	}
	live := cloneLiveSet(liveOut)
	collectTerminatorUses(live, block.Term)
	for i := len(block.Instrs) - 1; i >= 0; i-- {
		stmt := block.Instrs[i]
		info, ok := ValueStmtInfo(stmt)
		if !ok {
			transferCFGStmtLiveness(live, stmt, false)
			continue
		}
		if info.Name == "_" {
			transferCFGStmtLiveness(live, stmt, false)
			continue
		}
		nameLive := containsLive(live, info.Name)
		if !nameLive && ValueStmtMayHaveSideEffects(stmt) {
			SetValueStmtBindingName(stmt, "_")
		}
		transferCFGStmtLiveness(live, stmt, false)
	}
}

func transferCFGStmtLiveness(live map[string]struct{}, stmt Stmt, markDead bool) bool {
	if info, ok := ValueStmtInfo(stmt); ok {
		if info.Name == "_" {
			if ValueStmtMayHaveSideEffects(stmt) {
				CollectValueStmtUses(live, stmt)
				return true
			}
			return !markDead
		}
		nameLive := containsLive(live, info.Name)
		delete(live, info.Name)
		if nameLive || ValueStmtMayHaveSideEffects(stmt) {
			CollectValueStmtUses(live, stmt)
			return true
		}
		return !markDead
	}
	if _, ok := LinearStmtInfo(stmt); ok {
		if StmtMayHaveSideEffects(stmt) {
			CollectStmtUses(live, stmt)
			return true
		}
		return !markDead
	}
	return true
}

func cloneLiveSet(in map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for name := range in {
		out[name] = struct{}{}
	}
	return out
}

func sameLiveSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for name := range a {
		if _, ok := b[name]; !ok {
			return false
		}
	}
	return true
}

func collectTerminatorUses(live map[string]struct{}, term Terminator) {
	WalkTerminatorExprs(term, func(expr Expr) {
		collectExprUses(live, expr)
	})
}
