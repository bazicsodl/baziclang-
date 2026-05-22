package mir

import "fmt"

func validateMatchExprArms(ctx *typeContext, ex *MatchExpr) error {
	return validateMatchArms(ctx, ex.Arms,
		"mir: match expression has no arms",
		"mir: match expression arm missing variant",
		"mir: match expression guard has type %s, expected bool",
		func(arm MatchExprArm, variant string, guard Expr) error {
			if err := validateExpr(ctx, arm.Value); err != nil {
				return err
			}
			if valueType, ok := ctx.inferExprType(arm.Value); ok && !typesCompatible(ex.Type, valueType) {
				return fmt.Errorf("mir: match expression arm '%s' has type %s, expected %s", variant, valueType, ex.Type)
			}
			return nil
		},
	)
}
