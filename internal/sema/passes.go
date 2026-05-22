package sema

import (
	"fmt"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/diag"
	"baziclang/internal/source"
)

type declCollector struct {
	c *Checker
}

func newDeclCollector(c *Checker) declCollector {
	return declCollector{c: c}
}

func (p declCollector) run(prog *ast.Program) error {
	for _, d := range prog.Decls {
		switch decl := d.(type) {
		case *ast.ImportDecl, *ast.GlobalLetDecl:
			continue
		case *ast.StructDecl:
			if err := p.collectStruct(decl); err != nil {
				return err
			}
		case *ast.InterfaceDecl:
			if err := p.collectInterface(decl); err != nil {
				return err
			}
		case *ast.EnumDecl:
			if err := p.collectEnum(decl); err != nil {
				return err
			}
		case *ast.FuncDecl:
			if err := p.collectFunc(decl); err != nil {
				return err
			}
		case *ast.ImplDecl:
			canonStruct, err := p.c.canonicalizeTypeRef(decl.StructType, nil, false)
			if err != nil {
				return err
			}
			decl.StructType = canonStruct
			canonIface, err := p.c.canonicalizeTypeRef(ast.Type(decl.InterfaceName), nil, false)
			if err != nil {
				return err
			}
			decl.InterfaceName = string(canonIface)
			p.c.impls = append(p.c.impls, *decl)
		}
	}
	return nil
}

func (p declCollector) collectStruct(decl *ast.StructDecl) error {
	prevPackage := p.c.currentPackage
	p.c.currentPackage = decl.PackageID
	defer func() { p.c.currentPackage = prevPackage }()
	tpMap, err := p.validateTypeParams(decl, "struct", decl.Name, decl.TypeParams, &decl.TypeParamBounds)
	if err != nil {
		return err
	}
	for tp, bound := range decl.TypeParamBounds {
		canon, err := p.c.canonicalizeTypeRef(bound, tpMap, false)
		if err != nil {
			return fmt.Errorf("in struct '%s': %w", decl.Name, err)
		}
		decl.TypeParamBounds[tp] = canon
	}
	fields := map[string]ast.Type{}
	for i := range decl.Fields {
		f := &decl.Fields[i]
		if _, ok := fields[f.Name]; ok {
			return p.c.fieldError(f.Range, "duplicate field '%s' in struct '%s'", f.Name, decl.Name)
		}
		canon, err := p.c.canonicalizeTypeRef(f.Type, tpMap, false)
		if err != nil {
			return fmt.Errorf("in struct '%s': %w", decl.Name, err)
		}
		f.Type = canon
		fields[f.Name] = f.Type
	}
	sig := structSig(decl.TypeParams, decl.TypeParamBounds, fields)
	sig.PackageID = decl.PackageID
	sig.Public = decl.Public
	sig.InternalName = firstNonEmpty(decl.InternalName, decl.Name)
	if err := p.c.registerStruct(decl.Name, sig); err != nil {
		return p.c.nodeError(decl, "%s", err.Error())
	}
	return nil
}

func (p declCollector) collectInterface(decl *ast.InterfaceDecl) error {
	prevPackage := p.c.currentPackage
	p.c.currentPackage = decl.PackageID
	defer func() { p.c.currentPackage = prevPackage }()
	methods := map[string]InterfaceMethodSig{}
	for i := range decl.Methods {
		m := &decl.Methods[i]
		if _, exists := methods[m.Name]; exists {
			return p.c.fieldError(m.Range, "duplicate method '%s' in interface '%s'", m.Name, decl.Name)
		}
		params := make([]ast.Type, 0, len(m.Params))
		for i := range m.Params {
			param := &m.Params[i]
			canon, err := p.c.canonicalizeTypeRef(param.Type, nil, false)
			if err != nil {
				return fmt.Errorf("in interface '%s' method '%s': %w", decl.Name, m.Name, err)
			}
			param.Type = canon
			params = append(params, param.Type)
		}
		ret, err := p.c.canonicalizeTypeRef(m.Return, nil, true)
		if err != nil {
			return fmt.Errorf("in interface '%s' method '%s': %w", decl.Name, m.Name, err)
		}
		m.Return = ret
		methods[m.Name] = interfaceMethodSig(params, m.Return)
	}
	if err := p.c.registerInterface(decl.Name, InterfaceSig{Methods: methods, PackageID: decl.PackageID, Public: decl.Public, InternalName: firstNonEmpty(decl.InternalName, decl.Name)}); err != nil {
		return p.c.nodeError(decl, "%s", err.Error())
	}
	return nil
}

func (p declCollector) collectEnum(decl *ast.EnumDecl) error {
	variants := map[string]bool{}
	for _, variant := range decl.Variants {
		if variants[variant] {
			return p.c.nodeError(decl, "duplicate enum variant '%s' in enum '%s'", variant, decl.Name)
		}
		variants[variant] = true
		if err := p.c.registerGlobal(variant, GlobalSymbol{
			Type:      ast.Type(firstNonEmpty(decl.InternalName, decl.Name)),
			Const:     true,
			PackageID: decl.PackageID,
			Public:    decl.Public,
			InternalName: internalGlobalName(decl.PackageID, variant),
		}); err != nil {
			return p.c.nodeError(decl, "%s", err.Error())
		}
	}
	if err := p.c.registerEnum(decl.Name, EnumSig{Variants: variants, PackageID: decl.PackageID, Public: decl.Public, InternalName: firstNonEmpty(decl.InternalName, decl.Name)}); err != nil {
		return p.c.nodeError(decl, "%s", err.Error())
	}
	return nil
}

func (p declCollector) collectFunc(decl *ast.FuncDecl) error {
	prevPackage := p.c.currentPackage
	p.c.currentPackage = decl.PackageID
	defer func() { p.c.currentPackage = prevPackage }()
	tpMap, err := p.validateTypeParams(decl, "function", decl.Name, decl.TypeParams, &decl.TypeParamBounds)
	if err != nil {
		return err
	}
	for tp, bound := range decl.TypeParamBounds {
		canon, err := p.c.canonicalizeTypeRef(bound, tpMap, false)
		if err != nil {
			return fmt.Errorf("in function '%s': %w", decl.Name, err)
		}
		decl.TypeParamBounds[tp] = canon
	}
	params := make([]ast.Type, 0, len(decl.Params))
	for i := range decl.Params {
		param := &decl.Params[i]
		canon, err := p.c.canonicalizeTypeRef(param.Type, tpMap, false)
		if err != nil {
			return fmt.Errorf("in function '%s': %w", decl.Name, err)
		}
		param.Type = canon
		params = append(params, param.Type)
	}
	ret, err := p.c.canonicalizeTypeRef(decl.ReturnType, tpMap, true)
	if err != nil {
		return fmt.Errorf("in function '%s': %w", decl.Name, err)
	}
	decl.ReturnType = ret
	sig := genericFuncSig(decl.TypeParams, decl.TypeParamBounds, params, decl.ReturnType)
	sig.PackageID = decl.PackageID
	sig.Public = decl.Public
	sig.InternalName = decl.InternalName
	if err := p.c.registerFunction(decl.Name, sig); err != nil {
		return p.c.nodeError(decl, "%s", err.Error())
	}
	if decl.Name == "main" {
		p.c.mainDecl = decl
	}
	return nil
}

func (p declCollector) validateTypeParams(node ast.Node, kind string, name string, typeParams []string, bounds *map[string]ast.Type) (map[string]bool, error) {
	tpMap := map[string]bool{}
	for _, tp := range typeParams {
		if tpMap[tp] {
			return nil, p.c.nodeError(node, "duplicate type parameter '%s' in %s '%s'", tp, kind, name)
		}
		tpMap[tp] = true
	}
	if *bounds == nil {
		*bounds = map[string]ast.Type{}
	}
	for tp := range *bounds {
		if !tpMap[tp] {
			return nil, p.c.nodeError(node, "unknown type parameter '%s' in bounds for %s '%s'", tp, kind, name)
		}
	}
	return tpMap, nil
}

type globalCollector struct {
	c *Checker
}

func newGlobalCollector(c *Checker) globalCollector {
	return globalCollector{c: c}
}

func (p globalCollector) run(prog *ast.Program) error {
	for _, d := range prog.Decls {
		decl, ok := d.(*ast.GlobalLetDecl)
		if !ok {
			continue
		}
		if err := p.collectGlobal(decl); err != nil {
			return err
		}
	}
	return nil
}

func (p globalCollector) collectGlobal(decl *ast.GlobalLetDecl) error {
	prevPackage := p.c.currentPackage
	p.c.currentPackage = decl.PackageID
	defer func() {
		p.c.currentPackage = prevPackage
	}()
	t, err := p.c.exprType(decl.Init)
	if err != nil {
		return err
	}
	if decl.Type == ast.TypeInvalid {
		decl.Type = t
	}
	canon, err := p.c.canonicalizeTypeRef(decl.Type, nil, false)
	if err != nil {
		return err
	}
	decl.Type = canon
	if decl.Type != t && decl.Type != ast.TypeAny {
		return p.c.nodeError(decl, "global '%s' expected %s but got %s", decl.Name, decl.Type, t)
	}
	if err := p.c.registerGlobal(decl.Name, GlobalSymbol{
		Type:      decl.Type,
		Const:     decl.IsConst,
		PackageID: decl.PackageID,
		Public:    decl.Public,
		InternalName: decl.InternalName,
	}); err != nil {
		return p.c.nodeError(decl, "%s", err.Error())
	}
	return nil
}

func internalGlobalName(packageID, name string) string {
	if packageID == "" {
		return name
	}
	clean := strings.ReplaceAll(packageID, "pkg:", "pkg_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	return "__" + clean + "__" + name
}

type programShapeValidator struct {
	c *Checker
}

func newProgramShapeValidator(c *Checker) programShapeValidator {
	return programShapeValidator{c: c}
}

func (p programShapeValidator) run(_ *ast.Program) error {
	if p.c.mainDecl == nil {
		return diag.New("type error", "missing required 'main' function", source.Span{})
	}
	if err := p.c.validateMainSignature(); err != nil {
		return err
	}
	return p.c.validateTypeParamBounds()
}

type functionPass struct {
	c *Checker
}

func newFunctionPass(c *Checker) functionPass {
	return functionPass{c: c}
}

func (p functionPass) run(prog *ast.Program) error {
	for _, d := range prog.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if err := p.c.checkFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

type implPass struct {
	c *Checker
}

func newImplPass(c *Checker) implPass {
	return implPass{c: c}
}

func (p implPass) run(_ *ast.Program) error {
	return p.c.checkImpls()
}
