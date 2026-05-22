package codegen

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
	"baziclang/internal/mir"
	"baziclang/internal/parser"
	"baziclang/internal/sema"
	baztypes "baziclang/internal/types"
)

func TestGenerateGoLowersThroughMIRForMatchAndCalls(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn User_label(name: string): string { return name; }

fn main(): void {
    let role: Role = Admin;
    let label = match role {
        Guest: "guest",
        Admin: "admin",
    };
    println(User_label(label));
}`
	prog := parseAndCheckCodegen(t, src)
	out, err := GenerateGo(prog)
	if err != nil {
		t.Fatalf("generate go failed: %v", err)
	}
	if !strings.Contains(out, "func() string {") && !strings.Contains(out, "var label string = \"admin\"") {
		t.Fatalf("expected lowered or fully folded match result in generated go:\n%s", out)
	}
	if !strings.Contains(out, "switch __bazic_block {") &&
		!strings.Contains(out, "var let__mir") &&
		!strings.Contains(out, "var label string = func() string {") &&
		!strings.Contains(out, "var label string = \"admin\"") {
		t.Fatalf("expected mir-backed or stronger folded codegen shape in generated go:\n%s", out)
	}
	if !strings.Contains(out, "func User_label(") || !strings.Contains(out, "println(") {
		t.Fatalf("expected lowered function and print call in generated go:\n%s", out)
	}
}

func TestGenerateGoIncludesRuntimeHelpersFromMIRCallUsage(t *testing.T) {
	src := `struct Error {
    Message: string
}
struct Result[T, E] {
    Is_ok: bool
    Value: T
    Err: E
}

fn main(): void {
    let _ = __std_session_init("app.db");
    let _ = __std_jwt_sign_hs256("{}", "{}", "secret");
    let _ = __std_bcrypt_hash("pw", 12);
    let _ = __std_http_serve_app(":8080");
}`
	prog := parseAndCheckCodegen(t, src)
	out, err := GenerateGo(prog)
	if err != nil {
		t.Fatalf("generate go failed: %v", err)
	}
	for _, want := range []string{
		"\"database/sql\"",
		"\"sync\"",
		"\"crypto/hmac\"",
		"\"golang.org/x/crypto/bcrypt\"",
		"func __std_session_init(",
		"func __std_jwt_sign_hs256(",
		"func __std_bcrypt_hash(",
		"func __std_http_serve_app(",
		"func __std_db_exec(",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected generated go to contain %q:\n%s", want, out)
		}
	}
}

func TestGenValueStmtRHSMIRUsesStructuredStmtForms(t *testing.T) {
	tests := []struct {
		name string
		stmt mir.Stmt
		want string
	}{
		{
			name: "const",
			stmt: &mir.ConstStmt{Name: "k", Type: baztypes.MustParse(ast.TypeInt), Value: &mir.IntExpr{Value: 1}},
			want: "int64(1)",
		},
		{
			name: "copy",
			stmt: &mir.CopyStmt{Name: "cp", Type: baztypes.MustParse(ast.TypeInt), Source: &mir.IdentExpr{Name: "src"}},
			want: "src",
		},
		{
			name: "unary",
			stmt: &mir.UnaryOpStmt{Name: "u", Type: baztypes.MustParse(ast.TypeInt), Op: "-", Right: &mir.IntExpr{Value: 1}},
			want: "(-int64(1))",
		},
		{
			name: "binary",
			stmt: &mir.BinaryOpStmt{Name: "b", Type: baztypes.MustParse(ast.TypeInt), Left: &mir.IntExpr{Value: 1}, Op: "+", Right: &mir.IntExpr{Value: 2}},
			want: "(int64(1) + int64(2))",
		},
		{
			name: "call",
			stmt: &mir.CallStmt{Name: "c", Type: baztypes.MustParse(ast.TypeInt), Func: "len", Args: []mir.Expr{&mir.StringExpr{Value: "x"}}},
			want: "bazic_len(\"x\")",
		},
		{
			name: "field",
			stmt: &mir.FieldAccessStmt{Name: "f", Type: baztypes.MustParse(ast.TypeInt), Object: &mir.IdentExpr{Name: "pair"}, Field: "left"},
			want: "pair.Left",
		},
		{
			name: "struct",
			stmt: &mir.StructLitStmt{
				Name:     "s",
				Type:     baztypes.MustParse(ast.Type("Pair")),
				TypeName: "Pair",
				Fields: []mir.StructLitField{
					{Name: "right", Value: &mir.IntExpr{Value: 2}},
					{Name: "left", Value: &mir.IntExpr{Value: 1}},
				},
			},
			want: "Pair{Left: int64(1), Right: int64(2)}",
		},
	}
	for _, tc := range tests {
		got, err := genValueStmtRHSMIR(tc.stmt)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestGoDeclEmissionUsesMIRDecls(t *testing.T) {
	structOut, err := genStructMIR(&mir.StructDecl{
		Name:       "pkg__User",
		TypeParams: []string{"T"},
		Fields: []mir.StructField{
			{Name: "value", Type: baztypes.MustParse(ast.TypeString)},
		},
	})
	if err != nil {
		t.Fatalf("struct emit error: %v", err)
	}
	if !strings.Contains(structOut, "type pkg__User[T any] struct") || !strings.Contains(structOut, "Value string") {
		t.Fatalf("unexpected struct output:\n%s", structOut)
	}

	ifaceOut, err := genInterfaceMIR(&mir.InterfaceDecl{
		Name: "pkg__Greeter",
		Methods: []mir.InterfaceMethod{
			{
				Name:   "greet",
				Params: []mir.Param{{Name: "name", Type: baztypes.MustParse(ast.TypeString)}},
				Return: baztypes.MustParse(ast.TypeString),
			},
		},
	})
	if err != nil {
		t.Fatalf("interface emit error: %v", err)
	}
	if !strings.Contains(ifaceOut, "type pkg__Greeter interface") || !strings.Contains(ifaceOut, "Greet(name string) string") {
		t.Fatalf("unexpected interface output:\n%s", ifaceOut)
	}

	enumOut, err := genEnumMIR(&mir.EnumDecl{
		Name:     "pkg__Role",
		Variants: []string{"Guest", "Admin"},
	})
	if err != nil {
		t.Fatalf("enum emit error: %v", err)
	}
	if !strings.Contains(enumOut, "type pkg__Role int") || !strings.Contains(enumOut, "Guest pkg__Role = iota") || !strings.Contains(enumOut, "Admin pkg__Role = iota") {
		t.Fatalf("unexpected enum output:\n%s", enumOut)
	}
}

func TestGenFuncMIRRejectsMissingCFG(t *testing.T) {
	_, err := genFuncMIR(&mir.FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &mir.Block{},
	})
	if err == nil || !strings.Contains(err.Error(), "missing mir cfg") {
		t.Fatalf("expected missing mir cfg error, got %v", err)
	}
}

func TestGenFuncMIRRejectsNonUniqueCFGBindings(t *testing.T) {
	fn := &mir.FuncDecl{
		Name:       "dup",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Params:     []mir.Param{{Name: "x", Type: baztypes.MustParse(ast.TypeInt)}},
		Body:       &mir.Block{},
		CFG: &mir.CFG{
			Entry: "entry_0",
			Blocks: []*mir.BasicBlock{
				{
					Name: "entry_0",
					Instrs: []mir.Stmt{
						&mir.LetStmt{
							Name: "x",
							Type: baztypes.MustParse(ast.TypeInt),
							Init: &mir.IntExpr{Value: 1},
						},
					},
					Term: &mir.ReturnTerminator{},
				},
			},
		},
	}
	_, err := genFuncMIR(fn)
	if err == nil || !strings.Contains(err.Error(), "non-unique mir cfg bindings") {
		t.Fatalf("expected non-unique mir cfg bindings error, got %v", err)
	}
}

func TestGenFuncMIRSkipsDeadCFGTempsAndPureExprsInMultiBlockPath(t *testing.T) {
	fn := &mir.FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &mir.Block{},
		CFG: &mir.CFG{
			Entry: "entry",
			Blocks: []*mir.BasicBlock{
				{
					Name: "entry",
					Instrs: []mir.Stmt{
						&mir.BinaryOpStmt{
							Name:  "cond__mir1",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &mir.IntExpr{Value: 1},
							Op:    "+",
							Right: &mir.IntExpr{Value: 2},
						},
						&mir.ExprStmt{Expr: &mir.StringExpr{Value: "cfg-dead"}},
					},
					Term: &mir.JumpTerminator{Target: "done"},
				},
				{
					Name: "done",
					Instrs: []mir.Stmt{
						&mir.CallStmt{
							Name: "_",
							Type: baztypes.MustParse(ast.TypeVoid),
							Func: "println",
							Args: []mir.Expr{&mir.StringExpr{Value: "ok"}},
						},
					},
					Term: &mir.ReturnTerminator{},
				},
			},
		},
	}
	out, err := genFuncMIR(fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, unwanted := range []string{"var cond__mir1 ", "cfg-dead", "cond__mir1 ="} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("expected multiblock go emission to skip dead cfg work %q:\n%s", unwanted, out)
		}
	}
	if !strings.Contains(out, "println(\"ok\")") {
		t.Fatalf("expected live side-effecting cfg call to remain:\n%s", out)
	}
}

func TestGenFuncMIRDoesNotDeclareUnusedEffectOnlyCFGBindings(t *testing.T) {
	fn := &mir.FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &mir.Block{},
		CFG: &mir.CFG{
			Entry: "entry",
			Blocks: []*mir.BasicBlock{
				{
					Name: "entry",
					Instrs: []mir.Stmt{
						&mir.CallStmt{
							Name: "unused_effect",
							Type: baztypes.MustParse(ast.TypeInt),
							Func: "parse_int",
							Args: []mir.Expr{&mir.StringExpr{Value: "42"}},
						},
					},
					Term: &mir.ReturnTerminator{},
				},
			},
		},
	}
	mir.DiscardUnusedCFGValueBindings(fn)
	out, err := genFuncMIR(fn)
	if err != nil {
		t.Fatalf("gen func error: %v", err)
	}
	if strings.Contains(out, "var unused_effect ") {
		t.Fatalf("expected unused effect-only cfg binding to avoid local declaration:\n%s", out)
	}
	if !strings.Contains(out, "parse_int(\"42\")") {
		t.Fatalf("expected effectful call to remain in generated go:\n%s", out)
	}
}

func TestGenGlobalMIRUsesDirectMIRExprEmission(t *testing.T) {
	line, err := genGlobalMIR(&mir.GlobalLetDecl{
		Name: "label",
		Type: baztypes.MustParse(ast.TypeString),
		Init: &mir.MatchExpr{
			Subject: &mir.IdentExpr{Name: "Admin"},
			Type:    baztypes.MustParse(ast.TypeString),
			Arms: []mir.MatchExprArm{
				{Variant: "Guest", Value: &mir.StringExpr{Value: "guest"}},
				{Variant: "Admin", Value: &mir.StringExpr{Value: "admin"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(line, "var label string = func() string {") || !strings.Contains(line, "case Admin:") {
		t.Fatalf("expected direct mir match initializer in generated global, got:\n%s", line)
	}
}

func parseAndCheckCodegen(t *testing.T, src string) *ast.Program {
	t.Helper()
	tokens, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	prog, err := parser.New(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := sema.New().Check(prog); err != nil {
		t.Fatalf("check: %v", err)
	}
	return prog
}
