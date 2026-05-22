package mir

import (
	"fmt"

	"baziclang/internal/ast"
)

func validateBoolExpr(ctx *typeContext, e Expr, msg string) error {
	if err := validateExpr(ctx, e); err != nil {
		return err
	}
	if boolType, ok := ctx.inferExprType(e); ok && boolType.String() != string(ast.TypeBool) {
		return fmt.Errorf(msg, boolType)
	}
	return nil
}
