package mir

type MatchGroup[T matchArmInfo] struct {
	Variant string
	Arms    []T
}

func GroupMatchExprArms(arms []MatchExprArm) []MatchGroup[MatchExprArm] {
	return GroupMatchArms(arms)
}

func GroupMatchTerminatorArms(arms []MatchTerminatorArm) []MatchGroup[MatchTerminatorArm] {
	return GroupMatchArms(arms)
}

func GroupMatchArms[T matchArmInfo](arms []T) []MatchGroup[T] {
	order := []string{}
	by := map[string][]T{}
	for _, arm := range arms {
		variant := MatchArmVariant(arm)
		if _, ok := by[variant]; !ok {
			order = append(order, variant)
		}
		by[variant] = append(by[variant], arm)
	}
	out := make([]MatchGroup[T], 0, len(order))
	for _, v := range order {
		out = append(out, MatchGroup[T]{Variant: v, Arms: by[v]})
	}
	return out
}
