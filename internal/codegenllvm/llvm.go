package codegenllvm

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/backendmeta"
	"baziclang/internal/hir"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

const (
	anyTagInt    = intrinsics.LLVMAnyTagInt
	anyTagFloat  = intrinsics.LLVMAnyTagFloat
	anyTagBool   = intrinsics.LLVMAnyTagBool
	anyTagString = intrinsics.LLVMAnyTagString
	anyTagOther  = intrinsics.LLVMAnyTagOther
)

type llvmProgramPlan struct {
	shape    backendmeta.ProgramShapeMeta
	enums    enumInfo
	structs  structPool
	ifaces   interfacePool
	globals  globalSet
	funcSigs map[string]llvmFuncSig
	strs     stringPool
}

type llvmFunctionRenderer struct {
	funcs   map[string]llvmFuncSig
	globals map[string]globalSlot
	enums   enumInfo
	structs structPool
	ifaces  interfacePool
	strs    stringPool
}

type llvmMainRenderPlan struct {
	fn         *mir.FuncDecl
	hasGlobals bool
}

type llvmFuncRenderPlan struct {
	fn     *mir.FuncDecl
	abi    intrinsics.LLVMFunctionABI
	params []intrinsics.LLVMNamedType
}

type llvmFuncBodyPlan struct {
	fn          *mir.FuncDecl
	topology    *mir.CFGTopology
	deadByBlock map[string]map[int]bool
	localABIs   []intrinsics.LLVMNamedStorageABI
}

type llvmCFGBlockRenderPlan struct {
	fnName  string
	block   *mir.BasicBlock
	deadCFG map[int]bool
}

type llvmValueStmtEmitPlan struct {
	stmt  mir.Stmt
	ctx   *funcCtx
	funcs map[string]llvmFuncSig
	name  string
}

type llvmCFGInstrEmitPlan struct {
	stmt  mir.Stmt
	ctx   *funcCtx
	funcs map[string]llvmFuncSig
}

type llvmTerminatorEmitPlan struct {
	term          mir.Terminator
	ctx           *funcCtx
	funcs         map[string]llvmFuncSig
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

type llvmExprEmitPlan struct {
	ctx   *funcCtx
	expr  mir.Expr
	funcs map[string]llvmFuncSig
}

type llvmUnaryEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.UnaryOpStmt
	funcs map[string]llvmFuncSig
}

type llvmBinaryEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.BinaryOpStmt
	funcs map[string]llvmFuncSig
}

type llvmCallEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.CallStmt
	funcs map[string]llvmFuncSig
}

type llvmFieldAccessEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.FieldAccessStmt
	funcs map[string]llvmFuncSig
}

type llvmStructLitEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.StructLitStmt
	funcs map[string]llvmFuncSig
}

type llvmMatchValueEmitPlan struct {
	ctx   *funcCtx
	stmt  *mir.MatchValueStmt
	funcs map[string]llvmFuncSig
}

type llvmAtomicExprEmitPlan struct {
	ctx  *funcCtx
	expr mir.Expr
}

type llvmAssignTargetEmitPlan struct {
	ctx    *funcCtx
	target mir.Expr
}

type llvmValueCoercionPlan struct {
	ctx        *funcCtx
	value      string
	valueType  ast.Type
	targetType ast.Type
}

type llvmStoreEmitPlan struct {
	ctx        *funcCtx
	ptr        string
	targetType ast.Type
	value      string
	valueType  ast.Type
}

type llvmLoadValuePlan struct {
	ctx *funcCtx
	ptr string
	typ ast.Type
}

type llvmBoolConvertPlan struct {
	ctx    *funcCtx
	value  string
	target string
}

type llvmStringPtrPlan struct {
	ctx   *funcCtx
	value string
}

type llvmBindingLookupPlan struct {
	ctx  *funcCtx
	name string
}

type llvmFieldPathPtrPlan struct {
	ctx    *funcCtx
	ptr    string
	typ    ast.Type
	fields []string
}

type llvmStructFieldResolvePlan struct {
	ctx   *funcCtx
	typ   ast.Type
	field string
}

type llvmAllocaPlan struct {
	ctx *funcCtx
	b   *strings.Builder
	typ ast.Type
}

type llvmAggregateExtractPlan struct {
	ctx  *funcCtx
	base string
	agg  string
	ref  llvmStructFieldRef
}

type llvmAggregateInsertPlan struct {
	ctx        *funcCtx
	structType string
	agg        string
	ref        llvmStructFieldRef
	value      string
}

type llvmGuardedMatchValuePlan struct {
	b          *strings.Builder
	ctx        *funcCtx
	arms       []mir.MatchExprArm
	funcs      map[string]llvmFuncSig
	mergeLabel string
	resolved   ast.Type
	caseLabel  string
}

type llvmGuardedMatchTerminatorPlan struct {
	b            *strings.Builder
	ctx          *funcCtx
	arms         []mir.MatchTerminatorArm
	defaultLabel string
	funcs        map[string]llvmFuncSig
}

type llvmDeclPreludePlan struct {
	shape   backendmeta.ProgramShapeMeta
	structs structPool
	ifaces  interfacePool
	enums   enumInfo
}

type llvmGlobalPreludePlan struct {
	globals globalSet
	funcs   map[string]llvmFuncSig
	enums   enumInfo
	structs structPool
	ifaces  interfacePool
	strs    stringPool
}

type llvmFunctionLoopPlan struct {
	funcs      []*mir.FuncDecl
	hasGlobals bool
	renderer   llvmFunctionRenderer
}

type llvmRuntimePreludePlan struct {
	shape   backendmeta.ProgramShapeMeta
	structs structPool
	ifaces  interfacePool
	strs    stringPool
}

type llvmDeclCommentItemPlan struct {
	kind string
	name string
}

type llvmStructTypeItemPlan struct {
	name    string
	info    structInfo
	enums   enumInfo
	structs structPool
	ifaces  interfacePool
}

type llvmInterfaceTypeItemPlan struct {
	name string
}

type llvmStringGlobalItemPlan struct {
	name  string
	value string
}

type llvmRuntimePreludeRenderer struct {
	section intrinsics.LLVMRuntimePreludeSection
	render  func(llvmRuntimePreludePlan) string
}

type llvmBuiltinRuntimeRenderer struct {
	section intrinsics.LLVMBuiltinRuntimeSection
	render  func(structPool, stringPool) string
}

type llvmModuleTargetInfo struct {
	DataLayout string
	Triple     string
}

type llvmGlobalDeclItemPlan struct {
	global globalInfo
	abiEnv llvmABIEnv
}

type llvmGlobalInitItemPlan struct {
	global globalInfo
	slot   globalSlot
	ctx    *funcCtx
	funcs  map[string]llvmFuncSig
	abiEnv llvmABIEnv
}

var llvmRuntimePreludeRenderers = []llvmRuntimePreludeRenderer{
	{section: intrinsics.LLVMRuntimePreludeStringGlobals, render: func(p llvmRuntimePreludePlan) string {
		return renderLLVMStringGlobals(p.strs)
	}},
	{section: intrinsics.LLVMRuntimePreludeRouteTable, render: func(p llvmRuntimePreludePlan) string {
		return emitRouteTable(p.shape.Runtime.Routes.Handlers, p.strs)
	}},
	{section: intrinsics.LLVMRuntimePreludeStringRuntime, render: func(p llvmRuntimePreludePlan) string {
		return emitStringRuntime(p.strs)
	}},
	{section: intrinsics.LLVMRuntimePreludeBuiltin, render: func(p llvmRuntimePreludePlan) string {
		return emitBuiltinRuntime(p.shape.RuntimeShape.LLVMRuntimeSurface.BuiltinSections, p.structs, p.ifaces, p.strs)
	}},
	{section: intrinsics.LLVMRuntimePreludeAnyRuntime, render: func(p llvmRuntimePreludePlan) string {
		return emitAnyRuntime(p.strs)
	}},
	{section: intrinsics.LLVMRuntimePreludeStdDecls, render: func(p llvmRuntimePreludePlan) string {
		return emitStdDecls(p.shape.RuntimeShape.LLVMRuntimeSurface.TypeAliases, p.structs)
	}},
}

var llvmBuiltinRuntimeRenderers = []llvmBuiltinRuntimeRenderer{
	{section: intrinsics.LLVMBuiltinRuntimeContains, render: func(structs structPool, strs stringPool) string { return emitContains(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeStartsWith, render: func(structs structPool, strs stringPool) string { return emitStartsWith(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeEndsWith, render: func(structs structPool, strs stringPool) string { return emitEndsWith(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeToUpper, render: func(structs structPool, strs stringPool) string {
		return emitCaseTransform(intrinsics.LLVMRuntimeToUpperFunc, "toupper", strs)
	}},
	{section: intrinsics.LLVMBuiltinRuntimeToLower, render: func(structs structPool, strs stringPool) string {
		return emitCaseTransform(intrinsics.LLVMRuntimeToLowerFunc, "tolower", strs)
	}},
	{section: intrinsics.LLVMBuiltinRuntimeTrimSpace, render: func(structs structPool, strs stringPool) string { return emitTrimSpace(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeRepeat, render: func(structs structPool, strs stringPool) string { return emitRepeat(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeReplace, render: func(structs structPool, strs stringPool) string { return emitReplace(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeIntToStr, render: func(structs structPool, strs stringPool) string { return emitIntToStr(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeFloatToStr, render: func(structs structPool, strs stringPool) string { return emitFloatToStr(strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeParseInt, render: func(structs structPool, strs stringPool) string { return emitParseInt(structs, strs) }},
	{section: intrinsics.LLVMBuiltinRuntimeParseFloat, render: func(structs structPool, strs stringPool) string { return emitParseFloat(structs, strs) }},
}

func buildLLVMProgramPlan(mp *mir.Program) llvmProgramPlan {
	shape := backendmeta.CollectProgramShapeMeta(mp)
	enums := collectEnums(shape.Enums)
	structs := ensureRuntimeStructs(collectStructs(shape.Structs), shape.Runtime.Types.HTTPResponseType)
	ifaces := collectInterfaces(shape.Interfaces)
	globals := collectGlobals(shape.Globals)
	funcSigs := map[string]llvmFuncSig{}
	for name, sig := range shape.ProgramFuncSigs {
		funcSigs[name] = llvmFuncSig(sig)
	}
	for name, sig := range shape.RuntimeShape.LLVMRuntimeSurface.FuncSigs {
		funcSigs[name] = llvmFuncSig(sig)
	}
	return llvmProgramPlan{
		shape:    shape,
		enums:    enums,
		structs:  structs,
		ifaces:   ifaces,
		globals:  globals,
		funcSigs: funcSigs,
		strs:     collectStringLiteralsFromMIR(mp, shape.RuntimeShape.RouteStrings),
	}
}

func (p llvmProgramPlan) buildDeclPreludePlan() llvmDeclPreludePlan {
	return llvmDeclPreludePlan{
		shape:   p.shape,
		structs: p.structs,
		ifaces:  p.ifaces,
		enums:   p.enums,
	}
}

func (p llvmProgramPlan) buildGlobalPreludePlan() llvmGlobalPreludePlan {
	return llvmGlobalPreludePlan{
		globals: p.globals,
		funcs:   p.funcSigs,
		enums:   p.enums,
		structs: p.structs,
		ifaces:  p.ifaces,
		strs:    p.strs,
	}
}

func (p llvmProgramPlan) buildRuntimePreludePlan() llvmRuntimePreludePlan {
	return llvmRuntimePreludePlan{
		shape:   p.shape,
		structs: p.structs,
		ifaces:  p.ifaces,
		strs:    p.strs,
	}
}

func (p llvmProgramPlan) buildFunctionLoopPlan() llvmFunctionLoopPlan {
	return llvmFunctionLoopPlan{
		funcs:      p.shape.OrderedFuncs,
		hasGlobals: len(p.globals.order) > 0,
		renderer: llvmFunctionRenderer{
			funcs:   p.funcSigs,
			globals: p.globals.slots,
			enums:   p.enums,
			structs: p.structs,
			ifaces:  p.ifaces,
			strs:    p.strs,
		},
	}
}

func (p llvmProgramPlan) renderPrelude() (string, error) {
	decls, err := p.renderDeclPrelude()
	if err != nil {
		return "", err
	}
	runtime, err := p.renderRuntimePrelude()
	if err != nil {
		return "", err
	}
	globals, err := p.renderGlobalsPrelude()
	if err != nil {
		return "", err
	}
	return decls + runtime + globals, nil
}

func (p llvmProgramPlan) renderDeclPrelude() (string, error) {
	return p.buildDeclPreludePlan().render()
}

func (p llvmDeclPreludePlan) render() (string, error) {
	shape := p.shape
	var b strings.Builder
	b.WriteString("; Bazic LLVM IR (early backend)\n")
	b.WriteString("source_filename = \"bazic_module\"\n\n")
	if target, ok := llvmHostModuleTarget(); ok {
		b.WriteString(fmt.Sprintf("target datalayout = %q\n", target.DataLayout))
		b.WriteString(fmt.Sprintf("target triple = %q\n\n", target.Triple))
	}
	for _, decl := range shape.RuntimeShape.LLVMRuntimeSurface.PreludeDecls {
		b.WriteString(decl)
	}
	b.WriteString("\n")

	for _, decl := range shape.StructNodes {
		b.WriteString(llvmDeclCommentItemPlan{kind: "struct", name: decl.Name}.render())
	}
	for _, decl := range shape.InterfaceNodes {
		b.WriteString(llvmDeclCommentItemPlan{kind: "interface", name: decl.Name}.render())
	}
	for _, decl := range shape.EnumNodes {
		b.WriteString(llvmDeclCommentItemPlan{kind: "enum", name: decl.Name}.render())
	}
	if len(shape.StructNodes)+len(shape.InterfaceNodes)+len(shape.EnumNodes)+len(shape.GlobalNodes)+len(shape.OrderedFuncs) > 0 {
		b.WriteString("\n")
	}

	if len(p.structs.order) > 0 {
		for _, name := range p.structs.order {
			info := p.structs.byName[name]
			decl, err := llvmStructTypeItemPlan{
				name:    name,
				info:    info,
				enums:   p.enums,
				structs: p.structs,
				ifaces:  p.ifaces,
			}.render()
			if err != nil {
				return "", err
			}
			b.WriteString(decl)
		}
		b.WriteString("\n")
	}

	if len(p.ifaces.order) > 0 {
		for _, name := range p.ifaces.order {
			b.WriteString(llvmInterfaceTypeItemPlan{name: name}.render())
		}
		b.WriteString("\n")
	}

	b.WriteString("%Any = type { i64, ptr }\n\n")
	return b.String(), nil
}

func (p llvmProgramPlan) renderRuntimePrelude() (string, error) {
	return p.buildRuntimePreludePlan().render()
}

func (p llvmRuntimePreludePlan) render() (string, error) {
	var b strings.Builder
	for _, section := range p.shape.RuntimeShape.LLVMRuntimeSurface.PreludeSections {
		segment := p.renderSection(section)
		b.WriteString(segment)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (p llvmRuntimePreludePlan) renderSection(section intrinsics.LLVMRuntimePreludeSection) string {
	for _, renderer := range llvmRuntimePreludeRenderers {
		if renderer.section == section {
			return renderer.render(p)
		}
	}
	return ""
}

func renderLLVMStringGlobals(strs stringPool) string {
	if len(strs.ordered) == 0 {
		return ""
	}
	var b strings.Builder
	for _, lit := range strs.ordered {
		name := strs.names[lit]
		b.WriteString(llvmStringGlobalItemPlan{name: name, value: lit}.render())
	}
	return b.String()
}

func (p llvmProgramPlan) renderGlobalsPrelude() (string, error) {
	return p.buildGlobalPreludePlan().render()
}

func (p llvmGlobalPreludePlan) render() (string, error) {
	var b strings.Builder
	if len(p.globals.order) > 0 {
		decls, err := emitGlobalDecls(p.globals, p.enums, p.structs, p.ifaces)
		if err != nil {
			return "", err
		}
		b.WriteString(decls)
		b.WriteString("\n")
		initIR, err := emitGlobalInit(p.globals, p.funcs, p.enums, p.structs, p.ifaces, p.strs)
		if err != nil {
			return "", err
		}
		b.WriteString(initIR)
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (p llvmProgramPlan) renderFunctions() (string, error) {
	return p.buildFunctionLoopPlan().render()
}

func (p llvmFunctionLoopPlan) render() (string, error) {
	var b strings.Builder
	emittedFuncs := map[string]bool{}
	for _, fn := range p.funcs {
		if emittedFuncs[fn.Name] {
			continue
		}
		emittedFuncs[fn.Name] = true
		if len(fn.TypeParams) > 0 {
			return "", fmt.Errorf("llvm backend: unresolved generic function '%s'", fn.Name)
		}
		fnIR, err := p.renderer.render(fn, p.hasGlobals)
		if err != nil {
			return "", err
		}
		b.WriteString(fnIR)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (r llvmFunctionRenderer) render(fn *mir.FuncDecl, hasGlobals bool) (string, error) {
	if fn.Name == "main" {
		return r.renderMain(llvmMainRenderPlan{fn: fn, hasGlobals: hasGlobals})
	}
	return r.renderFunction(fn)
}

func (r llvmFunctionRenderer) renderMain(plan llvmMainRenderPlan) (string, error) {
	if err := requireMIRCFGLLVM(plan.fn); err != nil {
		return "", err
	}
	return renderLLVMMainBody(plan, r.funcs, r.globals, r.enums, r.structs, r.ifaces, r.strs)
}

func (r llvmFunctionRenderer) buildFunctionPlan(fn *mir.FuncDecl) (llvmFuncRenderPlan, error) {
	abiEnv := llvmABIEnv{enums: r.enums, structs: r.structs, ifaces: r.ifaces}
	params := make([]intrinsics.LLVMNamedType, 0, len(fn.Params))
	for _, p := range fn.Params {
		params = append(params, intrinsics.LLVMNamedType{Name: p.Name, Type: baztypes.ToAST(p.Type)})
	}
	abi, err := abiEnv.functionABIOrError(fn.Name, baztypes.ToAST(fn.ReturnType), params)
	if err != nil {
		return llvmFuncRenderPlan{}, err
	}
	return llvmFuncRenderPlan{fn: fn, abi: abi, params: params}, nil
}

func (r llvmFunctionRenderer) renderFunction(fn *mir.FuncDecl) (string, error) {
	plan, err := r.buildFunctionPlan(fn)
	if err != nil {
		return "", err
	}
	if err := requireMIRCFGLLVM(plan.fn); err != nil {
		return "", err
	}
	return renderLLVMFunctionBody(plan, r.funcs, r.globals, r.enums, r.structs, r.ifaces, r.strs)
}

func GenerateLLVMIR(p *ast.Program) (string, error) {
	p = monomorphizeProgram(p)
	hp, err := hir.Lower(p)
	if err != nil {
		return "", err
	}
	mp, err := mir.Lower(hp)
	if err != nil {
		return "", err
	}
	plan := buildLLVMProgramPlan(mp)
	prelude, err := plan.renderPrelude()
	if err != nil {
		return "", err
	}
	functions, err := plan.renderFunctions()
	if err != nil {
		return "", err
	}
	return prelude + functions, nil
}

type globalInfo struct {
	Name string
	Type ast.Type
	Init mir.Expr
}

type globalSlot struct {
	ptr string
	typ ast.Type
}

type globalSet struct {
	order []globalInfo
	slots map[string]globalSlot
}

func collectGlobals(globals []backendmeta.GlobalDecl) globalSet {
	out := globalSet{order: []globalInfo{}, slots: map[string]globalSlot{}}
	for _, g := range globals {
		name := g.Name
		typ := normalizeLLVMType(g.Type)
		info := globalInfo{Name: name, Type: typ, Init: g.Init}
		out.order = append(out.order, info)
		out.slots[name] = globalSlot{ptr: "@" + name, typ: typ}
	}
	return out
}

type enumInfo struct {
	variantIndex map[string]int
	variantType  map[string]string
	enumTypes    map[string]bool
}

func collectEnums(decls []backendmeta.EnumDecl) enumInfo {
	info := enumInfo{
		variantIndex: map[string]int{},
		variantType:  map[string]string{},
		enumTypes:    map[string]bool{},
	}
	for _, decl := range decls {
		enumName := decl.Name
		info.enumTypes[enumName] = true
		for i, v := range decl.Variants {
			info.variantIndex[v] = i
			info.variantType[v] = enumName
		}
	}
	return info
}

type structFieldInfo struct {
	Name string
	Type ast.Type
}

type structInfo struct {
	Fields     []structFieldInfo
	FieldIndex map[string]int
}

type structPool struct {
	byName map[string]structInfo
	order  []string
}

type interfacePool struct {
	names map[string]bool
	order []string
}

func collectStructs(decls []backendmeta.StructDecl) structPool {
	pool := structPool{
		byName: map[string]structInfo{},
		order:  []string{},
	}
	for _, decl := range decls {
		name := decl.Name
		if _, exists := pool.byName[name]; exists {
			continue
		}
		fields := make([]structFieldInfo, 0, len(decl.Fields))
		index := map[string]int{}
		for i, f := range decl.Fields {
			fields = append(fields, structFieldInfo{Name: f.Name, Type: f.Type})
			index[f.Name] = i
		}
		pool.byName[name] = structInfo{Fields: fields, FieldIndex: index}
		pool.order = append(pool.order, name)
	}
	return pool
}

func ensureRuntimeStructs(pool structPool, httpResponseType string) structPool {
	if _, ok := pool.byName["Error"]; !ok {
		return pool
	}
	addResult := func(name string, okType ast.Type) {
		if _, exists := pool.byName[name]; exists {
			return
		}
		pool.byName[name] = structInfo{
			Fields: []structFieldInfo{
				{Name: "is_ok", Type: ast.TypeBool},
				{Name: "value", Type: okType},
				{Name: "err", Type: ast.Type("Error")},
			},
			FieldIndex: map[string]int{
				"is_ok": 0,
				"value": 1,
				"err":   2,
			},
		}
		pool.order = append(pool.order, name)
	}
	addResult(intrinsics.LLVMResultStructName("string", "Error"), ast.TypeString)
	addResult(intrinsics.LLVMResultStructName("bool", "Error"), ast.TypeBool)
	addResult(intrinsics.LLVMResultStructName("int", "Error"), ast.TypeInt)
	addResult(intrinsics.LLVMResultStructName("float", "Error"), ast.TypeFloat)
	if _, ok := pool.byName[httpResponseType]; ok {
		addResult(intrinsics.LLVMResultStructName(httpResponseType, "Error"), ast.Type(httpResponseType))
	}
	return pool
}

func collectInterfaces(decls []backendmeta.InterfaceDecl) interfacePool {
	pool := interfacePool{
		names: map[string]bool{},
		order: []string{},
	}
	for _, decl := range decls {
		name := decl.Name
		if pool.names[name] {
			continue
		}
		pool.names[name] = true
		pool.order = append(pool.order, name)
	}
	return pool
}

func llvmHostModuleTarget() (llvmModuleTargetInfo, bool) {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			return llvmModuleTargetInfo{
				DataLayout: "e-m:w-p270:32:32-p271:32:32-p272:64:64-i64:64-i128:128-f80:128-n8:16:32:64-S128",
				Triple:     "x86_64-pc-windows-msvc19.33.0",
			}, true
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			return llvmModuleTargetInfo{
				DataLayout: "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-i128:128-f80:128-n8:16:32:64-S128",
				Triple:     "x86_64-unknown-linux-gnu",
			}, true
		}
	case "darwin":
		switch runtime.GOARCH {
		case "arm64":
			return llvmModuleTargetInfo{
				DataLayout: "e-m:o-i64:64-i128:128-n32:64-S128",
				Triple:     "arm64-apple-macosx15.0.0",
			}, true
		case "amd64":
			return llvmModuleTargetInfo{
				DataLayout: "e-m:o-i64:64-f80:128-n8:16:32:64-S128",
				Triple:     "x86_64-apple-macosx10.12.0",
			}, true
		}
	}
	return llvmModuleTargetInfo{}, false
}

func emitStructType(name string, info structInfo, enums enumInfo, structs structPool, ifaces interfacePool) (string, error) {
	abiEnv := llvmABIEnv{enums: enums, structs: structs, ifaces: ifaces}
	parts := make([]string, 0, len(info.Fields))
	for _, f := range info.Fields {
		abi, err := abiEnv.valueABIOrError(f.Type, "llvm backend: unsupported field type '%s.%s' (%s)", name, f.Name, f.Type)
		if err != nil {
			return "", err
		}
		parts = append(parts, abi.LLVMType)
	}
	return fmt.Sprintf("%%%s = type { %s }\n", name, strings.Join(parts, ", ")), nil
}

type stringPool struct {
	names   map[string]string
	ordered []string
}

func (p *stringPool) add(lit string) {
	if p == nil {
		return
	}
	if _, ok := p.names[lit]; ok {
		return
	}
	name := fmt.Sprintf(".str%d", len(p.ordered))
	p.names[lit] = name
	p.ordered = append(p.ordered, lit)
}

func collectStringLiteralsFromMIR(p *mir.Program, extra []string) stringPool {
	pool := stringPool{
		names:   map[string]string{},
		ordered: []string{},
	}
	add := pool.add
	add("%ld")
	add("%ld\n")
	add("%g")
	add("%g\n")
	add("%s")
	add("%s\n")
	add("true")
	add("false")
	add("")
	add("invalid int")
	add("invalid float")
	add("std unavailable")
	add("<any>")
	for _, s := range extra {
		add(s)
	}
	augmentStringPoolFromMIR(&pool, p)
	return pool
}

func augmentStringPoolFromMIR(pool *stringPool, p *mir.Program) {
	if pool == nil || p == nil {
		return
	}
	for _, d := range p.Decls {
		switch decl := d.(type) {
		case *mir.FuncDecl:
			augmentStringPoolFromMIRBlock(pool, decl.Body)
			augmentStringPoolFromCFG(pool, decl.CFG)
		case *mir.GlobalLetDecl:
			augmentStringPoolFromMIRExpr(pool, decl.Init)
		}
	}
}

func augmentStringPoolFromCFG(pool *stringPool, cfg *mir.CFG) {
	if pool == nil || cfg == nil {
		return
	}
	liveness := mir.AnalyzeCFGLivenessFromCFG(cfg)
	deadByBlock := map[string]map[int]bool{}
	if liveness != nil {
		deadByBlock = liveness.DeadByBlock
	}
	for _, block := range cfg.Blocks {
		if block == nil {
			continue
		}
		deadCFG := deadByBlock[block.Name]
		for i, instr := range block.Instrs {
			if deadCFG[i] {
				continue
			}
			augmentStringPoolFromMIRStmt(pool, instr)
		}
		augmentStringPoolFromMIRTerminator(pool, block.Term)
	}
}

func augmentStringPoolFromMIRBlock(pool *stringPool, blk *mir.Block) {
	if pool == nil || blk == nil {
		return
	}
	for _, stmt := range blk.Stmts {
		augmentStringPoolFromMIRStmt(pool, stmt)
	}
}

func augmentStringPoolFromMIRStmt(pool *stringPool, stmt mir.Stmt) {
	if pool == nil || stmt == nil {
		return
	}
	mir.WalkStmtExprs(stmt, func(expr mir.Expr) {
		augmentStringPoolFromMIRExpr(pool, expr)
	})
	mir.WalkStmtChildBlocks(stmt, func(block *mir.Block) {
		augmentStringPoolFromMIRBlock(pool, block)
	})
}

func augmentStringPoolFromMIRTerminator(pool *stringPool, term mir.Terminator) {
	if pool == nil || term == nil {
		return
	}
	mir.WalkTerminatorExprs(term, func(expr mir.Expr) {
		augmentStringPoolFromMIRExpr(pool, expr)
	})
}

func augmentStringPoolFromMIRExpr(pool *stringPool, expr mir.Expr) {
	if pool == nil || expr == nil {
		return
	}
	mir.WalkExpr(expr, func(expr mir.Expr) {
		if str, ok := expr.(*mir.StringExpr); ok {
			pool.add(str.Value)
		}
	})
}

func emitAnyRuntime(strs stringPool) string {
	var b strings.Builder
	trueName, trueLen := stringGlobalRef(strs, "true")
	falseName, falseLen := stringGlobalRef(strs, "false")
	anyName, anyLen := stringGlobalRef(strs, "<any>")

	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeAnyToStrFunc + "(%Any %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %tag = extractvalue %Any %v, 0\n")
	b.WriteString("  %payload = extractvalue %Any %v, 1\n")
	b.WriteString("  switch i64 %tag, label %any_fallback [\n")
	b.WriteString("    i64 1, label %any_int\n")
	b.WriteString("    i64 2, label %any_float\n")
	b.WriteString("    i64 3, label %any_bool\n")
	b.WriteString("    i64 4, label %any_string\n")
	b.WriteString("  ]\n")
	b.WriteString("any_int:\n")
	b.WriteString("  %ival = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %istr = call ptr @" + intrinsics.LLVMRuntimeIntToStrFunc + "(i64 %ival)\n")
	b.WriteString("  ret ptr %istr\n")
	b.WriteString("any_float:\n")
	b.WriteString("  %fbits = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %fval = bitcast i64 %fbits to double\n")
	b.WriteString("  %fstr = call ptr @" + intrinsics.LLVMRuntimeFloatToStrFunc + "(double %fval)\n")
	b.WriteString("  ret ptr %fstr\n")
	b.WriteString("any_bool:\n")
	b.WriteString("  %bval = ptrtoint ptr %payload to i64\n")
	b.WriteString("  %btrue = icmp ne i64 %bval, 0\n")
	b.WriteString(fmt.Sprintf("  %%btrue_ptr = %s\n", stringGEP(trueName, trueLen)))
	b.WriteString(fmt.Sprintf("  %%bfalse_ptr = %s\n", stringGEP(falseName, falseLen)))
	b.WriteString("  %bstr = select i1 %btrue, ptr %btrue_ptr, ptr %bfalse_ptr\n")
	b.WriteString("  ret ptr %bstr\n")
	b.WriteString("any_string:\n")
	b.WriteString("  ret ptr %payload\n")
	b.WriteString("any_fallback:\n")
	b.WriteString(fmt.Sprintf("  %%any_ptr = %s\n", stringGEP(anyName, anyLen)))
	b.WriteString("  ret ptr %any_ptr\n")
	b.WriteString("}\n\n")

	b.WriteString("define i8 @" + intrinsics.LLVMRuntimeAnyEqFunc + "(%Any %a, %Any %b) {\n")
	b.WriteString("entry:\n")
	b.WriteString("  %tagA = extractvalue %Any %a, 0\n")
	b.WriteString("  %tagB = extractvalue %Any %b, 0\n")
	b.WriteString("  %tagsame = icmp eq i64 %tagA, %tagB\n")
	b.WriteString("  br i1 %tagsame, label %any_eq_switch, label %any_eq_false\n")
	b.WriteString("any_eq_false:\n")
	b.WriteString("  ret i8 0\n")
	b.WriteString("any_eq_switch:\n")
	b.WriteString("  %payloadA = extractvalue %Any %a, 1\n")
	b.WriteString("  %payloadB = extractvalue %Any %b, 1\n")
	b.WriteString("  switch i64 %tagA, label %any_eq_ptr [\n")
	b.WriteString("    i64 1, label %any_eq_int\n")
	b.WriteString("    i64 2, label %any_eq_float\n")
	b.WriteString("    i64 3, label %any_eq_bool\n")
	b.WriteString("    i64 4, label %any_eq_string\n")
	b.WriteString("  ]\n")
	b.WriteString("any_eq_int:\n")
	b.WriteString("  %aint = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bint = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %eqi = icmp eq i64 %aint, %bint\n")
	b.WriteString("  %eqi8 = zext i1 %eqi to i8\n")
	b.WriteString("  ret i8 %eqi8\n")
	b.WriteString("any_eq_float:\n")
	b.WriteString("  %abits = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bbits = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %af = bitcast i64 %abits to double\n")
	b.WriteString("  %bf = bitcast i64 %bbits to double\n")
	b.WriteString("  %eqf = fcmp oeq double %af, %bf\n")
	b.WriteString("  %eqf8 = zext i1 %eqf to i8\n")
	b.WriteString("  ret i8 %eqf8\n")
	b.WriteString("any_eq_bool:\n")
	b.WriteString("  %ab = ptrtoint ptr %payloadA to i64\n")
	b.WriteString("  %bb = ptrtoint ptr %payloadB to i64\n")
	b.WriteString("  %eqb = icmp eq i64 %ab, %bb\n")
	b.WriteString("  %eqb8 = zext i1 %eqb to i8\n")
	b.WriteString("  ret i8 %eqb8\n")
	b.WriteString("any_eq_string:\n")
	b.WriteString("  %cmp = call i32 @" + intrinsics.LLVMRuntimeStrCmpFunc + "(ptr %payloadA, ptr %payloadB)\n")
	b.WriteString("  %eqs = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %eqs8 = zext i1 %eqs to i8\n")
	b.WriteString("  ret i8 %eqs8\n")
	b.WriteString("any_eq_ptr:\n")
	b.WriteString("  %eqp = icmp eq ptr %payloadA, %payloadB\n")
	b.WriteString("  %eqp8 = zext i1 %eqp to i8\n")
	b.WriteString("  ret i8 %eqp8\n")
	b.WriteString("}\n")
	return b.String()
}

func emitGlobalDecls(globals globalSet, enums enumInfo, structs structPool, ifaces interfacePool) (string, error) {
	abiEnv := llvmABIEnv{enums: enums, structs: structs, ifaces: ifaces}
	var b strings.Builder
	for _, g := range globals.order {
		line, err := llvmGlobalDeclItemPlan{global: g, abiEnv: abiEnv}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}
	return b.String(), nil
}

func emitGlobalInit(globals globalSet, funcs map[string]llvmFuncSig, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	abiEnv := llvmABIEnv{enums: enums, structs: structs, ifaces: ifaces}
	var b strings.Builder
	b.WriteString("define void @" + intrinsics.LLVMRuntimeInitGlobalsFunc + "() {\n")
	b.WriteString("entry:\n")
	ctx := newFuncCtx(enums, structs, ifaces, strs, false, globals.slots)
	for _, g := range globals.order {
		slot, ok := globals.slots[g.Name]
		if !ok {
			continue
		}
		line, err := llvmGlobalInitItemPlan{
			global: g,
			slot:   slot,
			ctx:    ctx,
			funcs:  funcs,
			abiEnv: abiEnv,
		}.render()
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}
	b.WriteString("  ret void\n")
	b.WriteString("}\n")
	return b.String(), nil
}

func (p llvmGlobalDeclItemPlan) render() (string, error) {
	abi, err := p.abiEnv.storageABIOrError(p.global.Type, "llvm backend: unsupported global type '%s' (%s)", p.global.Name, p.global.Type)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@%s = global %s %s\n", p.global.Name, abi.LLVMType, abi.DefaultValue), nil
}

func (p llvmGlobalInitItemPlan) render() (string, error) {
	code, value, t, ok := emitExprMIRLLVM(p.ctx, p.global.Init, p.funcs)
	if !ok {
		return "", fmt.Errorf("llvm backend: unsupported global init for '%s' (%T)", p.global.Name, p.global.Init)
	}
	coerceCode, coerced, abi, ok := p.ctx.coerceTypedLLVMValue(p.slot.typ, value, t)
	if !ok {
		return "", fmt.Errorf("llvm backend: global init type mismatch for '%s' (got %s, expected %s)", p.global.Name, t, p.slot.typ)
	}
	code += coerceCode
	value = coerced
	_, err := p.abiEnv.storageABIOrError(p.slot.typ, "llvm backend: unsupported global storage type '%s' for '%s'", p.slot.typ, p.global.Name)
	if err != nil {
		return "", err
	}
	return code + fmt.Sprintf("  store %s %s, ptr %s\n", abi.LLVMType, value, p.slot.ptr), nil
}

func (p llvmDeclCommentItemPlan) render() string {
	return fmt.Sprintf("; %s %s\n", p.kind, p.name)
}

func (p llvmStructTypeItemPlan) render() (string, error) {
	return emitStructType(p.name, p.info, p.enums, p.structs, p.ifaces)
}

func (p llvmInterfaceTypeItemPlan) render() string {
	return fmt.Sprintf("%%%s = type { ptr, ptr }\n", p.name)
}

func (p llvmStringGlobalItemPlan) render() string {
	return emitStringGlobal(p.name, p.value)
}

func emitStringGlobal(name string, value string) string {
	escaped := escapeLLVMString(value)
	length := len([]byte(value)) + 1
	return fmt.Sprintf("@%s = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n", name, length, escaped)
}

func escapeLLVMString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\\':
			b.WriteString("\\5C")
		case '"':
			b.WriteString("\\22")
		case '\n':
			b.WriteString("\\0A")
		case '\r':
			b.WriteString("\\0D")
		case '\t':
			b.WriteString("\\09")
		default:
			if ch < 32 || ch >= 127 {
				b.WriteString(fmt.Sprintf("\\%02X", ch))
			} else {
				b.WriteByte(ch)
			}
		}
	}
	return b.String()
}

func stringGlobalRef(strs stringPool, lit string) (string, int) {
	name := strs.names[lit]
	return name, len([]byte(lit)) + 1
}

func stringGEP(name string, length int) string {
	return fmt.Sprintf("getelementptr inbounds [%d x i8], ptr @%s, i64 0, i64 0", length, name)
}

func stringGEPConst(name string, length int) string {
	return fmt.Sprintf("getelementptr inbounds ([%d x i8], ptr @%s, i64 0, i64 0)", length, name)
}

func emitNormalizedRuntimeStringPtr(b *strings.Builder, strs stringPool, dst string, src string) {
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString(fmt.Sprintf("  %%%s_empty = %s\n", dst, stringGEP(emptyName, emptyLen)))
	b.WriteString(fmt.Sprintf("  %%%s_isnull = icmp eq ptr %%%s, null\n", dst, src))
	b.WriteString(fmt.Sprintf("  %%%s = select i1 %%%s_isnull, ptr %%%s_empty, ptr %%%s\n", dst, dst, dst, src))
}

func emitRouteTable(handlers []intrinsics.HTTPHandlerSpec, strs stringPool) string {
	var b strings.Builder
	b.WriteString("%" + intrinsics.LLVMRuntimeRouteType + " = type { ptr, ptr, ptr }\n")
	if len(handlers) == 0 {
		b.WriteString("@" + intrinsics.LLVMRuntimeRoutesGlobal + " = global [0 x %" + intrinsics.LLVMRuntimeRouteType + "] []\n")
		b.WriteString("@" + intrinsics.LLVMRuntimeRoutesLenGlobal + " = global i64 0\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("@%s = global [%d x %%%s] [\n", intrinsics.LLVMRuntimeRoutesGlobal, len(handlers), intrinsics.LLVMRuntimeRouteType))
	for i, h := range handlers {
		methodName, methodLen := stringGlobalRef(strs, h.Method)
		path := intrinsics.HTTPRoutePattern(h)
		pathName, pathLen := stringGlobalRef(strs, path)
		methodPtr := stringGEPConst(methodName, methodLen)
		pathPtr := stringGEPConst(pathName, pathLen)
		b.WriteString(fmt.Sprintf("  %%%s { ptr %s, ptr %s, ptr @%s }", intrinsics.LLVMRuntimeRouteType, methodPtr, pathPtr, h.FuncName))
		if i+1 < len(handlers) {
			b.WriteString(",\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString("]\n")
	b.WriteString(fmt.Sprintf("@%s = global i64 %d\n", intrinsics.LLVMRuntimeRoutesLenGlobal, len(handlers)))
	return b.String()
}

func emitStringRuntime(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeStrConcatFunc + "(ptr %a, ptr %b) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "a_safe", "a")
	emitNormalizedRuntimeStringPtr(&b, strs, "b_safe", "b")
	b.WriteString("  %lenA = call i64 @strlen(ptr %a_safe)\n")
	b.WriteString("  %lenB = call i64 @strlen(ptr %b_safe)\n")
	b.WriteString("  %aempty = icmp eq i64 %lenA, 0\n")
	b.WriteString("  br i1 %aempty, label %dup_b, label %check_b\n")
	b.WriteString("dup_b:\n")
	b.WriteString("  %dupBTotal = add i64 %lenB, 1\n")
	b.WriteString("  %dupB = call ptr @malloc(i64 %dupBTotal)\n")
	b.WriteString("  %dupBNull = icmp eq ptr %dupB, null\n")
	b.WriteString("  br i1 %dupBNull, label %oom, label %dup_b_copy\n")
	b.WriteString("dup_b_copy:\n")
	b.WriteString("  call ptr @memcpy(ptr %dupB, ptr %b_safe, i64 %lenB)\n")
	b.WriteString("  %dupBEnd = getelementptr i8, ptr %dupB, i64 %lenB\n")
	b.WriteString("  store i8 0, ptr %dupBEnd\n")
	b.WriteString("  ret ptr %dupB\n")
	b.WriteString("check_b:\n")
	b.WriteString("  %bempty = icmp eq i64 %lenB, 0\n")
	b.WriteString("  br i1 %bempty, label %dup_a, label %cont\n")
	b.WriteString("dup_a:\n")
	b.WriteString("  %dupATotal = add i64 %lenA, 1\n")
	b.WriteString("  %dupA = call ptr @malloc(i64 %dupATotal)\n")
	b.WriteString("  %dupANull = icmp eq ptr %dupA, null\n")
	b.WriteString("  br i1 %dupANull, label %oom, label %dup_a_copy\n")
	b.WriteString("dup_a_copy:\n")
	b.WriteString("  call ptr @memcpy(ptr %dupA, ptr %a_safe, i64 %lenA)\n")
	b.WriteString("  %dupAEnd = getelementptr i8, ptr %dupA, i64 %lenA\n")
	b.WriteString("  store i8 0, ptr %dupAEnd\n")
	b.WriteString("  ret ptr %dupA\n")
	b.WriteString("cont:\n")
	b.WriteString("  %sum = add i64 %lenA, %lenB\n")
	b.WriteString("  %total = add i64 %sum, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %oom, label %cont_copy\n")
	b.WriteString("cont_copy:\n")
	b.WriteString("  call ptr @memcpy(ptr %buf, ptr %a_safe, i64 %lenA)\n")
	b.WriteString("  %dstB = getelementptr i8, ptr %buf, i64 %lenA\n")
	b.WriteString("  call ptr @memcpy(ptr %dstB, ptr %b_safe, i64 %lenB)\n")
	b.WriteString("  %end = getelementptr i8, ptr %buf, i64 %sum\n")
	b.WriteString("  store i8 0, ptr %end\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("oom:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n\n")
	b.WriteString("define i32 @" + intrinsics.LLVMRuntimeStrCmpFunc + "(ptr %a, ptr %b) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "a_safe", "a")
	emitNormalizedRuntimeStringPtr(&b, strs, "b_safe", "b")
	b.WriteString("  %c = call i32 @strcmp(ptr %a_safe, ptr %b_safe)\n")
	b.WriteString("  ret i32 %c\n")
	b.WriteString("}\n")
	return b.String()
}

func emitBuiltinRuntime(sections []intrinsics.LLVMBuiltinRuntimeSection, structs structPool, ifaces interfacePool, strs stringPool) string {
	var b strings.Builder
	_ = ifaces
	for _, section := range sections {
		body := emitBuiltinRuntimeSection(section, structs, strs)
		b.WriteString(body)
		b.WriteString("\n")
	}
	return b.String()
}

func emitBuiltinRuntimeSection(section intrinsics.LLVMBuiltinRuntimeSection, structs structPool, strs stringPool) string {
	for _, renderer := range llvmBuiltinRuntimeRenderers {
		if renderer.section == section {
			return renderer.render(structs, strs)
		}
	}
	return ""
}

func emitContains(strs stringPool) string {
	var b strings.Builder
	b.WriteString("define i8 @" + intrinsics.LLVMRuntimeContainsFunc + "(ptr %s, ptr %sub) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	emitNormalizedRuntimeStringPtr(&b, strs, "sub_safe", "sub")
	b.WriteString("  %found = call ptr @strstr(ptr %s_safe, ptr %sub_safe)\n")
	b.WriteString("  %ok = icmp ne ptr %found, null\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n")
	return b.String()
}

func emitStartsWith(strs stringPool) string {
	var b strings.Builder
	b.WriteString("define i8 @" + intrinsics.LLVMRuntimeStartsWithFunc + "(ptr %s, ptr %prefix) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	emitNormalizedRuntimeStringPtr(&b, strs, "prefix_safe", "prefix")
	b.WriteString("  %len = call i64 @strlen(ptr %prefix_safe)\n")
	b.WriteString("  %cmp = call i32 @strncmp(ptr %s_safe, ptr %prefix_safe, i64 %len)\n")
	b.WriteString("  %ok = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n")
	return b.String()
}

func emitEndsWith(strs stringPool) string {
	var b strings.Builder
	b.WriteString("define i8 @" + intrinsics.LLVMRuntimeEndsWithFunc + "(ptr %s, ptr %suffix) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	emitNormalizedRuntimeStringPtr(&b, strs, "suffix_safe", "suffix")
	b.WriteString("  %lenS = call i64 @strlen(ptr %s_safe)\n")
	b.WriteString("  %lenT = call i64 @strlen(ptr %suffix_safe)\n")
	b.WriteString("  %short = icmp ult i64 %lenS, %lenT\n")
	b.WriteString("  br i1 %short, label %retfalse, label %cont\n")
	b.WriteString("retfalse:\n")
	b.WriteString("  ret i8 0\n")
	b.WriteString("cont:\n")
	b.WriteString("  %start = sub i64 %lenS, %lenT\n")
	b.WriteString("  %ptr = getelementptr i8, ptr %s_safe, i64 %start\n")
	b.WriteString("  %cmp = call i32 @strncmp(ptr %ptr, ptr %suffix_safe, i64 %lenT)\n")
	b.WriteString("  %ok = icmp eq i32 %cmp, 0\n")
	b.WriteString("  %ok8 = zext i1 %ok to i8\n")
	b.WriteString("  ret i8 %ok8\n")
	b.WriteString("}\n")
	return b.String()
}

func emitStdDecls(typeAliases map[string]ast.Type, structs structPool) string {
	return intrinsics.FormatLLVMStdDecls(typeAliases, func(name string) bool {
		_, ok := structs.byName[name]
		return ok
	})
}

func emitCaseTransform(name string, fn string, strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString(fmt.Sprintf("define ptr @%s(ptr %%s) {\n", name))
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	b.WriteString("  %len = call i64 @strlen(ptr %s_safe)\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %oom, label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %i = phi i64 [ 0, %entry ], [ %next, %body ]\n")
	b.WriteString("  %done = icmp eq i64 %i, %len\n")
	b.WriteString("  br i1 %done, label %end, label %body\n")
	b.WriteString("body:\n")
	b.WriteString("  %srcPtr = getelementptr i8, ptr %s_safe, i64 %i\n")
	b.WriteString("  %ch = load i8, ptr %srcPtr\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString(fmt.Sprintf("  %%conv = call i32 @%s(i32 %%ch32)\n", fn))
	b.WriteString("  %out = trunc i32 %conv to i8\n")
	b.WriteString("  %dstPtr = getelementptr i8, ptr %buf, i64 %i\n")
	b.WriteString("  store i8 %out, ptr %dstPtr\n")
	b.WriteString("  %next = add i64 %i, 1\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("end:\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %len\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("oom:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitTrimSpace(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeTrimSpaceFunc + "(ptr %s) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	b.WriteString("  %len = call i64 @strlen(ptr %s_safe)\n")
	b.WriteString("  %start = alloca i64\n")
	b.WriteString("  store i64 0, ptr %start\n")
	b.WriteString("  br label %loop_start\n")
	b.WriteString("loop_start:\n")
	b.WriteString("  %i = load i64, ptr %start\n")
	b.WriteString("  %done = icmp uge i64 %i, %len\n")
	b.WriteString("  br i1 %done, label %allspace, label %check_start\n")
	b.WriteString("check_start:\n")
	b.WriteString("  %ptr = getelementptr i8, ptr %s_safe, i64 %i\n")
	b.WriteString("  %ch = load i8, ptr %ptr\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %is = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iss = icmp ne i32 %is, 0\n")
	b.WriteString("  br i1 %iss, label %inc_start, label %start_done\n")
	b.WriteString("inc_start:\n")
	b.WriteString("  %ni = add i64 %i, 1\n")
	b.WriteString("  store i64 %ni, ptr %start\n")
	b.WriteString("  br label %loop_start\n")
	b.WriteString("start_done:\n")
	b.WriteString("  %startVal = load i64, ptr %start\n")
	b.WriteString("  %end = alloca i64\n")
	b.WriteString("  %last = sub i64 %len, 1\n")
	b.WriteString("  store i64 %last, ptr %end\n")
	b.WriteString("  br label %loop_end\n")
	b.WriteString("loop_end:\n")
	b.WriteString("  %j = load i64, ptr %end\n")
	b.WriteString("  %lt = icmp ult i64 %j, %startVal\n")
	b.WriteString("  br i1 %lt, label %allspace, label %check_end\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %ptr2 = getelementptr i8, ptr %s_safe, i64 %j\n")
	b.WriteString("  %ch2 = load i8, ptr %ptr2\n")
	b.WriteString("  %ch32b = zext i8 %ch2 to i32\n")
	b.WriteString("  %is2 = call i32 @isspace(i32 %ch32b)\n")
	b.WriteString("  %iss2 = icmp ne i32 %is2, 0\n")
	b.WriteString("  br i1 %iss2, label %dec_end, label %end_done\n")
	b.WriteString("dec_end:\n")
	b.WriteString("  %nj = sub i64 %j, 1\n")
	b.WriteString("  store i64 %nj, ptr %end\n")
	b.WriteString("  br label %loop_end\n")
	b.WriteString("end_done:\n")
	b.WriteString("  %endVal = load i64, ptr %end\n")
	b.WriteString("  %newLen = sub i64 %endVal, %startVal\n")
	b.WriteString("  %newLen2 = add i64 %newLen, 1\n")
	b.WriteString("  %total = add i64 %newLen2, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %allspace, label %copy\n")
	b.WriteString("copy:\n")
	b.WriteString("  %src = getelementptr i8, ptr %s_safe, i64 %startVal\n")
	b.WriteString("  call ptr @memcpy(ptr %buf, ptr %src, i64 %newLen2)\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %newLen2\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("allspace:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitRepeat(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeRepeatFunc + "(ptr %s, i64 %count) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	b.WriteString("  %nonpos = icmp sle i64 %count, 0\n")
	b.WriteString("  br i1 %nonpos, label %repeat_empty, label %check_one\n")
	b.WriteString("repeat_empty:\n")
	b.WriteString(fmt.Sprintf("  %%empty_ptr = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty_ptr\n")
	b.WriteString("check_one:\n")
	b.WriteString("  %one = icmp eq i64 %count, 1\n")
	b.WriteString("  br i1 %one, label %repeat_single, label %cont\n")
	b.WriteString("repeat_single:\n")
	b.WriteString("  ret ptr %s_safe\n")
	b.WriteString("cont:\n")
	b.WriteString("  %len = call i64 @strlen(ptr %s_safe)\n")
	b.WriteString("  %len0 = icmp eq i64 %len, 0\n")
	b.WriteString("  br i1 %len0, label %repeat_empty, label %cont_work\n")
	b.WriteString("cont_work:\n")
	b.WriteString("  %total = mul i64 %len, %count\n")
	b.WriteString("  %alloc = add i64 %total, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %alloc)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %repeat_empty, label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %i = phi i64 [ 0, %cont_work ], [ %next, %body ]\n")
	b.WriteString("  %done = icmp eq i64 %i, %count\n")
	b.WriteString("  br i1 %done, label %end, label %body\n")
	b.WriteString("body:\n")
	b.WriteString("  %offset = mul i64 %i, %len\n")
	b.WriteString("  %dst = getelementptr i8, ptr %buf, i64 %offset\n")
	b.WriteString("  call ptr @memcpy(ptr %dst, ptr %s_safe, i64 %len)\n")
	b.WriteString("  %next = add i64 %i, 1\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("end:\n")
	b.WriteString("  %endPtr = getelementptr i8, ptr %buf, i64 %total\n")
	b.WriteString("  store i8 0, ptr %endPtr\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("}\n")
	return b.String()
}

func emitReplace(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeReplaceFunc + "(ptr %s, ptr %old, ptr %new) {\n")
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	emitNormalizedRuntimeStringPtr(&b, strs, "old_safe", "old")
	emitNormalizedRuntimeStringPtr(&b, strs, "new_safe", "new")
	b.WriteString("  %oldLen = call i64 @strlen(ptr %old_safe)\n")
	b.WriteString("  %zero = icmp eq i64 %oldLen, 0\n")
	b.WriteString("  br i1 %zero, label %retorig, label %count_block\n")
	b.WriteString("retorig:\n")
	b.WriteString("  ret ptr %s_safe\n")
	b.WriteString("count_block:\n")
	b.WriteString("  %count = alloca i64\n")
	b.WriteString("  store i64 0, ptr %count\n")
	b.WriteString("  %cursor = alloca ptr\n")
	b.WriteString("  store ptr %s_safe, ptr %cursor\n")
	b.WriteString("  br label %count_loop\n")
	b.WriteString("count_loop:\n")
	b.WriteString("  %cur = load ptr, ptr %cursor\n")
	b.WriteString("  %found = call ptr @strstr(ptr %cur, ptr %old_safe)\n")
	b.WriteString("  %isnull = icmp eq ptr %found, null\n")
	b.WriteString("  br i1 %isnull, label %count_done, label %count_hit\n")
	b.WriteString("count_hit:\n")
	b.WriteString("  %c = load i64, ptr %count\n")
	b.WriteString("  %c1 = add i64 %c, 1\n")
	b.WriteString("  store i64 %c1, ptr %count\n")
	b.WriteString("  %next = getelementptr i8, ptr %found, i64 %oldLen\n")
	b.WriteString("  store ptr %next, ptr %cursor\n")
	b.WriteString("  br label %count_loop\n")
	b.WriteString("count_done:\n")
	b.WriteString("  %cfinal = load i64, ptr %count\n")
	b.WriteString("  %noccur = icmp eq i64 %cfinal, 0\n")
	b.WriteString("  br i1 %noccur, label %retorig, label %alloc\n")
	b.WriteString("alloc:\n")
	b.WriteString("  %lenS = call i64 @strlen(ptr %s_safe)\n")
	b.WriteString("  %lenN = call i64 @strlen(ptr %new_safe)\n")
	b.WriteString("  %diff = sub i64 %lenN, %oldLen\n")
	b.WriteString("  %extra = mul i64 %diff, %cfinal\n")
	b.WriteString("  %newLen = add i64 %lenS, %extra\n")
	b.WriteString("  %total = add i64 %newLen, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %oom, label %alloc_ok\n")
	b.WriteString("alloc_ok:\n")
	b.WriteString("  %src = alloca ptr\n")
	b.WriteString("  %dst = alloca ptr\n")
	b.WriteString("  store ptr %s_safe, ptr %src\n")
	b.WriteString("  store ptr %buf, ptr %dst\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("loop:\n")
	b.WriteString("  %srcv = load ptr, ptr %src\n")
	b.WriteString("  %found2 = call ptr @strstr(ptr %srcv, ptr %old_safe)\n")
	b.WriteString("  %isnull2 = icmp eq ptr %found2, null\n")
	b.WriteString("  br i1 %isnull2, label %copy_tail, label %copy_seg\n")
	b.WriteString("copy_seg:\n")
	b.WriteString("  %srcInt = ptrtoint ptr %srcv to i64\n")
	b.WriteString("  %foundInt = ptrtoint ptr %found2 to i64\n")
	b.WriteString("  %segLen = sub i64 %foundInt, %srcInt\n")
	b.WriteString("  %dstv = load ptr, ptr %dst\n")
	b.WriteString("  call ptr @memcpy(ptr %dstv, ptr %srcv, i64 %segLen)\n")
	b.WriteString("  %dst2 = getelementptr i8, ptr %dstv, i64 %segLen\n")
	b.WriteString("  call ptr @memcpy(ptr %dst2, ptr %new_safe, i64 %lenN)\n")
	b.WriteString("  %dst3 = getelementptr i8, ptr %dst2, i64 %lenN\n")
	b.WriteString("  store ptr %dst3, ptr %dst\n")
	b.WriteString("  %nextSrc = getelementptr i8, ptr %found2, i64 %oldLen\n")
	b.WriteString("  store ptr %nextSrc, ptr %src\n")
	b.WriteString("  br label %loop\n")
	b.WriteString("copy_tail:\n")
	b.WriteString("  %srcv2 = load ptr, ptr %src\n")
	b.WriteString("  %tailLen = call i64 @strlen(ptr %srcv2)\n")
	b.WriteString("  %dstv2 = load ptr, ptr %dst\n")
	b.WriteString("  call ptr @memcpy(ptr %dstv2, ptr %srcv2, i64 %tailLen)\n")
	b.WriteString("  %end = getelementptr i8, ptr %dstv2, i64 %tailLen\n")
	b.WriteString("  store i8 0, ptr %end\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("oom:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitIntToStr(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	fmtName, fmtLen := stringGlobalRef(strs, "%ld")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeIntToStrFunc + "(i64 %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString(fmt.Sprintf("  %%fmt = %s\n", stringGEP(fmtName, fmtLen)))
	b.WriteString("  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, i64 %v)\n")
	b.WriteString("  %len = sext i32 %len32 to i64\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %oom, label %format\n")
	b.WriteString("format:\n")
	b.WriteString("  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, i64 %v)\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("oom:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitFloatToStr(strs stringPool) string {
	var b strings.Builder
	emptyName, emptyLen := stringGlobalRef(strs, "")
	fmtName, fmtLen := stringGlobalRef(strs, "%g")
	b.WriteString("define ptr @" + intrinsics.LLVMRuntimeFloatToStrFunc + "(double %v) {\n")
	b.WriteString("entry:\n")
	b.WriteString(fmt.Sprintf("  %%fmt = %s\n", stringGEP(fmtName, fmtLen)))
	b.WriteString("  %len32 = call i32 (ptr, i64, ptr, ...) @snprintf(ptr null, i64 0, ptr %fmt, double %v)\n")
	b.WriteString("  %len = sext i32 %len32 to i64\n")
	b.WriteString("  %total = add i64 %len, 1\n")
	b.WriteString("  %buf = call ptr @malloc(i64 %total)\n")
	b.WriteString("  %bufNull = icmp eq ptr %buf, null\n")
	b.WriteString("  br i1 %bufNull, label %oom, label %format\n")
	b.WriteString("format:\n")
	b.WriteString("  call i32 (ptr, i64, ptr, ...) @snprintf(ptr %buf, i64 %total, ptr %fmt, double %v)\n")
	b.WriteString("  ret ptr %buf\n")
	b.WriteString("oom:\n")
	b.WriteString(fmt.Sprintf("  %%empty = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  ret ptr %empty\n")
	b.WriteString("}\n")
	return b.String()
}

func emitParseInt(structs structPool, strs stringPool) string {
	var b strings.Builder
	resultName := intrinsics.LLVMResultStructName("int", "Error")
	emptyName, emptyLen := stringGlobalRef(strs, "")
	errName, errLen := stringGlobalRef(strs, "invalid int")
	b.WriteString(fmt.Sprintf("define %%%s @%s(ptr %%s) {\n", resultName, intrinsics.LLVMRuntimeParseIntFunc))
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	b.WriteString("  %endptr = alloca ptr\n")
	b.WriteString("  %val = call i64 @strtol(ptr %s_safe, ptr %endptr, i32 10)\n")
	b.WriteString("  %end = load ptr, ptr %endptr\n")
	b.WriteString("  %same = icmp eq ptr %end, %s_safe\n")
	b.WriteString("  br i1 %same, label %fail, label %checktail\n")
	b.WriteString("checktail:\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("skip_ws:\n")
	b.WriteString("  %cur = load ptr, ptr %endptr\n")
	b.WriteString("  %ch = load i8, ptr %cur\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %isws = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iswsb = icmp ne i32 %isws, 0\n")
	b.WriteString("  br i1 %iswsb, label %skip_inc, label %check_end\n")
	b.WriteString("skip_inc:\n")
	b.WriteString("  %next = getelementptr i8, ptr %cur, i64 1\n")
	b.WriteString("  store ptr %next, ptr %endptr\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %end2 = load ptr, ptr %endptr\n")
	b.WriteString("  %ch2 = load i8, ptr %end2\n")
	b.WriteString("  %ch2_32 = zext i8 %ch2 to i32\n")
	b.WriteString("  %iszero = icmp eq i32 %ch2_32, 0\n")
	b.WriteString("  br i1 %iszero, label %ok, label %fail\n")
	b.WriteString("ok:\n")
	b.WriteString(fmt.Sprintf("  %%okmsg = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  %okerr = insertvalue %Error undef, ptr %okmsg, 0\n")
	b.WriteString("  %r0 = insertvalue %" + resultName + " undef, i8 1, 0\n")
	b.WriteString("  %r1 = insertvalue %" + resultName + " %r0, i64 %val, 1\n")
	b.WriteString("  %r2 = insertvalue %" + resultName + " %r1, %Error %okerr, 2\n")
	b.WriteString("  ret %" + resultName + " %r2\n")
	b.WriteString("fail:\n")
	b.WriteString(fmt.Sprintf("  %%errmsg = %s\n", stringGEP(errName, errLen)))
	b.WriteString("  %errv = insertvalue %Error undef, ptr %errmsg, 0\n")
	b.WriteString("  %f0 = insertvalue %" + resultName + " undef, i8 0, 0\n")
	b.WriteString("  %f1 = insertvalue %" + resultName + " %f0, i64 0, 1\n")
	b.WriteString("  %f2 = insertvalue %" + resultName + " %f1, %Error %errv, 2\n")
	b.WriteString("  ret %" + resultName + " %f2\n")
	b.WriteString("}\n")
	return b.String()
}

func emitParseFloat(structs structPool, strs stringPool) string {
	var b strings.Builder
	resultName := intrinsics.LLVMResultStructName("float", "Error")
	emptyName, emptyLen := stringGlobalRef(strs, "")
	errName, errLen := stringGlobalRef(strs, "invalid float")
	b.WriteString(fmt.Sprintf("define %%%s @%s(ptr %%s) {\n", resultName, intrinsics.LLVMRuntimeParseFloatFunc))
	b.WriteString("entry:\n")
	emitNormalizedRuntimeStringPtr(&b, strs, "s_safe", "s")
	b.WriteString("  %endptr = alloca ptr\n")
	b.WriteString("  %val = call double @strtod(ptr %s_safe, ptr %endptr)\n")
	b.WriteString("  %end = load ptr, ptr %endptr\n")
	b.WriteString("  %same = icmp eq ptr %end, %s_safe\n")
	b.WriteString("  br i1 %same, label %fail, label %checktail\n")
	b.WriteString("checktail:\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("skip_ws:\n")
	b.WriteString("  %cur = load ptr, ptr %endptr\n")
	b.WriteString("  %ch = load i8, ptr %cur\n")
	b.WriteString("  %ch32 = zext i8 %ch to i32\n")
	b.WriteString("  %isws = call i32 @isspace(i32 %ch32)\n")
	b.WriteString("  %iswsb = icmp ne i32 %isws, 0\n")
	b.WriteString("  br i1 %iswsb, label %skip_inc, label %check_end\n")
	b.WriteString("skip_inc:\n")
	b.WriteString("  %next = getelementptr i8, ptr %cur, i64 1\n")
	b.WriteString("  store ptr %next, ptr %endptr\n")
	b.WriteString("  br label %skip_ws\n")
	b.WriteString("check_end:\n")
	b.WriteString("  %end2 = load ptr, ptr %endptr\n")
	b.WriteString("  %ch2 = load i8, ptr %end2\n")
	b.WriteString("  %ch2_32 = zext i8 %ch2 to i32\n")
	b.WriteString("  %iszero = icmp eq i32 %ch2_32, 0\n")
	b.WriteString("  br i1 %iszero, label %ok, label %fail\n")
	b.WriteString("ok:\n")
	b.WriteString(fmt.Sprintf("  %%okmsg = %s\n", stringGEP(emptyName, emptyLen)))
	b.WriteString("  %okerr = insertvalue %Error undef, ptr %okmsg, 0\n")
	b.WriteString("  %r0 = insertvalue %" + resultName + " undef, i8 1, 0\n")
	b.WriteString("  %r1 = insertvalue %" + resultName + " %r0, double %val, 1\n")
	b.WriteString("  %r2 = insertvalue %" + resultName + " %r1, %Error %okerr, 2\n")
	b.WriteString("  ret %" + resultName + " %r2\n")
	b.WriteString("fail:\n")
	b.WriteString(fmt.Sprintf("  %%errmsg = %s\n", stringGEP(errName, errLen)))
	b.WriteString("  %errv = insertvalue %Error undef, ptr %errmsg, 0\n")
	b.WriteString("  %f0 = insertvalue %" + resultName + " undef, i8 0, 0\n")
	b.WriteString("  %f1 = insertvalue %" + resultName + " %f0, double 0.0, 1\n")
	b.WriteString("  %f2 = insertvalue %" + resultName + " %f1, %Error %errv, 2\n")
	b.WriteString("  ret %" + resultName + " %f2\n")
	b.WriteString("}\n")
	return b.String()
}
func renderLLVMMainHeader() string {
	return "define i32 @main(i32 %argc, ptr %argv) {\nentry:\n"
}

func renderLLVMMainPrelude(b *strings.Builder, hasGlobals bool) {
	b.WriteString("  call void @" + intrinsics.LLVMRuntimeSetArgsFunc + "(i32 %argc, ptr %argv)\n")
	if hasGlobals {
		b.WriteString("  call void @" + intrinsics.LLVMRuntimeInitGlobalsFunc + "()\n")
	}
}

func renderLLVMMainFinalizer(b *strings.Builder, ctx *funcCtx) {
	if !ctx.terminated {
		b.WriteString("  ret i32 0\n")
	}
}

func renderLLVMMainBody(plan llvmMainRenderPlan, funcs map[string]llvmFuncSig, globals map[string]globalSlot, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	var b strings.Builder
	b.WriteString(renderLLVMMainHeader())
	renderLLVMMainPrelude(&b, plan.hasGlobals)
	ctx := newFuncCtx(enums, structs, ifaces, strs, true, globals)
	ctx.returnType = ast.TypeVoid
	if err := emitFunctionBodyMIR(&b, ctx, plan.fn, funcs); err != nil {
		return "", err
	}
	renderLLVMMainFinalizer(&b, ctx)
	b.WriteString("}\n")
	return b.String(), nil
}

func renderLLVMFunctionHeader(plan llvmFuncRenderPlan) string {
	paramParts := make([]string, 0, len(plan.abi.Params))
	for _, p := range plan.abi.Params {
		paramParts = append(paramParts, fmt.Sprintf("%s %%%s", p.LLVMType, p.Name))
	}
	return fmt.Sprintf("define %s @%s(%s) {\nentry:\n", plan.abi.LLVMRetType, plan.fn.Name, strings.Join(paramParts, ", "))
}

func renderLLVMFunctionFinalizer(b *strings.Builder, ctx *funcCtx, plan llvmFuncRenderPlan, abiEnv llvmABIEnv) error {
	if ctx.terminated {
		return nil
	}
	if ctx.returnType != ast.TypeVoid {
		b.WriteString(fmt.Sprintf("  ; missing return in function %s, defaulting\n", plan.fn.Name))
	}
	storageABI := intrinsics.LLVMStorageABI{}
	if ctx.returnType != ast.TypeVoid {
		var err error
		storageABI, err = abiEnv.storageABIOrError(ctx.returnType, "llvm backend: unsupported default return type '%s' for '%s'", ctx.returnType, plan.fn.Name)
		if err != nil {
			return err
		}
	}
	b.WriteString(intrinsics.FormatLLVMDefaultReturn(ctx.returnType, plan.abi.LLVMRetType, storageABI.DefaultValue))
	return nil
}

func renderLLVMFunctionBody(plan llvmFuncRenderPlan, funcs map[string]llvmFuncSig, globals map[string]globalSlot, enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool) (string, error) {
	abiEnv := llvmABIEnv{enums: enums, structs: structs, ifaces: ifaces}
	var b strings.Builder
	b.WriteString(renderLLVMFunctionHeader(plan))
	ctx := newFuncCtx(enums, structs, ifaces, strs, false, globals)
	ctx.returnType = plan.abi.NormalizedRet
	for _, p := range plan.abi.Params {
		ptr, err := ctx.allocaOrError(&b, p.NormalizedType, "llvm backend: unsupported param type '%s' for '%s.%s'", p.NormalizedType, plan.fn.Name, p.Name)
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("  store %s %%%s, ptr %s\n", p.LLVMType, p.Name, ptr))
		ctx.vars[p.Name] = varSlot{ptr: ptr, typ: p.NormalizedType}
	}
	if err := emitFunctionBodyMIR(&b, ctx, plan.fn, funcs); err != nil {
		return "", err
	}
	if err := renderLLVMFunctionFinalizer(&b, ctx, plan, abiEnv); err != nil {
		return "", err
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type irCtx struct {
	tmp int
	lbl int
}

type llvmFuncSig = backendmeta.FuncSig

func newIRCtx() *irCtx { return &irCtx{} }

func (c *irCtx) nextTmp() string {
	c.tmp++
	return fmt.Sprintf("%%t%d", c.tmp)
}

func (c *irCtx) nextLabel(prefix string) string {
	c.lbl++
	return fmt.Sprintf("%s%d", prefix, c.lbl)
}

type varSlot struct {
	ptr string
	typ ast.Type
}

type funcCtx struct {
	ir         *irCtx
	vars       map[string]varSlot
	globals    map[string]globalSlot
	returnType ast.Type
	enums      enumInfo
	structs    structPool
	ifaces     interfacePool
	strs       stringPool
	terminated bool
	isMain     bool
}

type llvmABIEnv struct {
	enums   enumInfo
	structs structPool
	ifaces  interfacePool
}

type llvmStructFieldRef struct {
	Base string
	Info structFieldInfo
	Idx  int
	ABI  intrinsics.LLVMValueABI
}

func newFuncCtx(enums enumInfo, structs structPool, ifaces interfacePool, strs stringPool, isMain bool, globals map[string]globalSlot) *funcCtx {
	return &funcCtx{
		ir:      newIRCtx(),
		vars:    map[string]varSlot{},
		globals: globals,
		enums:   enums,
		structs: structs,
		ifaces:  ifaces,
		strs:    strs,
		isMain:  isMain,
	}
}

func (c *funcCtx) abi() llvmABIEnv {
	return llvmABIEnv{enums: c.enums, structs: c.structs, ifaces: c.ifaces}
}

func (c *funcCtx) bindingSlot(name string) (varSlot, bool) {
	return llvmBindingLookupPlan{ctx: c, name: name}.lookup()
}

func (c *funcCtx) resolveStructField(t ast.Type, field string) (llvmStructFieldRef, bool) {
	return llvmStructFieldResolvePlan{ctx: c, typ: t, field: field}.resolve()
}

func (c *funcCtx) resolveStructFieldOrError(t ast.Type, field string, format string, args ...any) (llvmStructFieldRef, error) {
	ref, ok := c.resolveStructField(t, field)
	if ok {
		return ref, nil
	}
	return llvmStructFieldRef{}, fmt.Errorf(format, args...)
}

func (c *funcCtx) emitLoadLLVMValue(ptr string, t ast.Type) (string, string, ast.Type, bool) {
	return llvmLoadValuePlan{ctx: c, ptr: ptr, typ: t}.emit()
}

func (p llvmLoadValuePlan) emit() (string, string, ast.Type, bool) {
	c := p.ctx
	ptr := p.ptr
	t := p.typ
	abi, ok := c.abi().valueABI(t)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	tmp := c.ir.nextTmp()
	code := fmt.Sprintf("  %s = load %s, ptr %s\n", tmp, abi.LLVMType, ptr)
	return code, tmp, abi.NormalizedType, true
}

func (c *funcCtx) allocaOrError(b *strings.Builder, t ast.Type, format string, args ...any) (string, error) {
	abi, err := c.abi().storageABIOrError(t, format, args...)
	if err != nil {
		return "", err
	}
	return llvmAllocaPlan{ctx: c, b: b, typ: abi.NormalizedType}.emitWithLLVMType(abi.LLVMType), nil
}

func (c *funcCtx) alloca(b *strings.Builder, t ast.Type) string {
	abi, ok := c.abi().storageABI(t)
	if !ok {
		return ""
	}
	return llvmAllocaPlan{ctx: c, b: b, typ: abi.NormalizedType}.emitWithLLVMType(abi.LLVMType)
}

func (c *funcCtx) emitFieldPathPtr(ptr string, typ ast.Type, fields []string) (string, string, ast.Type, bool) {
	return llvmFieldPathPtrPlan{ctx: c, ptr: ptr, typ: typ, fields: fields}.emit()
}

func (p llvmBindingLookupPlan) lookup() (varSlot, bool) {
	if slot, ok := p.ctx.vars[p.name]; ok {
		return slot, true
	}
	if g, ok := p.ctx.globals[p.name]; ok {
		return varSlot{ptr: g.ptr, typ: g.typ}, true
	}
	return varSlot{}, false
}

func (p llvmFieldPathPtrPlan) emit() (string, string, ast.Type, bool) {
	code := ""
	currentPtr := p.ptr
	currentType := p.typ
	for _, field := range p.fields {
		fieldRef, ok := p.ctx.resolveStructField(currentType, field)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		tmp := p.ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = getelementptr inbounds %%%s, ptr %s, i32 0, i32 %d\n", tmp, fieldRef.Base, currentPtr, fieldRef.Idx)
		currentPtr = tmp
		currentType = fieldRef.Info.Type
	}
	return code, currentPtr, currentType, true
}

func (p llvmStructFieldResolvePlan) resolve() (llvmStructFieldRef, bool) {
	base, _ := intrinsics.LLVMGenericBase(p.typ)
	if base == "" {
		base = string(normalizeLLVMType(p.typ))
	}
	info, ok := p.ctx.structs.byName[base]
	if !ok {
		return llvmStructFieldRef{}, false
	}
	idx, ok := info.FieldIndex[p.field]
	if !ok {
		return llvmStructFieldRef{}, false
	}
	fieldInfo := info.Fields[idx]
	abi, ok := p.ctx.abi().valueABI(fieldInfo.Type)
	if !ok {
		return llvmStructFieldRef{}, false
	}
	return llvmStructFieldRef{Base: base, Info: fieldInfo, Idx: idx, ABI: abi}, true
}

func (p llvmAllocaPlan) emitWithLLVMType(llvmType string) string {
	ptr := p.ctx.ir.nextTmp()
	p.b.WriteString(fmt.Sprintf("  %s = alloca %s\n", ptr, llvmType))
	return ptr
}

func (p llvmAggregateExtractPlan) emit() (string, string, ast.Type) {
	tmp := p.ctx.ir.nextTmp()
	code := fmt.Sprintf("  %s = extractvalue %%%s %s, %d\n", tmp, p.base, p.agg, p.ref.Idx)
	return code, tmp, p.ref.Info.Type
}

func (p llvmAggregateInsertPlan) emit() (string, string) {
	next := p.ctx.ir.nextTmp()
	code := fmt.Sprintf("  %s = insertvalue %s %s, %s %s, %d\n", next, p.structType, p.agg, p.ref.ABI.LLVMType, p.value, p.ref.Idx)
	return code, next
}

func (c *funcCtx) emitAggregateFieldExtract(base string, agg string, ref llvmStructFieldRef) (string, string, ast.Type) {
	return llvmAggregateExtractPlan{ctx: c, base: base, agg: agg, ref: ref}.emit()
}

func (c *funcCtx) emitAggregateFieldInsert(structType string, agg string, ref llvmStructFieldRef, value string) (string, string) {
	return llvmAggregateInsertPlan{ctx: c, structType: structType, agg: agg, ref: ref, value: value}.emit()
}

func (c *funcCtx) emitEnumSwitchDispatch(b *strings.Builder, subjVal string, subjType ast.Type, defaultLabel string, variants []string, labelPrefix string) ([]string, bool) {
	subjABI, err := c.abi().valueABIOrError(subjType, "llvm backend: unsupported match subject type '%s'", subjType)
	if err != nil {
		return nil, false
	}
	caseLabels := make([]string, 0, len(variants))
	b.WriteString(fmt.Sprintf("  switch %s %s, label %%%s [\n", subjABI.LLVMType, subjVal, defaultLabel))
	for _, variant := range variants {
		lbl := c.ir.nextLabel(labelPrefix)
		caseLabels = append(caseLabels, lbl)
		idx, ok := c.enums.variantIndex[variant]
		if !ok {
			return nil, false
		}
		b.WriteString(fmt.Sprintf("    %s %d, label %%%s\n", subjABI.LLVMType, idx, lbl))
	}
	b.WriteString("  ]\n")
	return caseLabels, true
}

func (c *funcCtx) emitTypedReturn(b *strings.Builder, value string, valueType ast.Type) bool {
	coerceCode, coerced, abi, ok := c.coerceTypedLLVMValue(c.returnType, value, valueType)
	if !ok {
		return false
	}
	b.WriteString(coerceCode)
	b.WriteString(fmt.Sprintf("  ret %s %s\n", abi.LLVMType, coerced))
	c.terminated = true
	return true
}

func (c *funcCtx) coerceTypedLLVMValue(targetType ast.Type, value string, valueType ast.Type) (string, string, intrinsics.LLVMValueABI, bool) {
	coerceCode, coerced, _, ok := coerceLLVMValueForTarget(c, value, valueType, targetType)
	if !ok {
		return "", "", intrinsics.LLVMValueABI{}, false
	}
	abi, err := c.abi().valueABIOrError(targetType, "llvm backend: unsupported value type '%s'", targetType)
	if err != nil {
		return "", "", intrinsics.LLVMValueABI{}, false
	}
	return coerceCode, coerced, abi, true
}

func emitFunctionBodyMIR(b *strings.Builder, ctx *funcCtx, fn *mir.FuncDecl, funcs map[string]llvmFuncSig) error {
	if fn == nil {
		return nil
	}
	plan, err := buildLLVMFuncBodyPlan(ctx, fn)
	if err != nil {
		return err
	}
	renderLLVMFuncLocalAllocas(b, ctx, plan.localABIs)
	b.WriteString(fmt.Sprintf("  br label %%%s\n", cfgLabelLLVM(plan.fn.CFG.Entry)))
	for _, name := range plan.topology.ReversePostOrderNames() {
		if err := renderLLVMCFGBlock(b, ctx, llvmCFGBlockRenderPlan{
			fnName:  plan.fn.Name,
			block:   plan.topology.Blocks[name],
			deadCFG: plan.deadByBlock[name],
		}, funcs); err != nil {
			return err
		}
	}
	ctx.terminated = true
	return nil
}

func buildLLVMFuncBodyPlan(ctx *funcCtx, fn *mir.FuncDecl) (llvmFuncBodyPlan, error) {
	if fn == nil {
		return llvmFuncBodyPlan{}, nil
	}
	if err := requireMIRCFGLLVM(fn); err != nil {
		return llvmFuncBodyPlan{}, err
	}
	lets := collectCFGLetTypesLLVM(fn)
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
	deadNames := mir.DeadCFGValueNamesFromAnalysis(fn, liveness)
	liveLets := map[string]ast.Type{}
	for name, typ := range lets {
		if _, dead := deadNames[name]; dead {
			continue
		}
		liveLets[name] = typ
	}
	localABIs, err := ctx.abi().sortedNamedStorageABIsOrError(fn.Name, liveLets)
	if err != nil {
		return llvmFuncBodyPlan{}, err
	}
	return llvmFuncBodyPlan{
		fn:          fn,
		topology:    topology,
		deadByBlock: deadByBlock,
		localABIs:   localABIs,
	}, nil
}

func renderLLVMFuncLocalAllocas(b *strings.Builder, ctx *funcCtx, localABIs []intrinsics.LLVMNamedStorageABI) {
	for _, local := range localABIs {
		ptr := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = alloca %s\n", ptr, local.Storage.LLVMType))
		ctx.vars[local.Name] = varSlot{ptr: ptr, typ: local.Storage.NormalizedType}
	}
}

func renderLLVMCFGBlock(b *strings.Builder, ctx *funcCtx, plan llvmCFGBlockRenderPlan, funcs map[string]llvmFuncSig) error {
	b.WriteString(fmt.Sprintf("%s:\n", cfgLabelLLVM(plan.block.Name)))
	for i, instr := range plan.block.Instrs {
		if plan.deadCFG[i] {
			continue
		}
		if !emitCFGInstrMIRLLVM(b, ctx, instr, funcs) {
			return fmt.Errorf("llvm backend: unsupported mir cfg instruction in function '%s': %T", plan.fnName, instr)
		}
	}
	if !emitTerminatorMIRLLVM(b, ctx, plan.block.Term, funcs) {
		return fmt.Errorf("llvm backend: unsupported mir cfg terminator in function '%s': %T", plan.fnName, plan.block.Term)
	}
	return nil
}

func requireMIRCFGLLVM(fn *mir.FuncDecl) error {
	if fn == nil {
		return fmt.Errorf("llvm backend: nil mir function")
	}
	if fn.CFG == nil {
		return fmt.Errorf("llvm backend: function '%s' missing mir cfg", fn.Name)
	}
	if !mir.HasUniqueCFGBindings(fn) {
		return fmt.Errorf("llvm backend: function '%s' has non-unique mir cfg bindings", fn.Name)
	}
	return nil
}

func collectCFGLetTypesLLVM(fn *mir.FuncDecl) map[string]ast.Type {
	out := map[string]ast.Type{}
	for name, typ := range mir.CollectCFGLetTypes(fn) {
		out[name] = normalizeLLVMType(baztypes.ToAST(typ))
	}
	return out
}

func cfgLabelLLVM(name string) string {
	return "mir_" + name
}

func emitUnaryValueStmtMIRLLVM(ctx *funcCtx, st *mir.UnaryOpStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmUnaryEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmUnaryEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	code, value, t, ok := emitExprMIRLLVM(ctx, st.Right, funcs)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	switch st.Op {
	case "-":
		if t == ast.TypeInt {
			tmp := ctx.ir.nextTmp()
			return code + fmt.Sprintf("  %s = sub i64 0, %s\n", tmp, value), tmp, ast.TypeInt, true
		}
		if t == ast.TypeFloat {
			tmp := ctx.ir.nextTmp()
			return code + fmt.Sprintf("  %s = fsub double 0.0, %s\n", tmp, value), tmp, ast.TypeFloat, true
		}
	case "!":
		if t == ast.TypeBool {
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = icmp eq i8 %s, 0\n", tmp, value)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
	}
	return "", "", ast.TypeInvalid, false
}

func emitBinaryValueStmtMIRLLVM(ctx *funcCtx, st *mir.BinaryOpStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmBinaryEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmBinaryEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	lc, lv, lt, ok := emitExprMIRLLVM(ctx, st.Left, funcs)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	rc, rv, rt, ok := emitExprMIRLLVM(ctx, st.Right, funcs)
	if !ok || lt != rt {
		return "", "", ast.TypeInvalid, false
	}
	code := lc + rc
	if lt == ast.TypeInt {
		if op, ok := mapIntArithOp(st.Op); ok {
			tmp := ctx.ir.nextTmp()
			return code + fmt.Sprintf("  %s = %s i64 %s, %s\n", tmp, op, lv, rv), tmp, ast.TypeInt, true
		}
		if op, ok := mapIntCmpOp(st.Op); ok {
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = icmp %s i64 %s, %s\n", tmp, op, lv, rv)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
		return "", "", ast.TypeInvalid, false
	}
	if lt == ast.TypeFloat {
		if op, ok := mapFloatArithOp(st.Op); ok {
			tmp := ctx.ir.nextTmp()
			return code + fmt.Sprintf("  %s = %s double %s, %s\n", tmp, op, lv, rv), tmp, ast.TypeFloat, true
		}
		if op, ok := mapFloatCmpOp(st.Op); ok {
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = fcmp %s double %s, %s\n", tmp, op, lv, rv)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
		return "", "", ast.TypeInvalid, false
	}
	if _, ok := ctx.enums.enumTypes[string(lt)]; ok {
		if st.Op == "==" || st.Op == "!=" {
			tmp := ctx.ir.nextTmp()
			cmp := "eq"
			if st.Op == "!=" {
				cmp = "ne"
			}
			code += fmt.Sprintf("  %s = icmp %s i64 %s, %s\n", tmp, cmp, lv, rv)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
	}
	if lt == ast.TypeBool {
		if st.Op == "&&" || st.Op == "||" {
			tmp := ctx.ir.nextTmp()
			irOp := "and"
			if st.Op == "||" {
				irOp = "or"
			}
			return code + fmt.Sprintf("  %s = %s i8 %s, %s\n", tmp, irOp, lv, rv), tmp, ast.TypeBool, true
		}
		if st.Op == "==" || st.Op == "!=" {
			tmp := ctx.ir.nextTmp()
			cmp := "eq"
			if st.Op == "!=" {
				cmp = "ne"
			}
			code += fmt.Sprintf("  %s = icmp %s i8 %s, %s\n", tmp, cmp, lv, rv)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
		return "", "", ast.TypeInvalid, false
	}
	if lt == ast.TypeString {
		if st.Op == "+" {
			tmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = call ptr @%s(ptr %s, ptr %s)\n", tmp, intrinsics.LLVMRuntimeStrConcatFunc, lv, rv)
			return code, tmp, ast.TypeString, true
		}
		if st.Op == "==" || st.Op == "!=" || st.Op == "<" || st.Op == "<=" || st.Op == ">" || st.Op == ">=" {
			cmpTmp := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = call i32 @%s(ptr %s, ptr %s)\n", cmpTmp, intrinsics.LLVMRuntimeStrCmpFunc, lv, rv)
			tmp := ctx.ir.nextTmp()
			cmp := "eq"
			switch st.Op {
			case "!=":
				cmp = "ne"
			case "<":
				cmp = "slt"
			case "<=":
				cmp = "sle"
			case ">":
				cmp = "sgt"
			case ">=":
				cmp = "sge"
			}
			code += fmt.Sprintf("  %s = icmp %s i32 %s, 0\n", tmp, cmp, cmpTmp)
			zextCode, out := boolToI8(ctx, tmp)
			return code + zextCode, out, ast.TypeBool, true
		}
	}
	if lt == ast.TypeAny && rt == ast.TypeAny && (st.Op == "==" || st.Op == "!=") {
		tmp := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = call i8 @%s(%%Any %s, %%Any %s)\n", tmp, intrinsics.LLVMRuntimeAnyEqFunc, lv, rv)
		if st.Op == "!=" {
			tmp2 := ctx.ir.nextTmp()
			code += fmt.Sprintf("  %s = xor i8 %s, 1\n", tmp2, tmp)
			return code, tmp2, ast.TypeBool, true
		}
		return code, tmp, ast.TypeBool, true
	}
	return "", "", ast.TypeInvalid, false
}

func emitCallValueStmtMIRLLVM(ctx *funcCtx, st *mir.CallStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmCallEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmCallEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	if intrinsics.IsBuiltinVoidCall(st.Func) {
		if len(st.Args) != 1 {
			return "", "", ast.TypeInvalid, false
		}
		argCode, argVal, argType, ok := emitExprMIRLLVM(ctx, st.Args[0], funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		isPrintln := st.Func == "println"
		fmtLit, ok := intrinsics.LLVMPrintfFormat(isPrintln, argType, ctx.enums.enumTypes[string(argType)])
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		switch {
		case argType == ast.TypeInt || ctx.enums.enumTypes[string(argType)]:
			fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			return argCode + fmtCode + fmt.Sprintf("  call i32 @printf(ptr %s, i64 %s)\n", fmtPtr, argVal), "", ast.TypeVoid, true
		case argType == ast.TypeFloat:
			fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			return argCode + fmtCode + fmt.Sprintf("  call i32 @printf(ptr %s, double %s)\n", fmtPtr, argVal), "", ast.TypeVoid, true
		case argType == ast.TypeBool:
			fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			trueCode, truePtr, ok := stringPtr(ctx, "true")
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			falseCode, falsePtr, ok := stringPtr(ctx, "false")
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			condCode, cond := boolToI1(ctx, argVal)
			code := argCode + fmtCode + trueCode + falseCode + condCode
			code += fmt.Sprintf("  %s = select i1 %s, ptr %s, ptr %s\n", tmp, cond, truePtr, falsePtr)
			code += fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, tmp)
			return code, "", ast.TypeVoid, true
		case argType == ast.TypeString:
			fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			nonnullCode, nonnullPtr, ok := nonNullStringPtr(ctx, argVal)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			return argCode + fmtCode + nonnullCode + fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, nonnullPtr), "", ast.TypeVoid, true
		case argType == ast.TypeAny:
			fmtCode, fmtPtr, ok := stringPtr(ctx, fmtLit)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			code := argCode + fmtCode
			code += fmt.Sprintf("  %s = call ptr @%s(%%Any %s)\n", tmp, intrinsics.LLVMRuntimeAnyToStrFunc, argVal)
			nonnullCode, nonnullPtr, ok := nonNullStringPtr(ctx, tmp)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			code += nonnullCode
			code += fmt.Sprintf("  call i32 @printf(ptr %s, ptr %s)\n", fmtPtr, nonnullPtr)
			return code, "", ast.TypeVoid, true
		default:
			return "", "", ast.TypeInvalid, false
		}
	}
	if code, value, typ, ok := emitLoweredBuiltinExprMIR(ctx, st.Func, st.Args, funcs); ok {
		return code, value, typ, true
	}
	sig, ok := funcs[st.Func]
	if !ok || len(sig.Params) != len(st.Args) {
		return "", "", ast.TypeInvalid, false
	}
	var b strings.Builder
	argParts := make([]string, 0, len(st.Args))
	for i, a := range st.Args {
		code, value, t, ok := emitExprMIRLLVM(ctx, a, funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		coerceCode, coerced, argABI, ok := ctx.coerceTypedLLVMValue(sig.Params[i], value, t)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		code += coerceCode
		value = coerced
		paramLLVMType, ok := intrinsics.MapLLVMCallParamType(st.Func, sig.Params[i], func(t ast.Type) (string, bool) {
			return ctx.abi().runtimeType(t)
		}, func(name string) bool {
			_, ok := ctx.structs.byName[name]
			return ok
		})
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		if paramLLVMType == "i1" && argABI.LLVMType == "i8" {
			i1Code, i1Value := boolToI1(ctx, value)
			code += i1Code
			value = i1Value
		}
		b.WriteString(code)
		argParts = append(argParts, fmt.Sprintf("%s%s %s", paramLLVMType, intrinsics.LLVMCallParamAttrs(st.Func, sig.Params[i]), value))
	}
	abi, err := ctx.abi().callABIOrError(st.Func, sig.Ret)
	if err != nil {
		return "", "", ast.TypeInvalid, false
	}
	ret := abi.NormalizedRet
	retType := abi.LLVMRetType
	switch abi.Convention {
	case intrinsics.LLVMCallVoid:
		b.WriteString(fmt.Sprintf("  call %s @%s(%s)\n", retType, st.Func, strings.Join(argParts, ", ")))
		return b.String(), "", ast.TypeVoid, true
	case intrinsics.LLVMCallSRet:
		tmpPtr := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = alloca %s\n", tmpPtr, retType))
		args := append([]string{fmt.Sprintf("ptr sret(%s) align 8 %s", retType, tmpPtr)}, argParts...)
		b.WriteString(fmt.Sprintf("  call void @%s(%s)\n", st.Func, strings.Join(args, ", ")))
		loadCode, tmp, loadType, ok := ctx.emitLoadLLVMValue(tmpPtr, ret)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		b.WriteString(loadCode)
		if strings.HasPrefix(st.Func, "__std_") {
			normCode, normValue, changed, ok := normalizeExternalStringValue(ctx, tmp, loadType)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			if changed {
				b.WriteString(normCode)
				tmp = normValue
			}
		}
		return b.String(), tmp, loadType, true
	case intrinsics.LLVMCallValue:
		valueABI, err := ctx.abi().valueABIOrError(ret, "llvm backend: unsupported return type '%s' for '%s'", ret, st.Func)
		if err != nil {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = call %s%s @%s(%s)\n", tmp, intrinsics.LLVMCallRetPrefix(st.Func, ret), retType, st.Func, strings.Join(argParts, ", ")))
		if code, value, typ, ok := decodeLLVMCompactBoolResultValue(ctx, tmp, retType, ret); ok {
			b.WriteString(code)
			return b.String(), value, typ, true
		}
		if ret == ast.TypeBool && retType == "i1" && valueABI.LLVMType == "i8" {
			tmp8 := ctx.ir.nextTmp()
			b.WriteString(fmt.Sprintf("  %s = zext i1 %s to i8\n", tmp8, tmp))
			return b.String(), tmp8, ret, true
		}
		if strings.HasPrefix(st.Func, "__std_") && retType == valueABI.LLVMType {
			normCode, normValue, changed, ok := normalizeExternalStringValue(ctx, tmp, ret)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			if changed {
				b.WriteString(normCode)
				return b.String(), normValue, ret, true
			}
		}
		if retType == valueABI.LLVMType {
			return b.String(), tmp, ret, true
		}
		tmpPtr := ctx.ir.nextTmp()
		b.WriteString(fmt.Sprintf("  %s = alloca %s\n", tmpPtr, retType))
		b.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", retType, tmp, tmpPtr))
		loadCode, loadTmp, loadType, ok := ctx.emitLoadLLVMValue(tmpPtr, ret)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		b.WriteString(loadCode)
		if strings.HasPrefix(st.Func, "__std_") {
			normCode, normValue, changed, ok := normalizeExternalStringValue(ctx, loadTmp, loadType)
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			if changed {
				b.WriteString(normCode)
				loadTmp = normValue
			}
		}
		return b.String(), loadTmp, loadType, true
	}
	return "", "", ast.TypeInvalid, false
}

func decodeLLVMCompactBoolResultValue(ctx *funcCtx, value string, abiType string, logicalType ast.Type) (string, string, ast.Type, bool) {
	logicalType = normalizeLLVMType(logicalType)
	if logicalType != ast.Type(intrinsics.LLVMResultStructName("bool", "Error")) {
		return "", "", ast.TypeInvalid, false
	}
	valueABI, ok := ctx.abi().valueABI(logicalType)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}

	word0 := ctx.ir.nextTmp()
	code := ""
	var errPtr string
	switch abiType {
	case "[2 x i64]":
		code += fmt.Sprintf("  %s = extractvalue [2 x i64] %s, 0\n", word0, value)
		errBits := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = extractvalue [2 x i64] %s, 1\n", errBits, value)
		errPtr = ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", errPtr, errBits)
	case "{ i64, ptr }":
		code += fmt.Sprintf("  %s = extractvalue { i64, ptr } %s, 0\n", word0, value)
		errPtr = ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = extractvalue { i64, ptr } %s, 1\n", errPtr, value)
	default:
		return "", "", ast.TypeInvalid, false
	}

	ok8 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = trunc i64 %s to i8\n", ok8, word0)
	word1 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = lshr i64 %s, 8\n", word1, word0)
	val8 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = trunc i64 %s to i8\n", val8, word1)
	errValue := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = insertvalue %%Error undef, ptr %s, 0\n", errValue, errPtr)
	result0 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = insertvalue %s undef, i8 %s, 0\n", result0, valueABI.LLVMType, ok8)
	result1 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = insertvalue %s %s, i8 %s, 1\n", result1, valueABI.LLVMType, result0, val8)
	result2 := ctx.ir.nextTmp()
	code += fmt.Sprintf("  %s = insertvalue %s %s, %%Error %s, 2\n", result2, valueABI.LLVMType, result1, errValue)
	return code, result2, valueABI.NormalizedType, true
}

func emitFieldAccessValueStmtMIRLLVM(ctx *funcCtx, st *mir.FieldAccessStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmFieldAccessEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmFieldAccessEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	objCode, objVal, objType, ok := emitExprMIRLLVM(ctx, st.Object, funcs)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	fieldRef, err := ctx.resolveStructFieldOrError(objType, st.Field, "llvm backend: unsupported field access '%s.%s'", objType, st.Field)
	if err != nil {
		return "", "", ast.TypeInvalid, false
	}
	extractCode, tmp, outType := ctx.emitAggregateFieldExtract(fieldRef.Base, objVal, fieldRef)
	return objCode + extractCode, tmp, outType, true
}

func emitStructLitValueStmtMIRLLVM(ctx *funcCtx, st *mir.StructLitStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmStructLitEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmStructLitEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	base, ok := intrinsics.LLVMGenericBase(ast.Type(st.TypeName))
	if !ok {
		base = st.TypeName
	}
	if _, ok := ctx.structs.byName[base]; !ok {
		return "", "", ast.TypeInvalid, false
	}
	structType := "%" + base
	code := ""
	curr := "undef"
	for _, f := range st.Fields {
		fieldRef, err := ctx.resolveStructFieldOrError(ast.Type(base), f.Name, "llvm backend: unsupported struct field '%s.%s'", base, f.Name)
		if err != nil {
			return "", "", ast.TypeInvalid, false
		}
		fcode, fval, ftype, ok := emitExprMIRLLVM(ctx, f.Value, funcs)
		if !ok || ftype != fieldRef.Info.Type {
			return "", "", ast.TypeInvalid, false
		}
		code += fcode
		insertCode, next := ctx.emitAggregateFieldInsert(structType, curr, fieldRef, fval)
		code += insertCode
		curr = next
	}
	return code, curr, ast.Type(base), true
}

func emitMatchValueStmtMIRLLVM(ctx *funcCtx, st *mir.MatchValueStmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmMatchValueEmitPlan{ctx: ctx, stmt: st, funcs: funcs}.emit()
}

func (p llvmMatchValueEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	st := p.stmt
	funcs := p.funcs
	resolved := baztypes.ToAST(st.Type)
	if resolved == ast.TypeInvalid {
		return "", "", ast.TypeInvalid, false
	}
	code, subjVal, subjType, ok := emitExprMIRLLVM(ctx, st.Subject, funcs)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	if _, ok := ctx.enums.enumTypes[string(subjType)]; !ok {
		return "", "", ast.TypeInvalid, false
	}
	storageABI, err := ctx.abi().storageABIOrError(resolved, "llvm backend: unsupported match value type '%s'", resolved)
	if err != nil {
		return "", "", ast.TypeInvalid, false
	}
	llvmType := storageABI.LLVMType
	mergeLabel := ctx.ir.nextLabel("match_expr_end")
	defaultLabel := ctx.ir.nextLabel("match_expr_default")
	grouped := mir.GroupMatchExprArms(st.Arms)
	var b strings.Builder
	b.WriteString(code)
	variants := make([]string, 0, len(grouped))
	for _, g := range grouped {
		variants = append(variants, g.Variant)
	}
	caseLabels, ok := ctx.emitEnumSwitchDispatch(&b, subjVal, subjType, defaultLabel, variants, "match_expr_arm")
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	phiVals := make([]string, 0, len(st.Arms))
	for i, g := range grouped {
		b.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
		entries, ok := emitGuardedMatchValueExprMIR(&b, ctx, g.Arms, funcs, mergeLabel, resolved, caseLabels[i])
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		phiVals = append(phiVals, entries...)
	}
	b.WriteString(fmt.Sprintf("%s:\n", defaultLabel))
	defVal := storageABI.DefaultValue
	b.WriteString(fmt.Sprintf("  br label %%%s\n", mergeLabel))
	phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", defVal, defaultLabel))
	b.WriteString(fmt.Sprintf("%s:\n", mergeLabel))
	tmp := ctx.ir.nextTmp()
	b.WriteString(fmt.Sprintf("  %s = phi %s %s\n", tmp, llvmType, strings.Join(phiVals, ", ")))
	return b.String(), tmp, resolved, true
}

func emitValueStmtExprMIRLLVM(ctx *funcCtx, s mir.Stmt, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	plan, ok := buildLLVMValueStmtEmitPlan(ctx, s, funcs)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	return plan.emitExpr()
}

func emitValueStmtMIRLLVM(b *strings.Builder, ctx *funcCtx, s mir.Stmt, funcs map[string]llvmFuncSig) bool {
	plan, ok := buildLLVMValueStmtEmitPlan(ctx, s, funcs)
	if !ok {
		return false
	}
	return plan.emitInto(b)
}

func coerceLLVMValueForTarget(ctx *funcCtx, value string, valueType ast.Type, targetType ast.Type) (string, string, ast.Type, bool) {
	return llvmValueCoercionPlan{ctx: ctx, value: value, valueType: valueType, targetType: targetType}.emit()
}

func (p llvmValueCoercionPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	value := p.value
	valueType := p.valueType
	targetType := p.targetType
	kind, coercedType, ok := intrinsics.ClassifyLLVMValueCoercion(targetType, valueType)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	switch kind {
	case intrinsics.LLVMValueDirect:
		return "", value, coercedType, true
	case intrinsics.LLVMValueBoxAny:
		boxCode, boxed, ok := boxToAny(ctx, value, valueType)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		return boxCode, boxed, coercedType, true
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func emitStoreLLVMValue(b *strings.Builder, ctx *funcCtx, ptr string, targetType ast.Type, value string, valueType ast.Type) bool {
	return llvmStoreEmitPlan{ctx: ctx, ptr: ptr, targetType: targetType, value: value, valueType: valueType}.emitInto(b)
}

func (p llvmStoreEmitPlan) emitInto(b *strings.Builder) bool {
	ctx := p.ctx
	code, coerced, abi, ok := ctx.coerceTypedLLVMValue(p.targetType, p.value, p.valueType)
	if !ok {
		return false
	}
	b.WriteString(code)
	b.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", abi.LLVMType, coerced, p.ptr))
	return true
}

func emitCFGInstrMIRLLVM(b *strings.Builder, ctx *funcCtx, s mir.Stmt, funcs map[string]llvmFuncSig) bool {
	if name, ok := mir.ValueStmtBindingName(s); ok && name == "_" {
		if expr, ok := mir.ValueStmtExpr(s); ok {
			code, _, _, ok := emitExprMIRLLVM(ctx, expr, funcs)
			if ok {
				b.WriteString(code)
				return true
			}
		}
	}
	if emitValueStmtMIRLLVM(b, ctx, s, funcs) {
		return true
	}
	plan, ok := buildLLVMCFGInstrEmitPlan(ctx, s, funcs)
	if !ok {
		return false
	}
	return plan.emitInto(b)
}

func buildLLVMValueStmtEmitPlan(ctx *funcCtx, s mir.Stmt, funcs map[string]llvmFuncSig) (llvmValueStmtEmitPlan, bool) {
	info, ok := mir.ValueStmtInfo(s)
	if !ok {
		return llvmValueStmtEmitPlan{}, false
	}
	return llvmValueStmtEmitPlan{
		stmt:  s,
		ctx:   ctx,
		funcs: funcs,
		name:  info.Name,
	}, true
}

func (p llvmValueStmtEmitPlan) emitExpr() (string, string, ast.Type, bool) {
	expr, ok := mir.ValueStmtExpr(p.stmt)
	if !ok {
		return "", "", ast.TypeInvalid, false
	}
	return emitExprMIRLLVM(p.ctx, expr, p.funcs)
}

func (p llvmValueStmtEmitPlan) emitInto(b *strings.Builder) bool {
	code, value, outType, ok := p.emitExpr()
	if !ok {
		return false
	}
	b.WriteString(code)
	if p.name == "_" {
		return true
	}
	slot, ok := p.ctx.vars[p.name]
	if !ok {
		return false
	}
	return emitStoreLLVMValue(b, p.ctx, slot.ptr, slot.typ, value, outType)
}

func buildLLVMCFGInstrEmitPlan(ctx *funcCtx, s mir.Stmt, funcs map[string]llvmFuncSig) (llvmCFGInstrEmitPlan, bool) {
	if _, ok := mir.LinearStmtInfo(s); !ok {
		return llvmCFGInstrEmitPlan{}, false
	}
	return llvmCFGInstrEmitPlan{stmt: s, ctx: ctx, funcs: funcs}, true
}

func (p llvmCFGInstrEmitPlan) emitInto(b *strings.Builder) bool {
	info, ok := mir.LinearStmtInfo(p.stmt)
	if !ok {
		return false
	}
	if info.Target != nil {
		ptrCode, ptr, targetType, ok := emitAssignTargetPtrMIRLLVM(p.ctx, info.Target)
		if !ok {
			return false
		}
		code, value, t, ok := emitExprMIRLLVM(p.ctx, info.Value, p.funcs)
		if !ok {
			return false
		}
		b.WriteString(ptrCode)
		b.WriteString(code)
		return emitStoreLLVMValue(b, p.ctx, ptr, targetType, value, t)
	}
	code, _, _, ok := emitExprMIRLLVM(p.ctx, info.Value, p.funcs)
	if !ok {
		if !mir.StmtMayHaveSideEffects(p.stmt) {
			return true
		}
		return false
	}
	b.WriteString(code)
	return true
}

func emitTerminatorMIRLLVM(b *strings.Builder, ctx *funcCtx, term mir.Terminator, funcs map[string]llvmFuncSig) bool {
	plan, ok := buildLLVMTerminatorEmitPlan(ctx, term, funcs)
	if !ok {
		return false
	}
	return plan.emitInto(b)
}

func buildLLVMTerminatorEmitPlan(ctx *funcCtx, term mir.Terminator, funcs map[string]llvmFuncSig) (llvmTerminatorEmitPlan, bool) {
	info, ok := mir.TerminatorInfo(term)
	if !ok {
		return llvmTerminatorEmitPlan{}, false
	}
	return llvmTerminatorEmitPlan{
		term:          term,
		ctx:           ctx,
		funcs:         funcs,
		kind:          info.Kind,
		value:         info.Value,
		cond:          info.Cond,
		subject:       info.Subject,
		target:        info.Target,
		thenTarget:    info.Then,
		elseTarget:    info.Else,
		defaultTarget: info.Default,
		matchArms:     info.Arms,
	}, true
}

func (p llvmTerminatorEmitPlan) emitReturn(b *strings.Builder) bool {
	if p.ctx.isMain {
		if p.value != nil {
			return false
		}
		b.WriteString("  ret i32 0\n")
		p.ctx.terminated = true
		return true
	}
	if p.value == nil {
		b.WriteString("  ret void\n")
		p.ctx.terminated = true
		return true
	}
	code, emittedValue, retType, ok := emitExprMIRLLVM(p.ctx, p.value, p.funcs)
	if !ok {
		return false
	}
	b.WriteString(code)
	return p.ctx.emitTypedReturn(b, emittedValue, retType)
}

func (p llvmTerminatorEmitPlan) emitJump(b *strings.Builder) bool {
	b.WriteString(fmt.Sprintf("  br label %%%s\n", cfgLabelLLVM(p.target)))
	return true
}

func (p llvmTerminatorEmitPlan) emitCond(b *strings.Builder) bool {
	code, condVal, condType, ok := emitExprMIRLLVM(p.ctx, p.cond, p.funcs)
	if !ok || condType != ast.TypeBool {
		return false
	}
	condCode, cond := boolToI1(p.ctx, condVal)
	b.WriteString(code)
	b.WriteString(condCode)
	b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", cond, cfgLabelLLVM(p.thenTarget), cfgLabelLLVM(p.elseTarget)))
	return true
}

func (p llvmTerminatorEmitPlan) emitMatch(b *strings.Builder) bool {
	code, subjVal, subjType, ok := emitExprMIRLLVM(p.ctx, p.subject, p.funcs)
	if !ok {
		return false
	}
	if _, ok := p.ctx.enums.enumTypes[string(subjType)]; !ok {
		return false
	}
	defaultLabel := p.defaultTarget
	unreachableLabel := ""
	if defaultLabel == "" {
		unreachableLabel = p.ctx.ir.nextLabel("match_unreachable")
		defaultLabel = unreachableLabel
	}
	grouped := mir.GroupMatchTerminatorArms(p.matchArms)
	b.WriteString(code)
	variants := make([]string, 0, len(grouped))
	for _, g := range grouped {
		variants = append(variants, g.Variant)
	}
	caseLabels, ok := p.ctx.emitEnumSwitchDispatch(b, subjVal, subjType, cfgLabelLLVM(defaultLabel), variants, "match_term_case")
	if !ok {
		return false
	}
	for i, g := range grouped {
		b.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
		if !emitGuardedMatchTerminatorLLVM(b, p.ctx, g.Arms, defaultLabel, p.funcs) {
			return false
		}
	}
	if unreachableLabel != "" {
		b.WriteString(fmt.Sprintf("%s:\n", cfgLabelLLVM(unreachableLabel)))
		b.WriteString("  unreachable\n")
	}
	return true
}

func (p llvmTerminatorEmitPlan) emitInto(b *strings.Builder) bool {
	switch p.kind {
	case "return":
		return p.emitReturn(b)
	case "jump":
		return p.emitJump(b)
	case "cond":
		return p.emitCond(b)
	case "match":
		return p.emitMatch(b)
	default:
		return false
	}
}

func emitExprMIRLLVM(ctx *funcCtx, e mir.Expr, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	return llvmExprEmitPlan{ctx: ctx, expr: e, funcs: funcs}.emit()
}

func emitAtomicExprMIRLLVM(ctx *funcCtx, e mir.Expr) (string, string, ast.Type, bool) {
	return llvmAtomicExprEmitPlan{ctx: ctx, expr: e}.emit()
}

func (p llvmAtomicExprEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	switch ex := p.expr.(type) {
	case *mir.IntExpr:
		return "", strconv.FormatInt(ex.Value, 10), ast.TypeInt, true
	case *mir.FloatExpr:
		return "", strconv.FormatFloat(ex.Value, 'f', -1, 64), ast.TypeFloat, true
	case *mir.BoolExpr:
		if ex.Value {
			return "", "1", ast.TypeBool, true
		}
		return "", "0", ast.TypeBool, true
	case *mir.StringExpr:
		code, ptr, ok := stringPtr(ctx, ex.Value)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		return code, ptr, ast.TypeString, true
	case *mir.NilExpr:
		return "", "", ast.TypeInvalid, false
	case *mir.IdentExpr:
		name := ex.Name
		if slot, ok := ctx.bindingSlot(name); ok {
			return ctx.emitLoadLLVMValue(slot.ptr, slot.typ)
		}
		if idx, ok := ctx.enums.variantIndex[name]; ok {
			enumName := ctx.enums.variantType[name]
			if enumName == "" {
				return "", "", ast.TypeInvalid, false
			}
			return "", strconv.Itoa(idx), ast.Type(enumName), true
		}
		return "", "", ast.TypeInvalid, false
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func (p llvmExprEmitPlan) emit() (string, string, ast.Type, bool) {
	if code, value, typ, ok := emitAtomicExprMIRLLVM(p.ctx, p.expr); ok {
		return code, value, typ, true
	}
	switch ex := p.expr.(type) {
	case *mir.UnaryExpr:
		return emitUnaryValueStmtMIRLLVM(p.ctx, &mir.UnaryOpStmt{NodeInfo: ex.NodeInfo, Op: ex.Op, Right: ex.Right}, p.funcs)
	case *mir.BinaryExpr:
		return emitBinaryValueStmtMIRLLVM(p.ctx, &mir.BinaryOpStmt{NodeInfo: ex.NodeInfo, Op: ex.Op, Left: ex.Left, Right: ex.Right}, p.funcs)
	case *mir.CallExpr:
		return emitCallValueStmtMIRLLVM(p.ctx, &mir.CallStmt{NodeInfo: ex.NodeInfo, Func: ex.Func, Args: ex.Args}, p.funcs)
	case *mir.FieldAccessExpr:
		return emitFieldAccessValueStmtMIRLLVM(p.ctx, &mir.FieldAccessStmt{NodeInfo: ex.NodeInfo, Object: ex.Object, Field: ex.Field}, p.funcs)
	case *mir.StructLitExpr:
		return emitStructLitValueStmtMIRLLVM(p.ctx, &mir.StructLitStmt{NodeInfo: ex.NodeInfo, TypeName: ex.TypeName, Fields: ex.Fields}, p.funcs)
	case *mir.MatchExpr:
		return emitMatchValueStmtMIRLLVM(p.ctx, &mir.MatchValueStmt{NodeInfo: ex.NodeInfo, Subject: ex.Subject, Arms: ex.Arms, Type: ex.Type}, p.funcs)
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func emitAssignTargetPtrMIRLLVM(ctx *funcCtx, target mir.Expr) (string, string, ast.Type, bool) {
	return llvmAssignTargetEmitPlan{ctx: ctx, target: target}.emit()
}

func (p llvmAssignTargetEmitPlan) emit() (string, string, ast.Type, bool) {
	ctx := p.ctx
	switch t := p.target.(type) {
	case *mir.IdentExpr:
		name := t.Name
		if slot, ok := ctx.bindingSlot(name); ok {
			return "", slot.ptr, slot.typ, true
		}
		return "", "", ast.TypeInvalid, false
	case *mir.FieldAccessExpr:
		fields := []string{}
		cur := p.target
		for {
			fa, ok := cur.(*mir.FieldAccessExpr)
			if !ok {
				break
			}
			fields = append(fields, fa.Field)
			cur = fa.Object
		}
		for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
			fields[i], fields[j] = fields[j], fields[i]
		}
		baseIdent, ok := cur.(*mir.IdentExpr)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		baseSlot, ok := ctx.bindingSlot(baseIdent.Name)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		return ctx.emitFieldPathPtr(baseSlot.ptr, baseSlot.typ, fields)
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func emitGuardedMatchValueExprMIR(b *strings.Builder, ctx *funcCtx, arms []mir.MatchExprArm, funcs map[string]llvmFuncSig, mergeLabel string, resolved ast.Type, caseLabel string) ([]string, bool) {
	return llvmGuardedMatchValuePlan{
		b:          b,
		ctx:        ctx,
		arms:       arms,
		funcs:      funcs,
		mergeLabel: mergeLabel,
		resolved:   resolved,
		caseLabel:  caseLabel,
	}.emit()
}

func (p llvmGuardedMatchValuePlan) emit() ([]string, bool) {
	phiVals := []string{}
	unguarded := -1
	for i, arm := range p.arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	hasGuard := false
	for _, arm := range p.arms {
		if arm.Guard != nil {
			hasGuard = true
			break
		}
	}
	if !hasGuard && unguarded >= 0 {
		valCode, val, valType, ok := emitExprMIRLLVM(p.ctx, p.arms[unguarded].Value, p.funcs)
		if !ok || valType != p.resolved {
			return nil, false
		}
		p.b.WriteString(valCode)
		p.b.WriteString(fmt.Sprintf("  br label %%%s\n", p.mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, p.caseLabel))
		return phiVals, true
	}
	unguardedLabel := ""
	unguardedLabelEmitted := false
	if unguarded >= 0 {
		unguardedLabel = p.ctx.ir.nextLabel("match_unguarded")
	}
	guarded := []int{}
	for i, arm := range p.arms {
		if arm.Guard != nil {
			guarded = append(guarded, i)
		}
	}
	nextLabel := ""
	for gi, idx := range guarded {
		arm := p.arms[idx]
		condCode, condVal, condType, ok := emitExprMIRLLVM(p.ctx, arm.Guard, p.funcs)
		if !ok || condType != ast.TypeBool {
			return nil, false
		}
		condI1Code, condI1 := boolToI1(p.ctx, condVal)
		thenLabel := p.ctx.ir.nextLabel("match_guard_then")
		if gi == len(guarded)-1 {
			if unguarded >= 0 {
				nextLabel = unguardedLabel
			} else {
				nextLabel = p.mergeLabel
			}
		} else {
			nextLabel = p.ctx.ir.nextLabel("match_guard_next")
		}
		p.b.WriteString(condCode)
		p.b.WriteString(condI1Code)
		p.b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", condI1, thenLabel, nextLabel))
		p.b.WriteString(fmt.Sprintf("%s:\n", thenLabel))
		valCode, val, valType, ok := emitExprMIRLLVM(p.ctx, arm.Value, p.funcs)
		if !ok || valType != p.resolved {
			return nil, false
		}
		p.b.WriteString(valCode)
		p.b.WriteString(fmt.Sprintf("  br label %%%s\n", p.mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, thenLabel))
		p.b.WriteString(fmt.Sprintf("%s:\n", nextLabel))
		if nextLabel == unguardedLabel {
			unguardedLabelEmitted = true
		}
	}
	if unguarded >= 0 {
		if unguardedLabel != "" && !unguardedLabelEmitted {
			p.b.WriteString(fmt.Sprintf("%s:\n", unguardedLabel))
		}
		valCode, val, valType, ok := emitExprMIRLLVM(p.ctx, p.arms[unguarded].Value, p.funcs)
		if !ok || valType != p.resolved {
			return nil, false
		}
		p.b.WriteString(valCode)
		p.b.WriteString(fmt.Sprintf("  br label %%%s\n", p.mergeLabel))
		phiVals = append(phiVals, fmt.Sprintf("[ %s, %%%s ]", val, unguardedLabel))
	}
	return phiVals, true
}

func emitGuardedMatchTerminatorLLVM(b *strings.Builder, ctx *funcCtx, arms []mir.MatchTerminatorArm, defaultLabel string, funcs map[string]llvmFuncSig) bool {
	return llvmGuardedMatchTerminatorPlan{
		b:            b,
		ctx:          ctx,
		arms:         arms,
		defaultLabel: defaultLabel,
		funcs:        funcs,
	}.emit()
}

func (p llvmGuardedMatchTerminatorPlan) emit() bool {
	unguarded := -1
	for i, arm := range p.arms {
		if arm.Guard == nil {
			unguarded = i
			break
		}
	}
	hasGuard := false
	for _, arm := range p.arms {
		if arm.Guard != nil {
			hasGuard = true
			break
		}
	}
	if !hasGuard && unguarded >= 0 {
		p.b.WriteString(fmt.Sprintf("  br label %%%s\n", cfgLabelLLVM(p.arms[unguarded].Target)))
		return true
	}
	guarded := []int{}
	for i, arm := range p.arms {
		if arm.Guard != nil {
			guarded = append(guarded, i)
		}
	}
	for gi, idx := range guarded {
		arm := p.arms[idx]
		condCode, condVal, condType, ok := emitExprMIRLLVM(p.ctx, arm.Guard, p.funcs)
		if !ok || condType != ast.TypeBool {
			return false
		}
		condI1Code, condI1 := boolToI1(p.ctx, condVal)
		nextLabel := p.defaultLabel
		if gi < len(guarded)-1 {
			nextLabel = p.ctx.ir.nextLabel("match_guard_next")
		} else if unguarded >= 0 {
			nextLabel = p.arms[unguarded].Target
		}
		p.b.WriteString(condCode)
		p.b.WriteString(condI1Code)
		p.b.WriteString(fmt.Sprintf("  br i1 %s, label %%%s, label %%%s\n", condI1, cfgLabelLLVM(arm.Target), cfgLabelLLVM(nextLabel)))
		if gi < len(guarded)-1 {
			p.b.WriteString(fmt.Sprintf("%s:\n", cfgLabelLLVM(nextLabel)))
		}
	}
	return true
}

func (e llvmABIEnv) runtimeType(t ast.Type) (string, bool) {
	return intrinsics.MapLLVMRuntimeType(
		t,
		func(name string) bool { return e.enums.enumTypes[name] },
		func(name string) bool {
			_, ok := e.structs.byName[name]
			return ok
		},
		func(name string) bool { return e.ifaces.names[name] },
	)
}

func (e llvmABIEnv) defaultValue(t ast.Type) string {
	return intrinsics.DefaultLLVMValue(
		t,
		func(name string) bool { return e.enums.enumTypes[name] },
		func(name string) bool {
			_, ok := e.structs.byName[name]
			return ok
		},
		func(name string) bool { return e.ifaces.names[name] },
	)
}

func (e llvmABIEnv) valueABI(t ast.Type) (intrinsics.LLVMValueABI, bool) {
	return intrinsics.BuildLLVMValueABI(
		t,
		func(t ast.Type) (string, bool) { return e.runtimeType(t) },
	)
}

func (e llvmABIEnv) valueABIOrError(t ast.Type, format string, args ...any) (intrinsics.LLVMValueABI, error) {
	abi, ok := e.valueABI(t)
	if ok {
		return abi, nil
	}
	return intrinsics.LLVMValueABI{}, fmt.Errorf(format, args...)
}

func (e llvmABIEnv) storageABI(t ast.Type) (intrinsics.LLVMStorageABI, bool) {
	return intrinsics.BuildLLVMStorageABI(
		t,
		func(t ast.Type) (string, bool) { return e.runtimeType(t) },
		func(t ast.Type) string { return e.defaultValue(t) },
	)
}

func (e llvmABIEnv) storageABIOrError(t ast.Type, format string, args ...any) (intrinsics.LLVMStorageABI, error) {
	abi, ok := e.storageABI(t)
	if ok {
		return abi, nil
	}
	return intrinsics.LLVMStorageABI{}, fmt.Errorf(format, args...)
}

func (e llvmABIEnv) sortedNamedStorageABIs(types map[string]ast.Type) ([]intrinsics.LLVMNamedStorageABI, bool) {
	return intrinsics.BuildLLVMSortedNamedStorageABIs(
		types,
		func(t ast.Type) (string, bool) { return e.runtimeType(t) },
		func(t ast.Type) string { return e.defaultValue(t) },
	)
}

func (e llvmABIEnv) sortedNamedStorageABIsOrError(fnName string, types map[string]ast.Type) ([]intrinsics.LLVMNamedStorageABI, error) {
	abis, ok := e.sortedNamedStorageABIs(types)
	if ok {
		return abis, nil
	}
	for name, typ := range types {
		if _, ok := e.storageABI(typ); !ok {
			return nil, fmt.Errorf("llvm backend: unsupported local type '%s' for '%s.%s'", typ, fnName, name)
		}
	}
	return nil, fmt.Errorf("llvm backend: unsupported local storage abi for '%s'", fnName)
}

func (e llvmABIEnv) functionABI(ret ast.Type, params []intrinsics.LLVMNamedType) (intrinsics.LLVMFunctionABI, bool) {
	return intrinsics.BuildLLVMFunctionABI(
		ret,
		params,
		func(t ast.Type) (string, bool) { return e.runtimeType(t) },
	)
}

func (e llvmABIEnv) callABI(callee string, ret ast.Type) (intrinsics.LLVMCallABI, bool) {
	return intrinsics.ClassifyLLVMCallABI(
		callee,
		ret,
		func(t ast.Type) (string, bool) { return e.runtimeType(t) },
		func(name string) bool {
			_, ok := e.structs.byName[name]
			return ok
		},
	)
}

func (e llvmABIEnv) callABIOrError(callee string, ret ast.Type) (intrinsics.LLVMCallABI, error) {
	abi, ok := e.callABI(callee, ret)
	if ok {
		return abi, nil
	}
	return intrinsics.LLVMCallABI{}, fmt.Errorf("llvm backend: unsupported call abi for '%s' (%s)", callee, ret)
}

func (e llvmABIEnv) anyBoxClass(t ast.Type) (intrinsics.LLVMAnyBoxClass, bool) {
	return intrinsics.ClassifyLLVMAnyBoxing(
		t,
		func(name string) bool { return e.enums.enumTypes[name] },
		func(name string) bool {
			_, ok := e.structs.byName[name]
			return ok
		},
		func(name string) bool { return e.ifaces.names[name] },
	)
}

func (e llvmABIEnv) anyHeapCopyType(t ast.Type) (string, bool) {
	return intrinsics.MapLLVMAnyHeapCopyType(
		t,
		func(name string) bool { return e.enums.enumTypes[name] },
		func(name string) bool {
			_, ok := e.structs.byName[name]
			return ok
		},
		func(name string) bool { return e.ifaces.names[name] },
	)
}

func (e llvmABIEnv) functionABIOrError(fnName string, ret ast.Type, params []intrinsics.LLVMNamedType) (intrinsics.LLVMFunctionABI, error) {
	abi, ok := e.functionABI(ret, params)
	if ok {
		return abi, nil
	}
	if _, ok := e.runtimeType(ret); !ok {
		return intrinsics.LLVMFunctionABI{}, fmt.Errorf("llvm backend: unsupported return type '%s' for '%s'", ret, fnName)
	}
	for _, p := range params {
		pt := normalizeLLVMType(p.Type)
		if _, ok := e.runtimeType(pt); !ok {
			return intrinsics.LLVMFunctionABI{}, fmt.Errorf("llvm backend: unsupported param type '%s' for '%s.%s'", pt, fnName, p.Name)
		}
	}
	return intrinsics.LLVMFunctionABI{}, fmt.Errorf("llvm backend: unsupported function abi for '%s'", fnName)
}

func findStructNameBySuffix(pool structPool, sourceName string) (string, bool) {
	if _, ok := pool.byName[sourceName]; ok {
		return sourceName, true
	}
	suffix := "__" + sourceName
	for name := range pool.byName {
		if strings.HasSuffix(name, suffix) {
			return name, true
		}
	}
	return "", false
}

func boolToI1(ctx *funcCtx, val string) (string, string) {
	return llvmBoolConvertPlan{ctx: ctx, value: val, target: "i1"}.emit()
}

func boolToI8(ctx *funcCtx, val string) (string, string) {
	return llvmBoolConvertPlan{ctx: ctx, value: val, target: "i8"}.emit()
}

func stringPtr(ctx *funcCtx, value string) (string, string, bool) {
	return llvmStringPtrPlan{ctx: ctx, value: value}.emit()
}

func nonNullStringPtr(ctx *funcCtx, value string) (string, string, bool) {
	emptyCode, emptyPtr, ok := stringPtr(ctx, "")
	if !ok {
		return "", value, true
	}
	isNull := ctx.ir.nextTmp()
	tmp := ctx.ir.nextTmp()
	code := emptyCode
	code += fmt.Sprintf("  %s = icmp eq ptr %s, null\n", isNull, value)
	code += fmt.Sprintf("  %s = select i1 %s, ptr %s, ptr %s\n", tmp, isNull, emptyPtr, value)
	return code, tmp, true
}

func normalizeExternalStringValue(ctx *funcCtx, value string, t ast.Type) (string, string, bool, bool) {
	normalized := normalizeLLVMType(t)
	if normalized == ast.TypeString {
		code, out, ok := nonNullStringPtr(ctx, value)
		return code, out, true, ok
	}

	info, ok := ctx.structs.byName[string(normalized)]
	if !ok {
		return "", value, false, true
	}
	llvmType, ok := ctx.abi().runtimeType(normalized)
	if !ok {
		return "", "", false, false
	}

	current := value
	code := ""
	changed := false
	for idx, field := range info.Fields {
		fieldLLVMType, ok := ctx.abi().runtimeType(field.Type)
		if !ok {
			return "", "", false, false
		}
		fieldTmp := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = extractvalue %s %s, %d\n", fieldTmp, llvmType, current, idx)
		fieldCode, fieldValue, fieldChanged, ok := normalizeExternalStringValue(ctx, fieldTmp, field.Type)
		if !ok {
			return "", "", false, false
		}
		if !fieldChanged {
			continue
		}
		changed = true
		code += fieldCode
		next := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = insertvalue %s %s, %s %s, %d\n", next, llvmType, current, fieldLLVMType, fieldValue, idx)
		current = next
	}
	return code, current, changed, true
}

func (p llvmBoolConvertPlan) emit() (string, string) {
	tmp := p.ctx.ir.nextTmp()
	if p.target == "i1" {
		return fmt.Sprintf("  %s = icmp ne i8 %s, 0\n", tmp, p.value), tmp
	}
	return fmt.Sprintf("  %s = zext i1 %s to i8\n", tmp, p.value), tmp
}

func (p llvmStringPtrPlan) emit() (string, string, bool) {
	name, ok := p.ctx.strs.names[p.value]
	if !ok {
		return "", "", false
	}
	length := len([]byte(p.value)) + 1
	tmp := p.ctx.ir.nextTmp()
	code := fmt.Sprintf("  %s = %s\n", tmp, stringGEP(name, length))
	return code, tmp, true
}

func boxToAny(ctx *funcCtx, value string, t ast.Type) (string, string, bool) {
	class, ok := ctx.abi().anyBoxClass(t)
	if !ok {
		return "", "", false
	}
	var code strings.Builder
	payload := value

	switch class.Payload {
	case intrinsics.LLVMAnyPayloadIntToPtr:
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp, value))
		payload = tmp
	case intrinsics.LLVMAnyPayloadBoolToPtr:
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = zext i8 %s to i64\n", tmp, value))
		tmp2 := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp2, tmp))
		payload = tmp2
	case intrinsics.LLVMAnyPayloadFloatBits:
		tmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = bitcast double %s to i64\n", tmp, value))
		tmp2 := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = inttoptr i64 %s to ptr\n", tmp2, tmp))
		payload = tmp2
	case intrinsics.LLVMAnyPayloadPtr:
	case intrinsics.LLVMAnyPayloadHeapCopy:
		llvmType, ok := ctx.abi().anyHeapCopyType(t)
		if !ok {
			return "", "", false
		}
		sizePtr := ctx.ir.nextTmp()
		sizeTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = getelementptr %s, ptr null, i32 1\n", sizePtr, llvmType))
		code.WriteString(fmt.Sprintf("  %s = ptrtoint ptr %s to i64\n", sizeTmp, sizePtr))
		memTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = call ptr @malloc(i64 %s)\n", memTmp, sizeTmp))
		castTmp := ctx.ir.nextTmp()
		code.WriteString(fmt.Sprintf("  %s = bitcast ptr %s to %s*\n", castTmp, memTmp, llvmType))
		code.WriteString(fmt.Sprintf("  store %s %s, ptr %s\n", llvmType, value, castTmp))
		payload = memTmp
	default:
		return "", "", false
	}

	tmp0 := ctx.ir.nextTmp()
	tmp1 := ctx.ir.nextTmp()
	code.WriteString(fmt.Sprintf("  %s = insertvalue %%Any undef, i64 %d, 0\n", tmp0, class.Tag))
	code.WriteString(fmt.Sprintf("  %s = insertvalue %%Any %s, ptr %s, 1\n", tmp1, tmp0, payload))
	return code.String(), tmp1, true
}

func emitLoweredBuiltinExprMIR(ctx *funcCtx, funcName string, args []mir.Expr, funcs map[string]llvmFuncSig) (string, string, ast.Type, bool) {
	spec, ok := intrinsics.LookupLoweredBuiltin(funcName)
	if !ok || len(args) != spec.Arity {
		return "", "", ast.TypeInvalid, false
	}
	switch spec.Category {
	case intrinsics.LoweredBuiltinLen:
		code, val, t, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = call i64 @%s(ptr %s)\n", tmp, spec.LLVMTarget, val)
		return code, tmp, spec.ReturnType, true
	case intrinsics.LoweredBuiltinStringUnary:
		code, val, t, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code += fmt.Sprintf("  %s = call ptr @%s(ptr %s)\n", tmp, spec.LLVMTarget, val)
		return code, tmp, spec.ReturnType, true
	case intrinsics.LoweredBuiltinStringBinaryPredicate:
		c1, v1, t1, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t1 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		c2, v2, t2, ok := emitExprMIRLLVM(ctx, args[1], funcs)
		if !ok || t2 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code := c1 + c2 + fmt.Sprintf("  %s = call i8 @%s(ptr %s, ptr %s)\n", tmp, spec.LLVMTarget, v1, v2)
		return code, tmp, spec.ReturnType, true
	case intrinsics.LoweredBuiltinStringTernary:
		c1, v1, t1, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t1 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		c2, v2, t2, ok := emitExprMIRLLVM(ctx, args[1], funcs)
		if !ok || t2 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		c3, v3, t3, ok := emitExprMIRLLVM(ctx, args[2], funcs)
		if !ok || t3 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code := c1 + c2 + c3 + fmt.Sprintf("  %s = call ptr @%s(ptr %s, ptr %s, ptr %s)\n", tmp, spec.LLVMTarget, v1, v2, v3)
		return code, tmp, spec.ReturnType, true
	case intrinsics.LoweredBuiltinStringRepeat:
		c1, v1, t1, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t1 != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		c2, v2, t2, ok := emitExprMIRLLVM(ctx, args[1], funcs)
		if !ok || t2 != ast.TypeInt {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		code := c1 + c2 + fmt.Sprintf("  %s = call ptr @%s(ptr %s, i64 %s)\n", tmp, spec.LLVMTarget, v1, v2)
		return code, tmp, spec.ReturnType, true
	case intrinsics.LoweredBuiltinStringify:
		c1, v1, t1, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok {
			return "", "", ast.TypeInvalid, false
		}
		if t1 == ast.TypeString {
			return c1, v1, ast.TypeString, true
		}
		if t1 == ast.TypeBool {
			trueCode, truePtr, ok := stringPtr(ctx, "true")
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			falseCode, falsePtr, ok := stringPtr(ctx, "false")
			if !ok {
				return "", "", ast.TypeInvalid, false
			}
			tmp := ctx.ir.nextTmp()
			condCode, cond := boolToI1(ctx, v1)
			code := c1 + trueCode + falseCode + condCode
			code += fmt.Sprintf("  %s = select i1 %s, ptr %s, ptr %s\n", tmp, cond, truePtr, falsePtr)
			return code, tmp, ast.TypeString, true
		}
		if t1 == ast.TypeInt || ctx.enums.enumTypes[string(t1)] {
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @%s(i64 %s)\n", tmp, intrinsics.LLVMRuntimeIntToStrFunc, v1)
			return code, tmp, ast.TypeString, true
		}
		if t1 == ast.TypeFloat {
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @%s(double %s)\n", tmp, intrinsics.LLVMRuntimeFloatToStrFunc, v1)
			return code, tmp, ast.TypeString, true
		}
		if t1 == ast.TypeAny {
			tmp := ctx.ir.nextTmp()
			code := c1 + fmt.Sprintf("  %s = call ptr @%s(%%Any %s)\n", tmp, intrinsics.LLVMRuntimeAnyToStrFunc, v1)
			return code, tmp, ast.TypeString, true
		}
		return "", "", ast.TypeInvalid, false
	case intrinsics.LoweredBuiltinParseString:
		code, val, t, ok := emitExprMIRLLVM(ctx, args[0], funcs)
		if !ok || t != ast.TypeString {
			return "", "", ast.TypeInvalid, false
		}
		tmp := ctx.ir.nextTmp()
		typeName := intrinsics.LLVMResultStructName(spec.ParseKind, "Error")
		code += fmt.Sprintf("  %s = call %%%s @%s(ptr %s)\n", tmp, typeName, spec.LLVMTarget, val)
		return code, tmp, ast.Type(typeName), true
	default:
		return "", "", ast.TypeInvalid, false
	}
}

func mapIntArithOp(op string) (string, bool) {
	switch op {
	case "+":
		return "add", true
	case "-":
		return "sub", true
	case "*":
		return "mul", true
	case "/":
		return "sdiv", true
	case "%":
		return "srem", true
	default:
		return "", false
	}
}

func mapFloatArithOp(op string) (string, bool) {
	switch op {
	case "+":
		return "fadd", true
	case "-":
		return "fsub", true
	case "*":
		return "fmul", true
	case "/":
		return "fdiv", true
	default:
		return "", false
	}
}

func mapIntCmpOp(op string) (string, bool) {
	switch op {
	case "==":
		return "eq", true
	case "!=":
		return "ne", true
	case "<":
		return "slt", true
	case "<=":
		return "sle", true
	case ">":
		return "sgt", true
	case ">=":
		return "sge", true
	default:
		return "", false
	}
}

func mapFloatCmpOp(op string) (string, bool) {
	switch op {
	case "==":
		return "oeq", true
	case "!=":
		return "one", true
	case "<":
		return "olt", true
	case "<=":
		return "ole", true
	case ">":
		return "ogt", true
	case ">=":
		return "oge", true
	default:
		return "", false
	}
}

func normalizeLLVMType(t ast.Type) ast.Type {
	return intrinsics.NormalizeLLVMType(t)
}
