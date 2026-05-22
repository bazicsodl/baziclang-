package backendmeta

import (
	"baziclang/internal/ast"
	"baziclang/internal/intrinsics"
	"baziclang/internal/mir"
	baztypes "baziclang/internal/types"
)

type FuncSig struct {
	Params []ast.Type
	Ret    ast.Type
}

type GlobalDecl struct {
	Name string
	Type ast.Type
	Init mir.Expr
}

type EnumDecl struct {
	Name     string
	Variants []string
}

type StructField struct {
	Name string
	Type ast.Type
}

type StructDecl struct {
	Name   string
	Fields []StructField
}

type InterfaceDecl struct {
	Name string
}

type LLVMRuntimeSurfaceMeta struct {
	TypeAliases          map[string]ast.Type
	FuncSigs             map[string]FuncSig
	PreludeDecls         []string
	HasStringGlobals     bool
	HasParseIntRuntime   bool
	HasParseFloatRuntime bool
	PreludeSections      []intrinsics.LLVMRuntimePreludeSection
	BuiltinSections      []intrinsics.LLVMBuiltinRuntimeSection
}

type RuntimeShapeMeta struct {
	LLVMRuntimeSurface LLVMRuntimeSurfaceMeta
	RouteStrings       []string
}

type ProgramShapeMeta struct {
	Runtime         ProgramRuntimeMeta
	RuntimeShape    RuntimeShapeMeta
	OrderedFuncs    []*mir.FuncDecl
	StructNodes     []*mir.StructDecl
	InterfaceNodes  []*mir.InterfaceDecl
	EnumNodes       []*mir.EnumDecl
	GlobalNodes     []*mir.GlobalLetDecl
	ProgramFuncSigs map[string]FuncSig
	Globals         []GlobalDecl
	Enums           []EnumDecl
	Structs         []StructDecl
	Interfaces      []InterfaceDecl
}

func CollectLLVMRuntimeSurfaceMeta(runtime ProgramRuntimeMeta) LLVMRuntimeSurfaceMeta {
	hasStringGlobals := HasRuntimeFeature(runtime.Features, RuntimeFeatureLLVMStringGlobals)
	hasParseIntRuntime := HasRuntimeFeature(runtime.Features, RuntimeFeatureLLVMParseInt)
	hasParseFloatRuntime := HasRuntimeFeature(runtime.Features, RuntimeFeatureLLVMParseFloat)
	preludeSections := collectLLVMRuntimePreludeSections(runtime.Features)
	builtinSections := collectLLVMBuiltinRuntimeSections(runtime.Features)
	surface := intrinsics.BuildLLVMRuntimeSurface(
		runtime.Types.HTTPResponseType,
		runtime.Types.ServerRequestType,
		runtime.Types.ServerResponseType,
		hasStringGlobals,
		hasParseIntRuntime,
		hasParseFloatRuntime,
	)
	return LLVMRuntimeSurfaceMeta{
		TypeAliases:          surface.TypeAliases,
		FuncSigs:             collectRuntimeFuncSigs(surface.Functions),
		PreludeDecls:         append([]string(nil), surface.PreludeDecls...),
		HasStringGlobals:     surface.HasStringGlobals,
		HasParseIntRuntime:   surface.HasParseIntRuntime,
		HasParseFloatRuntime: surface.HasParseFloatRuntime,
		PreludeSections:      append([]intrinsics.LLVMRuntimePreludeSection(nil), preludeSections...),
		BuiltinSections:      append([]intrinsics.LLVMBuiltinRuntimeSection(nil), builtinSections...),
	}
}

func CollectProgramGlobals(p *mir.Program) []GlobalDecl {
	if p == nil {
		return nil
	}
	out := []GlobalDecl{}
	for _, d := range p.Decls {
		g, ok := d.(*mir.GlobalLetDecl)
		if !ok {
			continue
		}
		out = append(out, GlobalDecl{
			Name: g.Name,
			Type: baztypes.ToAST(g.Type),
			Init: g.Init,
		})
	}
	return out
}

func CollectProgramStructNodes(p *mir.Program) []*mir.StructDecl {
	if p == nil {
		return nil
	}
	out := []*mir.StructDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.StructDecl)
		if !ok {
			continue
		}
		out = append(out, decl)
	}
	return out
}

func CollectProgramInterfaceNodes(p *mir.Program) []*mir.InterfaceDecl {
	if p == nil {
		return nil
	}
	out := []*mir.InterfaceDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.InterfaceDecl)
		if !ok {
			continue
		}
		out = append(out, decl)
	}
	return out
}

func CollectProgramEnumNodes(p *mir.Program) []*mir.EnumDecl {
	if p == nil {
		return nil
	}
	out := []*mir.EnumDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.EnumDecl)
		if !ok {
			continue
		}
		out = append(out, decl)
	}
	return out
}

func CollectProgramGlobalNodes(p *mir.Program) []*mir.GlobalLetDecl {
	if p == nil {
		return nil
	}
	out := []*mir.GlobalLetDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.GlobalLetDecl)
		if !ok {
			continue
		}
		out = append(out, decl)
	}
	return out
}

func CollectProgramEnums(p *mir.Program) []EnumDecl {
	if p == nil {
		return nil
	}
	out := []EnumDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.EnumDecl)
		if !ok {
			continue
		}
		variants := make([]string, 0, len(decl.Variants))
		variants = append(variants, decl.Variants...)
		out = append(out, EnumDecl{Name: decl.Name, Variants: variants})
	}
	return out
}

func CollectProgramStructs(p *mir.Program) []StructDecl {
	if p == nil {
		return nil
	}
	out := []StructDecl{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.StructDecl)
		if !ok || len(decl.TypeParams) > 0 {
			continue
		}
		fields := make([]StructField, 0, len(decl.Fields))
		skip := false
		for _, f := range decl.Fields {
			fieldType := baztypes.ToAST(f.Type)
			if fieldType == ast.TypeInvalid {
				skip = true
				break
			}
			fields = append(fields, StructField{Name: f.Name, Type: fieldType})
		}
		if skip {
			continue
		}
		out = append(out, StructDecl{Name: decl.Name, Fields: fields})
	}
	return out
}

func CollectProgramInterfaces(p *mir.Program) []InterfaceDecl {
	if p == nil {
		return nil
	}
	out := []InterfaceDecl{}
	seen := map[string]bool{}
	for _, d := range p.Decls {
		decl, ok := d.(*mir.InterfaceDecl)
		if !ok || seen[decl.Name] {
			continue
		}
		seen[decl.Name] = true
		out = append(out, InterfaceDecl{Name: decl.Name})
	}
	return out
}

func RuntimeRouteStrings(handlers []intrinsics.HTTPHandlerSpec) []string {
	if len(handlers) == 0 {
		return nil
	}
	out := make([]string, 0, len(handlers)*2)
	for _, h := range handlers {
		out = append(out, h.Method, intrinsics.HTTPRoutePattern(h))
	}
	return out
}

func CollectOrderedFuncs(p *mir.Program) []*mir.FuncDecl {
	if p == nil {
		return nil
	}
	ordered := make([]*mir.FuncDecl, 0, len(p.Decls))
	index := map[string]int{}
	for _, d := range p.Decls {
		fn, ok := d.(*mir.FuncDecl)
		if !ok {
			continue
		}
		if i, ok := index[fn.Name]; ok {
			ordered[i] = fn
			continue
		}
		index[fn.Name] = len(ordered)
		ordered = append(ordered, fn)
	}
	return ordered
}

func CollectProgramFuncSigs(p *mir.Program) map[string]FuncSig {
	out := map[string]FuncSig{}
	if p == nil {
		return out
	}
	for _, fn := range CollectOrderedFuncs(p) {
		params := make([]ast.Type, 0, len(fn.Params))
		for _, p := range fn.Params {
			params = append(params, baztypes.ToAST(p.Type))
		}
		out[fn.Name] = FuncSig{Params: params, Ret: baztypes.ToAST(fn.ReturnType)}
	}
	return out
}

func collectRuntimeFuncSigs(functions []intrinsics.FunctionSpec) map[string]FuncSig {
	out := map[string]FuncSig{}
	for _, fn := range functions {
		out[fn.Name] = FuncSig{Params: fn.Params, Ret: intrinsics.NormalizeLLVMType(fn.Ret)}
	}
	return out
}

func CollectRuntimeShapeMeta(runtime ProgramRuntimeMeta) RuntimeShapeMeta {
	return RuntimeShapeMeta{
		LLVMRuntimeSurface: CollectLLVMRuntimeSurfaceMeta(runtime),
		RouteStrings:       runtime.Routes.RouteStrings,
	}
}

func CollectProgramShapeMeta(p *mir.Program) ProgramShapeMeta {
	runtime := CollectProgramRuntimeMeta(p)
	return ProgramShapeMeta{
		Runtime:         runtime,
		RuntimeShape:    CollectRuntimeShapeMeta(runtime),
		OrderedFuncs:    CollectOrderedFuncs(p),
		StructNodes:     CollectProgramStructNodes(p),
		InterfaceNodes:  CollectProgramInterfaceNodes(p),
		EnumNodes:       CollectProgramEnumNodes(p),
		GlobalNodes:     CollectProgramGlobalNodes(p),
		ProgramFuncSigs: CollectProgramFuncSigs(p),
		Globals:         CollectProgramGlobals(p),
		Enums:           CollectProgramEnums(p),
		Structs:         CollectProgramStructs(p),
		Interfaces:      CollectProgramInterfaces(p),
	}
}
