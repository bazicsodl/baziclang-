package mir

import (
	"strings"
	"testing"

	"baziclang/internal/ast"
	"baziclang/internal/hir"
	"baziclang/internal/lexer"
	"baziclang/internal/parser"
	"baziclang/internal/sema"
	baztypes "baziclang/internal/types"
)

func TestLowerProducesValidatedTypedCallAndMatch(t *testing.T) {
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
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	mainFn := out.Decls[len(out.Decls)-1].(*FuncDecl)
	if mainFn.CFG == nil {
		t.Fatalf("expected lowered cfg on main function")
	}
	if mainFn.CFG.Entry == "" || len(mainFn.CFG.Blocks) == 0 {
		t.Fatalf("expected populated cfg, got %+v", mainFn.CFG)
	}
	var matchValue *MatchValueStmt
	var callStmt *CallStmt
	for _, stmt := range mainFn.Body.Stmts {
		if matchStmt, ok := stmt.(*MatchValueStmt); ok {
			matchValue = matchStmt
			if matchStmt.Type.String() != string(ast.TypeString) {
				t.Fatalf("expected match expr type string, got %s", matchStmt.Type)
			}
		}
		if stmt, ok := stmt.(*CallStmt); ok {
			callStmt = stmt
		}
	}
	if matchValue == nil {
		if !blockContainsExpr(mainFn.Body, func(e Expr) bool {
			s, ok := e.(*StringExpr)
			return ok && s.Value == "admin"
		}) {
			t.Fatalf("expected lowered match expr value stmt or stronger folded admin string in main body")
		}
	}
	if callStmt == nil {
		t.Fatalf("expected lowered call stmt in main body")
	}
	if callStmt.Name != "_" {
		t.Fatalf("expected side-effect call binding '_', got %q", callStmt.Name)
	}
	if callStmt.Func != "println" {
		t.Fatalf("expected call func println, got %s", callStmt.Func)
	}
}

func TestValidateRejectsUnresolvedCall(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name: "main",
				Body: &Block{
					Stmts: []Stmt{
						&ExprStmt{Expr: &CallExpr{}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected unresolved call rejection")
	}
	if !strings.Contains(err.Error(), "unresolved call") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsStatementAfterReturn(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{},
						&ExprStmt{Expr: &StringExpr{Value: "dead"}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected unreachable statement rejection")
	}
	if !strings.Contains(err.Error(), "unreachable statement") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsMissingReturnOnNonVoidFunction(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&ExprStmt{Expr: &StringExpr{Value: "side-effect"}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected missing return rejection")
	}
	if !strings.Contains(err.Error(), "falls through without returning") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsInvalidAssignmentTarget(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body: &Block{
					Stmts: []Stmt{
						&AssignStmt{
							Target: &CallExpr{Func: "bad"},
							Value:  &IntExpr{Value: 1},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected invalid assignment target rejection")
	}
	if !strings.Contains(err.Error(), "invalid assignment target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsMatchExprArmWithoutVariant(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{
							Value: &MatchExpr{
								Subject: &IdentExpr{Name: "role"},
								Arms: []MatchExprArm{
									{Value: &StringExpr{Value: "guest"}},
								},
								Type: baztypes.MustParse(ast.TypeString),
							},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected missing variant rejection")
	}
	if !strings.Contains(err.Error(), "match expression arm missing variant") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsMissingCFGJumpTarget(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &JumpTerminator{Target: "missing"},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected missing cfg target rejection")
	}
	if !strings.Contains(err.Error(), "jump target 'missing' not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsValueReturnInVoidFunction(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{Value: &IntExpr{Value: 1}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected void return-value rejection")
	}
	if !strings.Contains(err.Error(), "void function cannot return a value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsBareReturnInNonVoidFunction(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected bare return rejection")
	}
	if !strings.Contains(err.Error(), "non-void function must return a value of type string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsUnreachableCFGBlock(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &ReturnTerminator{},
						},
						{
							Name: "dead",
							Term: &ReturnTerminator{},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected unreachable cfg block rejection")
	}
	if !strings.Contains(err.Error(), "unreachable cfg block 'dead'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsConditionalWithSameTarget(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &CondTerminator{
								Cond: &BoolExpr{Value: true},
								Then: "join",
								Else: "join",
							},
						},
						{
							Name: "join",
							Term: &ReturnTerminator{},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected malformed conditional rejection")
	}
	if !strings.Contains(err.Error(), "branches to the same target 'join'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsDuplicateUnguardedMatchArmInCFG(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &MatchTerminator{
								Subject: &IdentExpr{Name: "role"},
								Arms: []MatchTerminatorArm{
									{Variant: "Guest", Target: "guest1"},
									{Variant: "Guest", Target: "guest2"},
								},
							},
						},
						{Name: "guest1", Term: &ReturnTerminator{}},
						{Name: "guest2", Term: &ReturnTerminator{}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected duplicate unguarded match arm rejection")
	}
	if !strings.Contains(err.Error(), "duplicate unguarded match arm for variant 'Guest'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsGuardedMatchArmAfterUnguardedInCFG(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &MatchTerminator{
								Subject: &IdentExpr{Name: "role"},
								Arms: []MatchTerminatorArm{
									{Variant: "Guest", Target: "guest1"},
									{Variant: "Guest", Guard: &BoolExpr{Value: true}, Target: "guest2"},
								},
							},
						},
						{Name: "guest1", Term: &ReturnTerminator{}},
						{Name: "guest2", Term: &ReturnTerminator{}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected guarded-after-unguarded match arm rejection")
	}
	if !strings.Contains(err.Error(), "guarded match arm for variant 'Guest' appears after unguarded arm") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsCFGValueReturnInVoidFunction(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &ReturnTerminator{Value: &IntExpr{Value: 1}},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected cfg void return-value rejection")
	}
	if !strings.Contains(err.Error(), "void function cfg cannot return a value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsCFGBareReturnInNonVoidFunction(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{Value: &StringExpr{Value: "ok"}},
					},
				},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &ReturnTerminator{},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected cfg bare return rejection")
	}
	if !strings.Contains(err.Error(), "non-void function cfg must return a value of type string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHasUniqueCFGBindingsTracksParamAndLocalCollisions(t *testing.T) {
	unique := &FuncDecl{
		Name: "ok",
		Params: []Param{
			{Name: "input", Type: baztypes.MustParse(ast.TypeString)},
		},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&LetStmt{Name: "label", Type: baztypes.MustParse(ast.TypeString), Init: &StringExpr{Value: "x"}},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	if !HasUniqueCFGBindings(unique) {
		t.Fatalf("expected unique cfg bindings to be allowed")
	}

	colliding := &FuncDecl{
		Name: "bad",
		Params: []Param{
			{Name: "input", Type: baztypes.MustParse(ast.TypeString)},
		},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&LetStmt{Name: "input", Type: baztypes.MustParse(ast.TypeString), Init: &StringExpr{Value: "shadow"}},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	if HasUniqueCFGBindings(colliding) {
		t.Fatalf("expected param/local binding collision to be rejected")
	}
}

func TestCollectCFGLetTypesAggregatesCFGLocals(t *testing.T) {
	fn := &FuncDecl{
		Name: "main",
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&LetStmt{Name: "left", Type: baztypes.MustParse(ast.TypeInt), Init: &IntExpr{Value: 1}},
					},
					Term: &JumpTerminator{Target: "next"},
				},
				{
					Name: "next",
					Instrs: []Stmt{
						&LetStmt{Name: "right", Type: baztypes.MustParse(ast.TypeString), Init: &StringExpr{Value: "ok"}},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	types := CollectCFGLetTypes(fn)
	if len(types) != 2 {
		t.Fatalf("expected 2 collected cfg let types, got %d", len(types))
	}
	if got := baztypes.ToAST(types["left"]); got != ast.TypeInt {
		t.Fatalf("expected left type int, got %s", got)
	}
	if got := baztypes.ToAST(types["right"]); got != ast.TypeString {
		t.Fatalf("expected right type string, got %s", got)
	}
}

func TestAnalyzeCFGBuildsSharedTopology(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &CondTerminator{
					Cond: &BoolExpr{Value: true},
					Then: "left",
					Else: "right",
				},
			},
			{
				Name: "left",
				Term: &JumpTerminator{Target: "join"},
			},
			{
				Name: "right",
				Term: &JumpTerminator{Target: "join"},
			},
			{
				Name: "join",
				Term: &ReturnTerminator{},
			},
		},
	}
	topology, err := AnalyzeCFG(cfg)
	if err != nil {
		t.Fatalf("analyze cfg failed: %v", err)
	}
	if len(topology.Successors["entry"]) != 2 || topology.Successors["entry"][0] != "left" || topology.Successors["entry"][1] != "right" {
		t.Fatalf("unexpected entry successors: %v", topology.Successors["entry"])
	}
	if len(topology.Predecessors["join"]) != 2 {
		t.Fatalf("expected join predecessors from both branches, got %v", topology.Predecessors["join"])
	}
	if !topology.IsJoinBlock("join") {
		t.Fatalf("expected join to be recognized as a join block")
	}
	if got := topology.ReversePostOrderNames(); len(got) != 4 || got[0] != "entry" || got[len(got)-1] != "join" {
		t.Fatalf("unexpected reverse postorder traversal: %v", got)
	}
	if got := topology.ReachableBlockNames(); len(got) != 4 || got[0] != "entry" || got[3] != "join" {
		t.Fatalf("unexpected reachable block order: %v", got)
	}
	if !topology.Reachable["join"] {
		t.Fatalf("expected join block to be reachable")
	}
}

func TestSimplifyCFGRemovesTrivialJumpBlocks(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &JumpTerminator{Target: "mid"},
			},
			{
				Name: "mid",
				Term: &JumpTerminator{Target: "end"},
			},
			{
				Name: "end",
				Term: &ReturnTerminator{},
			},
		},
	}
	simplifyCFG(nil, cfg)
	if cfg.Entry != "end" {
		t.Fatalf("expected entry to collapse to end, got %q", cfg.Entry)
	}
	if len(cfg.Blocks) != 1 || cfg.Blocks[0].Name != "end" {
		t.Fatalf("expected only end block to remain, got %+v", cfg.Blocks)
	}
}

func TestSimplifyCFGCollapsesSameTargetConditionToJump(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &CondTerminator{
					Cond: &BoolExpr{Value: true},
					Then: "then",
					Else: "else",
				},
			},
			{
				Name: "then",
				Term: &JumpTerminator{Target: "join"},
			},
			{
				Name: "else",
				Term: &JumpTerminator{Target: "join"},
			},
			{
				Name: "join",
				Term: &ReturnTerminator{},
			},
		},
	}
	simplifyCFG(nil, cfg)
	entry := cfg.Blocks[0]
	if len(entry.Instrs) > 1 {
		t.Fatalf("expected at most one preserved condition expr, got %d", len(entry.Instrs))
	}
	if len(entry.Instrs) == 1 {
		if _, ok := entry.Instrs[0].(*ExprStmt); !ok {
			t.Fatalf("expected preserved condition expr stmt, got %T", entry.Instrs[0])
		}
	}
	switch term := entry.Term.(type) {
	case *JumpTerminator:
		if term.Target != "join" {
			t.Fatalf("expected collapsed jump target join, got %q", term.Target)
		}
	case *ReturnTerminator:
		// acceptable stronger form: jump collapse followed by linear merge into join return
	default:
		t.Fatalf("expected entry terminator to collapse to jump or merged return, got %T %+v", entry.Term, entry.Term)
	}
}

func TestSimplifyCFGCollapsesConstantConditionToJump(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &CondTerminator{
					Cond: &BoolExpr{Value: true},
					Then: "yes",
					Else: "no",
				},
			},
			{
				Name: "yes",
				Instrs: []Stmt{
					&CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "println", Args: []Expr{&StringExpr{Value: "yes"}}},
				},
				Term: &ReturnTerminator{},
			},
			{
				Name: "no",
				Instrs: []Stmt{
					&CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "println", Args: []Expr{&StringExpr{Value: "no"}}},
				},
				Term: &ReturnTerminator{},
			},
		},
	}
	simplifyCFG(nil, cfg)
	if _, ok := cfg.Blocks[0].Term.(*CondTerminator); ok {
		t.Fatalf("expected constant cond terminator to be simplified away")
	}
	for _, block := range cfg.Blocks {
		if block.Name == "no" {
			t.Fatalf("expected false branch block to be removed")
		}
	}
}

func TestSimplifyCFGRetargetsLoopPredecessorsBeforeRemovingTrivialCondBlock(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &JumpTerminator{Target: "while_cond"},
			},
			{
				Name: "while_cond",
				Term: &CondTerminator{
					Cond: &BoolExpr{Value: true},
					Then: "while_body",
					Else: "while_exit",
				},
			},
			{
				Name: "while_body",
				Instrs: []Stmt{
					&CallStmt{Name: "_", Type: baztypes.MustParse(ast.TypeVoid), Func: "println", Args: []Expr{&StringExpr{Value: "tick"}}},
				},
				Term: &JumpTerminator{Target: "while_cond"},
			},
			{
				Name: "while_exit",
				Term: &ReturnTerminator{},
			},
		},
	}

	simplifyCFG(nil, cfg)

	for _, block := range cfg.Blocks {
		for _, target := range TerminatorSuccessors(block.Term) {
			if target == "while_cond" {
				t.Fatalf("expected loop predecessors to be retargeted before trivial cond block removal")
			}
		}
	}
	if err := validateCFG(nil, cfg, baztypes.MustParse(ast.TypeVoid)); err != nil {
		t.Fatalf("expected simplified loop cfg to remain valid, got %v", err)
	}
}

func TestSimplifyCFGMergesLinearSuccessorBlock(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Instrs: []Stmt{
					&LetStmt{Name: "x", Type: baztypes.MustParse(ast.TypeInt), Init: &IntExpr{Value: 1}},
				},
				Term: &JumpTerminator{Target: "next"},
			},
			{
				Name: "next",
				Instrs: []Stmt{
					&ExprStmt{Expr: &CallExpr{Func: "println", Args: []Expr{&StringExpr{Value: "ok"}}}},
				},
				Term: &ReturnTerminator{},
			},
		},
	}
	simplifyCFG(nil, cfg)
	if len(cfg.Blocks) != 1 {
		t.Fatalf("expected merged cfg to keep one block, got %d", len(cfg.Blocks))
	}
	entry := cfg.Blocks[0]
	if entry.Name != "entry" {
		t.Fatalf("expected merged block to remain entry, got %q", entry.Name)
	}
	if len(entry.Instrs) != 2 {
		t.Fatalf("expected merged entry instrs to include successor body, got %d", len(entry.Instrs))
	}
	if _, ok := entry.Term.(*ReturnTerminator); !ok {
		t.Fatalf("expected merged block terminator to become successor return, got %T", entry.Term)
	}
}

func TestSimplifyCFGPrunesConstantGuardedMatchTerminatorArms(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &MatchTerminator{
					Subject: &IdentExpr{Name: "role"},
					Arms: []MatchTerminatorArm{
						{Variant: "Guest", Guard: &BoolExpr{Value: false}, Target: "guest_dead"},
						{Variant: "Guest", Guard: &BoolExpr{Value: true}, Target: "guest_live"},
						{Variant: "Guest", Guard: &CallExpr{Func: "is_guest", Args: []Expr{&IdentExpr{Name: "role"}}}, Target: "guest_later"},
						{Variant: "Admin", Target: "admin"},
					},
				},
			},
			{Name: "guest_dead", Term: &ReturnTerminator{}},
			{Name: "guest_live", Term: &ReturnTerminator{}},
			{Name: "guest_later", Term: &ReturnTerminator{}},
			{Name: "admin", Term: &ReturnTerminator{}},
		},
	}
	simplifyCFG(nil, cfg)
	entry := cfg.Blocks[0]
	term, ok := entry.Term.(*MatchTerminator)
	if !ok {
		t.Fatalf("expected entry to remain match terminator, got %T", entry.Term)
	}
	if len(term.Arms) != 2 {
		t.Fatalf("expected 2 reachable match terminator arms, got %d", len(term.Arms))
	}
	if term.Arms[0].Variant != "Guest" || term.Arms[0].Guard != nil || term.Arms[0].Target != "guest_live" {
		t.Fatalf("unexpected first surviving guest arm: %#v", term.Arms[0])
	}
	if term.Arms[1].Variant != "Admin" || term.Arms[1].Guard != nil || term.Arms[1].Target != "admin" {
		t.Fatalf("unexpected admin arm: %#v", term.Arms[1])
	}
}

func TestSimplifyCFGCollapsesConstantMatchTerminatorToChosenTarget(t *testing.T) {
	ctx := newTypeContext(newTypeIndex(&Program{
		Decls: []Decl{
			&EnumDecl{Name: "Role", Variants: []string{"Guest", "Admin"}},
		},
	}))
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &MatchTerminator{
					Subject: &IdentExpr{Name: "Guest"},
					Arms: []MatchTerminatorArm{
						{Variant: "Guest", Guard: &BoolExpr{Value: false}, Target: "guest_dead"},
						{Variant: "Guest", Guard: &BoolExpr{Value: true}, Target: "guest_live"},
						{Variant: "Admin", Target: "admin"},
					},
					Default: "fallback",
				},
			},
			{Name: "guest_dead", Term: &ReturnTerminator{}},
			{Name: "guest_live", Term: &ReturnTerminator{}},
			{Name: "admin", Term: &ReturnTerminator{}},
			{Name: "fallback", Term: &ReturnTerminator{}},
		},
	}
	simplifyCFG(ctx, cfg)
	entry := cfg.Blocks[0]
	switch term := entry.Term.(type) {
	case *JumpTerminator:
		if term.Target != "guest_live" {
			t.Fatalf("expected constant match to jump to guest_live, got %q", term.Target)
		}
	case *ReturnTerminator:
		// acceptable stronger form: jump collapse followed by linear merge into chosen return
	default:
		t.Fatalf("expected constant match terminator to collapse, got %T", entry.Term)
	}
	for _, block := range cfg.Blocks {
		if block.Name == "guest_dead" || block.Name == "admin" || block.Name == "fallback" {
			t.Fatalf("expected non-chosen constant match targets to be pruned, still found %q", block.Name)
		}
	}
}

func TestSimplifyCFGCollapsesSameTargetMatchToJump(t *testing.T) {
	cfg := &CFG{
		Entry: "entry",
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Term: &MatchTerminator{
					Subject: &IdentExpr{Name: "role"},
					Arms: []MatchTerminatorArm{
						{Variant: "Guest", Guard: &BoolExpr{Value: false}, Target: "dead"},
						{Variant: "Guest", Guard: &BoolExpr{Value: true}, Target: "join"},
						{Variant: "Admin", Target: "join"},
					},
					Default: "join",
				},
			},
			{Name: "dead", Term: &ReturnTerminator{}},
			{Name: "join", Term: &ReturnTerminator{}},
		},
	}
	simplifyCFG(nil, cfg)
	entry := cfg.Blocks[0]
	switch term := entry.Term.(type) {
	case *JumpTerminator:
		if term.Target != "join" {
			t.Fatalf("expected collapsed match jump target join, got %q", term.Target)
		}
	case *ReturnTerminator:
		// acceptable stronger form: jump collapse followed by linear merge into join return
	default:
		t.Fatalf("expected same-target match terminator to collapse, got %T", entry.Term)
	}
}

func TestLowerNormalizesShadowedLocalsToUniqueMIRBindings(t *testing.T) {
	src := `fn shadow(): bool {
    let x = 1;
    if true {
        let x = 2;
        if (x != 2) {
            return false;
        }
    }
    return (x == 1);
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	if !HasUniqueCFGBindings(fn) {
		t.Fatalf("expected shadowed locals to be normalized into unique cfg bindings")
	}
	var outerName string
	var innerLet *LetStmt
	for _, stmt := range fn.Body.Stmts {
		switch st := stmt.(type) {
		case *LetStmt:
			if st.Name == "x" {
				outerName = st.Name
			}
			if strings.HasPrefix(st.Name, "x__mir") {
				innerLet = st
			}
		case *ConstStmt:
			if st.Name == "x" {
				outerName = st.Name
			}
		}
	}
	if outerName == "" {
		t.Fatalf("expected outer let to keep source binding name x")
	}
	if outerName != "x" {
		t.Fatalf("expected outer let to keep source name, got %q", outerName)
	}
	if innerLet != nil && (innerLet.Name == "x" || !strings.HasPrefix(innerLet.Name, "x__mir")) {
		t.Fatalf("expected inner shadowed let to be renamed, got %q", innerLet.Name)
	}
	retIndex := len(fn.Body.Stmts) - 1
	ret := fn.Body.Stmts[retIndex].(*ReturnStmt)
	finalValue := ret.Value
	if ident, ok := finalValue.(*IdentExpr); ok && retIndex > 0 {
		prev := fn.Body.Stmts[retIndex-1]
		switch st := prev.(type) {
		case *LetStmt:
			if st.Name == ident.Name {
				finalValue = st.Init
			}
		case *UnaryOpStmt:
			if st.Name == ident.Name {
				finalValue = &UnaryExpr{NodeInfo: st.NodeInfo, Op: st.Op, Right: st.Right}
			}
		case *BinaryOpStmt:
			if st.Name == ident.Name {
				finalValue = &BinaryExpr{NodeInfo: st.NodeInfo, Left: st.Left, Op: st.Op, Right: st.Right}
			}
		}
	}
	switch finalCond := finalValue.(type) {
	case *BinaryExpr:
		finalLeft := finalCond.Left.(*IdentExpr)
		if finalLeft.Name != outerName {
			t.Fatalf("expected final return to reference outer local %q, got %q", outerName, finalLeft.Name)
		}
	case *BoolExpr:
		if !finalCond.Value {
			t.Fatalf("expected folded final return condition to remain true, got false")
		}
	default:
		t.Fatalf("expected final return value to be binary or folded bool, got %T", finalValue)
	}
}

func TestLowerANormalizesIfConditionsAndReturnsIntoTemps(t *testing.T) {
	src := `fn compute(a: int, b: int): int {
    if (a + b) > 10 {
        return a + b;
    }
    return a + b;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	if len(fn.Body.Stmts) < 3 {
		t.Fatalf("expected additional temp lets in normalized body, got %d stmts", len(fn.Body.Stmts))
	}
	var condTempName string
	var ifStmt *IfStmt
	for _, stmt := range fn.Body.Stmts {
		switch st := stmt.(type) {
		case *LetStmt:
			if strings.HasPrefix(st.Name, "cond__mir") {
				condTempName = st.Name
			}
		case *UnaryOpStmt:
			if strings.HasPrefix(st.Name, "cond__mir") {
				condTempName = st.Name
			}
		case *BinaryOpStmt:
			if strings.HasPrefix(st.Name, "cond__mir") {
				condTempName = st.Name
			}
		}
		if s, ok := stmt.(*IfStmt); ok {
			ifStmt = s
			break
		}
	}
	if condTempName == "" {
		t.Fatalf("expected condition temp statement in normalized body")
	}
	if ifStmt == nil {
		t.Fatalf("expected if stmt in normalized body")
	}
	if condIdent, ok := ifStmt.Cond.(*IdentExpr); !ok || condIdent.Name != condTempName {
		t.Fatalf("expected if condition to use temp %q, got %T", condTempName, ifStmt.Cond)
	}
	var thenRetTempName string
	switch st := ifStmt.Then.Stmts[0].(type) {
	case *LetStmt:
		thenRetTempName = st.Name
	case *UnaryOpStmt:
		thenRetTempName = st.Name
	case *BinaryOpStmt:
		thenRetTempName = st.Name
	}
	if !strings.HasPrefix(thenRetTempName, "ret__mir") {
		t.Fatalf("expected then branch to start with return temp stmt, got %T", ifStmt.Then.Stmts[0])
	}
	thenRet := ifStmt.Then.Stmts[1].(*ReturnStmt)
	if retIdent, ok := thenRet.Value.(*IdentExpr); !ok || retIdent.Name != thenRetTempName {
		t.Fatalf("expected then return to use temp %q", thenRetTempName)
	}
}

func TestLowerANormalizesNestedCallArgsIntoTemps(t *testing.T) {
	src := `fn add(a: int, b: int): int { return a + b; }

fn main(): void {
    let x = 1;
    let y = 2;
    let z = add(x + 1, y + 2);
    println(str(z));
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	argTemps := 0
	var callTempName string
	for _, stmt := range fn.Body.Stmts {
		switch st := stmt.(type) {
		case *LetStmt:
			if strings.HasPrefix(st.Name, "arg__mir") {
				argTemps++
			}
		case *UnaryOpStmt:
			if strings.HasPrefix(st.Name, "arg__mir") {
				argTemps++
			}
		case *BinaryOpStmt:
			if strings.HasPrefix(st.Name, "arg__mir") {
				argTemps++
			}
		case *CallStmt:
			if st.Func == "add" {
				callTempName = st.Name
				for i, arg := range st.Args {
					if _, ok := arg.(*IdentExpr); ok {
						continue
					}
					if _, ok := arg.(*IntExpr); ok {
						continue
					}
					t.Fatalf("expected call arg %d to be atomized ident or stronger folded int, got %T", i, arg)
				}
			}
		}
	}
	if argTemps < 2 && callTempName == "" {
		t.Fatalf("expected nested call args to be lifted into temps or folded, got temps=%d", argTemps)
	}
	if callTempName == "" {
		t.Fatalf("expected call result temp for normalized add call")
	}
}

func TestLowerSimplifiesConstantExpressions(t *testing.T) {
	src := `fn main(): void {
    let n = (1 + 2) * 3;
    let s = ("ba" + "zic");
    println(str(n));
    println(s);
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var sawInt9 bool
	var sawStringBazic bool
	for _, stmt := range fn.Body.Stmts {
		expr, ok := ValueStmtExpr(stmt)
		if !ok {
			continue
		}
		if got, ok := expr.(*IntExpr); ok && got.Value == 9 {
			sawInt9 = true
		}
		if got, ok := expr.(*StringExpr); ok && got.Value == "bazic" {
			sawStringBazic = true
		}
	}
	if !sawInt9 && !blockContainsExpr(fn.Body, func(e Expr) bool {
		got, ok := e.(*IntExpr)
		return ok && got.Value == 9
	}) && !blockContainsExpr(fn.Body, func(e Expr) bool {
		got, ok := e.(*StringExpr)
		return ok && got.Value == "9"
	}) {
		t.Fatalf("expected some lowered temp or binding to fold to int constant 9")
	}
	if !sawStringBazic && !blockContainsExpr(fn.Body, func(e Expr) bool {
		got, ok := e.(*StringExpr)
		return ok && got.Value == "bazic"
	}) {
		t.Fatalf("expected some lowered temp or binding to fold to string constant bazic")
	}
}

func TestLowerSimplifiesBooleanConditionTemps(t *testing.T) {
	src := `fn main(): void {
    if ((1 + 2) > 1 && true) {
        println("ok");
    }
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var condTemp *LetStmt
	var ifStmt *IfStmt
	var sawPrint bool
	for _, stmt := range fn.Body.Stmts {
		if let, ok := stmt.(*LetStmt); ok && strings.HasPrefix(let.Name, "cond__mir") {
			condTemp = let
		}
		if s, ok := stmt.(*IfStmt); ok {
			ifStmt = s
		}
		if callStmt, ok := stmt.(*CallStmt); ok && callStmt.Func == "println" {
			sawPrint = true
		}
	}
	if condTemp != nil {
		if got, ok := condTemp.Init.(*BoolExpr); !ok || !got.Value {
			t.Fatalf("expected folded bool true condition temp, got %T %#v", condTemp.Init, condTemp.Init)
		}
	}
	if ifStmt != nil {
		if got, ok := ifStmt.Cond.(*BoolExpr); !ok || !got.Value {
			t.Fatalf("expected surviving if condition to be literal true, got %T %#v", ifStmt.Cond, ifStmt.Cond)
		}
		return
	}
	if !sawPrint {
		t.Fatalf("expected constant-true branch to simplify to direct println side effect")
	}
}

func TestLowerPrunesDeadSyntheticTempsAfterFolding(t *testing.T) {
	src := `fn value(): int {
    return (1 + 2) * 3;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	if len(fn.Body.Stmts) != 1 {
		t.Fatalf("expected folded/pruned body to contain one stmt, got %d", len(fn.Body.Stmts))
	}
	ret, ok := fn.Body.Stmts[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected only stmt to be return, got %T", fn.Body.Stmts[0])
	}
	got, ok := ret.Value.(*IntExpr)
	if !ok || got.Value != 9 {
		t.Fatalf("expected pruned return literal 9, got %T %#v", ret.Value, ret.Value)
	}
}

func TestLowerSimplifiesConstantIfAndFalseWhile(t *testing.T) {
	src := `fn main(): void {
    if true {
        println("taken");
    } else {
        println("dead");
    }
    while false {
        println("loop");
    }
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	for _, stmt := range fn.Body.Stmts {
		switch stmt.(type) {
		case *IfStmt:
			t.Fatalf("expected constant if to be simplified away")
		case *WhileStmt:
			t.Fatalf("expected false while to be removed")
		}
	}
	var sawTaken bool
	for _, stmt := range fn.Body.Stmts {
		callStmt, ok := stmt.(*CallStmt)
		if !ok {
			continue
		}
		if callStmt.Func != "println" || len(callStmt.Args) != 1 {
			continue
		}
		if arg, ok := callStmt.Args[0].(*StringExpr); ok && arg.Value == "taken" {
			sawTaken = true
		}
	}
	if !sawTaken {
		t.Fatalf("expected kept constant-true branch body to remain")
	}
}

func TestLowerCanonicalizesCFGConditionAndReturnOperands(t *testing.T) {
	src := `fn compute(a: int, b: int): int {
    if (a + b) > b {
        return a + b;
    }
    return a + b;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	if fn.CFG == nil {
		t.Fatalf("expected cfg on lowered function")
	}
	var sawCFGCond bool
	var sawCFGReturn bool
	var sawStructuredCFGTemp bool
	for _, block := range fn.CFG.Blocks {
		for _, instr := range block.Instrs {
			switch st := instr.(type) {
			case *BinaryOpStmt:
				if strings.HasPrefix(st.Name, "cond__mir") || strings.HasPrefix(st.Name, "ret__mir") {
					sawStructuredCFGTemp = true
				}
			case *CallStmt:
				if strings.HasPrefix(st.Name, "cond__mir") || strings.HasPrefix(st.Name, "ret__mir") {
					sawStructuredCFGTemp = true
				}
			case *FieldAccessStmt:
				if strings.HasPrefix(st.Name, "cond__mir") || strings.HasPrefix(st.Name, "ret__mir") {
					sawStructuredCFGTemp = true
				}
			case *StructLitStmt:
				if strings.HasPrefix(st.Name, "cond__mir") || strings.HasPrefix(st.Name, "ret__mir") {
					sawStructuredCFGTemp = true
				}
			case *MatchValueStmt:
				if strings.HasPrefix(st.Name, "cond__mir") || strings.HasPrefix(st.Name, "ret__mir") {
					sawStructuredCFGTemp = true
				}
			}
		}
		switch term := block.Term.(type) {
		case *CondTerminator:
			sawCFGCond = true
			if _, ok := term.Cond.(*IdentExpr); !ok {
				t.Fatalf("expected cfg condition to be canonicalized to ident, got %T", term.Cond)
			}
		case *ReturnTerminator:
			if term.Value == nil {
				continue
			}
			sawCFGReturn = true
			if _, ok := term.Value.(*IdentExpr); !ok {
				t.Fatalf("expected cfg return value to be canonicalized to ident, got %T", term.Value)
			}
		}
	}
	if !sawCFGCond {
		t.Fatalf("expected at least one cfg conditional terminator")
	}
	if !sawCFGReturn {
		t.Fatalf("expected at least one non-void cfg return terminator")
	}
	if !sawStructuredCFGTemp {
		t.Fatalf("expected cfg operand temps to materialize into structured mir value statements")
	}
}

func TestLowerCanonicalizesCFGMatchSubjectOperand(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn current(flag: bool): Role {
    if flag {
        return Admin;
    }
    return Guest;
}

fn run(flag: bool): void {
    match current(flag) {
        Guest: { println("guest"); }
        Admin: { println("admin"); }
    }
}

fn main(): void {
    run(false);
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[2].(*FuncDecl)
	if fn.CFG == nil {
		t.Fatalf("expected cfg on lowered function")
	}
	var sawMatch bool
	var sawSubjectCallTemp bool
	for _, block := range fn.CFG.Blocks {
		for _, instr := range block.Instrs {
			if st, ok := instr.(*CallStmt); ok && (strings.HasPrefix(st.Name, "match__mir") || strings.HasPrefix(st.Name, "subject__mircfg")) && st.Func == "current" {
				sawSubjectCallTemp = true
			}
		}
		term, ok := block.Term.(*MatchTerminator)
		if !ok {
			continue
		}
		sawMatch = true
		if _, ok := term.Subject.(*IdentExpr); !ok {
			t.Fatalf("expected cfg match subject to be canonicalized to ident, got %T", term.Subject)
		}
	}
	if !sawMatch {
		t.Fatalf("expected at least one cfg match terminator")
	}
	if !sawSubjectCallTemp {
		t.Fatalf("expected cfg match subject temp to materialize into a structured call statement")
	}
}

func TestLowerMaterializesUnaryAndBinaryOpStatements(t *testing.T) {
	src := `fn compute(a: int, b: int): int {
    let sum = a + b;
    let neg = -sum;
    return neg;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var sawBinary bool
	var sawUnary bool
	for _, stmt := range fn.Body.Stmts {
		switch stmt.(type) {
		case *BinaryOpStmt:
			sawBinary = true
		case *UnaryOpStmt:
			sawUnary = true
		}
	}
	if !sawBinary {
		t.Fatalf("expected lowered body to contain binary op statement")
	}
	if !sawUnary {
		t.Fatalf("expected lowered body to contain unary op statement")
	}
}

func TestLowerMaterializesFieldAccessStatements(t *testing.T) {
	src := `struct Pair { left: int; right: int; }

fn compute(a: int, b: int): int {
    let pair = Pair { left: a, right: b };
    let left = pair.left;
    return left;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	var sawField bool
	for _, stmt := range fn.Body.Stmts {
		if _, ok := stmt.(*FieldAccessStmt); ok {
			sawField = true
			break
		}
	}
	if !sawField {
		t.Fatalf("expected lowered body to contain field access statement")
	}
}

func TestLowerMaterializesCallStatements(t *testing.T) {
	src := `fn add(a: int, b: int): int { return a + b; }

fn compute(a: int, b: int): int {
    let sum = add(a, b);
    return sum;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	var sawCall bool
	for _, stmt := range fn.Body.Stmts {
		if _, ok := stmt.(*CallStmt); ok {
			sawCall = true
			break
		}
	}
	if !sawCall {
		t.Fatalf("expected lowered body to contain call statement")
	}
}

func TestLowerMaterializesStructLiteralStatements(t *testing.T) {
	src := `struct Pair { left: int; right: int; }

fn compute(a: int, b: int): int {
    let pair = Pair { left: a, right: b };
    return pair.left;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	var sawStructLit bool
	for _, stmt := range fn.Body.Stmts {
		if _, ok := stmt.(*StructLitStmt); ok {
			sawStructLit = true
			break
		}
	}
	if !sawStructLit {
		t.Fatalf("expected lowered body to contain struct literal statement")
	}
}

func TestLowerMaterializesMatchValueStatements(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn compute(role: Role): string {
    let label = match role {
        Guest: "guest",
        Admin: "admin",
    };
    return label;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	var sawMatch bool
	for _, stmt := range fn.Body.Stmts {
		if _, ok := stmt.(*MatchValueStmt); ok {
			sawMatch = true
			break
		}
	}
	if !sawMatch {
		t.Fatalf("expected lowered body to contain match value statement")
	}
}

func TestLowerAnormalizesAssignRHSIntoTempValueStmt(t *testing.T) {
	src := `fn compute(a: int, b: int): int {
    let out = 0;
    out = a + b;
    return out;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var sawAssignTemp bool
	var sawAssignWithIdent bool
	for _, stmt := range fn.Body.Stmts {
		if st, ok := stmt.(*BinaryOpStmt); ok && strings.HasPrefix(st.Name, "assign__mir") {
			sawAssignTemp = true
		}
		if st, ok := stmt.(*AssignStmt); ok {
			if _, ok := st.Value.(*IdentExpr); ok {
				sawAssignWithIdent = true
			}
		}
	}
	if !sawAssignTemp {
		t.Fatalf("expected non-atomic assign rhs to lower through a named temp value stmt")
	}
	if !sawAssignWithIdent {
		t.Fatalf("expected assignment rhs to be rewritten to an ident temp")
	}
}

func TestLowerAnormalizesNestedAssignTargetReceiverIntoTemp(t *testing.T) {
	src := `struct Pair { left: int; right: int; }
struct Box { pair: Pair; }

fn main(): void {
    let box = Box { pair: Pair { left: 1, right: 2 } };
    box.pair.left = 3;
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[2].(*FuncDecl)
	var sawTargetTemp bool
	var sawAtomicTargetObject bool
	for _, stmt := range fn.Body.Stmts {
		if st, ok := stmt.(*FieldAccessStmt); ok && strings.HasPrefix(st.Name, "target__mir") {
			sawTargetTemp = true
		}
		assign, ok := stmt.(*AssignStmt)
		if !ok {
			continue
		}
		target, ok := assign.Target.(*FieldAccessExpr)
		if !ok {
			continue
		}
		if _, ok := target.Object.(*IdentExpr); ok {
			sawAtomicTargetObject = true
		}
	}
	if !sawAtomicTargetObject {
		t.Fatalf("expected assignment target receiver to be rewritten to an ident temp")
	}
	if !sawTargetTemp && !sawAtomicTargetObject {
		t.Fatalf("expected nested field assignment receiver normalization")
	}
}

func TestLowerPrunesPureExpressionStatements(t *testing.T) {
	src := `fn main(): void {
    1 + 2;
    "dead";
    println("ok");
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	if len(fn.Body.Stmts) != 1 {
		t.Fatalf("expected only side-effecting stmt to remain, got %d stmts", len(fn.Body.Stmts))
	}
	for _, stmt := range fn.Body.Stmts {
		switch st := stmt.(type) {
		case *ExprStmt:
			t.Fatalf("expected pure expression statements to be pruned from lowered mir body")
		case *UnaryOpStmt, *BinaryOpStmt, *FieldAccessStmt, *StructLitStmt, *MatchValueStmt:
			t.Fatalf("expected pure discard value statements to be pruned, got %T", st)
		}
	}
	var sawPrint bool
	for _, stmt := range fn.Body.Stmts {
		if st, ok := stmt.(*CallStmt); ok && st.Func == "println" {
			sawPrint = true
		}
	}
	if !sawPrint {
		t.Fatalf("expected side-effecting print call to remain after pruning pure expr statements")
	}
}

func TestLowerAnormalizesMatchStmtSubjectIntoTempValueStmt(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn choose(): Role {
    return Admin;
}

fn main(): void {
    match choose() {
        Guest: { println("guest"); }
        Admin: { println("admin"); }
    }
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[2].(*FuncDecl)
	var sawMatchTemp bool
	var sawMatchWithIdent bool
	for _, stmt := range fn.Body.Stmts {
		if st, ok := stmt.(*CallStmt); ok && strings.HasPrefix(st.Name, "match__mir") && st.Func == "choose" {
			sawMatchTemp = true
		}
		if st, ok := stmt.(*MatchStmt); ok {
			if _, ok := st.Subject.(*IdentExpr); ok {
				sawMatchWithIdent = true
			}
		}
	}
	if !sawMatchTemp {
		t.Fatalf("expected non-atomic match subject to lower through a named temp value stmt")
	}
	if !sawMatchWithIdent {
		t.Fatalf("expected match stmt subject to be rewritten to an ident temp")
	}
}

func TestLowerSimplifiesMaterializedBinaryStmtToConstantLet(t *testing.T) {
	src := `fn main(): void {
    let n = (1 + 2);
    println(str(n));
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var found bool
	for _, stmt := range fn.Body.Stmts {
		name, _, ok := ValueStmtBinding(stmt)
		if !ok || name != "n" {
			continue
		}
		expr, ok := ValueStmtExpr(stmt)
		if !ok {
			t.Fatalf("expected value stmt expr for n, got %T", stmt)
		}
		intExpr, ok := expr.(*IntExpr)
		if !ok || intExpr.Value != 3 {
			t.Fatalf("expected simplified value stmt n = 3, got %T %#v", expr, expr)
		}
		found = true
	}
	if !found && !blockContainsExpr(fn.Body, func(e Expr) bool {
		v, ok := e.(*IntExpr)
		return ok && v.Value == 3
	}) && !blockContainsExpr(fn.Body, func(e Expr) bool {
		v, ok := e.(*StringExpr)
		return ok && v.Value == "3"
	}) {
		t.Fatalf("expected simplified let binding for n")
	}
}

func TestLowerSimplifiesFieldAccessOnConstantStructLiteral(t *testing.T) {
	src := `struct Pair { left: int; right: int; }

fn main(): void {
    let n = (Pair { left: 1, right: 2 }).left;
    println(str(n));
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	var found bool
	for _, stmt := range fn.Body.Stmts {
		name, _, ok := ValueStmtBinding(stmt)
		if !ok || name != "n" {
			continue
		}
		expr, ok := ValueStmtExpr(stmt)
		if !ok {
			t.Fatalf("expected value stmt expr for n, got %T", stmt)
		}
		intExpr, ok := expr.(*IntExpr)
		if !ok || intExpr.Value != 1 {
			t.Fatalf("expected simplified value stmt n = 1, got %T %#v", expr, expr)
		}
		found = true
	}
	if !found && !blockContainsExpr(fn.Body, func(e Expr) bool {
		v, ok := e.(*IntExpr)
		return ok && v.Value == 1
	}) && !blockContainsExpr(fn.Body, func(e Expr) bool {
		v, ok := e.(*StringExpr)
		return ok && v.Value == "1"
	}) {
		t.Fatalf("expected simplified let binding for n")
	}
}

func TestLowerSimplifiesPureBuiltinCallsOnConstantArgs(t *testing.T) {
	src := `fn main(): void {
    let n = len("héy");
    let ok = contains("bazic", "zi");
    let s = replace("ba-zic", "-", "");
    let t = trim_space("  ok  ");
    let u = str(42);
    let v = repeat("ba", 3);
    println(str(n));
    println(str(ok));
    println(s + t + u + v);
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	got := map[string]Expr{}
	for _, stmt := range fn.Body.Stmts {
		name, _, ok := ValueStmtBinding(stmt)
		if !ok {
			continue
		}
		expr, ok := ValueStmtExpr(stmt)
		if !ok {
			continue
		}
		got[name] = expr
	}
	if v, ok := got["n"].(*IntExpr); !ok || v.Value != 3 {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			if v, ok := e.(*IntExpr); ok && v.Value == 3 {
				return true
			}
			if v, ok := e.(*StringExpr); ok && v.Value == "3" {
				return true
			}
			return false
		}) {
			t.Fatalf("expected n to fold to int 3 or equivalent folded usage, got %T %#v", got["n"], got["n"])
		}
	}
	if v, ok := got["ok"].(*BoolExpr); !ok || !v.Value {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*BoolExpr)
			return ok && v.Value
		}) && !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*StringExpr)
			return ok && v.Value == "true"
		}) {
			t.Fatalf("expected ok to fold to bool true or equivalent folded usage, got %T %#v", got["ok"], got["ok"])
		}
	}
	if v, ok := got["s"].(*StringExpr); !ok || v.Value != "bazic" {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*StringExpr)
			return ok && strings.Contains(v.Value, "bazic")
		}) {
			t.Fatalf("expected s to fold to string bazic or equivalent folded usage, got %T %#v", got["s"], got["s"])
		}
	}
	if v, ok := got["t"].(*StringExpr); !ok || v.Value != "ok" {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*StringExpr)
			return ok && strings.Contains(v.Value, "ok")
		}) {
			t.Fatalf("expected t to fold to string ok or equivalent folded usage, got %T %#v", got["t"], got["t"])
		}
	}
	if v, ok := got["u"].(*StringExpr); !ok || v.Value != "42" {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*StringExpr)
			return ok && strings.Contains(v.Value, "42")
		}) {
			t.Fatalf("expected u to fold to string 42 or equivalent folded usage, got %T %#v", got["u"], got["u"])
		}
	}
	if v, ok := got["v"].(*StringExpr); !ok || v.Value != "bababa" {
		if !blockContainsExpr(fn.Body, func(e Expr) bool {
			v, ok := e.(*StringExpr)
			return ok && strings.Contains(v.Value, "bababa")
		}) {
			t.Fatalf("expected v to fold to string bababa or equivalent folded usage, got %T %#v", got["v"], got["v"])
		}
	}
}

func TestValueStmtExprCoversExplicitMIRValueStatements(t *testing.T) {
	stmts := []Stmt{
		&ConstStmt{Name: "k", Type: baztypes.MustParse(ast.TypeInt), Value: &IntExpr{Value: 1}},
		&CopyStmt{Name: "cp", Type: baztypes.MustParse(ast.TypeInt), Source: &IdentExpr{Name: "src"}},
		&UnaryOpStmt{Name: "u", Type: baztypes.MustParse(ast.TypeInt), Op: "-", Right: &IntExpr{Value: 1}},
		&BinaryOpStmt{Name: "b", Type: baztypes.MustParse(ast.TypeInt), Left: &IntExpr{Value: 1}, Op: "+", Right: &IntExpr{Value: 2}},
		&CallStmt{Name: "c", Type: baztypes.MustParse(ast.TypeInt), Func: "len", Args: []Expr{&StringExpr{Value: "x"}}},
		&FieldAccessStmt{Name: "f", Type: baztypes.MustParse(ast.TypeInt), Object: &IdentExpr{Name: "pair"}, Field: "left"},
		&StructLitStmt{Name: "s", Type: baztypes.MustParse(ast.Type("Pair")), TypeName: "Pair", Fields: []StructLitField{{Name: "left", Value: &IntExpr{Value: 1}}}},
		&MatchValueStmt{Name: "m", Type: baztypes.MustParse(ast.TypeString), Subject: &IdentExpr{Name: "role"}, Arms: []MatchExprArm{{Variant: "Guest", Value: &StringExpr{Value: "guest"}}}},
	}
	for _, stmt := range stmts {
		expr, ok := ValueStmtExpr(stmt)
		if !ok || expr == nil {
			t.Fatalf("expected ValueStmtExpr to cover %T", stmt)
		}
		name, typ, ok := ValueStmtBinding(stmt)
		if !ok || name == "" || typ.Name == "" {
			t.Fatalf("expected ValueStmtBinding to cover %T", stmt)
		}
	}
}

func TestSetValueStmtBindingNameCoversExplicitMIRValueStatements(t *testing.T) {
	stmts := []Stmt{
		&ConstStmt{Name: "k", Type: baztypes.MustParse(ast.TypeInt), Value: &IntExpr{Value: 1}},
		&CopyStmt{Name: "cp", Type: baztypes.MustParse(ast.TypeInt), Source: &IdentExpr{Name: "src"}},
		&UnaryOpStmt{Name: "u", Type: baztypes.MustParse(ast.TypeInt), Op: "-", Right: &IntExpr{Value: 1}},
		&BinaryOpStmt{Name: "b", Type: baztypes.MustParse(ast.TypeInt), Left: &IntExpr{Value: 1}, Op: "+", Right: &IntExpr{Value: 2}},
		&CallStmt{Name: "c", Type: baztypes.MustParse(ast.TypeInt), Func: "len", Args: []Expr{&StringExpr{Value: "x"}}},
		&FieldAccessStmt{Name: "f", Type: baztypes.MustParse(ast.TypeInt), Object: &IdentExpr{Name: "pair"}, Field: "left"},
		&StructLitStmt{Name: "s", Type: baztypes.MustParse(ast.Type("Pair")), TypeName: "Pair", Fields: []StructLitField{{Name: "left", Value: &IntExpr{Value: 1}}}},
		&MatchValueStmt{Name: "m", Type: baztypes.MustParse(ast.TypeString), Subject: &IdentExpr{Name: "role"}, Arms: []MatchExprArm{{Variant: "Guest", Value: &StringExpr{Value: "guest"}}}},
	}
	for _, stmt := range stmts {
		if !SetValueStmtBindingName(stmt, "_") {
			t.Fatalf("expected SetValueStmtBindingName to cover %T", stmt)
		}
		name, _, ok := ValueStmtBinding(stmt)
		if !ok || name != "_" {
			t.Fatalf("expected renamed binding for %T, got %q", stmt, name)
		}
	}
	if SetValueStmtBindingName(&ExprStmt{Expr: &IntExpr{Value: 1}}, "_") {
		t.Fatalf("expected non-value stmt rename to be rejected")
	}
}

func TestMaterializeValueOpsConvertsSimpleConstAndCopyLets(t *testing.T) {
	stmts := []Stmt{
		&LetStmt{Name: "k", Type: baztypes.MustParse(ast.TypeInt), Init: &IntExpr{Value: 1}},
		&LetStmt{Name: "cp", Type: baztypes.MustParse(ast.TypeInt), Init: &IdentExpr{Name: "src"}},
	}
	out := materializeValueOpsStmts(nil, stmts)
	if _, ok := out[0].(*ConstStmt); !ok {
		t.Fatalf("expected int literal let to materialize into ConstStmt, got %T", out[0])
	}
	if _, ok := out[1].(*CopyStmt); !ok {
		t.Fatalf("expected ident let to materialize into CopyStmt, got %T", out[1])
	}
}

func TestValueStmtBookkeepingCoversExplicitMIRValueStatements(t *testing.T) {
	stmt := &MatchValueStmt{
		Name:    "m",
		Type:    baztypes.MustParse(ast.TypeString),
		Subject: &IdentExpr{Name: "role"},
		Arms: []MatchExprArm{
			{
				Variant: "Guest",
				Guard:   &CallExpr{Func: "is_guest", Args: []Expr{&IdentExpr{Name: "role"}}},
				Value:   &FieldAccessExpr{Object: &IdentExpr{Name: "user"}, Field: "name"},
			},
			{
				Variant: "Admin",
				Value:   &StringExpr{Value: "admin"},
			},
		},
	}
	live := map[string]struct{}{}
	if !CollectValueStmtUses(live, stmt) {
		t.Fatalf("expected CollectValueStmtUses to handle %T", stmt)
	}
	for _, want := range []string{"role", "user"} {
		if _, ok := live[want]; !ok {
			t.Fatalf("expected collected live use for %q, got %#v", want, live)
		}
	}
	if !ValueStmtMayHaveSideEffects(stmt) {
		t.Fatalf("expected guarded match value statement to be effectful because of call guard")
	}

	pure := &StructLitStmt{
		Name:     "pair",
		Type:     baztypes.MustParse(ast.Type("Pair")),
		TypeName: "Pair",
		Fields: []StructLitField{
			{Name: "left", Value: &IntExpr{Value: 1}},
			{Name: "right", Value: &IntExpr{Value: 2}},
		},
	}
	if ValueStmtMayHaveSideEffects(pure) {
		t.Fatalf("expected pure struct literal statement to be side-effect free")
	}
}

func TestLowerSimplifiesConstantMatchStmt(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn main(): void {
    match Admin {
        Guest: { println("guest"); }
        Admin: { println("admin"); }
    }
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	for _, stmt := range fn.Body.Stmts {
		if _, ok := stmt.(*MatchStmt); ok {
			t.Fatalf("expected constant match stmt to be simplified away")
		}
	}
	var sawAdmin bool
	for _, stmt := range fn.Body.Stmts {
		callStmt, ok := stmt.(*CallStmt)
		if !ok {
			continue
		}
		if callStmt.Func != "println" || len(callStmt.Args) != 1 {
			continue
		}
		if arg, ok := callStmt.Args[0].(*StringExpr); ok && arg.Value == "admin" {
			sawAdmin = true
		}
	}
	if !sawAdmin {
		t.Fatalf("expected chosen constant match arm body to remain")
	}
}

func TestLowerSimplifiesConstantMatchExpr(t *testing.T) {
	src := `enum Role { Guest, Admin }

fn label(): string {
    return match Admin {
        Guest: "guest",
        Admin: "admin",
    };
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	if len(fn.Body.Stmts) != 1 {
		t.Fatalf("expected simplified return-only body, got %d stmts", len(fn.Body.Stmts))
	}
	ret, ok := fn.Body.Stmts[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected simplified body to contain return, got %T", fn.Body.Stmts[0])
	}
	got, ok := ret.Value.(*StringExpr)
	if !ok || got.Value != "admin" {
		t.Fatalf("expected constant match expr to fold to admin string, got %T %#v", ret.Value, ret.Value)
	}
}

func TestSimplifyMatchStmtArmsDropsConstantFalseAndLaterSameVariantArms(t *testing.T) {
	arms := simplifyMatchArms([]MatchArm{
		{
			Variant: "Guest",
			Guard:   &BoolExpr{Value: false},
			Body:    &Block{Stmts: []Stmt{&ExprStmt{Expr: &StringExpr{Value: "dead"}}}},
		},
		{
			Variant: "Guest",
			Guard:   &BoolExpr{Value: true},
			Body:    &Block{Stmts: []Stmt{&ExprStmt{Expr: &StringExpr{Value: "live"}}}},
		},
		{
			Variant: "Guest",
			Guard:   &CallExpr{Func: "is_guest"},
			Body:    &Block{Stmts: []Stmt{&ExprStmt{Expr: &StringExpr{Value: "later-dead"}}}},
		},
		{
			Variant: "Admin",
			Body:    &Block{Stmts: []Stmt{&ExprStmt{Expr: &StringExpr{Value: "admin"}}}},
		},
	})
	if len(arms) != 2 {
		t.Fatalf("expected 2 reachable match arms, got %d", len(arms))
	}
	if arms[0].Variant != "Guest" || arms[0].Guard != nil {
		t.Fatalf("expected first surviving guest arm to become unguarded, got %#v", arms[0])
	}
	if arms[1].Variant != "Admin" || arms[1].Guard != nil {
		t.Fatalf("expected admin arm to remain unguarded, got %#v", arms[1])
	}
}

func TestSimplifyExprPrunesDeadMatchExprArms(t *testing.T) {
	expr := simplifyExpr(nil, &MatchExpr{
		Subject: &IdentExpr{Name: "role"},
		Arms: []MatchExprArm{
			{
				Variant: "Guest",
				Guard:   &BoolExpr{Value: false},
				Value:   &StringExpr{Value: "dead"},
			},
			{
				Variant: "Guest",
				Guard:   &BoolExpr{Value: true},
				Value:   &StringExpr{Value: "guest"},
			},
			{
				Variant: "Guest",
				Guard:   &CallExpr{Func: "is_guest", Args: []Expr{&IdentExpr{Name: "role"}}},
				Value:   &StringExpr{Value: "later-dead"},
			},
			{
				Variant: "Admin",
				Value:   &StringExpr{Value: "admin"},
			},
		},
	})
	match, ok := expr.(*MatchExpr)
	if !ok {
		t.Fatalf("expected match expr to remain a match expr, got %T", expr)
	}
	if len(match.Arms) != 2 {
		t.Fatalf("expected 2 reachable match expr arms, got %d", len(match.Arms))
	}
	if match.Arms[0].Variant != "Guest" || match.Arms[0].Guard != nil {
		t.Fatalf("expected first surviving guest arm to become unguarded, got %#v", match.Arms[0])
	}
	if match.Arms[1].Variant != "Admin" || match.Arms[1].Guard != nil {
		t.Fatalf("expected admin arm to remain unguarded, got %#v", match.Arms[1])
	}
}

func TestValidateRejectsEntryBlockWithIncomingEdge(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &JumpTerminator{Target: "loop"},
						},
						{
							Name: "loop",
							Term: &JumpTerminator{Target: "entry"},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected incoming-edge-on-entry rejection")
	}
	if !strings.Contains(err.Error(), "cfg entry block 'entry' has incoming edges") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsTypedLetInitializerMismatch(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body: &Block{
					Stmts: []Stmt{
						&LetStmt{
							Name: "label",
							Type: baztypes.MustParse(ast.TypeString),
							Init: &IntExpr{Value: 1},
						},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected typed let mismatch rejection")
	}
	if !strings.Contains(err.Error(), "let 'label' has type int, expected string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsTypedReturnMismatch(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&ReturnStmt{Value: &IntExpr{Value: 1}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected typed return mismatch rejection")
	}
	if !strings.Contains(err.Error(), "return value has type int, expected string") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsTypedCFGConditionMismatch(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&FuncDecl{
				Name:       "main",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Term: &CondTerminator{
								Cond: &IntExpr{Value: 1},
								Then: "yes",
								Else: "no",
							},
						},
						{Name: "yes", Term: &ReturnTerminator{}},
						{Name: "no", Term: &ReturnTerminator{}},
					},
				},
			},
		},
	}
	err := Validate(prog)
	if err == nil {
		t.Fatalf("expected cfg condition type rejection")
	}
	if !strings.Contains(err.Error(), "cfg condition has type int, expected bool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateInfersStructFieldAccessType(t *testing.T) {
	prog := &Program{
		Decls: []Decl{
			&StructDecl{
				Name: "User",
				Fields: []StructField{
					{Name: "name", Type: baztypes.MustParse(ast.TypeString)},
				},
			},
			&FuncDecl{
				Name:       "label",
				ReturnType: baztypes.MustParse(ast.TypeString),
				Body: &Block{
					Stmts: []Stmt{
						&LetStmt{
							Name: "user",
							Type: baztypes.MustParse(ast.Type("User")),
							Init: &StructLitExpr{
								TypeName: "User",
								Fields: []StructLitField{
									{Name: "name", Value: &StringExpr{Value: "Ipeh"}},
								},
							},
						},
						&ReturnStmt{
							Value: &FieldAccessExpr{
								Object: &IdentExpr{Name: "user"},
								Field:  "name",
							},
						},
					},
				},
			},
		},
	}
	if err := Validate(prog); err != nil {
		t.Fatalf("expected struct field type inference to validate, got %v", err)
	}
}

func TestGroupMatchArmsPreservesVariantEncounterOrder(t *testing.T) {
	exprGroups := GroupMatchExprArms([]MatchExprArm{
		{Variant: "B"},
		{Variant: "A"},
		{Variant: "B"},
		{Variant: "C"},
	})
	if len(exprGroups) != 3 {
		t.Fatalf("expected 3 expr groups, got %d", len(exprGroups))
	}
	if exprGroups[0].Variant != "B" || len(exprGroups[0].Arms) != 2 {
		t.Fatalf("unexpected first expr group: %#v", exprGroups[0])
	}
	if exprGroups[1].Variant != "A" || len(exprGroups[1].Arms) != 1 {
		t.Fatalf("unexpected second expr group: %#v", exprGroups[1])
	}
	if exprGroups[2].Variant != "C" || len(exprGroups[2].Arms) != 1 {
		t.Fatalf("unexpected third expr group: %#v", exprGroups[2])
	}

	termGroups := GroupMatchTerminatorArms([]MatchTerminatorArm{
		{Variant: "X"},
		{Variant: "Y"},
		{Variant: "X"},
	})
	if len(termGroups) != 2 {
		t.Fatalf("expected 2 terminator groups, got %d", len(termGroups))
	}
	if termGroups[0].Variant != "X" || len(termGroups[0].Arms) != 2 {
		t.Fatalf("unexpected first terminator group: %#v", termGroups[0])
	}
	if termGroups[1].Variant != "Y" || len(termGroups[1].Arms) != 1 {
		t.Fatalf("unexpected second terminator group: %#v", termGroups[1])
	}
}

func TestProgramUsesCallScansGlobalsBodiesAndCFG(t *testing.T) {
	pred := func(name string) bool { return name == "target_call" }
	prog := &Program{
		Decls: []Decl{
			&GlobalLetDecl{
				Name: "g",
				Type: baztypes.MustParse(ast.TypeInt),
				Init: &CallExpr{
					Func: "target_call",
				},
			},
			&FuncDecl{
				Name:       "body_fn",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body: &Block{
					Stmts: []Stmt{
						&ExprStmt{
							Expr: &MatchExpr{
								Subject: &IntExpr{Value: 1},
								Type:    baztypes.MustParse(ast.TypeInt),
								Arms: []MatchExprArm{
									{Variant: "A", Guard: &CallExpr{Func: "target_call"}, Value: &IntExpr{Value: 1}},
								},
							},
						},
					},
				},
			},
			&FuncDecl{
				Name:       "cfg_fn",
				ReturnType: baztypes.MustParse(ast.TypeVoid),
				Body:       &Block{},
				CFG: &CFG{
					Entry: "entry",
					Blocks: []*BasicBlock{
						{
							Name: "entry",
							Instrs: []Stmt{
								&CallStmt{Name: "_", Func: "other_call"},
							},
							Term: &MatchTerminator{
								Subject: &IntExpr{Value: 0},
								Arms: []MatchTerminatorArm{
									{Variant: "A", Guard: &CallExpr{Func: "target_call"}, Target: "done"},
								},
								Default: "done",
							},
						},
						{Name: "done", Term: &ReturnTerminator{}},
					},
				},
			},
		},
	}
	if !ProgramUsesCall(prog, pred) {
		t.Fatalf("expected ProgramUsesCall to detect target_call")
	}
	if ProgramUsesCall(prog, func(name string) bool { return name == "missing_call" }) {
		t.Fatalf("expected ProgramUsesCall to ignore missing_call")
	}
}

func TestLowerSimplifiesConstantGlobalInitializer(t *testing.T) {
	src := `const LABEL: string = to_upper("bazic")

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	global, ok := out.Decls[0].(*GlobalLetDecl)
	if !ok {
		t.Fatalf("expected first decl to be global, got %T", out.Decls[0])
	}
	got, ok := global.Init.(*StringExpr)
	if !ok || got.Value != "BAZIC" {
		t.Fatalf("expected simplified constant global initializer, got %T %#v", global.Init, global.Init)
	}
}

func TestLowerPropagatesSequentialConstGlobals(t *testing.T) {
	src := `const BASE: int = 1 + 2
const TOTAL: int = BASE + 4

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	total, ok := out.Decls[1].(*GlobalLetDecl)
	if !ok {
		t.Fatalf("expected second decl to be global, got %T", out.Decls[1])
	}
	got, ok := total.Init.(*IntExpr)
	if !ok || got.Value != 7 {
		t.Fatalf("expected propagated constant global initializer 7, got %T %#v", total.Init, total.Init)
	}
}

func TestLowerPropagatesConstGlobalsIntoNestedFunctionBlocks(t *testing.T) {
	src := `const FLAG: bool = 1 == 1

fn choose(): int {
    if FLAG {
        return 1;
    } else {
        return 2;
    }
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	if len(fn.Body.Stmts) != 1 {
		t.Fatalf("expected nested const-driven branch to simplify to one stmt, got %d", len(fn.Body.Stmts))
	}
	ret, ok := fn.Body.Stmts[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected simplified function body to contain return, got %T", fn.Body.Stmts[0])
	}
	got, ok := ret.Value.(*IntExpr)
	if !ok || got.Value != 1 {
		t.Fatalf("expected nested const-driven branch to simplify to return 1, got %T %#v", ret.Value, ret.Value)
	}
}

func TestLowerPropagatesImmutableConstLocals(t *testing.T) {
	src := `fn total(): int {
    const base: int = 1 + 2;
    let out = base + 4;
    return out;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var sawOutSeven bool
	for _, stmt := range fn.Body.Stmts {
		switch st := stmt.(type) {
		case *LetStmt:
			if st.Name == "out" {
				if got, ok := st.Init.(*IntExpr); ok && got.Value == 7 {
					sawOutSeven = true
				}
			}
		case *ConstStmt:
			if st.Name == "out" {
				if got, ok := st.Value.(*IntExpr); ok && got.Value == 7 {
					sawOutSeven = true
				}
			}
		}
	}
	if !sawOutSeven {
		t.Fatalf("expected immutable const local propagation to fold out to 7")
	}
}

func TestLowerKeepsFoldedValueStatementsExplicit(t *testing.T) {
	src := `fn total(): int {
    let n = 1 + 2;
    return n;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	for _, stmt := range fn.Body.Stmts {
		cst, ok := stmt.(*ConstStmt)
		if !ok || cst.Name != "n" {
			continue
		}
		got, ok := cst.Value.(*IntExpr)
		if !ok || got.Value != 3 {
			t.Fatalf("expected n to stay an explicit ConstStmt with value 3, got %T %#v", cst.Value, cst.Value)
		}
		return
	}
	t.Fatalf("expected folded local n to stay an explicit ConstStmt")
}

func TestLowerInvalidatesConstLikeLocalOnFieldAssignment(t *testing.T) {
	src := `struct Pair { left: int; right: int; }

fn value(): int {
    let pair = Pair { left: 1, right: 2 };
    pair.left = 9;
    return pair.left;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[1].(*FuncDecl)
	last, ok := fn.Body.Stmts[len(fn.Body.Stmts)-1].(*ReturnStmt)
	if !ok {
		t.Fatalf("expected function body to end in return, got %T", fn.Body.Stmts[len(fn.Body.Stmts)-1])
	}
	if got, ok := last.Value.(*IntExpr); ok && got.Value == 1 {
		t.Fatalf("expected field assignment to invalidate prior const-like struct binding; stale return %#v", last.Value)
	}
}

func TestLowerDoesNotFoldMutableLoopConditionToConstant(t *testing.T) {
	src := `fn count(): int {
    let i: int = 0;
    while i < 2 {
        i = i + 1;
    }
    return i;
}

fn main(): void {
}`
	prog := parseAndCheck(t, src)
	hp, err := hir.Lower(prog)
	if err != nil {
		t.Fatalf("hir lower failed: %v", err)
	}
	out, err := Lower(hp)
	if err != nil {
		t.Fatalf("mir lower failed: %v", err)
	}
	fn := out.Decls[0].(*FuncDecl)
	var foundWhileCond bool
	for _, block := range fn.CFG.Blocks {
		term, ok := block.Term.(*CondTerminator)
		if !ok {
			continue
		}
		if _, ok := term.Cond.(*BoolExpr); ok {
			t.Fatalf("expected mutable loop condition to remain non-constant in CFG, got %#v", term.Cond)
		}
		foundWhileCond = true
	}
	if !foundWhileCond {
		t.Fatalf("expected lowered cfg to retain loop condition block")
	}
}

func TestDeadSyntheticValueStmtIndexesDropsPureDiscardCFGValues(t *testing.T) {
	block := &BasicBlock{
		Name: "entry",
		Instrs: []Stmt{
			&BinaryOpStmt{
				Name:  "_",
				Type:  baztypes.MustParse(ast.TypeInt),
				Left:  &IntExpr{Value: 1},
				Op:    "+",
				Right: &IntExpr{Value: 2},
			},
			&CallStmt{
				Name: "_",
				Type: baztypes.MustParse(ast.TypeVoid),
				Func: "println",
				Args: []Expr{&StringExpr{Value: "ok"}},
			},
			&BinaryOpStmt{
				Name:  "cond__mir1",
				Type:  baztypes.MustParse(ast.TypeInt),
				Left:  &IntExpr{Value: 3},
				Op:    "+",
				Right: &IntExpr{Value: 4},
			},
		},
		Term: &ReturnTerminator{},
	}
	dead := DeadSyntheticValueStmtIndexes(block)
	if !dead[0] {
		t.Fatalf("expected pure discard cfg value stmt to be marked dead")
	}
	if dead[1] {
		t.Fatalf("expected side-effecting discard cfg call stmt to remain live")
	}
	if !dead[2] {
		t.Fatalf("expected dead synthetic temp to remain marked dead")
	}
}

func TestDeadSyntheticValueStmtIndexesKeepsNamedCFGValuesForInterblockSafety(t *testing.T) {
	block := &BasicBlock{
		Name: "entry",
		Instrs: []Stmt{
			&BinaryOpStmt{
				Name:  "folded_user_local",
				Type:  baztypes.MustParse(ast.TypeInt),
				Left:  &IntExpr{Value: 1},
				Op:    "+",
				Right: &IntExpr{Value: 2},
			},
			&CallStmt{
				Name: "unused_effect",
				Type: baztypes.MustParse(ast.TypeInt),
				Func: "parse_int",
				Args: []Expr{&StringExpr{Value: "42"}},
			},
		},
		Term: &ReturnTerminator{},
	}
	dead := DeadSyntheticValueStmtIndexes(block)
	if dead[0] {
		t.Fatalf("expected named cfg value stmt to remain live under block-local liveness")
	}
	if dead[1] {
		t.Fatalf("expected dead named side-effecting cfg value stmt to remain live")
	}
}

func TestDeadSyntheticValueStmtIndexesDropsPureCFGExprStmt(t *testing.T) {
	block := &BasicBlock{
		Name: "entry",
		Instrs: []Stmt{
			&ExprStmt{Expr: &StringExpr{Value: "cfg-dead"}},
			&ExprStmt{Expr: &CallExpr{Func: "println", Args: []Expr{&StringExpr{Value: "ok"}}}},
		},
		Term: &ReturnTerminator{},
	}
	dead := DeadSyntheticValueStmtIndexes(block)
	if !dead[0] {
		t.Fatalf("expected pure cfg expr stmt to be marked dead")
	}
	if dead[1] {
		t.Fatalf("expected side-effecting cfg expr stmt to remain live")
	}
}

func TestPruneBlockTempsDropsPureDeadNamedValueStmt(t *testing.T) {
	block := &Block{
		Stmts: []Stmt{
			&BinaryOpStmt{
				Name:  "folded_user_local",
				Type:  baztypes.MustParse(ast.TypeInt),
				Left:  &IntExpr{Value: 1},
				Op:    "+",
				Right: &IntExpr{Value: 2},
			},
			&CallStmt{
				Name: "_",
				Type: baztypes.MustParse(ast.TypeVoid),
				Func: "println",
				Args: []Expr{&StringExpr{Value: "ok"}},
			},
		},
	}
	pruneBlockTemps(block)
	if len(block.Stmts) != 1 {
		t.Fatalf("expected pure dead named value stmt to be pruned, got %d stmts", len(block.Stmts))
	}
	call, ok := block.Stmts[0].(*CallStmt)
	if !ok || call.Func != "println" {
		t.Fatalf("expected surviving stmt to be println call, got %T %#v", block.Stmts[0], block.Stmts[0])
	}
}

func TestPruneBlockTempsKeepsNamedValueNeededForLaterAssignment(t *testing.T) {
	block := &Block{
		Stmts: []Stmt{
			&LetStmt{
				Name: "passed",
				Type: baztypes.MustParse(ast.TypeInt),
				Init: &IntExpr{Value: 0},
			},
			&AssignStmt{
				Target: &IdentExpr{Name: "passed"},
				Value:  &IntExpr{Value: 1},
			},
			&CallStmt{
				Name: "_",
				Type: baztypes.MustParse(ast.TypeVoid),
				Func: "println",
				Args: []Expr{&IdentExpr{Name: "passed"}},
			},
		},
	}
	pruneBlockTemps(block)
	if len(block.Stmts) != 3 {
		t.Fatalf("expected declaration to survive later assignment use, got %d stmts", len(block.Stmts))
	}
	if let, ok := block.Stmts[0].(*LetStmt); !ok || let.Name != "passed" {
		t.Fatalf("expected first stmt to remain passed declaration, got %T %#v", block.Stmts[0], block.Stmts[0])
	}
}

func TestPruneBlockTempsRewritesUnusedEffectfulNamedValueToDiscard(t *testing.T) {
	block := &Block{
		Stmts: []Stmt{
			&CallStmt{
				Name: "unused_effect",
				Type: baztypes.MustParse(ast.TypeInt),
				Func: "parse_int",
				Args: []Expr{&StringExpr{Value: "42"}},
			},
		},
	}
	pruneBlockTemps(block)
	if len(block.Stmts) != 1 {
		t.Fatalf("expected effectful stmt to remain, got %d stmts", len(block.Stmts))
	}
	call, ok := block.Stmts[0].(*CallStmt)
	if !ok {
		t.Fatalf("expected surviving stmt to stay a call stmt, got %T %#v", block.Stmts[0], block.Stmts[0])
	}
	if call.Name != "_" {
		t.Fatalf("expected effectful dead named stmt to be rewritten to discard, got %q", call.Name)
	}
}

func TestDeadCFGValueNamesCollectsNamedDeadCFGTemps(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "cond__mir1",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &IntExpr{Value: 1},
							Op:    "+",
							Right: &IntExpr{Value: 2},
						},
						&CallStmt{
							Name: "_",
							Type: baztypes.MustParse(ast.TypeVoid),
							Func: "println",
							Args: []Expr{&StringExpr{Value: "ok"}},
						},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	dead := DeadCFGValueNames(fn)
	if _, ok := dead["cond__mir1"]; !ok {
		t.Fatalf("expected named dead cfg temp to be collected")
	}
}

func TestDeadCFGInstructionIndexesDropsPureNamedCFGValueAcrossBlocks(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "folded_user_local",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &IntExpr{Value: 1},
							Op:    "+",
							Right: &IntExpr{Value: 2},
						},
					},
					Term: &JumpTerminator{Target: "done"},
				},
				{
					Name: "done",
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	deadByBlock := DeadCFGInstructionIndexes(fn)
	if !deadByBlock["entry"][0] {
		t.Fatalf("expected pure named cfg value stmt to be dead across blocks")
	}
	deadNames := DeadCFGValueNames(fn)
	if _, ok := deadNames["folded_user_local"]; !ok {
		t.Fatalf("expected dead named cfg value to be collected")
	}
}

func TestDeadCFGInstructionIndexesKeepsValueUsedBySuccessorBlock(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeInt),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "folded_user_local",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &IntExpr{Value: 1},
							Op:    "+",
							Right: &IntExpr{Value: 2},
						},
					},
					Term: &JumpTerminator{Target: "done"},
				},
				{
					Name: "done",
					Term: &ReturnTerminator{Value: &IdentExpr{Name: "folded_user_local"}},
				},
			},
		},
	}
	deadByBlock := DeadCFGInstructionIndexes(fn)
	if deadByBlock["entry"][0] {
		t.Fatalf("expected cfg value used by successor block to remain live")
	}
}

func TestDiscardUnusedCFGValueBindingsRewritesUnusedEffectfulNames(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&CallStmt{
							Name: "unused_effect",
							Type: baztypes.MustParse(ast.TypeInt),
							Func: "parse_int",
							Args: []Expr{&StringExpr{Value: "42"}},
						},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	DiscardUnusedCFGValueBindings(fn)
	call, ok := fn.CFG.Blocks[0].Instrs[0].(*CallStmt)
	if !ok {
		t.Fatalf("expected call stmt, got %T", fn.CFG.Blocks[0].Instrs[0])
	}
	if call.Name != "_" {
		t.Fatalf("expected unused effectful cfg binding to be rewritten to discard, got %q", call.Name)
	}
}

func TestPruneDeadCFGInstructionsRemovesDeadCFGWork(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "cond__mir1",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &IntExpr{Value: 1},
							Op:    "+",
							Right: &IntExpr{Value: 2},
						},
						&ExprStmt{Expr: &StringExpr{Value: "dead"}},
						&CallStmt{
							Name: "_",
							Type: baztypes.MustParse(ast.TypeVoid),
							Func: "println",
							Args: []Expr{&StringExpr{Value: "live"}},
						},
					},
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	pruneDeadCFGInstructions(fn)
	block := fn.CFG.Blocks[0]
	if len(block.Instrs) != 1 {
		t.Fatalf("expected only live cfg instruction to remain, got %d", len(block.Instrs))
	}
	call, ok := block.Instrs[0].(*CallStmt)
	if !ok || call.Func != "println" {
		t.Fatalf("expected surviving cfg instruction to be live println call, got %T %#v", block.Instrs[0], block.Instrs[0])
	}
}

func TestFinalCFGSimplifyCollapsesEmptyJumpBlockAfterPrune(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "cond__mir1",
							Type:  baztypes.MustParse(ast.TypeInt),
							Left:  &IntExpr{Value: 1},
							Op:    "+",
							Right: &IntExpr{Value: 2},
						},
					},
					Term: &JumpTerminator{Target: "done"},
				},
				{
					Name: "done",
					Term: &ReturnTerminator{},
				},
			},
		},
	}
	pruneDeadCFGInstructions(fn)
	simplifyCFG(nil, fn.CFG)
	if fn.CFG.Entry != "done" {
		t.Fatalf("expected final cfg simplify to collapse empty pruned jump block to done, got %q", fn.CFG.Entry)
	}
	if len(fn.CFG.Blocks) != 1 || fn.CFG.Blocks[0].Name != "done" {
		t.Fatalf("expected final cfg simplify to remove empty pruned jump block, got %+v", fn.CFG.Blocks)
	}
}

func TestSimplifyCFGFuncPropagatesSyntheticTempIntoConditionalTerminator(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&BinaryOpStmt{
							Name:  "cond__mircfg1",
							Type:  baztypes.MustParse(ast.TypeBool),
							Left:  &IntExpr{Value: 1},
							Op:    "==",
							Right: &IntExpr{Value: 1},
						},
					},
					Term: &CondTerminator{
						Cond: &IdentExpr{Name: "cond__mircfg1"},
						Then: "yes",
						Else: "no",
					},
				},
				{Name: "yes", Term: &ReturnTerminator{}},
				{Name: "no", Term: &ReturnTerminator{}},
			},
		},
	}
	prog := &Program{Decls: []Decl{fn}}
	index := newTypeIndex(prog)
	simplifyCFGFunc(index, fn)
	simplifyCFG(newFuncTypeContext(index, fn), fn.CFG)
	switch term := fn.CFG.Blocks[0].Term.(type) {
	case *JumpTerminator:
		if term.Target != "yes" {
			t.Fatalf("expected propagated constant cfg condition to jump to yes, got %q", term.Target)
		}
	case *ReturnTerminator:
		// acceptable stronger form after linear merge
	default:
		t.Fatalf("expected cfg condition to simplify after synthetic-temp propagation, got %T", fn.CFG.Blocks[0].Term)
	}
}

func TestSimplifyCFGFuncPropagatesSyntheticTempIntoMatchTerminator(t *testing.T) {
	fn := &FuncDecl{
		Name:       "main",
		ReturnType: baztypes.MustParse(ast.TypeVoid),
		Body:       &Block{},
		CFG: &CFG{
			Entry: "entry",
			Blocks: []*BasicBlock{
				{
					Name: "entry",
					Instrs: []Stmt{
						&LetStmt{
							Name: "subject__mircfg1",
							Type: baztypes.MustParse(ast.Type("Role")),
							Init: &IdentExpr{Name: "Guest"},
						},
					},
					Term: &MatchTerminator{
						Subject: &IdentExpr{Name: "subject__mircfg1"},
						Arms: []MatchTerminatorArm{
							{Variant: "Guest", Target: "guest"},
							{Variant: "Admin", Target: "admin"},
						},
					},
				},
				{Name: "guest", Term: &ReturnTerminator{}},
				{Name: "admin", Term: &ReturnTerminator{}},
			},
		},
	}
	prog := &Program{
		Decls: []Decl{
			&EnumDecl{Name: "Role", Variants: []string{"Guest", "Admin"}},
			fn,
		},
	}
	index := newTypeIndex(prog)
	simplifyCFGFunc(index, fn)
	simplifyCFG(newFuncTypeContext(index, fn), fn.CFG)
	switch term := fn.CFG.Blocks[0].Term.(type) {
	case *JumpTerminator:
		if term.Target != "guest" {
			t.Fatalf("expected propagated constant cfg match subject to jump to guest, got %q", term.Target)
		}
	case *ReturnTerminator:
		// acceptable stronger form after linear merge
	default:
		t.Fatalf("expected cfg match to simplify after synthetic-temp propagation, got %T", fn.CFG.Blocks[0].Term)
	}
}

func blockContainsExpr(b *Block, pred func(Expr) bool) bool {
	if b == nil {
		return false
	}
	for _, stmt := range b.Stmts {
		if stmtContainsExpr(stmt, pred) {
			return true
		}
	}
	return false
}

func stmtContainsExpr(s Stmt, pred func(Expr) bool) bool {
	switch st := s.(type) {
	case *LetStmt:
		return exprContains(st.Init, pred)
	case *UnaryOpStmt, *BinaryOpStmt, *CallStmt, *FieldAccessStmt, *StructLitStmt, *MatchValueStmt:
		expr, ok := ValueStmtExpr(st)
		return ok && exprContains(expr, pred)
	case *AssignStmt:
		return exprContains(st.Target, pred) || exprContains(st.Value, pred)
	case *IfStmt:
		return exprContains(st.Cond, pred) || blockContainsExpr(st.Then, pred) || blockContainsExpr(st.Else, pred)
	case *WhileStmt:
		return exprContains(st.Cond, pred) || blockContainsExpr(st.Body, pred)
	case *MatchStmt:
		if exprContains(st.Subject, pred) {
			return true
		}
		for _, arm := range st.Arms {
			if exprContains(arm.Guard, pred) || blockContainsExpr(arm.Body, pred) {
				return true
			}
		}
	case *ReturnStmt:
		return exprContains(st.Value, pred)
	case *ExprStmt:
		return exprContains(st.Expr, pred)
	case *Block:
		return blockContainsExpr(st, pred)
	}
	return false
}

func exprContains(e Expr, pred func(Expr) bool) bool {
	if e == nil {
		return false
	}
	if pred(e) {
		return true
	}
	switch ex := e.(type) {
	case *UnaryExpr:
		return exprContains(ex.Right, pred)
	case *BinaryExpr:
		return exprContains(ex.Left, pred) || exprContains(ex.Right, pred)
	case *CallExpr:
		for _, arg := range ex.Args {
			if exprContains(arg, pred) {
				return true
			}
		}
	case *FieldAccessExpr:
		return exprContains(ex.Object, pred)
	case *StructLitExpr:
		for _, field := range ex.Fields {
			if exprContains(field.Value, pred) {
				return true
			}
		}
	case *MatchExpr:
		if exprContains(ex.Subject, pred) {
			return true
		}
		for _, arm := range ex.Arms {
			if exprContains(arm.Guard, pred) || exprContains(arm.Value, pred) {
				return true
			}
		}
	}
	return false
}

func parseAndCheck(t *testing.T, src string) *ast.Program {
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
