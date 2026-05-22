package parser

import (
	"fmt"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/diag"
	"baziclang/internal/lexer"
	"baziclang/internal/source"
)

type Parser struct {
	tokens             []lexer.Token
	pos                int
	allowStructLiteral bool
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, allowStructLiteral: true}
}

func (p *Parser) ParseProgram() (*ast.Program, error) {
	prog := &ast.Program{}
	if p.match(lexer.KwPackage) {
		pkg, err := p.parsePackageDecl()
		if err != nil {
			return nil, err
		}
		prog.Package = pkg
	}
	for !p.check(lexer.EOF) {
		for p.match(lexer.Semicolon) {
		}
		if p.check(lexer.EOF) {
			break
		}
		decl, err := p.parseDecl()
		if err != nil {
			return nil, err
		}
		prog.Decls = append(prog.Decls, decl)
	}
	if prog.Package != nil && len(prog.Decls) > 0 {
		prog.NodeInfo = ast.NodeInfo{Range: source.Join(prog.Package.Span(), prog.Decls[len(prog.Decls)-1].Span())}
	} else if prog.Package != nil {
		prog.NodeInfo = ast.NodeInfo{Range: prog.Package.Span()}
	} else if len(prog.Decls) > 0 {
		prog.NodeInfo = ast.NodeInfo{Range: source.Join(prog.Decls[0].Span(), prog.Decls[len(prog.Decls)-1].Span())}
	}
	return prog, nil
}

func (p *Parser) parsePackageDecl() (*ast.PackageDecl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected package name")
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.PackageDecl{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, nameTok.Span)},
		Name:     nameTok.Lexeme,
	}, nil
}

func (p *Parser) parseDecl() (ast.Decl, error) {
	public := p.match(lexer.KwPub)
	if p.match(lexer.KwImport) {
		if public {
			return nil, p.errorAtPrevious("import declarations cannot be public")
		}
		return p.parseImportDecl()
	}
	if p.match(lexer.KwStruct) {
		return p.parseStructDecl(public)
	}
	if p.match(lexer.KwEnum) {
		return p.parseEnumDecl(public)
	}
	if p.match(lexer.KwInterface) {
		return p.parseInterfaceDecl(public)
	}
	if p.match(lexer.KwImpl) {
		if public {
			return nil, p.errorAtPrevious("impl declarations cannot be public")
		}
		return p.parseImplDecl()
	}
	if p.match(lexer.KwFn) {
		return p.parseFuncDecl(public)
	}
	if p.match(lexer.KwConst) {
		return p.parseGlobalLetDecl(public, true)
	}
	if p.match(lexer.KwLet) {
		return p.parseGlobalLetDecl(public, false)
	}
	if public {
		return nil, p.errorAtPrevious("expected declaration after 'pub'")
	}
	return nil, p.errorAtCurrent("expected declaration: import/struct/enum/interface/impl/fn/let/const")
}

func (p *Parser) parseImportDecl() (ast.Decl, error) {
	start := p.previous()
	tok, err := p.consume(lexer.String, "expected import path string")
	if err != nil {
		return nil, err
	}
	alias := ""
	explicitAlias := false
	end := tok.Span
	if p.match(lexer.KwAs) {
		aliasTok, err := p.consume(lexer.Ident, "expected import alias after 'as'")
		if err != nil {
			return nil, err
		}
		alias = aliasTok.Lexeme
		explicitAlias = true
		end = aliasTok.Span
	}
	p.optionalSemicolons()
	return &ast.ImportDecl{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, end)},
		Path:     tok.Lexeme,
		Alias:    alias,
		ExplicitAlias: explicitAlias,
	}, nil
}

func (p *Parser) parseStructDecl(public bool) (ast.Decl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected struct name")
	if err != nil {
		return nil, err
	}
	typeParams, bounds, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LBrace, "expected '{' after struct name"); err != nil {
		return nil, err
	}
	fields := make([]ast.StructField, 0, 4)
	for !p.check(lexer.RBrace) {
		fieldName, err := p.consume(lexer.Ident, "expected struct field name")
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(lexer.Colon, "expected ':' after field name"); err != nil {
			return nil, err
		}
		fieldType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		p.optionalSemicolons()
		fields = append(fields, ast.StructField{
			Range: source.Join(fieldName.Span, p.previous().Span),
			Name:  fieldName.Lexeme,
			Type:  fieldType,
		})
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after struct body")
	if err != nil {
		return nil, err
	}
	return &ast.StructDecl{
		NodeInfo:        ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)},
		Public:          public,
		Name:            nameTok.Lexeme,
		TypeParams:      typeParams,
		TypeParamBounds: bounds,
		Fields:          fields,
	}, nil
}

func (p *Parser) parseEnumDecl(public bool) (ast.Decl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected enum name")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LBrace, "expected '{' after enum name"); err != nil {
		return nil, err
	}
	variants := make([]string, 0, 4)
	for !p.check(lexer.RBrace) {
		v, err := p.consume(lexer.Ident, "expected enum variant")
		if err != nil {
			return nil, err
		}
		variants = append(variants, v.Lexeme)
		if !p.match(lexer.Comma) {
			break
		}
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after enum body")
	if err != nil {
		return nil, err
	}
	return &ast.EnumDecl{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)},
		Public:   public,
		Name:     nameTok.Lexeme,
		Variants: variants,
	}, nil
}

func (p *Parser) parseInterfaceDecl(public bool) (ast.Decl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected interface name")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LBrace, "expected '{' after interface name"); err != nil {
		return nil, err
	}
	methods := make([]ast.InterfaceMethod, 0, 4)
	for !p.check(lexer.RBrace) {
		if _, err := p.consume(lexer.KwFn, "expected 'fn' in interface method"); err != nil {
			return nil, err
		}
		mname, err := p.consume(lexer.Ident, "expected method name")
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(lexer.LParen, "expected '(' after method name"); err != nil {
			return nil, err
		}
		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(lexer.RParen, "expected ')' after method params"); err != nil {
			return nil, err
		}
		ret := ast.TypeVoid
		if p.match(lexer.Colon) {
			ret, err = p.parseType()
			if err != nil {
				return nil, err
			}
		}
		p.optionalSemicolons()
		methods = append(methods, ast.InterfaceMethod{
			Range:  source.Join(mname.Span, p.previous().Span),
			Name:   mname.Lexeme,
			Params: params,
			Return: ret,
		})
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after interface body")
	if err != nil {
		return nil, err
	}
	return &ast.InterfaceDecl{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)},
		Public:   public,
		Name:     nameTok.Lexeme,
		Methods:  methods,
	}, nil
}

func (p *Parser) parseImplDecl() (ast.Decl, error) {
	start := p.previous()
	st, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.Colon, "expected ':' in impl declaration"); err != nil {
		return nil, err
	}
	iface, err := p.parseType()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ImplDecl{
		NodeInfo:      ast.NodeInfo{Range: source.Join(start.Span, p.previous().Span)},
		StructType:    st,
		InterfaceName: string(iface),
	}, nil
}

func (p *Parser) parseFuncDecl(public bool) (ast.Decl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected function name")
	if err != nil {
		return nil, err
	}
	typeParams, bounds, err := p.parseTypeParams()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LParen, "expected '(' after function name"); err != nil {
		return nil, err
	}
	params, err := p.parseParams()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.RParen, "expected ')' after parameters"); err != nil {
		return nil, err
	}
	ret := ast.TypeVoid
	if p.match(lexer.Colon) {
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		ret = t
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.FuncDecl{
		NodeInfo:        ast.NodeInfo{Range: source.Join(start.Span, body.Span())},
		Public:          public,
		Name:            nameTok.Lexeme,
		TypeParams:      typeParams,
		TypeParamBounds: bounds,
		Params:          params,
		ReturnType:      ret,
		Body:            body,
	}, nil
}

func (p *Parser) parseGlobalLetDecl(public bool, isConst bool) (ast.Decl, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected variable name")
	if err != nil {
		return nil, err
	}
	var typ ast.Type = ast.TypeInvalid
	if p.match(lexer.Colon) {
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		typ = t
	}
	if _, err := p.consume(lexer.Equal, "expected '=' in variable declaration"); err != nil {
		return nil, err
	}
	init, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.GlobalLetDecl{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, init.Span())},
		Public:   public,
		Name:     nameTok.Lexeme,
		Type:     typ,
		Init:     init,
		IsConst:  isConst,
	}, nil
}

func (p *Parser) parseBlock() (*ast.BlockStmt, error) {
	start, err := p.consume(lexer.LBrace, "expected '{' to start block")
	if err != nil {
		return nil, err
	}
	block := &ast.BlockStmt{}
	for !p.check(lexer.RBrace) && !p.check(lexer.EOF) {
		for p.match(lexer.Semicolon) {
		}
		if p.check(lexer.RBrace) || p.check(lexer.EOF) {
			break
		}
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		block.Stmts = append(block.Stmts, stmt)
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after block")
	if err != nil {
		return nil, err
	}
	block.NodeInfo = ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)}
	return block, nil
}

func (p *Parser) parseStmt() (ast.Stmt, error) {
	if p.match(lexer.KwLet) {
		return p.parseLetStmt(false)
	}
	if p.match(lexer.KwConst) {
		return p.parseLetStmt(true)
	}
	if p.match(lexer.KwIf) {
		return p.parseIfStmt()
	}
	if p.match(lexer.KwWhile) {
		return p.parseWhileStmt()
	}
	if p.match(lexer.KwMatch) {
		return p.parseMatchStmt()
	}
	if p.match(lexer.KwReturn) {
		return p.parseReturnStmt()
	}
	if p.isAssignStart() {
		return p.parseAssignStmt()
	}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ExprStmt{
		NodeInfo: ast.NodeInfo{Range: expr.Span()},
		Expr:     expr,
	}, nil
}

func (p *Parser) parseLetStmt(isConst bool) (ast.Stmt, error) {
	start := p.previous()
	nameTok, err := p.consume(lexer.Ident, "expected variable name")
	if err != nil {
		return nil, err
	}
	var typ ast.Type = ast.TypeInvalid
	if p.match(lexer.Colon) {
		t, err := p.parseType()
		if err != nil {
			return nil, err
		}
		typ = t
	}
	if _, err := p.consume(lexer.Equal, "expected '=' in variable declaration"); err != nil {
		return nil, err
	}
	init, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.LetStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, init.Span())},
		Name:     nameTok.Lexeme,
		Type:     typ,
		Init:     init,
		IsConst:  isConst,
	}, nil
}

func (p *Parser) parseAssignStmt() (ast.Stmt, error) {
	target, err := p.parseAssignTarget()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.Equal, "expected '=' in assignment"); err != nil {
		return nil, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.AssignStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(target.Span(), value.Span())},
		Target:   target,
		Value:    value,
	}, nil
}

func (p *Parser) parseAssignTarget() (ast.Expr, error) {
	nameTok, err := p.consume(lexer.Ident, "expected variable name")
	if err != nil {
		return nil, err
	}
	var expr ast.Expr = &ast.IdentExpr{
		NodeInfo: ast.NodeInfo{Range: nameTok.Span},
		Name:     nameTok.Lexeme,
	}
	for p.match(lexer.Dot) {
		field, err := p.consume(lexer.Ident, "expected field name after '.'")
		if err != nil {
			return nil, err
		}
		expr = &ast.FieldAccessExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), field.Span)},
			Object:   expr,
			Field:    field.Lexeme,
		}
	}
	return expr, nil
}

func (p *Parser) parseIfStmt() (ast.Stmt, error) {
	start := p.previous()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	var elseBlock *ast.BlockStmt
	p.optionalSemicolons()
	if p.match(lexer.KwElse) {
		if p.match(lexer.KwIf) {
			cond, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			thenBlock, err := p.parseBlock()
			if err != nil {
				return nil, err
			}
			var chainedElse *ast.BlockStmt
			p.optionalSemicolons()
			if p.match(lexer.KwElse) {
				if p.match(lexer.KwIf) {
					chainedIf, err := p.parseIfStmt()
					if err != nil {
						return nil, err
					}
					chainedElse = &ast.BlockStmt{
						NodeInfo: ast.NodeInfo{Range: chainedIf.Span()},
						Stmts:    []ast.Stmt{chainedIf},
					}
				} else {
					chainedElse, err = p.parseBlock()
					if err != nil {
						return nil, err
					}
				}
			}
			nestedIf := &ast.IfStmt{
				NodeInfo: ast.NodeInfo{Range: source.Join(cond.Span(), blockSpan(thenBlock, chainedElse))},
				Cond:     cond,
				Then:     thenBlock,
				Else:     chainedElse,
			}
			elseBlock = &ast.BlockStmt{
				NodeInfo: ast.NodeInfo{Range: nestedIf.Span()},
				Stmts:    []ast.Stmt{nestedIf},
			}
		} else {
			elseBlock, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
	}
	return &ast.IfStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, blockSpan(thenBlock, elseBlock))},
		Cond:     cond,
		Then:     thenBlock,
		Else:     elseBlock,
	}, nil
}

func (p *Parser) parseWhileStmt() (ast.Stmt, error) {
	start := p.previous()
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, body.Span())},
		Cond:     cond,
		Body:     body,
	}, nil
}

func (p *Parser) parseMatchStmt() (ast.Stmt, error) {
	start := p.previous()
	prevAllowStructLiteral := p.allowStructLiteral
	p.allowStructLiteral = false
	subject, err := p.parseExpr()
	p.allowStructLiteral = prevAllowStructLiteral
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LBrace, "expected '{' after match subject"); err != nil {
		return nil, err
	}
	arms := make([]ast.MatchArm, 0, 4)
	for !p.check(lexer.RBrace) && !p.check(lexer.EOF) {
		p.optionalSemicolons()
		if p.check(lexer.RBrace) {
			break
		}
		variantTok, err := p.consume(lexer.Ident, "expected enum variant in match arm")
		if err != nil {
			return nil, err
		}
		variant, variantSpan, err := p.parseQualifiedName(variantTok, "variant name")
		if err != nil {
			return nil, err
		}
		var guard ast.Expr
		if p.match(lexer.KwIf) {
			g, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			guard = g
		}
		if _, err := p.consume(lexer.Colon, "expected ':' after match arm variant"); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		armEnd := body.Span()
		if guard != nil {
			armEnd = body.Span()
		}
		arms = append(arms, ast.MatchArm{
			Range:   source.Join(variantSpan, armEnd),
			Variant: variant,
			Guard:   guard,
			Body:    body,
		})
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after match statement")
	if err != nil {
		return nil, err
	}
	return &ast.MatchStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)},
		Subject:  subject,
		Arms:     arms,
	}, nil
}

func (p *Parser) parseMatchExpr() (ast.Expr, error) {
	start := p.previous()
	prevAllowStructLiteral := p.allowStructLiteral
	p.allowStructLiteral = false
	subject, err := p.parseExpr()
	p.allowStructLiteral = prevAllowStructLiteral
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.LBrace, "expected '{' after match subject"); err != nil {
		return nil, err
	}
	arms := make([]ast.MatchExprArm, 0, 4)
	for !p.check(lexer.RBrace) && !p.check(lexer.EOF) {
		p.optionalSemicolons()
		if p.check(lexer.RBrace) {
			break
		}
		variantTok, err := p.consume(lexer.Ident, "expected enum variant in match arm")
		if err != nil {
			return nil, err
		}
		variant, variantSpan, err := p.parseQualifiedName(variantTok, "variant name")
		if err != nil {
			return nil, err
		}
		var guard ast.Expr
		if p.match(lexer.KwIf) {
			g, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			guard = g
		}
		if _, err := p.consume(lexer.Colon, "expected ':' after match arm variant"); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		arms = append(arms, ast.MatchExprArm{
			Range:   source.Join(variantSpan, value.Span()),
			Variant: variant,
			Guard:   guard,
			Value:   value,
		})
		p.optionalSemicolons()
		if p.match(lexer.Comma) {
			continue
		}
		if p.check(lexer.Ident) {
			continue
		}
		break
	}
	endTok, err := p.consume(lexer.RBrace, "expected '}' after match expression")
	if err != nil {
		return nil, err
	}
	return &ast.MatchExpr{
		NodeInfo:     ast.NodeInfo{Range: source.Join(start.Span, endTok.Span)},
		Subject:      subject,
		Arms:         arms,
		ResolvedType: ast.TypeInvalid,
	}, nil
}

func (p *Parser) parseReturnStmt() (ast.Stmt, error) {
	start := p.previous()
	if p.match(lexer.Semicolon) {
		return &ast.ReturnStmt{NodeInfo: ast.NodeInfo{Range: start.Span}}, nil
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ReturnStmt{
		NodeInfo: ast.NodeInfo{Range: source.Join(start.Span, value.Span())},
		Value:    value,
	}, nil
}

func (p *Parser) parseExpr() (ast.Expr, error) { return p.parseOr() }

func (p *Parser) parseOr() (ast.Expr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.OrOr) {
		op := p.previous().Lexeme
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseAnd() (ast.Expr, error) {
	expr, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.AndAnd) {
		op := p.previous().Lexeme
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseEquality() (ast.Expr, error) {
	expr, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.EqEq, lexer.NotEq) {
		op := p.previous().Lexeme
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseComparison() (ast.Expr, error) {
	expr, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.Less, lexer.LessEq, lexer.Greater, lexer.GreaterEq) {
		op := p.previous().Lexeme
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseTerm() (ast.Expr, error) {
	expr, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.Plus, lexer.Minus) {
		op := p.previous().Lexeme
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseFactor() (ast.Expr, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.match(lexer.Star, lexer.Slash, lexer.Percent) {
		op := p.previous().Lexeme
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		expr = &ast.BinaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), right.Span())},
			Left:     expr,
			Op:       op,
			Right:    right,
		}
	}
	return expr, nil
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.match(lexer.Bang, lexer.Minus) {
		opTok := p.previous()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{
			NodeInfo: ast.NodeInfo{Range: source.Join(opTok.Span, right.Span())},
			Op:       opTok.Lexeme,
			Right:    right,
		}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(lexer.LParen) {
			args := []ast.Expr{}
			if !p.check(lexer.RParen) {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if !p.match(lexer.Comma) {
						break
					}
				}
			}
			if _, err := p.consume(lexer.RParen, "expected ')' after arguments"); err != nil {
				return nil, err
			}
			endTok := p.previous()
			switch target := expr.(type) {
			case *ast.IdentExpr:
				expr = &ast.CallExpr{
					NodeInfo: ast.NodeInfo{Range: source.Join(target.Span(), endTok.Span)},
					Callee:   target.Name,
					Args:     args,
				}
			case *ast.FieldAccessExpr:
				expr = &ast.CallExpr{
					NodeInfo: ast.NodeInfo{Range: source.Join(target.Span(), endTok.Span)},
					Receiver: target.Object,
					Method:   target.Field,
					Args:     args,
				}
			default:
				return nil, p.errorAtCurrent("only functions or methods can be called")
			}
			continue
		}
		if p.match(lexer.Dot) {
			field, err := p.consume(lexer.Ident, "expected field name after '.'")
			if err != nil {
				return nil, err
			}
			expr = &ast.FieldAccessExpr{
				NodeInfo: ast.NodeInfo{Range: source.Join(expr.Span(), field.Span)},
				Object:   expr,
				Field:    field.Lexeme,
			}
			continue
		}
		break
	}
	return expr, nil
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	if p.match(lexer.KwTrue) {
		return &ast.BoolExpr{NodeInfo: ast.NodeInfo{Range: p.previous().Span}, Value: true}, nil
	}
	if p.match(lexer.KwFalse) {
		return &ast.BoolExpr{NodeInfo: ast.NodeInfo{Range: p.previous().Span}, Value: false}, nil
	}
	if p.match(lexer.KwNil) {
		return &ast.NilExpr{NodeInfo: ast.NodeInfo{Range: p.previous().Span}}, nil
	}
	if p.match(lexer.KwMatch) {
		return p.parseMatchExpr()
	}
	if p.match(lexer.Int) {
		tok := p.previous()
		v, err := strconv.ParseInt(tok.Lexeme, 10, 64)
		if err != nil {
			return nil, p.errorAtCurrent("invalid integer literal")
		}
		return &ast.IntExpr{NodeInfo: ast.NodeInfo{Range: tok.Span}, Value: v}, nil
	}
	if p.match(lexer.Float) {
		tok := p.previous()
		v, err := strconv.ParseFloat(tok.Lexeme, 64)
		if err != nil {
			return nil, p.errorAtCurrent("invalid float literal")
		}
		return &ast.FloatExpr{NodeInfo: ast.NodeInfo{Range: tok.Span}, Value: v}, nil
	}
	if p.match(lexer.String) {
		tok := p.previous()
		return &ast.StringExpr{NodeInfo: ast.NodeInfo{Range: tok.Span}, Value: tok.Lexeme}, nil
	}
	if p.match(lexer.Ident) {
		nameTok := p.previous()
		name, nameSpan, qualified, err := p.parseQualifiedTypeName(nameTok)
		if err != nil {
			return nil, err
		}
		if p.allowStructLiteral && p.check(lexer.LBrace) && p.looksLikeStructLiteral() {
			p.pos++
			fields := make([]ast.StructLitField, 0, 4)
			if !p.check(lexer.RBrace) {
				for {
					fName, err := p.consume(lexer.Ident, "expected struct field name")
					if err != nil {
						return nil, err
					}
					if _, err := p.consume(lexer.Colon, "expected ':' in struct literal"); err != nil {
						return nil, err
					}
					fVal, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					fields = append(fields, ast.StructLitField{
						Range: source.Join(fName.Span, fVal.Span()),
						Name:  fName.Lexeme,
						Value: fVal,
					})
					p.optionalSemicolons()
					if p.match(lexer.Comma) || p.match(lexer.Semicolon) {
						continue
					}
					if p.check(lexer.Ident) {
						continue
					}
					break
				}
			}
			if _, err := p.consume(lexer.RBrace, "expected '}' after struct literal"); err != nil {
				return nil, err
			}
			return &ast.StructLitExpr{
				NodeInfo: ast.NodeInfo{Range: source.Join(nameSpan, p.previous().Span)},
				TypeName: name,
				Fields:   fields,
			}, nil
		}
		if strings.Contains(name, "[") {
			return nil, p.errorAtCurrent("generic type expressions must be struct literals")
		}
		if qualified {
			return p.buildQualifiedExpr(name, nameSpan), nil
		}
		return &ast.IdentExpr{NodeInfo: ast.NodeInfo{Range: nameSpan}, Name: name}, nil
	}
	if p.match(lexer.LParen) {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(lexer.RParen, "expected ')' after expression"); err != nil {
			return nil, err
		}
		return expr, nil
	}
	return nil, p.errorAtCurrent("expected expression")
}

func (p *Parser) parseType() (ast.Type, error) {
	base, err := p.consume(lexer.Ident, "expected type name")
	if err != nil {
		return ast.TypeInvalid, err
	}
	name, _, _, err := p.parseQualifiedTypeName(base)
	if err != nil {
		return ast.TypeInvalid, err
	}
	return ast.Type(name), nil
}

func (p *Parser) parseTypeParams() ([]string, map[string]ast.Type, error) {
	typeParams := []string{}
	bounds := map[string]ast.Type{}
	if !p.match(lexer.LBracket) {
		return typeParams, bounds, nil
	}
	for {
		tok, err := p.consume(lexer.Ident, "expected type parameter")
		if err != nil {
			return nil, nil, err
		}
		if p.match(lexer.Colon) {
			boundType, err := p.parseType()
			if err != nil {
				return nil, nil, err
			}
			bounds[tok.Lexeme] = boundType
		}
		typeParams = append(typeParams, tok.Lexeme)
		if !p.match(lexer.Comma) {
			break
		}
	}
	if _, err := p.consume(lexer.RBracket, "expected ']' after type params"); err != nil {
		return nil, nil, err
	}
	return typeParams, bounds, nil
}

func (p *Parser) parseParams() ([]ast.Param, error) {
	params := []ast.Param{}
	if p.check(lexer.RParen) {
		return params, nil
	}
	for {
		paramName, err := p.consume(lexer.Ident, "expected parameter name")
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(lexer.Colon, "expected ':' after parameter name"); err != nil {
			return nil, err
		}
		paramType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		params = append(params, ast.Param{
			Range: source.Join(paramName.Span, p.previous().Span),
			Name:  paramName.Lexeme,
			Type:  paramType,
		})
		if !p.match(lexer.Comma) {
			break
		}
	}
	return params, nil
}

func (p *Parser) match(kinds ...lexer.TokenKind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.pos++
			return true
		}
	}
	return false
}

func (p *Parser) check(kind lexer.TokenKind) bool {
	if p.isAtEnd() {
		return kind == lexer.EOF
	}
	return p.peek().Kind == kind
}

func (p *Parser) consume(kind lexer.TokenKind, msg string) (lexer.Token, error) {
	if p.check(kind) {
		p.pos++
		return p.previous(), nil
	}
	return lexer.Token{}, p.errorAtCurrent(msg)
}

func (p *Parser) isAtEnd() bool     { return p.peek().Kind == lexer.EOF }
func (p *Parser) peek() lexer.Token { return p.tokens[p.pos] }
func (p *Parser) peekNext() lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos+1]
}
func (p *Parser) previous() lexer.Token { return p.tokens[p.pos-1] }

func (p *Parser) errorAtCurrent(msg string) error {
	tok := p.peek()
	return diag.New("parse error", fmt.Sprintf("%s (got '%s')", msg, tok.Lexeme), tok.Span)
}

func (p *Parser) errorAtPrevious(msg string) error {
	tok := p.previous()
	return diag.New("parse error", fmt.Sprintf("%s (got '%s')", msg, tok.Lexeme), tok.Span)
}

func (p *Parser) optionalSemicolons() {
	for p.match(lexer.Semicolon) {
	}
}

func (p *Parser) looksLikeStructLiteral() bool {
	if !p.check(lexer.LBrace) || p.pos+1 >= len(p.tokens) {
		return false
	}
	next := p.tokens[p.pos+1]
	if next.Kind == lexer.RBrace {
		return true
	}
	if next.Kind != lexer.Ident || p.pos+2 >= len(p.tokens) {
		return false
	}
	return p.tokens[p.pos+2].Kind == lexer.Colon
}

func (p *Parser) isAssignStart() bool {
	if !p.check(lexer.Ident) {
		return false
	}
	i := p.pos + 1
	for i < len(p.tokens) && p.tokens[i].Kind == lexer.Dot {
		i++
		if i >= len(p.tokens) || p.tokens[i].Kind != lexer.Ident {
			return false
		}
		i++
	}
	if i >= len(p.tokens) {
		return false
	}
	return p.tokens[i].Kind == lexer.Equal
}

func blockSpan(thenBlock, elseBlock *ast.BlockStmt) source.Span {
	if elseBlock != nil {
		return elseBlock.Span()
	}
	return thenBlock.Span()
}

func (p *Parser) parseQualifiedTypeName(first lexer.Token) (string, source.Span, bool, error) {
	name := first.Lexeme
	nameSpan := first.Span
	qualified := false
	for p.match(lexer.Dot) {
		part, err := p.consume(lexer.Ident, "expected type name after '.'")
		if err != nil {
			return "", source.Span{}, false, err
		}
		qualified = true
		name += "." + part.Lexeme
		nameSpan = source.Join(nameSpan, part.Span)
	}
	if p.match(lexer.LBracket) {
		args := make([]string, 0, 2)
		for {
			t, err := p.parseType()
			if err != nil {
				return "", source.Span{}, false, err
			}
			args = append(args, string(t))
			if !p.match(lexer.Comma) {
				break
			}
		}
		if _, err := p.consume(lexer.RBracket, "expected ']' after generic type args"); err != nil {
			return "", source.Span{}, false, err
		}
		name = fmt.Sprintf("%s[%s]", name, strings.Join(args, ","))
		nameSpan = source.Join(nameSpan, p.previous().Span)
	}
	return name, nameSpan, qualified, nil
}

func (p *Parser) parseQualifiedName(first lexer.Token, label string) (string, source.Span, error) {
	name := first.Lexeme
	nameSpan := first.Span
	for p.match(lexer.Dot) {
		part, err := p.consume(lexer.Ident, "expected "+label+" after '.'")
		if err != nil {
			return "", source.Span{}, err
		}
		name += "." + part.Lexeme
		nameSpan = source.Join(nameSpan, part.Span)
	}
	return name, nameSpan, nil
}

func (p *Parser) buildQualifiedExpr(name string, span source.Span) ast.Expr {
	parts := strings.Split(name, ".")
	expr := ast.Expr(&ast.IdentExpr{NodeInfo: ast.NodeInfo{Range: span}, Name: parts[0]})
	for _, part := range parts[1:] {
		expr = &ast.FieldAccessExpr{
			NodeInfo: ast.NodeInfo{Range: span},
			Object:   expr,
			Field:    part,
		}
	}
	return expr
}
