package sema

import (
	"fmt"
	"sort"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/diag"
	"baziclang/internal/intrinsics"
	"baziclang/internal/source"
	baztypes "baziclang/internal/types"
)

type FuncSig struct {
	TypeParams      []string
	TypeParamBounds map[string]baztypes.Type
	Params          []baztypes.Type
	Ret             baztypes.Type
	PackageID       string
	Public          bool
	InternalName    string
}

type StructSig struct {
	TypeParams      []string
	TypeParamBounds map[string]baztypes.Type
	Fields          map[string]baztypes.Type
	PackageID       string
	Public          bool
	InternalName    string
}

type InterfaceMethodSig struct {
	Params []baztypes.Type
	Ret    baztypes.Type
}

type InterfaceSig struct {
	Methods map[string]InterfaceMethodSig
	PackageID string
	Public    bool
	InternalName string
}

type EnumSig struct {
	Variants  map[string]bool
	PackageID string
	Public    bool
	InternalName string
}

type Checker struct {
	functions  map[string]FuncSig
	packageFunctions map[string]map[string]FuncSig
	structs    map[string]StructSig
	packageStructs map[string]map[string]StructSig
	internalStructs map[string]StructSig
	interfaces map[string]InterfaceSig
	packageInterfaces map[string]map[string]InterfaceSig
	internalInterfaces map[string]InterfaceSig
	enums      map[string]EnumSig
	packageEnums map[string]map[string]EnumSig
	internalEnums map[string]EnumSig
	globals    map[string]GlobalSymbol
	packageGlobals map[string]map[string]GlobalSymbol
	imports    map[string]map[string]importBinding
	bareImports map[string]map[string]bool
	impls      []ast.ImplDecl
	scopes     *scopeStack
	currentFn  ast.Type
	currentFnDecl *ast.FuncDecl
	currentPackage string
	mainDecl   *ast.FuncDecl
	fnTypes    map[string]bool
}

type GlobalSymbol struct {
	Type      ast.Type
	Const     bool
	PackageID string
	Public    bool
	InternalName string
}

type importBinding struct {
	TargetPackageID string
	ExplicitAlias   bool
}

type varInfo struct {
	typ      ast.Type
	used     bool
	declSpan source.Span
	isConst bool
}

func funcSig(params []ast.Type, ret ast.Type) FuncSig {
	return FuncSig{Params: astTypesToStructured(params), Ret: baztypes.MustParse(ret), Public: true}
}

func genericFuncSig(typeParams []string, bounds map[string]ast.Type, params []ast.Type, ret ast.Type) FuncSig {
	return FuncSig{
		TypeParams:      append([]string{}, typeParams...),
		TypeParamBounds: astTypeMapToStructured(bounds),
		Params:          astTypesToStructured(params),
		Ret:             baztypes.MustParse(ret),
		Public:          true,
	}
}

func structSig(typeParams []string, bounds map[string]ast.Type, fields map[string]ast.Type) StructSig {
	return StructSig{
		TypeParams:      append([]string{}, typeParams...),
		TypeParamBounds: astTypeMapToStructured(bounds),
		Fields:          astTypeMapToStructured(fields),
		Public:          true,
	}
}

func interfaceMethodSig(params []ast.Type, ret ast.Type) InterfaceMethodSig {
	return InterfaceMethodSig{Params: astTypesToStructured(params), Ret: baztypes.MustParse(ret)}
}

func astTypesToStructured(in []ast.Type) []baztypes.Type {
	if in == nil {
		return nil
	}
	out := make([]baztypes.Type, 0, len(in))
	for _, t := range in {
		out = append(out, baztypes.MustParse(t))
	}
	return out
}

func astTypeMapToStructured(in map[string]ast.Type) map[string]baztypes.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]baztypes.Type, len(in))
	for k, v := range in {
		if v == "" {
			out[k] = baztypes.Type{}
			continue
		}
		out[k] = baztypes.MustParse(v)
	}
	return out
}

func structuredTypeToAST(t baztypes.Type) ast.Type {
	if t.Kind == baztypes.KindInvalid && t.Name == "" && len(t.Args) == 0 {
		return ""
	}
	return baztypes.ToAST(t)
}

func structuredTypeMapToAST(in map[string]baztypes.Type) map[string]ast.Type {
	if in == nil {
		return nil
	}
	out := make(map[string]ast.Type, len(in))
	for k, v := range in {
		out[k] = structuredTypeToAST(v)
	}
	return out
}

func New() *Checker {
	c := &Checker{
		functions:  map[string]FuncSig{},
		packageFunctions: map[string]map[string]FuncSig{},
		structs:    map[string]StructSig{},
		packageStructs: map[string]map[string]StructSig{},
		internalStructs: map[string]StructSig{},
		interfaces: map[string]InterfaceSig{},
		packageInterfaces: map[string]map[string]InterfaceSig{},
		internalInterfaces: map[string]InterfaceSig{},
		enums:      map[string]EnumSig{},
		packageEnums: map[string]map[string]EnumSig{},
		internalEnums: map[string]EnumSig{},
		globals:    map[string]GlobalSymbol{},
		packageGlobals: map[string]map[string]GlobalSymbol{},
		imports:    map[string]map[string]importBinding{},
		bareImports: map[string]map[string]bool{},
		scopes:     newScopeStack(),
		fnTypes:    map[string]bool{},
	}
	for _, fn := range intrinsics.FunctionSpecs(nil) {
		c.functions[fn.Name] = funcSig(fn.Params, fn.Ret)
	}
	return c
}

func (c *Checker) diagAt(span source.Span, format string, args ...any) error {
	return diag.New("type error", fmt.Sprintf(format, args...), span)
}

func (c *Checker) nodeError(node ast.Node, format string, args ...any) error {
	if node == nil {
		return diag.New("type error", fmt.Sprintf(format, args...), source.Span{})
	}
	return c.diagAt(node.Span(), format, args...)
}

func (c *Checker) fieldError(span source.Span, format string, args ...any) error {
	return c.diagAt(span, format, args...)
}

type checkPass struct {
	name string
	run  func(*Checker, *ast.Program) error
}

type typeLookupStatus int

const (
	typeLookupMissing typeLookupStatus = iota
	typeLookupFound
	typeLookupAmbiguous
)

func (c *Checker) Check(p *ast.Program) error {
	c.collectImportRefs(p)
	passes := []checkPass{
		{name: "collect declarations", run: (*Checker).collectDecls},
		{name: "collect globals", run: (*Checker).collectGlobals},
		{name: "validate entry points", run: (*Checker).validateProgramShape},
		{name: "check functions", run: (*Checker).checkFunctions},
		{name: "check impls", run: (*Checker).checkProgramImpls},
	}
	for _, pass := range passes {
		if err := pass.run(c, p); err != nil {
			return err
		}
	}
	return nil
}

func (c *Checker) collectImportRefs(p *ast.Program) {
	c.imports = map[string]map[string]importBinding{}
	c.bareImports = map[string]map[string]bool{}
	if p == nil {
		return
	}
	for _, ref := range p.Imports {
		if ref.OwnerPackageID == "" || ref.Alias == "" || ref.TargetPackageID == "" {
			continue
		}
		byOwner := c.imports[ref.OwnerPackageID]
		if byOwner == nil {
			byOwner = map[string]importBinding{}
			c.imports[ref.OwnerPackageID] = byOwner
		}
		byOwner[ref.Alias] = importBinding{TargetPackageID: ref.TargetPackageID, ExplicitAlias: ref.ExplicitAlias}
		if ref.BareAllowed {
			bare := c.bareImports[ref.OwnerPackageID]
			if bare == nil {
				bare = map[string]bool{}
				c.bareImports[ref.OwnerPackageID] = bare
			}
			bare[ref.TargetPackageID] = true
		}
	}
}

func (c *Checker) collectDecls(p *ast.Program) error {
	return newDeclCollector(c).run(p)
}

func (c *Checker) collectGlobals(p *ast.Program) error {
	return newGlobalCollector(c).run(p)
}

func (c *Checker) validateProgramShape(_ *ast.Program) error {
	return newProgramShapeValidator(c).run(nil)
}

func (c *Checker) checkFunctions(p *ast.Program) error {
	return newFunctionPass(c).run(p)
}

func (c *Checker) checkProgramImpls(_ *ast.Program) error {
	return newImplPass(c).run(nil)
}

func (c *Checker) validateTypeParamBounds() error {
	for name, sig := range c.structs {
		if err := c.validateStructBounds(name, sig); err != nil {
			return err
		}
	}
	for _, byPkg := range c.packageStructs {
		for name, sig := range byPkg {
			if err := c.validateStructBounds(name, sig); err != nil {
				return err
			}
		}
	}
	for name, sig := range c.functions {
		if err := c.validateFuncBounds(name, sig); err != nil {
			return err
		}
	}
	for _, byPkg := range c.packageFunctions {
		for name, sig := range byPkg {
			if err := c.validateFuncBounds(name, sig); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Checker) validateStructBounds(name string, sig StructSig) error {
	prevPackage := c.currentPackage
	c.currentPackage = sig.PackageID
	defer func() { c.currentPackage = prevPackage }()
	for tp, bound := range sig.TypeParamBounds {
		rawBound := structuredTypeToAST(bound)
		if rawBound == "" {
			continue
		}
		if _, ok := c.resolveInterface(string(rawBound)); ok != typeLookupFound {
			return fmt.Errorf("type error: unknown interface '%s' bound for struct '%s' type param '%s'", rawBound, name, tp)
		}
	}
	return nil
}

func (c *Checker) validateFuncBounds(name string, sig FuncSig) error {
	prevPackage := c.currentPackage
	c.currentPackage = sig.PackageID
	defer func() { c.currentPackage = prevPackage }()
	for tp, bound := range sig.TypeParamBounds {
		rawBound := structuredTypeToAST(bound)
		if rawBound == "" {
			continue
		}
		if _, ok := c.resolveInterface(string(rawBound)); ok != typeLookupFound {
			return fmt.Errorf("type error: unknown interface '%s' bound for function '%s' type param '%s'", rawBound, name, tp)
		}
	}
	return nil
}

func (c *Checker) validateMainSignature() error {
	var sig FuncSig
	var ok bool
	if c.mainDecl != nil && c.mainDecl.PackageID != "" {
		if byPkg := c.packageFunctions[c.mainDecl.PackageID]; byPkg != nil {
			sig, ok = byPkg["main"]
		}
	} else {
		sig, ok = c.functions["main"]
	}
	if !ok {
		if byPkg := c.packageFunctions["main"]; byPkg != nil {
			sig, ok = byPkg["main"]
		}
	}
	if !ok {
		return c.nodeError(c.mainDecl, "missing main signature")
	}
	if len(sig.TypeParams) != 0 {
		return c.nodeError(c.mainDecl, "'main' cannot be generic")
	}
	if len(sig.Params) != 0 {
		return c.nodeError(c.mainDecl, "'main' must not take parameters")
	}
	if structuredTypeToAST(sig.Ret) != ast.TypeVoid {
		return c.nodeError(c.mainDecl, "'main' must return void")
	}
	return nil
}

func (c *Checker) checkFunc(fn *ast.FuncDecl) error {
	return newFuncChecker(c, fn).run()
}

func (c *Checker) checkStmt(s ast.Stmt) error {
	switch st := s.(type) {
	case *ast.LetStmt:
		t, err := c.exprType(st.Init)
		if err != nil {
			return err
		}
		if st.Type == ast.TypeInvalid {
			st.Type = t
		}
		canon, err := c.canonicalizeTypeRef(st.Type, c.fnTypes, false)
		if err != nil {
			return err
		}
		st.Type = canon
		if st.Type != t && st.Type != ast.TypeAny {
			return c.nodeError(st, "variable '%s' expected %s but got %s", st.Name, st.Type, t)
		}
		return c.declare(st.Name, st.Type, st.IsConst, st.Span())
	case *ast.AssignStmt:
		targetType, err := c.typeOfAssignTarget(st.Target)
		if err != nil {
			return err
		}
		if root, ok := rootIdent(st.Target); ok {
			if _, isConst, ok := c.resolveVar(root, false); ok && isConst {
				return c.nodeError(st, "cannot assign to const '%s'", root)
			}
		}
		rhs, err := c.exprType(st.Value)
		if err != nil {
			return err
		}
		if targetType != rhs && targetType != ast.TypeAny {
			return c.nodeError(st, "cannot assign %s to '%s' (%s)", rhs, formatAssignTargetName(st.Target), targetType)
		}
		return nil
	case *ast.IfStmt:
		cond, err := c.exprType(st.Cond)
		if err != nil {
			return err
		}
		if cond != ast.TypeBool {
			return c.nodeError(st.Cond, "if condition must be bool, got %s", cond)
		}
		if err := c.checkBlock(st.Then); err != nil {
			return err
		}
		if st.Else != nil {
			if err := c.checkBlock(st.Else); err != nil {
				return err
			}
		}
		return nil
	case *ast.WhileStmt:
		cond, err := c.exprType(st.Cond)
		if err != nil {
			return err
		}
		if cond != ast.TypeBool {
			return c.nodeError(st.Cond, "while condition must be bool, got %s", cond)
		}
		return c.checkBlock(st.Body)
	case *ast.MatchStmt:
		return c.checkMatchStmt(st)
	case *ast.ReturnStmt:
		if st.Value == nil {
			if c.currentFn != ast.TypeVoid {
				return c.nodeError(st, "return value required for function returning %s", c.currentFn)
			}
			return nil
		}
		t, err := c.exprType(st.Value)
		if err != nil {
			return err
		}
		if t != c.currentFn && c.currentFn != ast.TypeAny {
			return c.nodeError(st, "return type mismatch, expected %s got %s", c.currentFn, t)
		}
		return nil
	case *ast.ExprStmt:
		_, err := c.exprType(st.Expr)
		return err
	default:
		return c.nodeError(s, "unsupported statement")
	}
}

func (c *Checker) typeOfAssignTarget(target ast.Expr) (ast.Type, error) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		typ, _, ok := c.resolveVar(t.Name, false)
		if !ok {
			return ast.TypeInvalid, c.nodeError(t, "unknown variable '%s'%s", t.Name, c.suggestVisibleName(t.Name))
		}
		return typ, nil
	case *ast.FieldAccessExpr:
		return c.exprType(t)
	default:
		return ast.TypeInvalid, c.nodeError(target, "invalid assignment target")
	}
}

func rootIdent(target ast.Expr) (string, bool) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		return t.Name, true
	case *ast.FieldAccessExpr:
		return rootIdent(t.Object)
	default:
		return "", false
	}
}

func formatAssignTargetName(target ast.Expr) string {
	switch t := target.(type) {
	case *ast.IdentExpr:
		return t.Name
	case *ast.FieldAccessExpr:
		return formatAssignTargetName(t.Object) + "." + t.Field
	default:
		return "target"
	}
}

func (c *Checker) checkMatchStmt(st *ast.MatchStmt) error {
	enumName, enumSig, err := c.resolveMatchSubjectEnum(st.Subject)
	if err != nil {
		return err
	}
	unguarded := map[string]bool{}
	for i := range st.Arms {
		arm := &st.Arms[i]
		variant, err := c.validateMatchVariant(enumName, enumSig, unguarded, arm.Variant, arm.Guard, arm.Range)
		if err != nil {
			return err
		}
		arm.Variant = variant
		if arm.Guard != nil {
			t, err := c.exprType(arm.Guard)
			if err != nil {
				return err
			}
			if t != ast.TypeBool {
				return c.nodeError(arm.Guard, "match guard must be bool, got %s", t)
			}
		}
		if err := c.checkBlock(arm.Body); err != nil {
			return err
		}
	}
	if err := ensureMatchExhaustive(enumName, enumSig.Variants, unguarded); err != nil {
		return err
	}
	return nil
}

func (c *Checker) checkBlock(b *ast.BlockStmt) error {
	c.pushScope()
	for _, s := range b.Stmts {
		if err := c.checkStmt(s); err != nil {
			_ = c.popScope()
			return err
		}
	}
	return c.popScope()
}

func (c *Checker) exprType(e ast.Expr) (ast.Type, error) {
	switch ex := e.(type) {
	case *ast.IntExpr:
		return ast.TypeInt, nil
	case *ast.FloatExpr:
		return ast.TypeFloat, nil
	case *ast.BoolExpr:
		return ast.TypeBool, nil
	case *ast.StringExpr:
		return ast.TypeString, nil
	case *ast.NilExpr:
		return ast.TypeInvalid, c.nodeError(ex, "'nil' is not a value in Bazic; use Option[T], Result[T,E], or Error with explicit state")
	case *ast.IdentExpr:
		if t, ok := c.resolve(ex.Name, true); ok {
			if g, status := c.resolveGlobalSymbol(ex.Name); status == typeLookupFound && g.InternalName != "" && !c.isEnumVariantLiteral(ex.Name, g) {
				ex.Resolved = g.InternalName
			}
			return t, nil
		}
		return ast.TypeInvalid, c.nodeError(ex, "unknown identifier '%s'%s", ex.Name, c.suggestVisibleName(ex.Name))
	case *ast.StructLitExpr:
		canonType, err := c.canonicalizeTypeRef(ast.Type(ex.TypeName), c.fnTypes, false)
		if err != nil {
			return ast.TypeInvalid, c.nodeError(ex, "%s", err.Error())
		}
		ex.TypeName = string(canonType)
		base, args, _ := splitGenericType(string(ex.TypeName))
		if base == "" {
			base = ex.TypeName
		}
		sig, status := c.resolveStruct(base)
		if status == typeLookupAmbiguous {
			return ast.TypeInvalid, c.nodeError(ex, "ambiguous struct '%s'; qualify or disambiguate the package source", ex.TypeName)
		}
		if status != typeLookupFound {
			return ast.TypeInvalid, c.nodeError(ex, "unknown struct '%s'", ex.TypeName)
		}
		mapping, err := bindTypeParams(sig.TypeParams, args)
		if err != nil {
			return ast.TypeInvalid, c.nodeError(ex, "%s", err.Error())
		}
		for tp, bound := range sig.TypeParamBounds {
			rawBound := structuredTypeToAST(bound)
			if rawBound == "" {
				continue
			}
			if actual, ok := mapping[tp]; ok {
				if err := c.requireImplements(actual, rawBound); err != nil {
					return ast.TypeInvalid, c.nodeError(ex, "type argument '%s' does not satisfy bound %s: %v", actual, rawBound, err)
				}
			}
		}
		seen := map[string]bool{}
		for _, field := range ex.Fields {
			rawExpected, ok := sig.Fields[field.Name]
			if !ok {
				return ast.TypeInvalid, c.fieldError(field.Range, "unknown field '%s' on struct '%s'%s", field.Name, ex.TypeName, suggestNameSuffix(field.Name, mapKeys(sig.Fields)))
			}
			expected := substType(structuredTypeToAST(rawExpected), mapping)
			vt, err := c.exprType(field.Value)
			if err != nil {
				return ast.TypeInvalid, err
			}
			if vt != expected && expected != ast.TypeAny {
				return ast.TypeInvalid, c.fieldError(field.Range, "field '%s' on '%s' expected %s got %s", field.Name, ex.TypeName, expected, vt)
			}
			seen[field.Name] = true
		}
		for name := range sig.Fields {
			if !seen[name] {
				return ast.TypeInvalid, c.nodeError(ex, "missing field '%s' in struct literal '%s'", name, ex.TypeName)
			}
		}
		return canonType, nil
	case *ast.FieldAccessExpr:
		if pkg, ok := ex.Object.(*ast.IdentExpr); ok {
			if _, imported := c.resolveImportedPackage(pkg.Name); imported {
				if t, _, ok := c.resolveQualifiedGlobal(pkg.Name, ex.Field); ok {
					if byPkg, ok := c.packageGlobals[c.imports[c.currentPackage][pkg.Name].TargetPackageID]; ok {
						if g, ok := byPkg[ex.Field]; ok {
							ex.ResolvedGlobal = g.InternalName
						}
					}
					return t, nil
				}
				return ast.TypeInvalid, c.nodeError(ex, "package '%s' has no public value '%s'", pkg.Name, ex.Field)
			}
			if t, _, ok := c.resolveQualifiedGlobal(pkg.Name, ex.Field); ok {
				return t, nil
			}
		}
		objType, err := c.exprType(ex.Object)
		if err != nil {
			return ast.TypeInvalid, err
		}
		base, args, _ := splitGenericType(string(objType))
		if base == "" {
			base = string(objType)
		}
		sig, status := c.resolveStruct(base)
		if status == typeLookupAmbiguous {
			return ast.TypeInvalid, c.nodeError(ex, "ambiguous struct type '%s'; qualify or disambiguate the package source", objType)
		}
		if status != typeLookupFound {
			return ast.TypeInvalid, c.nodeError(ex, "field access requires struct type, got %s", objType)
		}
		mapping, err := bindTypeParams(sig.TypeParams, args)
		if err != nil {
			return ast.TypeInvalid, err
		}
		rawField, ok := sig.Fields[ex.Field]
		if !ok {
			return ast.TypeInvalid, c.nodeError(ex, "struct '%s' has no field '%s'%s", objType, ex.Field, suggestNameSuffix(ex.Field, mapKeys(sig.Fields)))
		}
		return substType(structuredTypeToAST(rawField), mapping), nil
	case *ast.UnaryExpr:
		r, err := c.exprType(ex.Right)
		if err != nil {
			return ast.TypeInvalid, err
		}
		switch ex.Op {
		case "-":
			if r == ast.TypeInt || r == ast.TypeFloat {
				return r, nil
			}
			return ast.TypeInvalid, c.nodeError(ex, "unary '-' requires numeric type")
		case "!":
			if r == ast.TypeBool {
				return ast.TypeBool, nil
			}
			return ast.TypeInvalid, c.nodeError(ex, "unary '!' requires bool")
		}
		return ast.TypeInvalid, c.nodeError(ex, "unsupported unary operator '%s'", ex.Op)
	case *ast.BinaryExpr:
		l, err := c.exprType(ex.Left)
		if err != nil {
			return ast.TypeInvalid, err
		}
		r, err := c.exprType(ex.Right)
		if err != nil {
			return ast.TypeInvalid, err
		}
		switch ex.Op {
		case "+", "-", "*", "/", "%":
			if l != r {
				return ast.TypeInvalid, c.nodeError(ex, "operator '%s' requires matching operands", ex.Op)
			}
			if ex.Op == "+" && l == ast.TypeString {
				return ast.TypeString, nil
			}
			if l == ast.TypeInt || l == ast.TypeFloat {
				if ex.Op == "%" && l != ast.TypeInt {
					return ast.TypeInvalid, fmt.Errorf("type error: '%%' only supports int")
				}
				return l, nil
			}
			return ast.TypeInvalid, c.nodeError(ex, "invalid operands for '%s'", ex.Op)
		case "==", "!=":
			if l != r {
				return ast.TypeInvalid, c.nodeError(ex, "comparison requires same types")
			}
			return ast.TypeBool, nil
		case "<", "<=", ">", ">=":
			if l != r || (l != ast.TypeInt && l != ast.TypeFloat && l != ast.TypeString) {
				return ast.TypeInvalid, c.nodeError(ex, "invalid operands for comparison")
			}
			return ast.TypeBool, nil
		case "&&", "||":
			if l == ast.TypeBool && r == ast.TypeBool {
				return ast.TypeBool, nil
			}
			return ast.TypeInvalid, c.nodeError(ex, "logical operators require bool")
		}
		return ast.TypeInvalid, c.nodeError(ex, "unsupported operator '%s'", ex.Op)
	case *ast.CallExpr:
		if ex.Receiver != nil {
			if pkg, ok := ex.Receiver.(*ast.IdentExpr); ok {
				if _, imported := c.resolveImportedPackage(pkg.Name); imported {
					if sig, ok := c.resolveQualifiedFunction(pkg.Name, ex.Method); ok {
						visibleName := ex.Method
						ex.Callee = firstNonEmpty(sig.InternalName, ex.Method)
						ex.Args = ex.Args
						ex.Receiver = nil
						ex.Method = ""
						return c.checkCallExpr(visibleName, ex.Args, sig)
					}
					return ast.TypeInvalid, c.nodeError(ex, "package '%s' has no public function '%s'", pkg.Name, ex.Method)
				}
				if sig, ok := c.resolveQualifiedFunction(pkg.Name, ex.Method); ok {
					visibleName := ex.Method
					ex.Callee = firstNonEmpty(sig.InternalName, ex.Method)
					ex.Args = ex.Args
					ex.Receiver = nil
					ex.Method = ""
					return c.checkCallExpr(visibleName, ex.Args, sig)
				}
			}
			receiverType, err := c.exprType(ex.Receiver)
			if err != nil {
				return ast.TypeInvalid, err
			}
			base, _, ok := splitGenericType(string(receiverType))
			if !ok {
				base = string(receiverType)
			}
			resolvedName := fmt.Sprintf("%s_%s", base, ex.Method)
			sig, status := c.resolveFunction(resolvedName)
			if status == typeLookupAmbiguous {
				return ast.TypeInvalid, c.nodeError(ex, "method '%s' on '%s' is ambiguous across packages", ex.Method, receiverType)
			}
			if status != typeLookupFound {
				if _, hidden := c.resolveHiddenFunction(resolvedName); hidden {
					return ast.TypeInvalid, c.nodeError(ex, "method '%s' on '%s' is not public in this package", ex.Method, receiverType)
				}
				return ast.TypeInvalid, c.nodeError(ex, "unknown method '%s' on '%s'%s (expected function '%s')", ex.Method, receiverType, c.suggestMethodName(base, ex.Method), resolvedName)
			}
			ex.Callee = firstNonEmpty(sig.InternalName, resolvedName)
			ex.Args = append([]ast.Expr{ex.Receiver}, ex.Args...)
			ex.Receiver = nil
			ex.Method = ""
			return c.checkCallExpr(resolvedName, ex.Args, sig)
		}
		sig, status := c.resolveFunction(ex.Callee)
		if status == typeLookupAmbiguous {
			return ast.TypeInvalid, c.nodeError(ex, "function '%s' is ambiguous across packages", ex.Callee)
		}
		if status != typeLookupFound {
			if _, hidden := c.resolveHiddenFunction(ex.Callee); hidden {
				return ast.TypeInvalid, c.nodeError(ex, "function '%s' is not public in this package", ex.Callee)
			}
			return ast.TypeInvalid, c.nodeError(ex, "unknown function '%s'%s", ex.Callee, c.suggestVisibleFunctionName(ex.Callee))
		}
		visibleName := ex.Callee
		ex.Callee = firstNonEmpty(sig.InternalName, ex.Callee)
		return c.checkCallExpr(visibleName, ex.Args, sig)
	case *ast.MatchExpr:
		enumName, enumSig, err := c.resolveMatchSubjectEnum(ex.Subject)
		if err != nil {
			return ast.TypeInvalid, err
		}
		unguarded := map[string]bool{}
		armType := ast.TypeInvalid
		for i := range ex.Arms {
			arm := &ex.Arms[i]
			variant, err := c.validateMatchVariant(enumName, enumSig, unguarded, arm.Variant, arm.Guard, arm.Range)
			if err != nil {
				return ast.TypeInvalid, err
			}
			arm.Variant = variant
			if arm.Guard != nil {
				t, err := c.exprType(arm.Guard)
				if err != nil {
					return ast.TypeInvalid, err
				}
				if t != ast.TypeBool {
					return ast.TypeInvalid, c.nodeError(arm.Guard, "match guard must be bool, got %s", t)
				}
			}
			t, err := c.exprType(arm.Value)
			if err != nil {
				return ast.TypeInvalid, err
			}
			if armType == ast.TypeInvalid {
				armType = t
				continue
			}
			if armType != t {
				return ast.TypeInvalid, c.fieldError(arm.Range, "match expression arm type mismatch, expected %s got %s", armType, t)
			}
		}
		if err := ensureMatchExhaustive(enumName, enumSig.Variants, unguarded); err != nil {
			return ast.TypeInvalid, err
		}
		if armType == ast.TypeInvalid {
			return ast.TypeInvalid, c.nodeError(ex, "match expression must have at least one arm")
		}
		ex.ResolvedType = armType
		return armType, nil
	default:
		return ast.TypeInvalid, c.nodeError(e, "unsupported expression")
	}
}

func (c *Checker) resolveMatchSubjectEnum(subject ast.Expr) (string, EnumSig, error) {
	subjectType, err := c.exprType(subject)
	if err != nil {
		return "", EnumSig{}, err
	}
	enumName := string(subjectType)
	enumSig, status := c.resolveEnum(enumName)
	if status == typeLookupAmbiguous {
		return "", EnumSig{}, c.nodeError(subject, "match subject enum '%s' is ambiguous across packages", subjectType)
	}
	if status != typeLookupFound {
		return "", EnumSig{}, c.nodeError(subject, "match subject must be enum, got %s", subjectType)
	}
	return enumName, enumSig, nil
}

func (c *Checker) validateMatchVariant(enumName string, enumSig EnumSig, unguarded map[string]bool, variant string, guard ast.Expr, span source.Span) (string, error) {
	variant, err := c.normalizeMatchVariant(enumName, enumSig, variant, span)
	if err != nil {
		return "", err
	}
	variants := enumSig.Variants
	if !variants[variant] {
		return "", c.diagAt(span, "unknown variant '%s' for enum '%s'", variant, enumName)
	}
	if guard == nil {
		if unguarded[variant] {
			return "", c.diagAt(span, "duplicate match arm for variant '%s'", variant)
		}
		unguarded[variant] = true
		return variant, nil
	}
	if unguarded[variant] {
		return "", c.nodeError(guard, "guarded match arm must appear before unguarded arm for variant '%s'", variant)
	}
	return variant, nil
}

func (c *Checker) normalizeMatchVariant(enumName string, enumSig EnumSig, variant string, span source.Span) (string, error) {
	if !strings.Contains(variant, ".") {
		return variant, nil
	}
	parts := strings.Split(variant, ".")
	if len(parts) < 2 {
		return variant, nil
	}
	alias := strings.Join(parts[:len(parts)-1], ".")
	base := parts[len(parts)-1]
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok || targetPkg != enumSig.PackageID {
		return "", c.diagAt(span, "match arm variant '%s' does not belong to enum '%s'", variant, enumName)
	}
	return base, nil
}

func ensureMatchExhaustive(enumName string, variants map[string]bool, seen map[string]bool) error {
	missing := make([]string, 0, len(variants))
	for v := range variants {
		if !seen[v] {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("type error: non-exhaustive match for enum '%s'; missing unguarded variants: %s", enumName, strings.Join(missing, ", "))
	}
	return nil
}

func (c *Checker) checkCallExpr(name string, args []ast.Expr, sig FuncSig) (ast.Type, error) {
	if len(args) != len(sig.Params) {
		return ast.TypeInvalid, c.nodeError(firstArgNode(args), "function '%s' expects %d args, got %d", name, len(sig.Params), len(args))
	}
	mapping := map[string]ast.Type{}
	typeParams := typeParamSet(sig.TypeParams)
	for i, arg := range args {
		at, err := c.exprType(arg)
		if err != nil {
			return ast.TypeInvalid, err
		}
		expected := sig.Params[i]
		rawExpected := structuredTypeToAST(expected)
		if canonExpected, err := c.canonicalizeTypeRef(rawExpected, typeParams, false); err == nil {
			rawExpected = canonExpected
		}
		if rawExpected == ast.TypeAny {
			continue
		}
		if !unifyTypeParams(rawExpected, at, mapping, sig.TypeParams) {
			return ast.TypeInvalid, c.nodeError(arg, "arg %d to '%s' expected %s got %s", i+1, name, rawExpected, at)
		}
	}
	ret := structuredTypeToAST(sig.Ret)
	if len(mapping) > 0 {
		ret = substType(ret, mapping)
	}
	if canonRet, err := c.canonicalizeTypeRef(ret, typeParams, ret == ast.TypeVoid); err == nil {
		ret = canonRet
	}
	if err := c.validateTypeParamBoundsForMapping(structuredTypeMapToAST(sig.TypeParamBounds), mapping, name); err != nil {
		return ast.TypeInvalid, err
	}
	for _, tp := range sig.TypeParams {
		if typeContainsParam(ret, tp) {
			return ast.TypeInvalid, diag.New("type error", fmt.Sprintf("could not infer return type for generic function '%s'", name), source.Span{})
		}
	}
	return ret, nil
}

func unifyTypeParams(expected ast.Type, actual ast.Type, mapping map[string]ast.Type, params []string) bool {
	return baztypes.Unify(expected, actual, mapping, params)
}

func (c *Checker) validateTypeParamBoundsForMapping(bounds map[string]ast.Type, mapping map[string]ast.Type, owner string) error {
	if bounds == nil {
		return nil
	}
	for tp, bound := range bounds {
		if bound == "" {
			continue
		}
		actual, ok := mapping[tp]
		if !ok {
			return fmt.Errorf("type error: could not infer type argument '%s' for '%s'", tp, owner)
		}
		if err := c.requireImplements(actual, bound); err != nil {
			return fmt.Errorf("type error: type argument '%s' for '%s' does not satisfy bound %s: %w", tp, owner, bound, err)
		}
	}
	return nil
}

func (c *Checker) requireImplements(t ast.Type, iface ast.Type) error {
	base, ok := baztypes.Base(t)
	if ok {
		t = ast.Type(base)
	}
	_, structStatus := c.resolveStruct(string(t))
	if structStatus == typeLookupAmbiguous {
		return fmt.Errorf("'%s' is an ambiguous struct type", t)
	}
	if structStatus != typeLookupFound {
		return fmt.Errorf("'%s' is not a struct type", t)
	}
	if !c.hasImpl(t, string(iface)) {
		return fmt.Errorf("'%s' does not implement '%s'", t, iface)
	}
	return nil
}

func (c *Checker) hasImpl(structType ast.Type, iface string) bool {
	base, ok := baztypes.Base(structType)
	if ok {
		structType = ast.Type(base)
	}
	for _, impl := range c.impls {
		implBase, ok := baztypes.Base(impl.StructType)
		if ok {
			if implBase == string(structType) && impl.InterfaceName == iface {
				return true
			}
			continue
		}
		if impl.StructType == structType && impl.InterfaceName == iface {
			return true
		}
	}
	return false
}

func (c *Checker) checkImpls() error {
	for _, impl := range c.impls {
		prevPackage := c.currentPackage
		c.currentPackage = impl.PackageID
		base, ok := baztypes.Base(impl.StructType)
		if !ok {
			base = string(impl.StructType)
		}
		_, structStatus := c.resolveStruct(base)
		if structStatus == typeLookupAmbiguous {
			c.currentPackage = prevPackage
			return fmt.Errorf("type error: impl target struct '%s' is ambiguous across packages", impl.StructType)
		}
		if structStatus != typeLookupFound {
			c.currentPackage = prevPackage
			return fmt.Errorf("type error: impl target struct '%s' not found", impl.StructType)
		}
		iface, interfaceStatus := c.resolveInterface(impl.InterfaceName)
		if interfaceStatus == typeLookupAmbiguous {
			c.currentPackage = prevPackage
			return fmt.Errorf("type error: interface '%s' is ambiguous across packages", impl.InterfaceName)
		}
		if interfaceStatus != typeLookupFound {
			c.currentPackage = prevPackage
			return fmt.Errorf("type error: interface '%s' not found for impl", impl.InterfaceName)
		}
		for mname, msig := range iface.Methods {
			fnName := fmt.Sprintf("%s_%s", base, mname)
			fn, fnStatus := c.resolveFunction(fnName)
			if fnStatus == typeLookupAmbiguous {
				c.currentPackage = prevPackage
				return fmt.Errorf("type error: impl %s:%s method function '%s' is ambiguous across packages", impl.StructType, impl.InterfaceName, fnName)
			}
			if fnStatus != typeLookupFound {
				c.currentPackage = prevPackage
				return fmt.Errorf("type error: impl %s:%s missing function '%s'", impl.StructType, impl.InterfaceName, fnName)
			}
			if len(msig.Params) > 0 && structuredTypeToAST(msig.Params[0]) == impl.StructType {
				if len(fn.Params) != len(msig.Params) {
					c.currentPackage = prevPackage
					return fmt.Errorf("type error: '%s' must have %d params", fnName, len(msig.Params))
				}
				for i := range msig.Params {
					if structuredTypeToAST(fn.Params[i]) != structuredTypeToAST(msig.Params[i]) {
						c.currentPackage = prevPackage
						return fmt.Errorf("type error: '%s' param %d mismatch", fnName, i+1)
					}
				}
			} else {
				if len(fn.Params) != len(msig.Params)+1 {
					c.currentPackage = prevPackage
					return fmt.Errorf("type error: '%s' must have receiver + %d params", fnName, len(msig.Params))
				}
				if structuredTypeToAST(fn.Params[0]) != impl.StructType {
					c.currentPackage = prevPackage
					return fmt.Errorf("type error: '%s' first param must be %s", fnName, impl.StructType)
				}
				for i := range msig.Params {
					if structuredTypeToAST(fn.Params[i+1]) != structuredTypeToAST(msig.Params[i]) {
						c.currentPackage = prevPackage
						return fmt.Errorf("type error: '%s' param %d mismatch", fnName, i+2)
					}
				}
			}
			if structuredTypeToAST(fn.Ret) != structuredTypeToAST(msig.Ret) {
				c.currentPackage = prevPackage
				return fmt.Errorf("type error: '%s' return type mismatch", fnName)
			}
		}
		c.currentPackage = prevPackage
	}
	return nil
}

func bindTypeParams(typeParams []string, args []string) (map[string]ast.Type, error) {
	actual := make([]ast.Type, 0, len(args))
	for _, arg := range args {
		actual = append(actual, ast.Type(arg))
	}
	return baztypes.BindTypeParams(typeParams, actual)
}

func substType(t ast.Type, mapping map[string]ast.Type) ast.Type {
	return baztypes.Substitute(t, mapping)
}

func (c *Checker) validateTypeRef(t ast.Type, typeParams map[string]bool, allowVoid bool) error {
	_, err := c.canonicalizeTypeRef(t, typeParams, allowVoid)
	return err
}

func (c *Checker) canonicalizeTypeRef(t ast.Type, typeParams map[string]bool, allowVoid bool) (ast.Type, error) {
	if t == ast.TypeInvalid {
		return "", fmt.Errorf("type error: invalid type")
	}
	if t == ast.TypeVoid && !allowVoid {
		return "", fmt.Errorf("type error: void cannot be used here")
	}
	name := string(t)
	if typeParams != nil && typeParams[name] {
		return t, nil
	}
	if isBuiltin(t) {
		return t, nil
	}
	if base, args, ok := splitGenericType(name); ok {
		if alias, target, qualified := splitQualifiedTypeBase(base); qualified {
			sig, found := c.resolveQualifiedStruct(alias, target)
			if !found {
				return "", fmt.Errorf("type error: package '%s' has no public type '%s'", alias, target)
			}
			if len(sig.TypeParams) != len(args) {
				return "", fmt.Errorf("type error: type '%s' expects %d args, got %d", base, len(sig.TypeParams), len(args))
			}
			canonArgs := make([]string, 0, len(args))
			for _, a := range args {
				canonArg, err := c.canonicalizeTypeRef(ast.Type(a), typeParams, false)
				if err != nil {
					return "", err
				}
				canonArgs = append(canonArgs, string(canonArg))
			}
			for i, tp := range sig.TypeParams {
				bound := sig.TypeParamBounds[tp]
				rawBound := structuredTypeToAST(bound)
				if rawBound == "" {
					continue
				}
				arg := ast.Type(args[i])
				if typeParams != nil && typeParams[string(arg)] {
					continue
				}
				if err := c.requireImplements(arg, rawBound); err != nil {
					return "", fmt.Errorf("type error: type argument '%s' does not satisfy bound %s: %w", arg, rawBound, err)
				}
			}
			return ast.Type(fmt.Sprintf("%s[%s]", firstNonEmpty(sig.InternalName, target), strings.Join(canonArgs, ","))), nil
		}
	}
	if alias, target, qualified := splitQualifiedTypeBase(name); qualified {
		if sig, ok := c.resolveQualifiedStruct(alias, target); ok {
			return ast.Type(firstNonEmpty(sig.InternalName, target)), nil
		}
		if sig, ok := c.resolveQualifiedEnum(alias, target); ok {
			return ast.Type(firstNonEmpty(sig.InternalName, target)), nil
		}
		if sig, ok := c.resolveQualifiedInterface(alias, target); ok {
			return ast.Type(firstNonEmpty(sig.InternalName, target)), nil
		}
		return "", fmt.Errorf("type error: package '%s' has no public type '%s'", alias, target)
	}
	if sig, status := c.resolveStruct(name); status == typeLookupFound {
		return ast.Type(firstNonEmpty(sig.InternalName, name)), nil
	}
	if sig, status := c.resolveEnum(name); status == typeLookupFound {
		return ast.Type(firstNonEmpty(sig.InternalName, name)), nil
	}
	if sig, status := c.resolveInterface(name); status == typeLookupFound {
		return ast.Type(firstNonEmpty(sig.InternalName, name)), nil
	}
	if _, status := c.resolveStruct(name); status == typeLookupAmbiguous {
		return "", fmt.Errorf("type error: ambiguous type '%s'", t)
	}
	if _, status := c.resolveEnum(name); status == typeLookupAmbiguous {
		return "", fmt.Errorf("type error: ambiguous type '%s'", t)
	}
	if _, status := c.resolveInterface(name); status == typeLookupAmbiguous {
		return "", fmt.Errorf("type error: ambiguous type '%s'", t)
	}
	if base, args, ok := splitGenericType(name); ok {
		sig, status := c.resolveStruct(base)
		if status == typeLookupAmbiguous {
			return "", fmt.Errorf("type error: ambiguous generic base type '%s'", base)
		}
		if status != typeLookupFound {
			return "", fmt.Errorf("type error: unknown generic base type '%s'", base)
		}
		if len(sig.TypeParams) != len(args) {
			return "", fmt.Errorf("type error: type '%s' expects %d args, got %d", base, len(sig.TypeParams), len(args))
		}
		canonArgs := make([]string, 0, len(args))
		for _, a := range args {
			canonArg, err := c.canonicalizeTypeRef(ast.Type(a), typeParams, false)
			if err != nil {
				return "", err
			}
			canonArgs = append(canonArgs, string(canonArg))
		}
		for i, tp := range sig.TypeParams {
			bound := sig.TypeParamBounds[tp]
			rawBound := structuredTypeToAST(bound)
			if rawBound == "" {
				continue
			}
			arg := ast.Type(args[i])
			if typeParams != nil && typeParams[string(arg)] {
				continue
			}
			if err := c.requireImplements(arg, rawBound); err != nil {
				return "", fmt.Errorf("type error: type argument '%s' does not satisfy bound %s: %w", arg, rawBound, err)
			}
		}
		return ast.Type(fmt.Sprintf("%s[%s]", firstNonEmpty(sig.InternalName, base), strings.Join(canonArgs, ","))), nil
	}
	return "", fmt.Errorf("type error: unknown type '%s'%s", t, c.suggestVisibleTypeName(name))
}

func splitGenericType(t string) (string, []string, bool) {
	base, ok := baztypes.Base(ast.Type(t))
	if !ok {
		return "", nil, false
	}
	args, ok := baztypes.Args(ast.Type(t))
	if !ok {
		return "", nil, false
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, string(arg))
	}
	return base, out, true
}

func splitTopLevel(s string) []string {
	args, ok := baztypes.Args(ast.Type("X[" + s + "]"))
	if !ok {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, string(arg))
	}
	return out
}

func splitQualifiedTypeBase(name string) (string, string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isBuiltin(t ast.Type) bool {
	switch t {
	case ast.TypeAny, ast.TypeBool, ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeVoid:
		return true
	default:
		return false
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func typeContainsParam(t ast.Type, param string) bool {
	return baztypes.ContainsParam(t, param)
}

func typeParamSet(params []string) map[string]bool {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]bool, len(params))
	for _, param := range params {
		out[param] = true
	}
	return out
}

func (c *Checker) suggestVisibleName(name string) string {
	candidates := map[string]bool{}
	for _, n := range c.scopes.visibleNames() {
		candidates[n] = true
	}
	for n, g := range c.globals {
		if c.canAccessPackage(g.PackageID, g.Public) {
			candidates[n] = true
		}
	}
	if c.currentPackage != "" {
		if byPkg := c.packageGlobals[c.currentPackage]; byPkg != nil {
			for n := range byPkg {
				candidates[n] = true
			}
		}
		for _, pkgID := range c.currentBareImportPackageIDs() {
			if byPkg := c.packageGlobals[pkgID]; byPkg != nil {
				for n, g := range byPkg {
					if c.canAccessPackage(g.PackageID, g.Public) {
						candidates[n] = true
					}
				}
			}
		}
	}
	c.addQualifiedGlobalCandidates(candidates)
	return suggestNameSuffix(name, mapKeys(candidates))
}

func (c *Checker) suggestVisibleFunctionName(name string) string {
	candidates := map[string]bool{}
	for n, sig := range c.functions {
		if c.canAccessPackage(sig.PackageID, sig.Public) {
			candidates[n] = true
		}
	}
	if c.currentPackage != "" {
		if byPkg := c.packageFunctions[c.currentPackage]; byPkg != nil {
			for n := range byPkg {
				candidates[n] = true
			}
		}
		for _, pkgID := range c.currentBareImportPackageIDs() {
			if byPkg := c.packageFunctions[pkgID]; byPkg != nil {
				for n, sig := range byPkg {
					if c.canAccessPackage(sig.PackageID, sig.Public) {
						candidates[n] = true
					}
				}
			}
		}
	}
	c.addQualifiedFunctionCandidates(candidates)
	return suggestNameSuffix(name, mapKeys(candidates))
}

func (c *Checker) suggestVisibleTypeName(name string) string {
	candidates := map[string]bool{}
	for n, sig := range c.structs {
		if c.canAccessPackage(sig.PackageID, sig.Public) {
			candidates[n] = true
		}
	}
	for n, sig := range c.interfaces {
		if c.canAccessPackage(sig.PackageID, sig.Public) {
			candidates[n] = true
		}
	}
	for n, sig := range c.enums {
		if c.canAccessPackage(sig.PackageID, sig.Public) {
			candidates[n] = true
		}
	}
	if c.currentPackage != "" {
		if byPkg := c.packageStructs[c.currentPackage]; byPkg != nil {
			for n := range byPkg {
				candidates[n] = true
			}
		}
		if byPkg := c.packageInterfaces[c.currentPackage]; byPkg != nil {
			for n := range byPkg {
				candidates[n] = true
			}
		}
		if byPkg := c.packageEnums[c.currentPackage]; byPkg != nil {
			for n := range byPkg {
				candidates[n] = true
			}
		}
		for _, pkgID := range c.currentBareImportPackageIDs() {
			if byPkg := c.packageStructs[pkgID]; byPkg != nil {
				for n, sig := range byPkg {
					if c.canAccessPackage(sig.PackageID, sig.Public) {
						candidates[n] = true
					}
				}
			}
			if byPkg := c.packageInterfaces[pkgID]; byPkg != nil {
				for n, sig := range byPkg {
					if c.canAccessPackage(sig.PackageID, sig.Public) {
						candidates[n] = true
					}
				}
			}
			if byPkg := c.packageEnums[pkgID]; byPkg != nil {
				for n, sig := range byPkg {
					if c.canAccessPackage(sig.PackageID, sig.Public) {
						candidates[n] = true
					}
				}
			}
		}
	}
	c.addQualifiedTypeCandidates(candidates)
	return suggestNameSuffix(name, mapKeys(candidates))
}

func (c *Checker) addQualifiedGlobalCandidates(candidates map[string]bool) {
	for alias, targetPkg := range c.currentImportTargets() {
		if !isUsableQualifiedAlias(alias) {
			continue
		}
		if byPkg := c.packageGlobals[targetPkg]; byPkg != nil {
			for name, g := range byPkg {
				if g.Public {
					candidates[alias+"."+name] = true
				}
			}
		}
	}
}

func (c *Checker) addQualifiedFunctionCandidates(candidates map[string]bool) {
	for alias, targetPkg := range c.currentImportTargets() {
		if !isUsableQualifiedAlias(alias) {
			continue
		}
		if byPkg := c.packageFunctions[targetPkg]; byPkg != nil {
			for name, sig := range byPkg {
				if sig.Public {
					candidates[alias+"."+name] = true
				}
			}
		}
	}
}

func (c *Checker) addQualifiedTypeCandidates(candidates map[string]bool) {
	for alias, targetPkg := range c.currentImportTargets() {
		if !isUsableQualifiedAlias(alias) {
			continue
		}
		if byPkg := c.packageStructs[targetPkg]; byPkg != nil {
			for name, sig := range byPkg {
				if sig.Public {
					candidates[alias+"."+name] = true
				}
			}
		}
		if byPkg := c.packageInterfaces[targetPkg]; byPkg != nil {
			for name, sig := range byPkg {
				if sig.Public {
					candidates[alias+"."+name] = true
				}
			}
		}
		if byPkg := c.packageEnums[targetPkg]; byPkg != nil {
			for name, sig := range byPkg {
				if sig.Public {
					candidates[alias+"."+name] = true
				}
			}
		}
	}
}

func (c *Checker) currentImportTargets() map[string]string {
	out := map[string]string{}
	if c.currentPackage == "" {
		return out
	}
	byAlias := c.imports[c.currentPackage]
	for alias, binding := range byAlias {
		out[alias] = binding.TargetPackageID
	}
	return out
}

func (c *Checker) canAccessPackage(packageID string, public bool) bool {
	return packageID == "" || packageID == c.currentPackage || public
}

func (c *Checker) resolveImportedPackage(alias string) (string, bool) {
	if c.currentPackage == "" {
		return "", false
	}
	byAlias := c.imports[c.currentPackage]
	if byAlias == nil {
		return "", false
	}
	binding, ok := byAlias[alias]
	return binding.TargetPackageID, ok
}

func (c *Checker) canUseBarePackage(packageID string) bool {
	if packageID == "" || packageID == c.currentPackage {
		return true
	}
	byPkg := c.bareImports[c.currentPackage]
	return byPkg != nil && byPkg[packageID]
}

func (c *Checker) currentBareImportPackageIDs() []string {
	if c.currentPackage == "" {
		return nil
	}
	byPkg := c.bareImports[c.currentPackage]
	if len(byPkg) == 0 {
		return nil
	}
	out := make([]string, 0, len(byPkg))
	for pkgID := range byPkg {
		out = append(out, pkgID)
	}
	return out
}

func (c *Checker) registerStruct(name string, sig StructSig) error {
	if sig.InternalName != "" {
		if existing, exists := c.internalStructs[sig.InternalName]; exists && existing.PackageID != sig.PackageID {
			return fmt.Errorf("duplicate struct '%s'", name)
		}
		c.internalStructs[sig.InternalName] = sig
	}
	if sig.PackageID == "" {
		if _, exists := c.structs[name]; exists {
			return fmt.Errorf("duplicate struct '%s'", name)
		}
		c.structs[name] = sig
		return nil
	}
	byPkg := c.packageStructs[sig.PackageID]
	if byPkg == nil {
		byPkg = map[string]StructSig{}
		c.packageStructs[sig.PackageID] = byPkg
	}
	if _, exists := byPkg[name]; exists {
		return fmt.Errorf("duplicate struct '%s'", name)
	}
	byPkg[name] = sig
	return nil
}

func (c *Checker) registerInterface(name string, sig InterfaceSig) error {
	if sig.InternalName != "" {
		if existing, exists := c.internalInterfaces[sig.InternalName]; exists && existing.PackageID != sig.PackageID {
			return fmt.Errorf("duplicate interface '%s'", name)
		}
		c.internalInterfaces[sig.InternalName] = sig
	}
	if sig.PackageID == "" {
		if _, exists := c.interfaces[name]; exists {
			return fmt.Errorf("duplicate interface '%s'", name)
		}
		c.interfaces[name] = sig
		return nil
	}
	byPkg := c.packageInterfaces[sig.PackageID]
	if byPkg == nil {
		byPkg = map[string]InterfaceSig{}
		c.packageInterfaces[sig.PackageID] = byPkg
	}
	if _, exists := byPkg[name]; exists {
		return fmt.Errorf("duplicate interface '%s'", name)
	}
	byPkg[name] = sig
	return nil
}

func (c *Checker) registerEnum(name string, sig EnumSig) error {
	if sig.InternalName != "" {
		if existing, exists := c.internalEnums[sig.InternalName]; exists && existing.PackageID != sig.PackageID {
			return fmt.Errorf("duplicate enum '%s'", name)
		}
		c.internalEnums[sig.InternalName] = sig
	}
	if sig.PackageID == "" {
		if _, exists := c.enums[name]; exists {
			return fmt.Errorf("duplicate enum '%s'", name)
		}
		c.enums[name] = sig
		return nil
	}
	byPkg := c.packageEnums[sig.PackageID]
	if byPkg == nil {
		byPkg = map[string]EnumSig{}
		c.packageEnums[sig.PackageID] = byPkg
	}
	if _, exists := byPkg[name]; exists {
		return fmt.Errorf("duplicate enum '%s'", name)
	}
	byPkg[name] = sig
	return nil
}

func (c *Checker) resolveStruct(name string) (StructSig, typeLookupStatus) {
	if sig, ok := c.internalStructs[name]; ok {
		return sig, typeLookupFound
	}
	if c.currentPackage != "" {
		if byPkg := c.packageStructs[c.currentPackage]; byPkg != nil {
			if sig, ok := byPkg[name]; ok {
				return sig, typeLookupFound
			}
		}
	}
	if sig, ok := c.structs[name]; ok && sig.PackageID == "" {
		return sig, typeLookupFound
	}
	return c.resolveVisibleImportedStruct(name)
}

func (c *Checker) resolveVisibleImportedStruct(name string) (StructSig, typeLookupStatus) {
	found := StructSig{}
	count := 0
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageStructs[pkgID]
		sig, ok := byPkg[name]
		if !ok || !c.canAccessPackage(sig.PackageID, sig.Public) {
			continue
		}
		found = sig
		count++
		if count > 1 {
			return StructSig{}, typeLookupAmbiguous
		}
	}
	if count == 1 {
		return found, typeLookupFound
	}
	return StructSig{}, typeLookupMissing
}

func (c *Checker) resolveInterface(name string) (InterfaceSig, typeLookupStatus) {
	if sig, ok := c.internalInterfaces[name]; ok {
		return sig, typeLookupFound
	}
	if c.currentPackage != "" {
		if byPkg := c.packageInterfaces[c.currentPackage]; byPkg != nil {
			if sig, ok := byPkg[name]; ok {
				return sig, typeLookupFound
			}
		}
	}
	if sig, ok := c.interfaces[name]; ok && sig.PackageID == "" {
		return sig, typeLookupFound
	}
	return c.resolveVisibleImportedInterface(name)
}

func (c *Checker) resolveVisibleImportedInterface(name string) (InterfaceSig, typeLookupStatus) {
	found := InterfaceSig{}
	count := 0
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageInterfaces[pkgID]
		sig, ok := byPkg[name]
		if !ok || !c.canAccessPackage(sig.PackageID, sig.Public) {
			continue
		}
		found = sig
		count++
		if count > 1 {
			return InterfaceSig{}, typeLookupAmbiguous
		}
	}
	if count == 1 {
		return found, typeLookupFound
	}
	return InterfaceSig{}, typeLookupMissing
}

func (c *Checker) resolveEnum(name string) (EnumSig, typeLookupStatus) {
	if sig, ok := c.internalEnums[name]; ok {
		return sig, typeLookupFound
	}
	if c.currentPackage != "" {
		if byPkg := c.packageEnums[c.currentPackage]; byPkg != nil {
			if sig, ok := byPkg[name]; ok {
				return sig, typeLookupFound
			}
		}
	}
	if sig, ok := c.enums[name]; ok && sig.PackageID == "" {
		return sig, typeLookupFound
	}
	return c.resolveVisibleImportedEnum(name)
}

func (c *Checker) resolveVisibleImportedEnum(name string) (EnumSig, typeLookupStatus) {
	found := EnumSig{}
	count := 0
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageEnums[pkgID]
		sig, ok := byPkg[name]
		if !ok || !c.canAccessPackage(sig.PackageID, sig.Public) {
			continue
		}
		found = sig
		count++
		if count > 1 {
			return EnumSig{}, typeLookupAmbiguous
		}
	}
	if count == 1 {
		return found, typeLookupFound
	}
	return EnumSig{}, typeLookupMissing
}

func (c *Checker) resolveFunction(name string) (FuncSig, typeLookupStatus) {
	if c.currentPackage != "" {
		if byPkg := c.packageFunctions[c.currentPackage]; byPkg != nil {
			if sig, ok := byPkg[name]; ok {
				return sig, typeLookupFound
			}
		}
	}
	if sig, ok := c.functions[name]; ok && sig.PackageID == "" {
		return sig, typeLookupFound
	}
	return c.resolveVisibleImportedFunction(name)
}

func (c *Checker) resolveVisibleImportedFunction(name string) (FuncSig, typeLookupStatus) {
	found := FuncSig{}
	count := 0
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageFunctions[pkgID]
		sig, ok := byPkg[name]
		if !ok || !c.canAccessPackage(sig.PackageID, sig.Public) {
			continue
		}
		found = sig
		count++
		if count > 1 {
			return FuncSig{}, typeLookupAmbiguous
		}
	}
	if count == 1 {
		return found, typeLookupFound
	}
	return FuncSig{}, typeLookupMissing
}

func (c *Checker) resolveHiddenFunction(name string) (FuncSig, bool) {
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageFunctions[pkgID]
		sig, ok := byPkg[name]
		if !ok {
			continue
		}
		if c.canAccessPackage(sig.PackageID, sig.Public) {
			continue
		}
		return sig, true
	}
	return FuncSig{}, false
}

func (c *Checker) resolveGlobalSymbol(name string) (GlobalSymbol, typeLookupStatus) {
	if c.currentPackage != "" {
		if byPkg := c.packageGlobals[c.currentPackage]; byPkg != nil {
			if g, ok := byPkg[name]; ok {
				return g, typeLookupFound
			}
		}
	}
	if g, ok := c.globals[name]; ok && g.PackageID == "" {
		return g, typeLookupFound
	}
	return c.resolveVisibleImportedGlobal(name)
}

func (c *Checker) resolveVisibleImportedGlobal(name string) (GlobalSymbol, typeLookupStatus) {
	found := GlobalSymbol{}
	count := 0
	for _, pkgID := range c.currentBareImportPackageIDs() {
		byPkg := c.packageGlobals[pkgID]
		g, ok := byPkg[name]
		if !ok || !c.canAccessPackage(g.PackageID, g.Public) {
			continue
		}
		found = g
		count++
		if count > 1 {
			return GlobalSymbol{}, typeLookupAmbiguous
		}
	}
	if count == 1 {
		return found, typeLookupFound
	}
	return GlobalSymbol{}, typeLookupMissing
}

func (c *Checker) registerFunction(name string, sig FuncSig) error {
	if sig.PackageID == "" {
		if _, exists := c.functions[name]; exists {
			return fmt.Errorf("duplicate function '%s'", name)
		}
		c.functions[name] = sig
		return nil
	}
	byPkg := c.packageFunctions[sig.PackageID]
	if byPkg == nil {
		byPkg = map[string]FuncSig{}
		c.packageFunctions[sig.PackageID] = byPkg
	}
	if _, exists := byPkg[name]; exists {
		return fmt.Errorf("duplicate function '%s'", name)
	}
	byPkg[name] = sig
	return nil
}

func (c *Checker) registerGlobal(name string, g GlobalSymbol) error {
	if g.PackageID == "" {
		if _, exists := c.globals[name]; exists {
			return fmt.Errorf("duplicate global '%s'", name)
		}
		c.globals[name] = g
		return nil
	}
	byPkg := c.packageGlobals[g.PackageID]
	if byPkg == nil {
		byPkg = map[string]GlobalSymbol{}
		c.packageGlobals[g.PackageID] = byPkg
	}
	if _, exists := byPkg[name]; exists {
		return fmt.Errorf("duplicate global '%s'", name)
	}
	byPkg[name] = g
	return nil
}

func (c *Checker) resolveQualifiedGlobal(alias, name string) (ast.Type, bool, bool) {
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok {
		return ast.TypeInvalid, false, false
	}
	byPkg := c.packageGlobals[targetPkg]
	if byPkg == nil {
		return ast.TypeInvalid, false, false
	}
	g, ok := byPkg[name]
	if !ok || g.PackageID != targetPkg || !g.Public {
		return ast.TypeInvalid, false, false
	}
	return g.Type, g.Const, true
}

func (c *Checker) resolveQualifiedFunction(alias, name string) (FuncSig, bool) {
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok {
		return FuncSig{}, false
	}
	byPkg := c.packageFunctions[targetPkg]
	if byPkg == nil {
		return FuncSig{}, false
	}
	sig, ok := byPkg[name]
	if !ok || sig.PackageID != targetPkg || !sig.Public {
		return FuncSig{}, false
	}
	return sig, true
}

func (c *Checker) resolveQualifiedStruct(alias, name string) (StructSig, bool) {
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok {
		return StructSig{}, false
	}
	byPkg := c.packageStructs[targetPkg]
	if byPkg == nil {
		return StructSig{}, false
	}
	sig, ok := byPkg[name]
	if !ok || sig.PackageID != targetPkg || !sig.Public {
		return StructSig{}, false
	}
	return sig, true
}

func (c *Checker) resolveQualifiedInterface(alias, name string) (InterfaceSig, bool) {
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok {
		return InterfaceSig{}, false
	}
	byPkg := c.packageInterfaces[targetPkg]
	if byPkg == nil {
		return InterfaceSig{}, false
	}
	sig, ok := byPkg[name]
	if !ok || sig.PackageID != targetPkg || !sig.Public {
		return InterfaceSig{}, false
	}
	return sig, true
}

func (c *Checker) resolveQualifiedEnum(alias, name string) (EnumSig, bool) {
	targetPkg, ok := c.resolveImportedPackage(alias)
	if !ok {
		return EnumSig{}, false
	}
	byPkg := c.packageEnums[targetPkg]
	if byPkg == nil {
		return EnumSig{}, false
	}
	sig, ok := byPkg[name]
	if !ok || sig.PackageID != targetPkg || !sig.Public {
		return EnumSig{}, false
	}
	return sig, true
}

func (c *Checker) suggestMethodName(base, name string) string {
	prefix := base + "_"
	candidates := []string{}
	for fn := range c.functions {
		if strings.HasPrefix(fn, prefix) {
			candidates = append(candidates, strings.TrimPrefix(fn, prefix))
		}
	}
	if c.currentPackage != "" {
		if byPkg := c.packageFunctions[c.currentPackage]; byPkg != nil {
			for fn := range byPkg {
				if strings.HasPrefix(fn, prefix) {
					candidates = append(candidates, strings.TrimPrefix(fn, prefix))
				}
			}
		}
		for _, pkgID := range c.currentBareImportPackageIDs() {
			if byPkg := c.packageFunctions[pkgID]; byPkg != nil {
				for fn, sig := range byPkg {
					if sig.Public && strings.HasPrefix(fn, prefix) {
						candidates = append(candidates, strings.TrimPrefix(fn, prefix))
					}
				}
			}
		}
	}
	return suggestNameSuffix(name, candidates)
}

func (c *Checker) isEnumVariantLiteral(name string, g GlobalSymbol) bool {
	if g.Type == ast.TypeInvalid {
		return false
	}
	sig, status := c.resolveEnum(string(g.Type))
	if status != typeLookupFound {
		return false
	}
	return sig.Variants[name]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func suggestNameSuffix(target string, candidates []string) string {
	for _, candidate := range candidates {
		if strings.HasSuffix(candidate, "."+target) {
			return fmt.Sprintf(" (did you mean '%s'?)", candidate)
		}
	}
	best, ok := closestName(target, candidates)
	if !ok {
		return ""
	}
	return fmt.Sprintf(" (did you mean '%s'?)", best)
}

func isUsableQualifiedAlias(alias string) bool {
	if alias == "" {
		return false
	}
	parts := strings.Split(alias, ".")
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i, r := range part {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && r != '_' && (i == 0 || r < '0' || r > '9') {
				return false
			}
		}
	}
	return true
}

func closestName(target string, candidates []string) (string, bool) {
	best := ""
	bestScore := 1 << 30
	for _, c := range candidates {
		if c == "" {
			continue
		}
		score := levenshtein(strings.ToLower(target), strings.ToLower(c))
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(target)) || strings.HasPrefix(strings.ToLower(target), strings.ToLower(c)) {
			score--
		}
		if score < bestScore {
			bestScore = score
			best = c
		}
	}
	if best == "" {
		return "", false
	}
	if bestScore > 2 {
		return "", false
	}
	return best, true
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev = curr
	}
	return prev[len(b)]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func mapKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func firstArgNode(args []ast.Expr) ast.Node {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

func guardOrVariantNode(guard ast.Expr) ast.Node {
	if guard == nil {
		return nil
	}
	return guard
}

func (c *Checker) declare(name string, t ast.Type, isConst bool, span source.Span) error {
	if err := c.scopes.declare(name, t, isConst, span); err != nil {
		return c.diagAt(span, "%s", err.Error())
	}
	return nil
}

func (c *Checker) resolve(name string, markUsed bool) (ast.Type, bool) {
	t, _, ok := c.resolveVar(name, markUsed)
	return t, ok
}

func (c *Checker) resolveVar(name string, markUsed bool) (ast.Type, bool, bool) {
	if t, isConst, ok := c.scopes.resolve(name, markUsed); ok {
		return t, isConst, true
	}
	if g, status := c.resolveGlobalSymbol(name); status == typeLookupFound {
		return g.Type, g.Const, true
	}
	return ast.TypeInvalid, false, false
}

func (c *Checker) pushScope() { c.scopes.push() }

func (c *Checker) popScope() error {
	scope := c.scopes.pop()
	if scope == nil {
		return nil
	}
	if err := c.scopes.validateScopeUsage(scope); err != nil {
		for _, info := range scope {
			if !info.used {
				return c.diagAt(info.declSpan, "%s", err.Error())
			}
		}
	}
	return nil
}

func blockAlwaysReturns(b *ast.BlockStmt) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Stmts {
		if stmtAlwaysReturns(s) {
			return true
		}
	}
	return false
}

func stmtAlwaysReturns(s ast.Stmt) bool {
	switch st := s.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		if st.Else == nil {
			return false
		}
		return blockAlwaysReturns(st.Then) && blockAlwaysReturns(st.Else)
	case *ast.MatchStmt:
		if len(st.Arms) == 0 {
			return false
		}
		for _, arm := range st.Arms {
			if !blockAlwaysReturns(arm.Body) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
