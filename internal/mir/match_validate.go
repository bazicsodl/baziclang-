package mir

import "errors"

func validateMatchArms[T matchArmInfo](ctx *typeContext, arms []T, emptyMsg string, missingVariantMsg string, guardTypeMsg string, validateArm func(T, string, Expr) error) error {
	if len(arms) == 0 {
		return errors.New(emptyMsg)
	}
	for _, arm := range arms {
		variant := MatchArmVariant(arm)
		if variant == "" {
			return errors.New(missingVariantMsg)
		}
		guard := MatchArmGuard(arm)
		if guard != nil {
			if err := validateBoolExpr(ctx, guard, guardTypeMsg); err != nil {
				return err
			}
		}
		if validateArm != nil {
			if err := validateArm(arm, variant, guard); err != nil {
				return err
			}
		}
	}
	return nil
}
