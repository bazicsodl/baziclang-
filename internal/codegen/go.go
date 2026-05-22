package codegen

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/backendmeta"
	"baziclang/internal/hir"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

func GenerateGo(p *ast.Program) (string, error) {
	hp, err := hir.Lower(p)
	if err != nil {
		return "", err
	}
	mp, err := mir.Lower(hp)
	if err != nil {
		return "", err
	}
	renderer := goProgramRenderer{plan: backendmeta.CollectGoProgramPlan(mp, os.Getenv("BAZIC_TARGET"))}
	return renderer.render()
}

type goProgramRenderer struct {
	plan backendmeta.GoProgramPlan
}

type goTypeDeclRenderPlan struct {
	structs    []*mir.StructDecl
	interfaces []*mir.InterfaceDecl
	enums      []*mir.EnumDecl
}

type goGlobalRenderPlan struct {
	globals []*mir.GlobalLetDecl
}

type goFunctionLoopPlan struct {
	funcs []*mir.FuncDecl
}

type goTypeDeclItemPlan struct {
	structDecl    *mir.StructDecl
	interfaceDecl *mir.InterfaceDecl
	enumDecl      *mir.EnumDecl
}

type goGlobalItemPlan struct {
	global *mir.GlobalLetDecl
}

func buildGoTypeDeclRenderPlan(shape backendmeta.ProgramShapeMeta) goTypeDeclRenderPlan {
	return goTypeDeclRenderPlan{
		structs:    shape.StructNodes,
		interfaces: shape.InterfaceNodes,
		enums:      shape.EnumNodes,
	}
}

func buildGoGlobalRenderPlan(shape backendmeta.ProgramShapeMeta) goGlobalRenderPlan {
	return goGlobalRenderPlan{globals: shape.GlobalNodes}
}

func buildGoFunctionLoopPlan(shape backendmeta.ProgramShapeMeta) goFunctionLoopPlan {
	return goFunctionLoopPlan{funcs: shape.OrderedFuncs}
}

func (r goProgramRenderer) renderPrelude() string {
	return r.plan.Prelude
}

func (r goProgramRenderer) renderTypeDecls() (string, error) {
	return buildGoTypeDeclRenderPlan(r.plan.Shape).render()
}

func (p goTypeDeclRenderPlan) render() (string, error) {
	var b strings.Builder
	for _, decl := range p.structs {
		s, err := goTypeDeclItemPlan{structDecl: decl}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(s + "\n")
	}
	for _, decl := range p.interfaces {
		s, err := goTypeDeclItemPlan{interfaceDecl: decl}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(s + "\n")
	}
	for _, decl := range p.enums {
		s, err := goTypeDeclItemPlan{enumDecl: decl}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(s + "\n")
	}
	return b.String(), nil
}

func (r goProgramRenderer) renderGlobals() (string, error) {
	return buildGoGlobalRenderPlan(r.plan.Shape).render()
}

func (p goGlobalRenderPlan) render() (string, error) {
	var b strings.Builder
	for _, g := range p.globals {
		line, err := goGlobalItemPlan{global: g}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(line + "\n")
	}
	return b.String(), nil
}

func (p goTypeDeclItemPlan) render() (string, error) {
	switch {
	case p.structDecl != nil:
		return genStructMIR(p.structDecl)
	case p.interfaceDecl != nil:
		return genInterfaceMIR(p.interfaceDecl)
	case p.enumDecl != nil:
		return genEnumMIR(p.enumDecl)
	default:
		return "", nil
	}
}

func (p goGlobalItemPlan) render() (string, error) {
	if p.global == nil {
		return "", nil
	}
	return genGlobalMIR(p.global)
}

func (r goProgramRenderer) renderFunctions() (string, error) {
	return buildGoFunctionLoopPlan(r.plan.Shape).render()
}

func (p goFunctionLoopPlan) render() (string, error) {
	var b strings.Builder
	for _, fn := range p.funcs {
		s, err := genFuncMIR(fn)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (r goProgramRenderer) render() (string, error) {
	typeDecls, err := r.renderTypeDecls()
	if err != nil {
		return "", err
	}
	globals, err := r.renderGlobals()
	if err != nil {
		return "", err
	}
	functions, err := r.renderFunctions()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(r.renderPrelude())
	b.WriteString(typeDecls)
	b.WriteString(globals)
	shape := r.plan.Shape
	if len(shape.StructNodes)+len(shape.InterfaceNodes)+len(shape.EnumNodes)+len(shape.GlobalNodes) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(functions)
	return b.String(), nil
}

func genStructMIR(s *mir.StructDecl) (string, error) {
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(s.Name)
	if len(s.TypeParams) > 0 {
		b.WriteString("[")
		for i, tp := range s.TypeParams {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(tp)
			b.WriteString(" any")
		}
		b.WriteString("]")
	}
	b.WriteString(" struct {\n")
	for _, f := range s.Fields {
		b.WriteString("\t")
		b.WriteString(exportName(f.Name))
		b.WriteString(" ")
		b.WriteString(mapHIRType(f.Type))
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func genInterfaceMIR(i *mir.InterfaceDecl) (string, error) {
	var b strings.Builder
	b.WriteString("type ")
	b.WriteString(i.Name)
	b.WriteString(" interface {\n")
	for _, m := range i.Methods {
		b.WriteString("\t")
		b.WriteString(exportName(m.Name))
		b.WriteString("(")
		for idx, p := range m.Params {
			if idx > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
			b.WriteString(" ")
			b.WriteString(mapHIRType(p.Type))
		}
		b.WriteString(")")
		if baztypes.ToAST(m.Return) != ast.TypeVoid {
			b.WriteString(" ")
			b.WriteString(mapHIRType(m.Return))
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func genEnumMIR(e *mir.EnumDecl) (string, error) {
	var b strings.Builder
	enumName := e.Name
	b.WriteString("type ")
	b.WriteString(enumName)
	b.WriteString(" int\n\n")
	b.WriteString("const (\n")
	for i, v := range e.Variants {
		b.WriteString("\t")
		b.WriteString(v)
		b.WriteString(" ")
		b.WriteString(enumName)
		b.WriteString(" = ")
		if i == 0 {
			b.WriteString("iota")
		} else {
			b.WriteString("iota")
		}
		b.WriteString("\n")
	}
	b.WriteString(")\n")
	return b.String(), nil
}

func genGlobalMIR(g *mir.GlobalLetDecl) (string, error) {
	rhs, err := genExprMIR(g.Init)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("var %s %s = %s", g.Name, mapHIRType(g.Type), rhs), nil
}

type goFuncRenderer struct {
	plan goFuncRenderPlan
}

type goFuncRenderPlan struct {
	fn          *mir.FuncDecl
	topology    *mir.CFGTopology
	deadByBlock map[string]map[int]bool
	deadNames   map[string]struct{}
}

type goCFGBlockRenderPlan struct {
	block   *mir.BasicBlock
	deadCFG map[int]bool
}

type goValueStmtRenderPlan struct {
	stmt    mir.Stmt
	name    string
	typ     baztypes.Type
	expr    mir.Expr
	effects bool
}

type goLinearStmtRenderPlan struct {
	stmt mir.Stmt
	info struct {
		Kind   string
		Target mir.Expr
		Value  mir.Expr
	}
}

type goTerminatorRenderPlan struct {
	term          mir.Terminator
	kind          string
	value         mir.Expr
	cond          mir.Expr
	subject       mir.Expr
	target        string
	thenTarget    string
	elseTarget    string
	defaultTarget string
	matchArms     []mir.MatchTerminatorArm
}

type goExprRenderPlan struct {
	expr mir.Expr
}

type goGuardedMatchExprPlan struct {
	arms []mir.MatchExprArm
}

type goGuardedMatchTerminatorPlan struct {
	arms          []mir.MatchTerminatorArm
	defaultTarget string
}

type goCallExprPlan struct {
	funcName string
	args     []mir.Expr
}

type goFieldAccessExprPlan struct {
	object mir.Expr
	field  string
}

type goStructLitExprPlan struct {
	typeName string
	fields   []mir.StructLitField
}

type goMatchExprPlan struct {
	subject mir.Expr
	arms    []mir.MatchExprArm
	typ     baztypes.Type
}

type goAtomicExprPlan struct {
	expr mir.Expr
}

type goAssignTargetPlan struct {
	expr mir.Expr
}

func buildGoFuncRenderPlan(fn *mir.FuncDecl) (goFuncRenderPlan, error) {
	if err := requireMIRCFG(fn); err != nil {
		return goFuncRenderPlan{}, err
	}
	liveness := mir.AnalyzeCFGLiveness(fn)
	topology := (*mir.CFGTopology)(nil)
	deadByBlock := map[string]map[int]bool{}
	if liveness != nil {
		topology = liveness.Topology
		deadByBlock = liveness.DeadByBlock
	}
	if topology == nil {
		topology, _ = mir.AnalyzeCFG(fn.CFG)
	}
	return goFuncRenderPlan{
		fn:          fn,
		topology:    topology,
		deadByBlock: deadByBlock,
		deadNames:   mir.DeadCFGValueNamesFromAnalysis(fn, liveness),
	}, nil
}

func genFuncMIR(fn *mir.FuncDecl) (string, error) {
	plan, err := buildGoFuncRenderPlan(fn)
	if err != nil {
		return "", err
	}
	return goFuncRenderer{plan: plan}.render()
}

func (r goFuncRenderer) renderSignature() string {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(r.plan.fn.Name)
	if len(r.plan.fn.TypeParams) > 0 {
		b.WriteString("[")
		for i, tp := range r.plan.fn.TypeParams {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(tp)
			b.WriteString(" any")
		}
		b.WriteString("]")
	}
	b.WriteString("(")
	for i, p := range r.plan.fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		b.WriteString(" ")
		b.WriteString(mapHIRType(p.Type))
	}
	b.WriteString(")")
	if !isHIRVoid(r.plan.fn.ReturnType) {
		b.WriteString(" ")
		b.WriteString(mapHIRType(r.plan.fn.ReturnType))
	}
	b.WriteString(" {\n")
	return b.String()
}

func (r goFuncRenderer) renderSingleBlockBody() (string, bool, error) {
	if r.plan.topology == nil || len(r.plan.topology.ReversePostOrderNames()) != 1 {
		return "", false, nil
	}
	blockName := r.plan.topology.ReversePostOrderNames()[0]
	body, err := genSingleBlockFuncMIR(r.plan.topology.Blocks[blockName], r.plan.deadByBlock[blockName])
	if err != nil {
		return "", true, err
	}
	return body, true, nil
}

func (r goFuncRenderer) renderLocalDecls() string {
	lets := collectCFGLetTypes(r.plan.fn)
	var b strings.Builder
	if len(lets) > 0 {
		names := make([]string, 0, len(lets))
		for name := range lets {
			if _, dead := r.plan.deadNames[name]; dead {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			b.WriteString("\tvar ")
			b.WriteString(name)
			b.WriteString(" ")
			b.WriteString(mapHIRType(lets[name]))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (r goFuncRenderer) renderCFGLoop() (string, error) {
	var b strings.Builder
	b.WriteString("\t__bazic_block := ")
	b.WriteString(strconv.Quote(r.plan.fn.CFG.Entry))
	b.WriteString("\n")
	b.WriteString("\tfor {\n")
	b.WriteString("\t\tswitch __bazic_block {\n")
	for _, name := range r.plan.topology.ReversePostOrderNames() {
		blockCode, err := renderGoCFGBlock(goCFGBlockRenderPlan{
			block:   r.plan.topology.Blocks[name],
			deadCFG: r.plan.deadByBlock[name],
		})
		if err != nil {
			return "", err
		}
		b.WriteString(blockCode)
	}
	b.WriteString("\t\tdefault:\n")
	b.WriteString("\t\t\tpanic(\"invalid mir cfg block\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	return b.String(), nil
}

func (r goFuncRenderer) renderMultiBlockBody() (string, error) {
	loop, err := r.renderCFGLoop()
	if err != nil {
		return "", err
	}
	return r.renderLocalDecls() + loop, nil
}

func (r goFuncRenderer) render() (string, error) {
	var b strings.Builder
	b.WriteString(r.renderSignature())
	if body, ok, err := r.renderSingleBlockBody(); err != nil {
		return "", err
	} else if ok {
		b.WriteString(indent(body))
		b.WriteString("}\n")
		return b.String(), nil
	}
	body, err := r.renderMultiBlockBody()
	if err != nil {
		return "", err
	}
	b.WriteString(body)
	b.WriteString("}\n")
	return b.String(), nil
}

func renderGoCFGBlock(plan goCFGBlockRenderPlan) (string, error) {
	var b strings.Builder
	b.WriteString("\t\tcase ")
	b.WriteString(strconv.Quote(plan.block.Name))
	b.WriteString(":\n")
	for i, instr := range plan.block.Instrs {
		if plan.deadCFG[i] {
			continue
		}
		line, err := genCFGInstrMIR(instr)
		if err != nil {
			return "", err
		}
		b.WriteString(indent(indent(line)))
	}
	term, err := genTerminatorMIR(plan.block.Term)
	if err != nil {
		return "", err
	}
	b.WriteString(indent(indent(term)))
	return b.String(), nil
}

func genSingleBlockFuncMIR(block *mir.BasicBlock, deadSynthetic map[int]bool) (string, error) {
	var b strings.Builder
	for i, instr := range block.Instrs {
		line, err := genLinearStmtMIR(instr, deadSynthetic[i])
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}
	term, err := genLinearTerminatorMIR(block.Term)
	if err != nil {
		return "", err
	}
	b.WriteString(term)
	return b.String(), nil
}

func requireMIRCFG(fn *mir.FuncDecl) error {
	if fn == nil {
		return fmt.Errorf("codegen: nil mir function")
	}
	if fn.CFG == nil {
		return fmt.Errorf("codegen: function '%s' missing mir cfg", fn.Name)
	}
	if !mir.HasUniqueCFGBindings(fn) {
		return fmt.Errorf("codegen: function '%s' has non-unique mir cfg bindings", fn.Name)
	}
	return nil
}

func collectCFGLetTypes(fn *mir.FuncDecl) map[string]baztypes.Type {
	return mir.CollectCFGLetTypes(fn)
}

func genCallExprMIRParts(funcName string, args []mir.Expr) (string, error) {
	return goCallExprPlan{funcName: funcName, args: args}.render()
}

func genFieldAccessExprMIRParts(object mir.Expr, field string) (string, error) {
	return goFieldAccessExprPlan{object: object, field: field}.render()
}

func genStructLitExprMIRParts(typeName string, fields []mir.StructLitField) (string, error) {
	return goStructLitExprPlan{typeName: typeName, fields: fields}.render()
}

func genMatchExprMIRParts(subject mir.Expr, arms []mir.MatchExprArm, typ baztypes.Type) (string, error) {
	return goMatchExprPlan{subject: subject, arms: arms, typ: typ}.render()
}

func (p goCallExprPlan) render() (string, error) {
	parts := make([]string, 0, len(p.args))
	for _, a := range p.args {
		s, err := genExprMIR(a)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	target := p.funcName
	if spec, ok := intrinsics.LookupLoweredBuiltin(p.funcName); ok && spec.GoTarget != "" {
		target = spec.GoTarget
	}
	return fmt.Sprintf("%s(%s)", target, strings.Join(parts, ", ")), nil
}

func (p goFieldAccessExprPlan) render() (string, error) {
	base, err := genExprMIR(p.object)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", base, exportName(p.field)), nil
}

func (p goStructLitExprPlan) render() (string, error) {
	parts := make([]string, 0, len(p.fields))
	ordered := append([]mir.StructLitField{}, p.fields...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, f := range ordered {
		v, err := genExprMIR(f.Value)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%s: %s", exportName(f.Name), v))
	}
	return fmt.Sprintf("%s{%s}", mapType(ast.Type(p.typeName)), strings.Join(parts, ", ")), nil
}

func (p goMatchExprPlan) render() (string, error) {
	subj, err := genExprMIR(p.subject)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("func() ")
	b.WriteString(mapHIRType(p.typ))
	b.WriteString(" {\n")
	b.WriteString("switch ")
	b.WriteString(subj)
	b.WriteString(" {\n")
	grouped := mir.GroupMatchExprArms(p.arms)
	for _, gv := range grouped {
		b.WriteString("case ")
		b.WriteString(gv.Variant)
		b.WriteString(":\n")
		chain, err := genGuardChainExprMIR(gv.Arms)
		if err != nil {
			return "", err
		}
		b.WriteString(indent(chain))
	}
	b.WriteString("}\n")
	b.WriteString("panic(\"unreachable match\")\n")
	b.WriteString("}()")
	return b.String(), nil
}

func genValueStmtRHSMIR(s mir.Stmt) (string, error) {
	plan, err := buildGoValueStmtRenderPlan(s)
	if err != nil {
		return "", err
	}
	return plan.renderRHS()
}

func genValueStmtAssignMIR(s mir.Stmt) (string, error) {
	plan, err := buildGoValueStmtRenderPlan(s)
	if err != nil {
		return "", err
	}
	return plan.renderAssign()
}

func genValueStmtDeclMIR(s mir.Stmt, dropBinding bool) (string, error) {
	plan, err := buildGoValueStmtRenderPlan(s)
	if err != nil {
		return "", err
	}
	return plan.renderDecl(dropBinding)
}

func genCFGInstrMIR(s mir.Stmt) (string, error) {
	if mir.IsValueStmt(s) {
		return genValueStmtAssignMIR(s)
	}
	plan, err := buildGoLinearStmtRenderPlan(s)
	if err != nil {
		return "", err
	}
	return plan.renderCFG()
}

func genLinearStmtMIR(s mir.Stmt, dropBinding bool) (string, error) {
	if mir.IsValueStmt(s) {
		return genValueStmtDeclMIR(s, dropBinding)
	}
	plan, err := buildGoLinearStmtRenderPlan(s)
	if err != nil {
		return "", err
	}
	return plan.renderLinear(dropBinding)
}

func buildGoValueStmtRenderPlan(s mir.Stmt) (goValueStmtRenderPlan, error) {
	info, ok := mir.ValueStmtInfo(s)
	if !ok {
		return goValueStmtRenderPlan{}, fmt.Errorf("codegen: unsupported mir value statement %T", s)
	}
	return goValueStmtRenderPlan{
		stmt:    s,
		name:    info.Name,
		typ:     info.Type,
		expr:    info.Expr,
		effects: mir.ValueStmtMayHaveSideEffects(s),
	}, nil
}

func (p goValueStmtRenderPlan) renderRHS() (string, error) {
	return genExprMIR(p.expr)
}

func (p goValueStmtRenderPlan) renderAssign() (string, error) {
	rhs, err := p.renderRHS()
	if err != nil {
		return "", err
	}
	if p.name == "_" {
		if !p.effects {
			return "_ = " + rhs + "\n", nil
		}
		return rhs + "\n", nil
	}
	return fmt.Sprintf("%s = %s\n", p.name, rhs), nil
}

func (p goValueStmtRenderPlan) renderDecl(dropBinding bool) (string, error) {
	rhs, err := p.renderRHS()
	if err != nil {
		return "", err
	}
	if p.name == "_" {
		if !p.effects {
			return "_ = " + rhs + "\n", nil
		}
		return rhs + "\n", nil
	}
	if dropBinding {
		if !p.effects {
			return "", nil
		}
		return rhs + "\n", nil
	}
	return fmt.Sprintf("var %s %s = %s\n", p.name, mapHIRType(p.typ), rhs), nil
}

func buildGoLinearStmtRenderPlan(s mir.Stmt) (goLinearStmtRenderPlan, error) {
	info, ok := mir.LinearStmtInfo(s)
	if !ok {
		return goLinearStmtRenderPlan{}, fmt.Errorf("codegen: unsupported linear mir statement %T", s)
	}
	return goLinearStmtRenderPlan{
		stmt: s,
		info: struct {
			Kind   string
			Target mir.Expr
			Value  mir.Expr
		}{Kind: info.Kind, Target: info.Target, Value: info.Value},
	}, nil
}

func (p goLinearStmtRenderPlan) renderAssignExpr() (string, error) {
	rhs, err := genExprMIR(p.info.Value)
	if err != nil {
		return "", err
	}
	target, err := genAssignTargetMIR(p.info.Target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s = %s\n", target, rhs), nil
}

func (p goLinearStmtRenderPlan) renderExpr() (string, error) {
	e, err := genExprMIR(p.info.Value)
	if err != nil {
		return "", err
	}
	return e + "\n", nil
}

func (p goLinearStmtRenderPlan) renderCFG() (string, error) {
	if p.info.Target != nil {
		return p.renderAssignExpr()
	}
	return p.renderExpr()
}

func (p goLinearStmtRenderPlan) renderLinear(dropBinding bool) (string, error) {
	if p.info.Target != nil {
		return p.renderAssignExpr()
	}
	if dropBinding {
		return "", nil
	}
	return p.renderExpr()
}

func genTerminatorMIR(t mir.Terminator) (string, error) {
	plan, err := buildGoTerminatorRenderPlan(t)
	if err != nil {
		return "", err
	}
	return plan.renderCFG()
}

func genLinearTerminatorMIR(t mir.Terminator) (string, error) {
	plan, err := buildGoTerminatorRenderPlan(t)
	if err != nil {
		return "", err
	}
	return plan.renderLinear()
}

func buildGoTerminatorRenderPlan(t mir.Terminator) (goTerminatorRenderPlan, error) {
	info, ok := mir.TerminatorInfo(t)
	if !ok {
		return goTerminatorRenderPlan{}, fmt.Errorf("codegen: unsupported mir terminator %T", t)
	}
	return goTerminatorRenderPlan{
		term:          t,
		kind:          info.Kind,
		value:         info.Value,
		cond:          info.Cond,
		subject:       info.Subject,
		target:        info.Target,
		thenTarget:    info.Then,
		elseTarget:    info.Else,
		defaultTarget: info.Default,
		matchArms:     info.Arms,
	}, nil
}

func (p goTerminatorRenderPlan) renderReturn() (string, error) {
	if p.value == nil {
		return "return\n", nil
	}
	e, err := genExprMIR(p.value)
	if err != nil {
		return "", err
	}
	return "return " + e + "\n", nil
}

func (p goTerminatorRenderPlan) renderJump() (string, error) {
	return fmt.Sprintf("__bazic_block = %q\ncontinue\n", p.target), nil
}

func (p goTerminatorRenderPlan) renderCond() (string, error) {
	cond, err := genExprMIR(p.cond)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("if %s {\n\t__bazic_block = %q\n\tcontinue\n}\n__bazic_block = %q\ncontinue\n", cond, p.thenTarget, p.elseTarget), nil
}

func (p goTerminatorRenderPlan) renderMatch() (string, error) {
	subject, err := genExprMIR(p.subject)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("switch ")
	b.WriteString(subject)
	b.WriteString(" {\n")
	grouped := mir.GroupMatchTerminatorArms(p.matchArms)
	for _, gv := range grouped {
		b.WriteString("case ")
		b.WriteString(gv.Variant)
		b.WriteString(":\n")
		chain, err := genGuardChainTerminatorMIR(gv.Arms, p.defaultTarget)
		if err != nil {
			return "", err
		}
		b.WriteString(indent(chain))
	}
	if p.defaultTarget != "" {
		b.WriteString("default:\n")
		b.WriteString(indent(fmt.Sprintf("__bazic_block = %q\ncontinue\n", p.defaultTarget)))
	}
	b.WriteString("}\n")
	b.WriteString("panic(\"unreachable match terminator\")\n")
	return b.String(), nil
}

func (p goTerminatorRenderPlan) renderCFG() (string, error) {
	switch p.kind {
	case "return":
		return p.renderReturn()
	case "jump":
		return p.renderJump()
	case "cond":
		return p.renderCond()
	case "match":
		return p.renderMatch()
	default:
		return "", fmt.Errorf("codegen: unsupported mir terminator %T", p.term)
	}
}

func (p goTerminatorRenderPlan) renderLinear() (string, error) {
	if p.kind != "return" {
		return "", fmt.Errorf("codegen: unsupported single-block mir terminator %T", p.term)
	}
	return p.renderReturn()
}

func genExprMIR(e mir.Expr) (string, error) {
	return goExprRenderPlan{expr: e}.render()
}

func genAtomicExprMIR(e mir.Expr) (string, bool, error) {
	return goAtomicExprPlan{expr: e}.render()
}

func (p goAtomicExprPlan) render() (string, bool, error) {
	switch ex := p.expr.(type) {
	case *mir.IdentExpr:
		return ex.Name, true, nil
	case *mir.IntExpr:
		return fmt.Sprintf("int64(%s)", strconv.FormatInt(ex.Value, 10)), true, nil
	case *mir.FloatExpr:
		return strconv.FormatFloat(ex.Value, 'f', -1, 64), true, nil
	case *mir.BoolExpr:
		if ex.Value {
			return "true", true, nil
		}
		return "false", true, nil
	case *mir.StringExpr:
		return strconv.Quote(ex.Value), true, nil
	case *mir.NilExpr:
		return "", true, fmt.Errorf("codegen: nil literal is unsupported; semantic validation should reject it")
	default:
		return "", false, nil
	}
}

func (p goExprRenderPlan) render() (string, error) {
	if out, ok, err := genAtomicExprMIR(p.expr); ok || err != nil {
		return out, err
	}
	switch ex := p.expr.(type) {
	case *mir.UnaryExpr:
		return p.renderUnary(ex)
	case *mir.BinaryExpr:
		return p.renderBinary(ex)
	case *mir.CallExpr:
		return genCallExprMIRParts(ex.Func, ex.Args)
	case *mir.FieldAccessExpr:
		return genFieldAccessExprMIRParts(ex.Object, ex.Field)
	case *mir.StructLitExpr:
		return genStructLitExprMIRParts(ex.TypeName, ex.Fields)
	case *mir.MatchExpr:
		return genMatchExprMIRParts(ex.Subject, ex.Arms, ex.Type)
	default:
		return "", fmt.Errorf("codegen: unsupported mir expression")
	}
}

func (p goExprRenderPlan) renderUnary(ex *mir.UnaryExpr) (string, error) {
	r, err := genExprMIR(ex.Right)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s%s)", ex.Op, r), nil
}

func (p goExprRenderPlan) renderBinary(ex *mir.BinaryExpr) (string, error) {
	l, err := genExprMIR(ex.Left)
	if err != nil {
		return "", err
	}
	r, err := genExprMIR(ex.Right)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s %s)", l, ex.Op, r), nil
}

func genAssignTargetMIR(e mir.Expr) (string, error) {
	return goAssignTargetPlan{expr: e}.render()
}

func (p goAssignTargetPlan) render() (string, error) {
	switch ex := p.expr.(type) {
	case *mir.IdentExpr:
		return ex.Name, nil
	case *mir.FieldAccessExpr:
		return genExprMIR(ex)
	default:
		return "", fmt.Errorf("codegen: invalid mir assignment target")
	}
}

func mapType(t ast.Type) string {
	switch t {
	case ast.TypeInt:
		return "int64"
	case ast.TypeFloat:
		return "float64"
	case ast.TypeBool:
		return "bool"
	case ast.TypeString:
		return "string"
	case ast.TypeVoid:
		return ""
	case ast.TypeAny:
		return "any"
	default:
		if base, args, ok := baztypes.SplitGenericTypeStrings(string(t)); ok {
			mapped := make([]string, 0, len(args))
			for _, a := range args {
				mapped = append(mapped, mapType(ast.Type(a)))
			}
			return fmt.Sprintf("%s[%s]", base, strings.Join(mapped, ", "))
		}
		return string(t)
	}
}

func mapHIRType(t baztypes.Type) string {
	return mapType(baztypes.ToAST(t))
}

func isHIRVoid(t baztypes.Type) bool {
	return baztypes.ToAST(t) == ast.TypeVoid
}

func exportName(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 32
	}
	return string(r)
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = "\t" + line
	}
	return strings.Join(lines, "\n")
}

func genGuardChainExprMIR(arms []mir.MatchExprArm) (string, error) {
	return goGuardedMatchExprPlan{arms: arms}.render()
}

func (p goGuardedMatchExprPlan) render() (string, error) {
	var b strings.Builder
	unguarded := -1
	for i, arm := range p.arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	for i, arm := range p.arms {
		if arm.Guard == nil {
			continue
		}
		cond, err := genExprMIR(arm.Guard)
		if err != nil {
			return "", err
		}
		val, err := genExprMIR(arm.Value)
		if err != nil {
			return "", err
		}
		if i == 0 {
			b.WriteString("if ")
		} else {
			b.WriteString("else if ")
		}
		b.WriteString(cond)
		b.WriteString(" {\n")
		b.WriteString("\treturn ")
		b.WriteString(val)
		b.WriteString("\n")
		b.WriteString("} ")
	}
	if unguarded >= 0 {
		if len(p.arms) > 0 && p.arms[0].Guard != nil {
			b.WriteString("else {\n")
			b.WriteString("\treturn ")
		} else {
			b.WriteString("return ")
		}
		val, err := genExprMIR(p.arms[unguarded].Value)
		if err != nil {
			return "", err
		}
		b.WriteString(val)
		if len(p.arms) > 0 && p.arms[0].Guard != nil {
			b.WriteString("\n}\n")
		} else {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func genGuardChainTerminatorMIR(arms []mir.MatchTerminatorArm, defaultTarget string) (string, error) {
	return goGuardedMatchTerminatorPlan{arms: arms, defaultTarget: defaultTarget}.render()
}

func (p goGuardedMatchTerminatorPlan) render() (string, error) {
	var b strings.Builder
	unguarded := -1
	for i, arm := range p.arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	for i, arm := range p.arms {
		if arm.Guard == nil {
			continue
		}
		cond, err := genExprMIR(arm.Guard)
		if err != nil {
			return "", err
		}
		if i == 0 {
			b.WriteString("if ")
		} else {
			b.WriteString("else if ")
		}
		b.WriteString(cond)
		b.WriteString(" {\n")
		b.WriteString("\t__bazic_block = ")
		b.WriteString(strconv.Quote(arm.Target))
		b.WriteString("\n\tcontinue\n")
		b.WriteString("} ")
	}
	if unguarded >= 0 {
		if len(p.arms) > 0 && p.arms[0].Guard != nil {
			b.WriteString("else {\n")
		}
		b.WriteString("\t__bazic_block = ")
		b.WriteString(strconv.Quote(p.arms[unguarded].Target))
		b.WriteString("\n\tcontinue\n")
		if len(p.arms) > 0 && p.arms[0].Guard != nil {
			b.WriteString("}\n")
		}
		return b.String(), nil
	}
	if p.defaultTarget != "" {
		if len(p.arms) > 0 {
			b.WriteString("else {\n")
			b.WriteString("\t__bazic_block = ")
			b.WriteString(strconv.Quote(p.defaultTarget))
			b.WriteString("\n\tcontinue\n")
			b.WriteString("}\n")
		} else {
			b.WriteString("__bazic_block = ")
			b.WriteString(strconv.Quote(p.defaultTarget))
			b.WriteString("\ncontinue\n")
		}
	}
	return b.String(), nil
}
