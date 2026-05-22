package mir

func SelectConstantMatchArm[T matchArmInfo](ctx *typeContext, subject Expr, arms []T) (T, bool) {
	var zero T
	variant, ok := constantEnumVariantName(ctx, subject)
	if !ok {
		return zero, false
	}
	for _, arm := range arms {
		armVariant := MatchArmVariant(arm)
		if armVariant != variant {
			continue
		}
		guard := MatchArmGuard(arm)
		if guard == nil {
			return arm, true
		}
		value, ok := BoolConstValue(guard)
		if !ok {
			return zero, false
		}
		if value {
			return arm, true
		}
	}
	return zero, false
}
