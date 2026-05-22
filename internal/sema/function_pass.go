package sema

import (
	"fmt"

	"baziclang/internal/ast"
)

type funcChecker struct {
	c  *Checker
	fn *ast.FuncDecl
}

func newFuncChecker(c *Checker, fn *ast.FuncDecl) funcChecker {
	return funcChecker{c: c, fn: fn}
}

func (p funcChecker) run() error {
	restore := p.enter()
	defer restore()

	p.c.pushScope()
	for _, param := range p.fn.Params {
		if err := p.c.declare(param.Name, param.Type, false, param.Range); err != nil {
			_ = p.c.popScope()
			return err
		}
	}
	for _, stmt := range p.fn.Body.Stmts {
		if err := p.c.checkStmt(stmt); err != nil {
			_ = p.c.popScope()
			return fmt.Errorf("in function '%s': %w", p.fn.Name, err)
		}
	}
	if p.fn.ReturnType != ast.TypeVoid && !blockAlwaysReturns(p.fn.Body) {
		_ = p.c.popScope()
		return p.c.nodeError(p.fn, "in function '%s': missing return on some control paths", p.fn.Name)
	}
	if err := p.c.popScope(); err != nil {
		return fmt.Errorf("in function '%s': %w", p.fn.Name, err)
	}
	return nil
}

func (p funcChecker) enter() func() {
	prevCurrentFn := p.c.currentFn
	prevCurrentFnDecl := p.c.currentFnDecl
	prevFnTypes := p.c.fnTypes
	prevCurrentPackage := p.c.currentPackage

	p.c.currentFn = p.fn.ReturnType
	p.c.currentFnDecl = p.fn
	p.c.currentPackage = p.fn.PackageID
	p.c.fnTypes = map[string]bool{}
	for _, tp := range p.fn.TypeParams {
		p.c.fnTypes[tp] = true
	}

	return func() {
		p.c.currentFn = prevCurrentFn
		p.c.currentFnDecl = prevCurrentFnDecl
		p.c.currentPackage = prevCurrentPackage
		p.c.fnTypes = prevFnTypes
	}
}
