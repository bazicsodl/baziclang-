package mir

func CommonMatchArmTarget[T matchArmTargetInfo](arms []T, defaultTarget string) (string, bool) {
	target := defaultTarget
	if len(arms) > 0 {
		target = MatchArmTarget(arms[0])
	}
	if target == "" {
		return "", false
	}
	for _, arm := range arms {
		armTarget := MatchArmTarget(arm)
		if armTarget != target {
			return "", false
		}
	}
	if defaultTarget != "" && defaultTarget != target {
		return "", false
	}
	if len(arms) == 0 && defaultTarget == "" {
		return "", false
	}
	return target, true
}
