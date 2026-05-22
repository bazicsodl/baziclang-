package mir

import baztypes "baziclang/internal/types"

type CFGTopology struct {
	Blocks       map[string]*BasicBlock
	Order        []string
	Successors   map[string][]string
	Predecessors map[string][]string
	Reachable    map[string]bool
	ReversePost  []string
}

func HasUniqueCFGBindings(fn *FuncDecl) bool {
	if fn == nil || fn.CFG == nil {
		return false
	}
	seen := map[string]struct{}{}
	for _, p := range fn.Params {
		seen[p.Name] = struct{}{}
	}
	for _, block := range fn.CFG.Blocks {
		for _, instr := range block.Instrs {
			name, _, ok := NamedValueStmtBinding(instr)
			if !ok {
				continue
			}
			if _, exists := seen[name]; exists {
				return false
			}
			seen[name] = struct{}{}
		}
	}
	return true
}

func CollectCFGLetTypes(fn *FuncDecl) map[string]baztypes.Type {
	out := map[string]baztypes.Type{}
	if fn == nil || fn.CFG == nil {
		return out
	}
	for _, block := range fn.CFG.Blocks {
		for _, instr := range block.Instrs {
			if name, typ, ok := NamedValueStmtBinding(instr); ok {
				out[name] = typ
			}
		}
	}
	return out
}

func AnalyzeCFG(cfg *CFG) (*CFGTopology, error) {
	if cfg == nil {
		return nil, nil
	}
	blocks := map[string]*BasicBlock{}
	for _, block := range cfg.Blocks {
		if block == nil || block.Name == "" {
			continue
		}
		blocks[block.Name] = block
	}
	top := &CFGTopology{
		Blocks:       blocks,
		Order:        []string{},
		Successors:   map[string][]string{},
		Predecessors: map[string][]string{},
		Reachable:    map[string]bool{},
	}
	for _, block := range cfg.Blocks {
		if block == nil || block.Name == "" || block.Term == nil {
			continue
		}
		top.Order = append(top.Order, block.Name)
		succs := dedupeStringsPreserveOrder(TerminatorSuccessors(block.Term))
		top.Successors[block.Name] = succs
		for _, succ := range succs {
			top.Predecessors[succ] = append(top.Predecessors[succ], block.Name)
		}
		if _, ok := top.Predecessors[block.Name]; !ok {
			top.Predecessors[block.Name] = []string{}
		}
	}
	if cfg.Entry == "" {
		return top, nil
	}
	post := []string{}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		if visited[name] || visiting[name] {
			return
		}
		visiting[name] = true
		for _, next := range top.Successors[name] {
			visit(next)
		}
		visiting[name] = false
		visited[name] = true
		post = append(post, name)
	}
	visit(cfg.Entry)
	queue := []string{cfg.Entry}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if top.Reachable[name] {
			continue
		}
		top.Reachable[name] = true
		for _, next := range top.Successors[name] {
			if !top.Reachable[next] {
				queue = append(queue, next)
			}
		}
	}
	top.ReversePost = make([]string, 0, len(post))
	for i := len(post) - 1; i >= 0; i-- {
		if top.Reachable[post[i]] {
			top.ReversePost = append(top.ReversePost, post[i])
		}
	}
	return top, nil
}

func (t *CFGTopology) PredecessorCount(name string) int {
	if t == nil {
		return 0
	}
	return len(t.Predecessors[name])
}

func (t *CFGTopology) IsJoinBlock(name string) bool {
	return t.PredecessorCount(name) > 1
}

func (t *CFGTopology) ReachableBlockNames() []string {
	if t == nil {
		return nil
	}
	out := make([]string, 0, len(t.Order))
	for _, name := range t.Order {
		if t.Reachable[name] {
			out = append(out, name)
		}
	}
	return out
}

func (t *CFGTopology) ReversePostOrderNames() []string {
	if t == nil {
		return nil
	}
	return append([]string(nil), t.ReversePost...)
}

func dedupeStringsPreserveOrder(in []string) []string {
	if len(in) < 2 {
		return append([]string(nil), in...)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
