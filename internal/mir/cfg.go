package mir

import (
	"fmt"

	"baziclang/internal/source"
	baztypes "baziclang/internal/types"
)

type cfgBuilder struct {
	order  []*BasicBlock
	byName map[string]*BasicBlock
	nextID int
}

type cfgBranchResult struct {
	end        string
	terminated bool
}

func lowerCFG(body *Block) (*CFG, error) {
	b := &cfgBuilder{byName: map[string]*BasicBlock{}}
	entry := b.newBlock("entry", body.Span())
	endName, terminated, err := b.lowerBlockInto(entry.Name, body)
	if err != nil {
		return nil, err
	}
	if !terminated {
		b.byName[endName].Term = &ReturnTerminator{
			NodeInfo: NodeInfo{Range: body.Span()},
		}
	}
	return &CFG{Entry: entry.Name, Blocks: b.order}, nil
}

func validateCFG(ctx *typeContext, cfg *CFG, ret baztypes.Type) error {
	if cfg == nil {
		return fmt.Errorf("mir: missing cfg")
	}
	if cfg.Entry == "" {
		return fmt.Errorf("mir: cfg missing entry block")
	}
	if len(cfg.Blocks) == 0 {
		return fmt.Errorf("mir: cfg has no blocks")
	}
	blocks := map[string]*BasicBlock{}
	for _, block := range cfg.Blocks {
		if block == nil {
			return fmt.Errorf("mir: cfg contains nil block")
		}
		if block.Name == "" {
			return fmt.Errorf("mir: cfg block missing name")
		}
		if _, exists := blocks[block.Name]; exists {
			return fmt.Errorf("mir: duplicate cfg block '%s'", block.Name)
		}
		blocks[block.Name] = block
		for _, instr := range block.Instrs {
			if !IsLinearStmt(instr) {
				return fmt.Errorf("mir: invalid cfg instruction %T in block '%s'", instr, block.Name)
			}
		}
		if block.Term == nil {
			return fmt.Errorf("mir: block '%s' missing terminator", block.Name)
		}
	}
	if _, ok := blocks[cfg.Entry]; !ok {
		return fmt.Errorf("mir: cfg entry block '%s' not found", cfg.Entry)
	}
	topology, _ := AnalyzeCFG(cfg)
	if topology.PredecessorCount(cfg.Entry) != 0 {
		return fmt.Errorf("mir: cfg entry block '%s' has incoming edges", cfg.Entry)
	}
	for _, block := range cfg.Blocks {
		if err := validateTerminator(ctx, block.Term, ret, blocks); err != nil {
			return err
		}
	}
	for _, block := range cfg.Blocks {
		if !topology.Reachable[block.Name] {
			return fmt.Errorf("mir: unreachable cfg block '%s'", block.Name)
		}
	}
	return nil
}

func validateTerminator(ctx *typeContext, term Terminator, ret baztypes.Type, blocks map[string]*BasicBlock) error {
	info, ok := TerminatorInfo(term)
	if !ok {
		return fmt.Errorf("mir: unsupported terminator %T", term)
	}
	if out, ok := MapTerminator[error](info,
		func(value Expr) error {
			return validateReturnValue(ctx, value, ret,
				"mir: void function cfg cannot return a value",
				"mir: non-void function cfg must return a value of type %s",
				"mir: cfg return value has type %s, expected %s",
			)
		},
		func(target string) error {
			if target == "" {
				return fmt.Errorf("mir: jump terminator missing target")
			}
			if _, ok := blocks[target]; !ok {
				return fmt.Errorf("mir: jump target '%s' not found", target)
			}
			return nil
		},
		func(cond Expr, thenTarget, elseTarget string) error {
			if err := validateBoolExpr(ctx, cond, "mir: cfg condition has type %s, expected bool"); err != nil {
				return err
			}
			if thenTarget == "" {
				return fmt.Errorf("mir: conditional terminator missing then-target")
			}
			if elseTarget == "" {
				return fmt.Errorf("mir: conditional terminator missing else-target")
			}
			if _, ok := blocks[thenTarget]; !ok {
				return fmt.Errorf("mir: conditional then-target '%s' not found", thenTarget)
			}
			if _, ok := blocks[elseTarget]; !ok {
				return fmt.Errorf("mir: conditional else-target '%s' not found", elseTarget)
			}
			if thenTarget == elseTarget {
				return fmt.Errorf("mir: conditional terminator branches to the same target '%s'", thenTarget)
			}
			return nil
		},
		func(subject Expr, defaultTarget string, arms []MatchTerminatorArm) error {
			return validateMatchTerminator(ctx, terminatorInfo{
				Kind:    "match",
				Subject: subject,
				Default: defaultTarget,
				Arms:    arms,
			}, blocks)
		},
	); ok {
		return out
	}
	return fmt.Errorf("mir: unsupported terminator %T", term)
}

func validateMatchTerminator(ctx *typeContext, info terminatorInfo, blocks map[string]*BasicBlock) error {
	if err := validateExpr(ctx, info.Subject); err != nil {
		return err
	}
	if info.Default != "" {
		if _, ok := blocks[info.Default]; !ok {
			return fmt.Errorf("mir: match default target '%s' not found", info.Default)
		}
	}
	seenUnguarded := map[string]bool{}
	return validateMatchArms(ctx, info.Arms,
		"mir: match terminator has no arms",
		"mir: match terminator arm missing variant",
		"mir: cfg match guard has type %s, expected bool",
		func(arm MatchTerminatorArm, variant string, guard Expr) error {
			if arm.Target == "" {
				return fmt.Errorf("mir: match terminator arm '%s' missing target", variant)
			}
			if guard != nil {
				if seenUnguarded[variant] {
					return fmt.Errorf("mir: guarded match arm for variant '%s' appears after unguarded arm", variant)
				}
			} else {
				if seenUnguarded[variant] {
					return fmt.Errorf("mir: duplicate unguarded match arm for variant '%s'", variant)
				}
				seenUnguarded[variant] = true
			}
			if _, ok := blocks[arm.Target]; !ok {
				return fmt.Errorf("mir: match arm target '%s' not found", arm.Target)
			}
			return nil
		},
	)
}

func (b *cfgBuilder) lowerBlockInto(current string, block *Block) (string, bool, error) {
	if block == nil {
		return current, false, nil
	}
	for _, stmt := range block.Stmts {
		if IsLinearStmt(stmt) {
			b.byName[current].Instrs = append(b.byName[current].Instrs, stmt)
			continue
		}
		if info, ok := StmtControlInfo(stmt); ok {
			next, terminated, err := b.lowerControlStmtInto(current, stmt, info)
			if err != nil {
				return "", false, err
			}
			current = next
			if terminated {
				return current, true, nil
			}
			continue
		}
		return "", false, fmt.Errorf("mir: unsupported cfg lowering statement %T", stmt)
	}
	return current, false, nil
}

func (b *cfgBuilder) lowerControlStmtInto(current string, stmt Stmt, info stmtControlInfo) (string, bool, error) {
	if out, ok := MapStmtControl[cfgLowerResult](info,
		func(block *Block) cfgLowerResult {
			next, terminated, err := b.lowerBlockInto(current, block)
			return cfgLowerResult{next: next, terminated: terminated, err: err}
		},
		func(cond Expr, then, els *Block) cfgLowerResult {
			thenBlock := b.newBlock("if_then", stmt.Span())
			elseBlock := b.newBlock("if_else", stmt.Span())
			thenEnd, thenTerminated, err := b.lowerBlockInto(thenBlock.Name, then)
			if err != nil {
				return cfgLowerResult{err: err}
			}
			elseEnd, elseTerminated, err := b.lowerBlockInto(elseBlock.Name, els)
			if err != nil {
				return cfgLowerResult{err: err}
			}
			b.byName[current].Term = &CondTerminator{
				NodeInfo: NodeInfo{Range: stmt.Span()},
				Cond:     cond,
				Then:     thenBlock.Name,
				Else:     elseBlock.Name,
			}
			if els == nil {
				elseTerminated = false
				elseEnd = elseBlock.Name
			}
			next, terminated := b.joinCFGBranches("if_join", stmt.Span(),
				cfgBranchResult{end: thenEnd, terminated: thenTerminated},
				cfgBranchResult{end: elseEnd, terminated: elseTerminated},
			)
			if terminated {
				return cfgLowerResult{next: current, terminated: true}
			}
			return cfgLowerResult{next: next}
		},
		func(cond Expr, body *Block) cfgLowerResult {
			condBlock := b.newBlock("while_cond", stmt.Span())
			bodyBlock := b.newBlock("while_body", stmt.Span())
			exitBlock := b.newBlock("while_exit", stmt.Span())
			b.byName[current].Term = &JumpTerminator{NodeInfo: NodeInfo{Range: stmt.Span()}, Target: condBlock.Name}
			b.byName[condBlock.Name].Term = &CondTerminator{
				NodeInfo: NodeInfo{Range: stmt.Span()},
				Cond:     cond,
				Then:     bodyBlock.Name,
				Else:     exitBlock.Name,
			}
			bodyEnd, bodyTerminated, err := b.lowerBlockInto(bodyBlock.Name, body)
			if err != nil {
				return cfgLowerResult{err: err}
			}
			if !bodyTerminated {
				b.byName[bodyEnd].Term = &JumpTerminator{NodeInfo: NodeInfo{Range: stmt.Span()}, Target: condBlock.Name}
			}
			return cfgLowerResult{next: exitBlock.Name}
		},
		func(subject Expr, armsIn []MatchArm) cfgLowerResult {
			arms := make([]MatchTerminatorArm, 0, len(armsIn))
			branches := make([]cfgBranchResult, 0, len(armsIn))
			for _, arm := range armsIn {
				armBlock := b.newBlock("match_arm", arm.Range)
				armEnd, armTerminated, err := b.lowerBlockInto(armBlock.Name, arm.Body)
				if err != nil {
					return cfgLowerResult{err: err}
				}
				branches = append(branches, cfgBranchResult{end: armEnd, terminated: armTerminated})
				arms = append(arms, MatchTerminatorArm{
					Range:   arm.Range,
					Variant: arm.Variant,
					Guard:   arm.Guard,
					Target:  armBlock.Name,
				})
			}
			b.byName[current].Term = &MatchTerminator{
				NodeInfo: NodeInfo{Range: stmt.Span()},
				Subject:  subject,
				Arms:     arms,
			}
			next, terminated := b.joinCFGBranches("match_join", stmt.Span(), branches...)
			if terminated {
				return cfgLowerResult{next: current, terminated: true}
			}
			return cfgLowerResult{next: next}
		},
		func(value Expr) cfgLowerResult {
			b.byName[current].Term = &ReturnTerminator{
				NodeInfo: NodeInfo{Range: stmt.Span()},
				Value:    value,
			}
			return cfgLowerResult{next: current, terminated: true}
		},
	); ok {
		return out.next, out.terminated, out.err
	}
	return "", false, fmt.Errorf("mir: unsupported cfg lowering statement %T", stmt)
}

type cfgLowerResult struct {
	next       string
	terminated bool
	err        error
}

func (b *cfgBuilder) joinCFGBranches(prefix string, span source.Span, branches ...cfgBranchResult) (string, bool) {
	if len(branches) == 0 {
		return "", true
	}
	allTerminated := true
	for _, branch := range branches {
		if !branch.terminated {
			allTerminated = false
			break
		}
	}
	if allTerminated {
		return "", true
	}
	joinBlock := b.newBlock(prefix, span)
	for _, branch := range branches {
		if branch.terminated || branch.end == "" {
			continue
		}
		b.byName[branch.end].Term = &JumpTerminator{NodeInfo: NodeInfo{Range: span}, Target: joinBlock.Name}
	}
	return joinBlock.Name, false
}

func (b *cfgBuilder) newBlock(prefix string, span source.Span) *BasicBlock {
	name := fmt.Sprintf("%s_%d", prefix, b.nextID)
	b.nextID++
	block := &BasicBlock{
		NodeInfo: NodeInfo{Range: span},
		Name:     name,
		Instrs:   []Stmt{},
	}
	b.order = append(b.order, block)
	b.byName[name] = block
	return block
}
