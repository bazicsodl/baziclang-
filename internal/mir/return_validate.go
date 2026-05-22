package mir

import (
	"errors"
	"fmt"
)

import "baziclang/internal/ast"
import baztypes "baziclang/internal/types"

func validateReturnValue(ctx *typeContext, value Expr, ret baztypes.Type, voidValueMsg string, missingMsg string, typeMsg string) error {
	if value != nil {
		if ctx != nil {
			if err := validateExpr(ctx, value); err != nil {
				return err
			}
		}
		if baztypes.ToAST(ret) == ast.TypeVoid {
			return errors.New(voidValueMsg)
		}
		if ctx != nil {
			if valueType, ok := ctx.inferExprType(value); ok && !typesCompatible(ret, valueType) {
				return fmt.Errorf(typeMsg, valueType, ret)
			}
		}
		return nil
	}
	if baztypes.ToAST(ret) != ast.TypeVoid {
		return fmt.Errorf(missingMsg, ret)
	}
	return nil
}
