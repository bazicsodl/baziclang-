package codegenllvm

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/backendmeta"
	"baziclang/internal/intrinsics"
	"baziclang/internal/lexer"
	"baziclang/internal/mir"
	"baziclang/internal/parser"
	"baziclang/internal/sema"
	baztypes "baziclang/internal/types"
)

func renderLLVMFuncForTest(fn *mir.FuncDecl, funcs map[string]llvmFuncSig, globals map[string]globalSlot, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	renderer := llvmFunctionRenderer{
		funcs:   funcs,
		globals: globals,
		enums:   enums,
		structs: structs,
		ifaces:  ifaces,
		strs:    strs,
	}
	return renderer.render(fn, len(globals) > 0)
}

func TestGenerateLLVMIRLowersThroughMIRForMatchAndCalls(t *testing.T) {
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
	prog := parseAndCheckLLVM(t, src)
	out, err := GenerateLLVMIR(prog)
	if err != nil {
		t.Fatalf("generate llvm failed: %v", err)
	}
	if !strings.Contains(out, "define i32 @main(") {
		t.Fatalf("expected main definition in generated llvm:\n%s", out)
	}
	if !strings.Contains(out, "@User_label") {
		t.Fatalf("expected lowered function call in generated llvm:\n%s", out)
	}
	if !strings.Contains(out, "br label %mir_entry_0") {
		t.Fatalf("expected direct mir cfg entry branch in generated llvm:\n%s", out)
	}
	if !strings.Contains(out, "mir_entry_0:") {
		t.Fatalf("expected direct mir cfg block labels in generated llvm:\n%s", out)
	}
}

func TestGenerateLLVMIREmitsDirectMIRCFGBlocks(t *testing.T) {
	src := `fn choose(flag: bool): int {
    if flag {
        return 1;
    } else {
        return 2;
    }
}

fn main(): void {
    println(choose(true));
}`
	prog := parseAndCheckLLVM(t, src)
	out, err := GenerateLLVMIR(prog)
	if err != nil {
		t.Fatalf("generate llvm failed: %v", err)
	}
	if !strings.Contains(out, "mir_if_then_1:") {
		t.Fatalf("expected direct mir cfg then block label in generated llvm:\n%s", out)
	}
	if !strings.Contains(out, "mir_if_else_2:") {
		t.Fatalf("expected direct mir cfg else block label in generated llvm:\n%s", out)
	}
}

func TestGenerateLLVMIREmitsHostTargetHeaderWhenKnown(t *testing.T) {
	src := `fn main(): void {}`
	prog := parseAndCheckLLVM(t, src)
	out, err := GenerateLLVMIR(prog)
	if err != nil {
		t.Fatalf("generate llvm failed: %v", err)
	}
	target, ok := llvmHostModuleTarget()
	if !ok {
		if strings.Contains(out, "target triple = ") || strings.Contains(out, "target datalayout = ") {
			t.Fatalf("did not expect target header for unknown host target:\n%s", out)
		}
		return
	}
	if !strings.Contains(out, `target datalayout = "`+target.DataLayout+`"`) {
		t.Fatalf("expected host target datalayout in generated llvm:\n%s", out)
	}
	if !strings.Contains(out, `target triple = "`+target.Triple+`"`) {
		t.Fatalf("expected host target triple in generated llvm:\n%s", out)
	}
}

func TestEmitValueStmtExprMIRLLVMUsesStructuredStmtForms(t *testing.T) {
	ctx := newFuncCtx(
		enumInfo{enumTypes: map[string]bool{"Role": true}, variantType: map[string]string{"Guest": "Role", "Admin": "Role"}, variantIndex: map[string]int{"Guest": 0, "Admin": 1}},
		structPool{byName: map[string]structInfo{
			"User": {
				Fields:     []structFieldInfo{{Name: "id", Type: ast.TypeInt}},
				FieldIndex: map[string]int{"id": 0},
			},
		}},
		interfacePool{names: map[string]bool{}},
		stringPool{names: map[string]string{"abc": ".str0", "guest": ".str1", "admin": ".str2"}, ordered: []string{"abc", "guest", "admin"}},
		false,
		nil,
	)
	ctx.vars["src"] = varSlot{ptr: "%src", typ: ast.TypeInt}
	ctx.vars["user"] = varSlot{ptr: "%user", typ: ast.Type("User")}

	code, value, typ, ok := emitValueStmtExprMIRLLVM(ctx, &mir.ConstStmt{
		Name:  "k",
		Type:  baztypes.MustParse(ast.TypeInt),
		Value: &mir.IntExpr{Value: 1},
	}, nil)
	if !ok {
		t.Fatalf("expected structured const stmt emission to succeed")
	}
	if typ != ast.TypeInt || value != "1" || code != "" {
		t.Fatalf("unexpected const stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}

	code, value, typ, ok = emitValueStmtExprMIRLLVM(ctx, &mir.CopyStmt{
		Name:   "cp",
		Type:   baztypes.MustParse(ast.TypeInt),
		Source: &mir.IdentExpr{Name: "src"},
	}, nil)
	if !ok {
		t.Fatalf("expected structured copy stmt emission to succeed")
	}
	if typ != ast.TypeInt || value == "" || !strings.Contains(code, "load i64, ptr %src") {
		t.Fatalf("unexpected copy stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}

	code, value, typ, ok = emitValueStmtExprMIRLLVM(ctx, &mir.BinaryOpStmt{
		Name:  "n",
		Op:    "+",
		Left:  &mir.IntExpr{Value: 1},
		Right: &mir.IntExpr{Value: 2},
	}, nil)
	if !ok {
		t.Fatalf("expected structured binary stmt emission to succeed")
	}
	if typ != ast.TypeInt || value == "" || !strings.Contains(code, "add i64") {
		t.Fatalf("unexpected binary stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}

	code, value, typ, ok = emitValueStmtExprMIRLLVM(ctx, &mir.CallStmt{
		Name: "s",
		Func: "len",
		Args: []mir.Expr{&mir.StringExpr{Value: "abc"}},
	}, nil)
	if !ok {
		t.Fatalf("expected structured call stmt emission to succeed")
	}
	if typ != ast.TypeInt || value == "" || !strings.Contains(code, "call i64 @bazic_len") {
		t.Fatalf("unexpected call stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}

	code, value, typ, ok = emitValueStmtExprMIRLLVM(ctx, &mir.FieldAccessStmt{
		Name:   "u",
		Object: &mir.IdentExpr{Name: "user"},
		Field:  "id",
	}, nil)
	if !ok {
		t.Fatalf("expected structured field-access stmt emission to succeed")
	}
	if typ != ast.TypeInt || value == "" || !strings.Contains(code, "load %User, ptr %user") || !strings.Contains(code, "extractvalue %User") {
		t.Fatalf("unexpected field-access stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}

	code, value, typ, ok = emitValueStmtExprMIRLLVM(ctx, &mir.MatchValueStmt{
		Name:    "m",
		Type:    baztypes.MustParse(ast.TypeString),
		Subject: &mir.IdentExpr{Name: "Admin"},
		Arms: []mir.MatchExprArm{
			{Variant: "Guest", Value: &mir.StringExpr{Value: "guest"}},
			{Variant: "Admin", Value: &mir.StringExpr{Value: "admin"}},
		},
	}, nil)
	if !ok {
		t.Fatalf("expected structured match-value stmt emission to succeed")
	}
	if typ != ast.TypeString || value == "" || !strings.Contains(code, "switch i64 1") || !strings.Contains(code, "phi ptr") {
		t.Fatalf("unexpected match-value stmt emission: code=%q value=%q typ=%s", code, value, typ)
	}
	if strings.Contains(code, "label %mir_match_expr_default") || !strings.Contains(code, "label %match_expr_default") {
		t.Fatalf("expected synthetic match-value default label to stay local, got:\n%s", code)
	}
}

func TestEmitValueStmtExprMIRLLVMRejectsNonValueStatements(t *testing.T) {
	ctx := newFuncCtx(enumInfo{}, structPool{byName: map[string]structInfo{}}, interfacePool{names: map[string]bool{}}, stringPool{}, false, nil)
	if _, _, _, ok := emitValueStmtExprMIRLLVM(ctx, &mir.ExprStmt{Expr: &mir.IntExpr{Value: 1}}, nil); ok {
		t.Fatalf("expected non-value statement to be rejected")
	}
}

func TestEmitCFGInstrMIRLLVMStoresLetThroughUnifiedValuePath(t *testing.T) {
	ctx := newFuncCtx(enumInfo{}, structPool{byName: map[string]structInfo{}}, interfacePool{names: map[string]bool{}}, stringPool{}, false, nil)
	ctx.vars["n"] = varSlot{ptr: "%n", typ: ast.TypeInt}
	var b strings.Builder
	ok := emitCFGInstrMIRLLVM(&b, ctx, &mir.LetStmt{
		Name: "n",
		Type: baztypes.MustParse(ast.TypeInt),
		Init: &mir.BinaryExpr{
			Left:  &mir.IntExpr{Value: 1},
			Op:    "+",
			Right: &mir.IntExpr{Value: 2},
		},
	}, nil)
	if !ok {
		t.Fatalf("expected let cfg instruction to emit successfully")
	}
	out := b.String()
	if !strings.Contains(out, "add i64") || !strings.Contains(out, "store i64") || !strings.Contains(out, "%n") {
		t.Fatalf("expected unified let store emission, got:\n%s", out)
	}
}

func TestEmitTerminatorMIRLLVMBoxesAnyReturnThroughSharedCoercion(t *testing.T) {
	ctx := newFuncCtx(enumInfo{}, structPool{byName: map[string]structInfo{}}, interfacePool{names: map[string]bool{}}, stringPool{}, false, nil)
	ctx.returnType = ast.TypeAny
	var b strings.Builder
	ok := emitTerminatorMIRLLVM(&b, ctx, &mir.ReturnTerminator{Value: &mir.IntExpr{Value: 7}}, nil)
	if !ok {
		t.Fatalf("expected any-return terminator emission to succeed")
	}
	out := b.String()
	if !strings.Contains(out, "inttoptr i64 7 to ptr") || !strings.Contains(out, "ret %Any") {
		t.Fatalf("expected boxed any return emission, got:\n%s", out)
	}
}

func TestEmitCallValueStmtMIRLLVMBoxesAnyArgsThroughSharedCoercion(t *testing.T) {
	ctx := newFuncCtx(enumInfo{}, structPool{byName: map[string]structInfo{}}, interfacePool{names: map[string]bool{}}, stringPool{}, false, nil)
	code, _, typ, ok := emitCallValueStmtMIRLLVM(ctx, &mir.CallStmt{
		Name: "n",
		Func: "accept_any",
		Args: []mir.Expr{&mir.IntExpr{Value: 7}},
	}, map[string]llvmFuncSig{
		"accept_any": {Params: []ast.Type{ast.TypeAny}, Ret: ast.TypeInt},
	})
	if !ok {
		t.Fatalf("expected any-arg call emission to succeed")
	}
	if typ != ast.TypeInt || !strings.Contains(code, "inttoptr i64 7 to ptr") || !strings.Contains(code, "call i64 @accept_any(%Any") {
		t.Fatalf("expected boxed any arg emission, got:\n%s", code)
	}
}

func TestBoxToAnyHeapCopiesStructPayloadThroughSharedABIType(t *testing.T) {
	ctx := newFuncCtx(
		enumInfo{},
		structPool{byName: map[string]structInfo{"User": {}}},
		interfacePool{names: map[string]bool{}},
		stringPool{},
		false,
		nil,
	)
	code, value, ok := boxToAny(ctx, "%user", ast.Type("User"))
	if !ok {
		t.Fatalf("expected struct any boxing to succeed")
	}
	if value == "" || !strings.Contains(code, "call ptr @malloc") || !strings.Contains(code, "bitcast ptr") || !strings.Contains(code, "to %User*") || !strings.Contains(code, "store %User %user") {
		t.Fatalf("expected heap-copy struct any boxing, got:\n%s", code)
	}
}

func TestEmitFunctionMIRRejectsMissingCFG(t *testing.T) {
	_, err := renderLLVMFuncForTest(&mir.FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &mir.Block{},
	}, nil, nil, enumInfo{}, structPool{}, interfacePool{}, stringPool{})
	if err == nil || !strings.Contains(err.Error(), "missing mir cfg") {
		t.Fatalf("expected missing mir cfg error, got %v", err)
	}
}

func TestEmitFunctionMIRRejectsNonUniqueCFGBindings(t *testing.T) {
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
	_, err := renderLLVMFuncForTest(fn, nil, nil, enumInfo{}, structPool{}, interfacePool{}, stringPool{})
	if err == nil || !strings.Contains(err.Error(), "non-unique mir cfg bindings") {
		t.Fatalf("expected non-unique mir cfg bindings error, got %v", err)
	}
}

func TestEmitFunctionBodyMIRDoesNotAllocateUnusedEffectOnlyCFGBindings(t *testing.T) {
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
	var b strings.Builder
	ctx := newFuncCtx(enumInfo{}, structPool{byName: map[string]structInfo{}}, interfacePool{names: map[string]bool{}}, stringPool{names: map[string]string{"42": ".str0"}, ordered: []string{"42"}}, false, nil)
	err := emitFunctionBodyMIR(&b, ctx, fn, map[string]llvmFuncSig{
		"parse_int": {Params: []ast.Type{ast.TypeString}, Ret: ast.TypeInt},
	})
	if err != nil {
		t.Fatalf("emit function body error: %v", err)
	}
	out := b.String()
	if strings.Contains(out, "unused_effect") {
		t.Fatalf("expected unused effect-only cfg binding to avoid local allocation:\n%s", out)
	}
	if !strings.Contains(out, "bazic_parse_int") {
		t.Fatalf("expected effectful call to remain in generated llvm:\n%s", out)
	}
}

func TestEmitFunctionMIRSkipsDeadCFGTempsAndPureExprs(t *testing.T) {
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
	out, err := renderLLVMFuncForTest(
		fn,
		nil,
		nil,
		enumInfo{},
		structPool{byName: map[string]structInfo{}},
		interfacePool{names: map[string]bool{}},
		stringPool{
			names: map[string]string{
				"ok":   ".str0",
				"%s":   ".str1",
				"%s\n": ".str2",
			},
			ordered: []string{"ok", "%s", "%s\n"},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, unwanted := range []string{"cfg-dead", "cond__mir1 = alloca"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("expected llvm mir emission to skip dead cfg work %q:\n%s", unwanted, out)
		}
	}
	if !strings.Contains(out, "@puts") && !strings.Contains(out, "@printf") {
		t.Fatalf("expected live side-effecting cfg call to remain:\n%s", out)
	}
}

func TestEmitGlobalInitUsesMIRExpressionsDirectly(t *testing.T) {
	globals := globalSet{
		order: []globalInfo{
			{
				Name: "label",
				Type: ast.TypeString,
				Init: &mir.MatchExpr{
					Subject: &mir.IdentExpr{Name: "Admin"},
					Type:    baztypes.MustParse(ast.TypeString),
					Arms: []mir.MatchExprArm{
						{Variant: "Guest", Value: &mir.StringExpr{Value: "guest"}},
						{Variant: "Admin", Value: &mir.StringExpr{Value: "admin"}},
					},
				},
			},
		},
		slots: map[string]globalSlot{
			"label": {ptr: "@label", typ: ast.TypeString},
		},
	}
	out, err := emitGlobalInit(
		globals,
		nil,
		enumInfo{
			enumTypes:    map[string]bool{"Role": true},
			variantType:  map[string]string{"Guest": "Role", "Admin": "Role"},
			variantIndex: map[string]int{"Guest": 0, "Admin": 1},
		},
		structPool{byName: map[string]structInfo{}},
		interfacePool{names: map[string]bool{}},
		stringPool{names: map[string]string{"guest": ".str0", "admin": ".str1"}, ordered: []string{"guest", "admin"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "define void @__bazic_init_globals()") || !strings.Contains(out, "switch i64 1") || !strings.Contains(out, "store ptr") {
		t.Fatalf("expected mir-native global init emission, got:\n%s", out)
	}
}

func TestEmitGlobalInitBoxesAnyValuesThroughSharedCoercion(t *testing.T) {
	globals := globalSet{
		order: []globalInfo{
			{Name: "payload", Type: ast.TypeAny, Init: &mir.IntExpr{Value: 7}},
		},
		slots: map[string]globalSlot{
			"payload": {ptr: "@payload", typ: ast.TypeAny},
		},
	}
	out, err := emitGlobalInit(
		globals,
		nil,
		enumInfo{},
		structPool{byName: map[string]structInfo{}},
		interfacePool{names: map[string]bool{}},
		stringPool{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "inttoptr i64 7 to ptr") || !strings.Contains(out, "store %Any") {
		t.Fatalf("expected boxed any global init emission, got:\n%s", out)
	}
}

func TestCollectStringLiteralsFromMIRIncludesLiveCFGStrings(t *testing.T) {
	pool := collectStringLiteralsFromMIR(&mir.Program{
		Decls: []mir.Decl{
			&mir.FuncDecl{
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
									Name: "_",
									Type: baztypes.MustParse(ast.TypeVoid),
									Func: "println",
									Args: []mir.Expr{&mir.StringExpr{Value: "cfg-live"}},
								},
							},
							Term: &mir.ReturnTerminator{},
						},
					},
				},
			},
		},
	}, []string{"extra-route"})

	if _, ok := pool.names["cfg-live"]; !ok {
		t.Fatalf("expected live cfg string to be collected from mir")
	}
	if _, ok := pool.names["extra-route"]; !ok {
		t.Fatalf("expected extra string to be preserved")
	}
	if _, ok := pool.names["%s"]; !ok {
		t.Fatalf("expected builtin runtime format strings to be preserved")
	}
}

func TestCollectStringLiteralsFromMIRSkipsDeadCFGStrings(t *testing.T) {
	pool := collectStringLiteralsFromMIR(&mir.Program{
		Decls: []mir.Decl{
			&mir.FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &mir.Block{},
				CFG: &mir.CFG{
					Entry: "entry",
					Blocks: []*mir.BasicBlock{
						{
							Name: "entry",
							Instrs: []mir.Stmt{
								&mir.ExprStmt{Expr: &mir.StringExpr{Value: "cfg-dead"}},
							},
							Term: &mir.ReturnTerminator{},
						},
					},
				},
			},
		},
	}, nil)

	if _, ok := pool.names["cfg-dead"]; ok {
		t.Fatalf("expected dead cfg string to be skipped")
	}
}

func TestEmitStdDeclsUsesIntrinsicRegistryAndAvailableResultStructs(t *testing.T) {
	httpResponseType := "pkg__HttpResponse"
	resultHttpResponseErr := intrinsics.LLVMResultStructName(httpResponseType, "Error")
	structs := structPool{
		byName: map[string]structInfo{
			intrinsics.LLVMResultStructName("string", "Error"): {},
			intrinsics.LLVMResultStructName("bool", "Error"):   {},
			intrinsics.LLVMResultStructName("int", "Error"):    {},
			httpResponseType:      {},
			resultHttpResponseErr: {},
		},
	}
	out := emitStdDecls(map[string]ast.Type{"HttpResponse": ast.Type(httpResponseType)}, structs)
	if !strings.Contains(out, "declare void @__std_json_get_int(ptr sret(%Result__int__Error), ptr, ptr)\n") {
		t.Fatalf("expected int result intrinsic decl in generated std decls:\n%s", out)
	}
	if strings.Contains(out, "__std_json_get_float") {
		t.Fatalf("did not expect float result intrinsic decl without matching result struct:\n%s", out)
	}
	expectedHttpRespDecl := "declare void @__std_http_get_opts_resp(ptr sret(%" + resultHttpResponseErr + "), ptr, i64, i64, ptr, ptr, i8, ptr)\n"
	if !strings.Contains(out, expectedHttpRespDecl) {
		t.Fatalf("expected HttpResponse result decl using internal type name:\n%s", out)
	}
	if !strings.Contains(out, "declare ptr @__std_args()\n") {
		t.Fatalf("expected plain pointer-return intrinsic decl in generated std decls:\n%s", out)
	}
}

func TestLLVMMetadataCollectorsUseMIRDecls(t *testing.T) {
	mp := &mir.Program{
		Decls: []mir.Decl{
			&mir.EnumDecl{
				Name:     "pkg__Role",
				Variants: []string{"Guest", "Admin"},
			},
			&mir.StructDecl{
				Name: "pkg__User",
				Fields: []mir.StructField{
					{Name: "name", Type: baztypes.MustParse(ast.TypeString)},
					{Name: "age", Type: baztypes.MustParse(ast.TypeInt)},
				},
			},
			&mir.InterfaceDecl{
				Name: "pkg__Greeter",
			},
		},
	}

	enums := collectEnums(backendmeta.CollectProgramEnums(mp))
	if !enums.enumTypes["pkg__Role"] || enums.variantType["Admin"] != "pkg__Role" || enums.variantIndex["Guest"] != 0 {
		t.Fatalf("unexpected enum info: %#v", enums)
	}

	structs := collectStructs(backendmeta.CollectProgramStructs(mp))
	user, ok := structs.byName["pkg__User"]
	if !ok || len(user.Fields) != 2 || user.FieldIndex["age"] != 1 || user.Fields[0].Type != ast.TypeString {
		t.Fatalf("unexpected struct pool: %#v", structs)
	}

	ifaces := collectInterfaces(backendmeta.CollectProgramInterfaces(mp))
	if !ifaces.names["pkg__Greeter"] || len(ifaces.order) != 1 || ifaces.order[0] != "pkg__Greeter" {
		t.Fatalf("unexpected interface pool: %#v", ifaces)
	}
}

func parseAndCheckLLVM(t *testing.T, src string) *ast.Program {
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
