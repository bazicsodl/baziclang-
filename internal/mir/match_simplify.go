package mir

func simplifyMatchArms(arms []MatchArm) []MatchArm {
	return simplifyMatchArmSlice(arms)
}

func simplifyMatchExprArms(arms []MatchExprArm) []MatchExprArm {
	return simplifyMatchArmSlice(arms)
}

func simplifyMatchTerminatorArms(arms []MatchTerminatorArm) []MatchTerminatorArm {
	return simplifyMatchArmSlice(arms)
}

func simplifyMatchArmSlice[T matchArmInfo](arms []T) []T {
	if len(arms) == 0 {
		return arms
	}
	out := make([]T, 0, len(arms))
	unguarded := map[string]bool{}
	for _, arm := range arms {
		variant := MatchArmVariant(arm)
		if unguarded[variant] {
			continue
		}
		guardExpr := MatchArmGuard(arm)
		if guard, ok := BoolConstValue(guardExpr); ok {
			if !guard {
				continue
			}
			SetMatchArmGuard(&arm, nil)
		}
		out = append(out, arm)
		guardExpr = MatchArmGuard(arm)
		if guardExpr == nil {
			unguarded[variant] = true
		}
	}
	return out
}
