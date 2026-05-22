package intrinsics

import "baziclang/internal/ast"

func SafetyPreludeDecls(hasStruct, hasFunc, hasGlobal map[string]bool) []ast.Decl {
	prelude := make([]ast.Decl, 0, 11)
	if !hasGlobal[PreludeAssertFailedGlobal] {
		prelude = append(prelude, &ast.GlobalLetDecl{
			Name: PreludeAssertFailedGlobal,
			Type: ast.TypeBool,
			Init: &ast.BoolExpr{Value: false},
		})
	}
	if !hasGlobal[PreludeAssertMessageGlobal] {
		prelude = append(prelude, &ast.GlobalLetDecl{
			Name: PreludeAssertMessageGlobal,
			Type: ast.TypeString,
			Init: &ast.StringExpr{Value: ""},
		})
	}
	if !hasStruct["Error"] {
		prelude = append(prelude, &ast.StructDecl{
			Name: "Error",
			Fields: []ast.StructField{
				{Name: "message", Type: ast.TypeString},
			},
		})
	}
	if !hasStruct["Option"] {
		prelude = append(prelude, &ast.StructDecl{
			Name:       "Option",
			TypeParams: []string{"T"},
			Fields: []ast.StructField{
				{Name: "is_some", Type: ast.TypeBool},
				{Name: "value", Type: ast.Type("T")},
			},
		})
	}
	if !hasStruct["Result"] {
		prelude = append(prelude, &ast.StructDecl{
			Name:       "Result",
			TypeParams: []string{"T", "E"},
			Fields: []ast.StructField{
				{Name: "is_ok", Type: ast.TypeBool},
				{Name: "value", Type: ast.Type("T")},
				{Name: "err", Type: ast.Type("E")},
			},
		})
	}
	if !hasFunc["some"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "some",
			TypeParams: []string{"T"},
			Params:     []ast.Param{{Name: "value", Type: ast.Type("T")}},
			ReturnType: ast.Type("Option[T]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Option[T]",
					Fields: []ast.StructLitField{
						{Name: "is_some", Value: &ast.BoolExpr{Value: true}},
						{Name: "value", Value: &ast.IdentExpr{Name: "value"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["none"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "none",
			TypeParams: []string{"T"},
			Params:     []ast.Param{{Name: "fallback", Type: ast.Type("T")}},
			ReturnType: ast.Type("Option[T]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Option[T]",
					Fields: []ast.StructLitField{
						{Name: "is_some", Value: &ast.BoolExpr{Value: false}},
						{Name: "value", Value: &ast.IdentExpr{Name: "fallback"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["ok"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "ok",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "value", Type: ast.Type("T")},
				{Name: "fallback_err", Type: ast.Type("E")},
			},
			ReturnType: ast.Type("Result[T,E]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Result[T,E]",
					Fields: []ast.StructLitField{
						{Name: "is_ok", Value: &ast.BoolExpr{Value: true}},
						{Name: "value", Value: &ast.IdentExpr{Name: "value"}},
						{Name: "err", Value: &ast.IdentExpr{Name: "fallback_err"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["err"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "err",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "fallback_value", Type: ast.Type("T")},
				{Name: "err_value", Type: ast.Type("E")},
			},
			ReturnType: ast.Type("Result[T,E]"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.ReturnStmt{Value: &ast.StructLitExpr{
					TypeName: "Result[T,E]",
					Fields: []ast.StructLitField{
						{Name: "is_ok", Value: &ast.BoolExpr{Value: false}},
						{Name: "value", Value: &ast.IdentExpr{Name: "fallback_value"}},
						{Name: "err", Value: &ast.IdentExpr{Name: "err_value"}},
					},
				}},
			}},
		})
	}
	if !hasFunc["assert"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "assert",
			Params:     []ast.Param{{Name: "cond", Type: ast.TypeBool}},
			ReturnType: ast.TypeVoid,
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.UnaryExpr{Op: "!", Right: &ast.IdentExpr{Name: "cond"}},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: PreludeAssertFailedGlobal}, Value: &ast.BoolExpr{Value: true}},
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: PreludeAssertMessageGlobal}, Value: &ast.StringExpr{Value: "assertion failed"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["assert_msg"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name: "assert_msg",
			Params: []ast.Param{
				{Name: "cond", Type: ast.TypeBool},
				{Name: "msg", Type: ast.TypeString},
			},
			ReturnType: ast.TypeVoid,
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.UnaryExpr{Op: "!", Right: &ast.IdentExpr{Name: "cond"}},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: PreludeAssertFailedGlobal}, Value: &ast.BoolExpr{Value: true}},
						&ast.AssignStmt{Target: &ast.IdentExpr{Name: PreludeAssertMessageGlobal}, Value: &ast.IdentExpr{Name: "msg"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["unwrap_or"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "unwrap_or",
			TypeParams: []string{"T"},
			Params: []ast.Param{
				{Name: "opt", Type: ast.Type("Option[T]")},
				{Name: "fallback", Type: ast.Type("T")},
			},
			ReturnType: ast.Type("T"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "opt"}, Field: "is_some"},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "opt"}, Field: "value"}},
					}},
					Else: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.IdentExpr{Name: "fallback"}},
					}},
				},
			}},
		})
	}
	if !hasFunc["result_or"] {
		prelude = append(prelude, &ast.FuncDecl{
			Name:       "result_or",
			TypeParams: []string{"T", "E"},
			Params: []ast.Param{
				{Name: "res", Type: ast.Type("Result[T,E]")},
				{Name: "fallback", Type: ast.Type("T")},
			},
			ReturnType: ast.Type("T"),
			Body: &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "res"}, Field: "is_ok"},
					Then: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.FieldAccessExpr{Object: &ast.IdentExpr{Name: "res"}, Field: "value"}},
					}},
					Else: &ast.BlockStmt{Stmts: []ast.Stmt{
						&ast.ReturnStmt{Value: &ast.IdentExpr{Name: "fallback"}},
					}},
				},
			}},
		})
	}
	return prelude
}
