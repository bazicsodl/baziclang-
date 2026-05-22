package mir

type terminatorTargetsMeta interface {
	terminatorSuccessors() []string
	rewriteTerminatorTargets(func(string) string) bool
}

type jumpTargetMeta interface {
	jumpTarget() string
}

type terminatorInfoMeta interface {
	terminatorInfo() terminatorInfo
	setTerminatorInfo(terminatorInfo) bool
}

func (t *ReturnTerminator) terminatorInfo() terminatorInfo {
	return terminatorInfo{Kind: "return", Value: t.Value}
}

func (t *ReturnTerminator) setTerminatorInfo(info terminatorInfo) bool {
	if info.Kind != "return" {
		return false
	}
	t.Value = info.Value
	return true
}

func (t *JumpTerminator) terminatorSuccessors() []string {
	return []string{t.Target}
}

func (t *JumpTerminator) rewriteTerminatorTargets(rewrite func(string) string) bool {
	target := rewrite(t.Target)
	if target == t.Target {
		return false
	}
	t.Target = target
	return true
}

func (t *JumpTerminator) jumpTarget() string {
	return t.Target
}

func (t *JumpTerminator) terminatorInfo() terminatorInfo {
	return terminatorInfo{Kind: "jump", Target: t.Target}
}

func (t *JumpTerminator) setTerminatorInfo(info terminatorInfo) bool {
	if info.Kind != "jump" {
		return false
	}
	t.Target = info.Target
	return true
}

func (t *CondTerminator) terminatorSuccessors() []string {
	return []string{t.Then, t.Else}
}

func (t *CondTerminator) rewriteTerminatorTargets(rewrite func(string) string) bool {
	changed := false
	thenTarget := rewrite(t.Then)
	elseTarget := rewrite(t.Else)
	if thenTarget != t.Then {
		t.Then = thenTarget
		changed = true
	}
	if elseTarget != t.Else {
		t.Else = elseTarget
		changed = true
	}
	return changed
}

func (t *CondTerminator) terminatorInfo() terminatorInfo {
	return terminatorInfo{Kind: "cond", Cond: t.Cond, Then: t.Then, Else: t.Else}
}

func (t *CondTerminator) setTerminatorInfo(info terminatorInfo) bool {
	if info.Kind != "cond" {
		return false
	}
	t.Cond = info.Cond
	t.Then = info.Then
	t.Else = info.Else
	return true
}

func (t *MatchTerminator) terminatorSuccessors() []string {
	out := make([]string, 0, len(t.Arms)+1)
	for _, arm := range t.Arms {
		out = append(out, arm.Target)
	}
	if t.Default != "" {
		out = append(out, t.Default)
	}
	return out
}

func (t *MatchTerminator) rewriteTerminatorTargets(rewrite func(string) string) bool {
	changed := false
	t.Arms = mapSlice(t.Arms, func(arm MatchTerminatorArm) MatchTerminatorArm {
		target := rewrite(arm.Target)
		if target != arm.Target {
			arm.Target = target
			changed = true
		}
		return arm
	})
	if t.Default != "" {
		target := rewrite(t.Default)
		if target != t.Default {
			t.Default = target
			changed = true
		}
	}
	return changed
}

func (t *MatchTerminator) terminatorInfo() terminatorInfo {
	return terminatorInfo{Kind: "match", Subject: t.Subject, Default: t.Default, Arms: t.Arms}
}

func (t *MatchTerminator) setTerminatorInfo(info terminatorInfo) bool {
	if info.Kind != "match" {
		return false
	}
	t.Subject = info.Subject
	t.Default = info.Default
	t.Arms = info.Arms
	return true
}

type terminatorInfo struct {
	Kind    string
	Value   Expr
	Cond    Expr
	Subject Expr
	Target  string
	Then    string
	Else    string
	Default string
	Arms    []MatchTerminatorArm
}

func TerminatorInfo(term Terminator) (terminatorInfo, bool) {
	if t, ok := term.(terminatorInfoMeta); ok {
		return t.terminatorInfo(), true
	}
	return terminatorInfo{}, false
}

func SetTerminatorInfo(term Terminator, info terminatorInfo) bool {
	if t, ok := term.(terminatorInfoMeta); ok {
		return t.setTerminatorInfo(info)
	}
	return false
}

func TerminatorSuccessors(term Terminator) []string {
	if t, ok := term.(terminatorTargetsMeta); ok {
		return t.terminatorSuccessors()
	}
	return nil
}

func JumpTarget(term Terminator) (string, bool) {
	if t, ok := term.(jumpTargetMeta); ok {
		return t.jumpTarget(), true
	}
	return "", false
}

func RewriteTerminatorTargets(term Terminator, rewrite func(string) string) bool {
	if term == nil || rewrite == nil {
		return false
	}
	if t, ok := term.(terminatorTargetsMeta); ok {
		return t.rewriteTerminatorTargets(rewrite)
	}
	return false
}

func JumpTerminatorLike(term Terminator, target string) Terminator {
	return &JumpTerminator{
		NodeInfo: NodeInfo{Range: term.Span()},
		Target:   target,
	}
}

func TrivialJumpTarget(block *BasicBlock) (string, bool) {
	if block == nil || len(block.Instrs) != 0 {
		return "", false
	}
	return JumpTarget(block.Term)
}

func MapTerminator[T any](
	info terminatorInfo,
	onReturn func(Expr) T,
	onJump func(string) T,
	onCond func(Expr, string, string) T,
	onMatch func(Expr, string, []MatchTerminatorArm) T,
) (T, bool) {
	switch info.Kind {
	case "return":
		if onReturn == nil {
			var zero T
			return zero, false
		}
		return onReturn(info.Value), true
	case "jump":
		if onJump == nil {
			var zero T
			return zero, false
		}
		return onJump(info.Target), true
	case "cond":
		if onCond == nil {
			var zero T
			return zero, false
		}
		return onCond(info.Cond, info.Then, info.Else), true
	case "match":
		if onMatch == nil {
			var zero T
			return zero, false
		}
		return onMatch(info.Subject, info.Default, info.Arms), true
	default:
		var zero T
		return zero, false
	}
}
