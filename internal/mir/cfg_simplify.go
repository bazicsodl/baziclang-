package mir

func simplifyCFG(ctx *typeContext, cfg *CFG) {
	if cfg == nil || len(cfg.Blocks) == 0 {
		return
	}
	for {
		changed := false
		originalEntry := cfg.Entry
		blockMap := map[string]*BasicBlock{}
		for _, block := range cfg.Blocks {
			if block != nil && block.Name != "" {
				blockMap[block.Name] = block
			}
		}
		for _, block := range cfg.Blocks {
			if block == nil {
				continue
			}
			if simplifyCFGTerminator(ctx, block, blockMap) {
				changed = true
			}
		}
		replacements := map[string]string{}
		for _, block := range cfg.Blocks {
			target, ok := TrivialJumpTarget(block)
			if !ok {
				continue
			}
			if target == "" || target == block.Name {
				continue
			}
			target = resolveJumpChain(blockMap, target, block.Name)
			if target == "" || target == block.Name {
				continue
			}
			replacements[block.Name] = target
		}
		if len(replacements) > 0 {
			for _, block := range cfg.Blocks {
				if block == nil || block.Term == nil {
					continue
				}
				if RewriteTerminatorTargets(block.Term, func(target string) string {
					if replacement, ok := replacements[target]; ok {
						return replacement
					}
					return target
				}) {
					changed = true
				}
			}
			if target, ok := replacements[originalEntry]; ok && canCollapseCFGEntryTo(cfg, originalEntry, target, replacements) {
				cfg.Entry = target
				changed = true
			}
		}
		remove := map[string]struct{}{}
		for _, block := range cfg.Blocks {
			if _, ok := replacements[block.Name]; !ok {
				continue
			}
			if block.Name == originalEntry && cfg.Entry == originalEntry {
				continue
			}
			remove[block.Name] = struct{}{}
			changed = true
		}
		if len(remove) > 0 {
			kept := make([]*BasicBlock, 0, len(cfg.Blocks))
			for _, block := range cfg.Blocks {
				if block == nil {
					continue
				}
				if _, ok := remove[block.Name]; ok {
					continue
				}
				kept = append(kept, block)
			}
			cfg.Blocks = kept
		}
		if mergeLinearCFGBlocks(cfg) {
			changed = true
		}
		if pruneUnreachableCFGBlocks(cfg) {
			changed = true
		}
		if !changed {
			return
		}
	}
}

func canCollapseCFGEntryTo(cfg *CFG, entry string, target string, replacements map[string]string) bool {
	if cfg == nil || entry == "" || target == "" || entry == target {
		return false
	}
	for _, block := range cfg.Blocks {
		if block == nil || block.Name == "" || block.Term == nil {
			continue
		}
		if block.Name == entry {
			continue
		}
		if _, ok := replacements[block.Name]; ok {
			continue
		}
		for _, succ := range TerminatorSuccessors(block.Term) {
			if succ == target {
				return false
			}
		}
	}
	return true
}

func collapseMatchTerminatorTarget(info terminatorInfo) (string, bool) {
	result, ok := MapTerminator[targetResult](info,
		nil,
		nil,
		nil,
		func(subject Expr, defaultTarget string, arms []MatchTerminatorArm) targetResult {
			target, ok := CommonMatchArmTarget(arms, defaultTarget)
			return targetResult{target: target, ok: ok}
		},
	)
	if !ok {
		return "", false
	}
	return result.target, result.ok
}

func simplifyCFGTerminator(ctx *typeContext, block *BasicBlock, blockMap map[string]*BasicBlock) bool {
	if block == nil || block.Term == nil {
		return false
	}
	info, ok := TerminatorInfo(block.Term)
	if !ok {
		return false
	}
	changed := false
	if out, ok := MapTerminator[bool](info,
		nil,
		func(target string) bool {
			return RewriteTerminatorTargets(block.Term, func(target string) string {
				return resolveJumpChain(blockMap, target, block.Name)
			})
		},
		func(cond Expr, thenTarget, elseTarget string) bool {
			changed = RewriteTerminatorTargets(block.Term, func(target string) string {
				return resolveJumpChain(blockMap, target, block.Name)
			})
			info, _ = TerminatorInfo(block.Term)
			if cond, ok := BoolConstValue(info.Cond); ok {
				target := info.Else
				if cond {
					target = info.Then
				}
				block.Term = JumpTerminatorLike(block.Term, target)
				return true
			}
			if info.Then == info.Else {
				block.Instrs = append(block.Instrs, &ExprStmt{NodeInfo: NodeInfo{Range: block.Term.Span()}, Expr: info.Cond})
				block.Term = JumpTerminatorLike(block.Term, info.Then)
				return true
			}
			return changed
		},
		func(subject Expr, defaultTarget string, arms []MatchTerminatorArm) bool {
			changed = RewriteTerminatorTargets(block.Term, func(target string) string {
				return resolveJumpChain(blockMap, target, block.Name)
			})
			info, _ = TerminatorInfo(block.Term)
			simplifiedArms := simplifyMatchTerminatorArms(info.Arms)
			if len(simplifiedArms) != len(info.Arms) {
				info.Arms = simplifiedArms
				changed = true
				SetTerminatorInfo(block.Term, info)
			}
			if target, ok := collapseConstantMatchTerminatorTarget(ctx, info); ok {
				block.Term = JumpTerminatorLike(block.Term, target)
				return true
			}
			if target, ok := collapseMatchTerminatorTarget(info); ok {
				block.Term = JumpTerminatorLike(block.Term, target)
				return true
			}
			return changed
		},
	); ok {
		return out
	}
	return false
}

func collapseConstantMatchTerminatorTarget(ctx *typeContext, info terminatorInfo) (string, bool) {
	if ctx == nil {
		return "", false
	}
	result, ok := MapTerminator[targetResult](info,
		nil,
		nil,
		nil,
		func(subject Expr, defaultTarget string, arms []MatchTerminatorArm) targetResult {
			arm, ok := SelectConstantMatchArm(ctx, subject, arms)
			if ok {
				return targetResult{target: arm.Target, ok: true}
			}
			if defaultTarget != "" {
				return targetResult{target: defaultTarget, ok: true}
			}
			return targetResult{}
		},
	)
	if !ok {
		return "", false
	}
	return result.target, result.ok
}

type targetResult struct {
	target string
	ok     bool
}

func cfgPredecessors(cfg *CFG) map[string][]string {
	out := map[string][]string{}
	if cfg == nil {
		return out
	}
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		if _, ok := out[block.Name]; !ok {
			out[block.Name] = []string{}
		}
		for _, succ := range TerminatorSuccessors(block.Term) {
			out[succ] = append(out[succ], block.Name)
		}
	}
	return out
}

func resolveJumpChain(blocks map[string]*BasicBlock, target string, self string) string {
	seen := map[string]struct{}{}
	for target != "" && target != self {
		if _, ok := seen[target]; ok {
			return target
		}
		seen[target] = struct{}{}
		block := blocks[target]
		next, ok := TrivialJumpTarget(block)
		if !ok {
			return target
		}
		if next == "" || next == target {
			return target
		}
		target = next
	}
	return target
}

func pruneUnreachableCFGBlocks(cfg *CFG) bool {
	topology, err := AnalyzeCFG(cfg)
	if err != nil || topology == nil {
		return false
	}
	kept := make([]*BasicBlock, 0, len(cfg.Blocks))
	changed := false
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		if !topology.Reachable[block.Name] {
			changed = true
			continue
		}
		kept = append(kept, block)
	}
	if changed {
		cfg.Blocks = kept
	}
	return changed
}

func mergeLinearCFGBlocks(cfg *CFG) bool {
	if cfg == nil || len(cfg.Blocks) < 2 {
		return false
	}
	preds := cfgPredecessors(cfg)
	blocks := map[string]*BasicBlock{}
	for _, block := range cfg.Blocks {
		if block != nil && block.Name != "" {
			blocks[block.Name] = block
		}
	}
	remove := map[string]struct{}{}
	changed := false
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		target, ok := JumpTarget(block.Term)
		if !ok {
			continue
		}
		if target == "" || target == block.Name {
			continue
		}
		succ := blocks[target]
		if succ == nil {
			continue
		}
		if len(preds[target]) != 1 || preds[target][0] != block.Name {
			continue
		}
		block.Instrs = append(block.Instrs, succ.Instrs...)
		block.Term = succ.Term
		remove[target] = struct{}{}
		delete(blocks, target)
		changed = true
	}
	if !changed {
		return false
	}
	kept := make([]*BasicBlock, 0, len(cfg.Blocks))
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		if _, ok := remove[block.Name]; ok {
			continue
		}
		kept = append(kept, block)
	}
	cfg.Blocks = kept
	return true
}
