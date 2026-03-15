package parser

import (
	"fmt"
	"strconv"
	"strings"

	"baziclang/internal/ast"
	"baziclang/internal/lexer"
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
	return prog, nil
}

func (p *Parser) parseDecl() (ast.Decl, error) {
	if p.match(lexer.KwImport) {
		return p.parseImportDecl()
	}
	if p.match(lexer.KwStruct) {
		return p.parseStructDecl()
	}
	if p.match(lexer.KwEnum) {
		return p.parseEnumDecl()
	}
	if p.match(lexer.KwInterface) {
		return p.parseInterfaceDecl()
	}
	if p.match(lexer.KwImpl) {
		return p.parseImplDecl()
	}
	if p.match(lexer.KwFn) {
		return p.parseFuncDecl()
	}
	if p.match(lexer.KwConst) {
		return p.parseGlobalLetDecl(true)
	}
	if p.match(lexer.KwLet) {
		return p.parseGlobalLetDecl(false)
	}
	return nil, p.errorAtCurrent("expected declaration: import/struct/enum/interface/impl/fn/let/const")
}

func (p *Parser) parseImportDecl() (ast.Decl, error) {
	tok, err := p.consume(lexer.String, "expected import path string")
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ImportDecl{Path: tok.Lexeme}, nil
}

func (p *Parser) parseStructDecl() (ast.Decl, error) {
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
		fields = append(fields, ast.StructField{Name: fieldName.Lexeme, Type: fieldType})
	}
	if _, err := p.consume(lexer.RBrace, "expected '}' after struct body"); err != nil {
		return nil, err
	}
	return &ast.StructDecl{Name: nameTok.Lexeme, TypeParams: typeParams, TypeParamBounds: bounds, Fields: fields}, nil
}

func (p *Parser) parseEnumDecl() (ast.Decl, error) {
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
	if _, err := p.consume(lexer.RBrace, "expected '}' after enum body"); err != nil {
		return nil, err
	}
	return &ast.EnumDecl{Name: nameTok.Lexeme, Variants: variants}, nil
}

func (p *Parser) parseInterfaceDecl() (ast.Decl, error) {
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
		methods = append(methods, ast.InterfaceMethod{Name: mname.Lexeme, Params: params, Return: ret})
	}
	if _, err := p.consume(lexer.RBrace, "expected '}' after interface body"); err != nil {
		return nil, err
	}
	return &ast.InterfaceDecl{Name: nameTok.Lexeme, Methods: methods}, nil
}

func (p *Parser) parseImplDecl() (ast.Decl, error) {
	st, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(lexer.Colon, "expected ':' in impl declaration"); err != nil {
		return nil, err
	}
	iface, err := p.consume(lexer.Ident, "expected interface name in impl declaration")
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ImplDecl{StructType: st, InterfaceName: iface.Lexeme}, nil
}

func (p *Parser) parseFuncDecl() (ast.Decl, error) {
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
	return &ast.FuncDecl{Name: nameTok.Lexeme, TypeParams: typeParams, TypeParamBounds: bounds, Params: params, ReturnType: ret, Body: body}, nil
}

func (p *Parser) parseGlobalLetDecl(isConst bool) (ast.Decl, error) {
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
	return &ast.GlobalLetDecl{Name: nameTok.Lexeme, Type: typ, Init: init, IsConst: isConst}, nil
}

func (p *Parser) parseBlock() (*ast.BlockStmt, error) {
	if _, err := p.consume(lexer.LBrace, "expected '{' to start block"); err != nil {
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
	if _, err := p.consume(lexer.RBrace, "expected '}' after block"); err != nil {
		return nil, err
	}
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
	return &ast.ExprStmt{Expr: expr}, nil
}

func (p *Parser) parseLetStmt(isConst bool) (ast.Stmt, error) {
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
	return &ast.LetStmt{Name: nameTok.Lexeme, Type: typ, Init: init, IsConst: isConst}, nil
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
	return &ast.AssignStmt{Target: target, Value: value}, nil
}

func (p *Parser) parseAssignTarget() (ast.Expr, error) {
	nameTok, err := p.consume(lexer.Ident, "expected variable name")
	if err != nil {
		return nil, err
	}
	var expr ast.Expr = &ast.IdentExpr{Name: nameTok.Lexeme}
	for p.match(lexer.Dot) {
		field, err := p.consume(lexer.Ident, "expected field name after '.'")
		if err != nil {
			return nil, err
		}
		expr = &ast.FieldAccessExpr{Object: expr, Field: field.Lexeme}
	}
	return expr, nil
}

func (p *Parser) parseIfStmt() (ast.Stmt, error) {
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
					chainedElse = &ast.BlockStmt{Stmts: []ast.Stmt{chainedIf}}
				} else {
					chainedElse, err = p.parseBlock()
					if err != nil {
						return nil, err
					}
				}
			}
			elseBlock = &ast.BlockStmt{Stmts: []ast.Stmt{
				&ast.IfStmt{Cond: cond, Then: thenBlock, Else: chainedElse},
			}}
		} else {
			elseBlock, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
	}
	return &ast.IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}, nil
}

func (p *Parser) parseWhileStmt() (ast.Stmt, error) {
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{Cond: cond, Body: body}, nil
}

func (p *Parser) parseMatchStmt() (ast.Stmt, error) {
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
		variant, err := p.consume(lexer.Ident, "expected enum variant in match arm")
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
		arms = append(arms, ast.MatchArm{Variant: variant.Lexeme, Guard: guard, Body: body})
	}
	if _, err := p.consume(lexer.RBrace, "expected '}' after match statement"); err != nil {
		return nil, err
	}
	return &ast.MatchStmt{Subject: subject, Arms: arms}, nil
}

func (p *Parser) parseMatchExpr() (ast.Expr, error) {
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
		variant, err := p.consume(lexer.Ident, "expected enum variant in match arm")
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
		arms = append(arms, ast.MatchExprArm{Variant: variant.Lexeme, Guard: guard, Value: value})
		p.optionalSemicolons()
		if p.match(lexer.Comma) {
			continue
		}
		if p.check(lexer.Ident) {
			continue
		}
		break
	}
	if _, err := p.consume(lexer.RBrace, "expected '}' after match expression"); err != nil {
		return nil, err
	}
	return &ast.MatchExpr{Subject: subject, Arms: arms, ResolvedType: ast.TypeInvalid}, nil
}

func (p *Parser) parseReturnStmt() (ast.Stmt, error) {
	if p.match(lexer.Semicolon) {
		return &ast.ReturnStmt{}, nil
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.optionalSemicolons()
	return &ast.ReturnStmt{Value: value}, nil
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
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
		expr = &ast.BinaryExpr{Left: expr, Op: op, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.match(lexer.Bang, lexer.Minus) {
		op := p.previous().Lexeme
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Op: op, Right: right}, nil
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
			switch target := expr.(type) {
			case *ast.IdentExpr:
				expr = &ast.CallExpr{Callee: target.Name, Args: args}
			case *ast.FieldAccessExpr:
				expr = &ast.CallExpr{Receiver: target.Object, Method: target.Field, Args: args}
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
			expr = &ast.FieldAccessExpr{Object: expr, Field: field.Lexeme}
			continue
		}
		break
	}
	return expr, nil
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	if p.match(lexer.KwTrue) {
		return &ast.BoolExpr{Value: true}, nil
	}
	if p.match(lexer.KwFalse) {
		return &ast.BoolExpr{Value: false}, nil
	}
	if p.match(lexer.KwNil) {
		return &ast.NilExpr{}, nil
	}
	if p.match(lexer.KwMatch) {
		return p.parseMatchExpr()
	}
	if p.match(lexer.Int) {
		v, err := strconv.ParseInt(p.previous().Lexeme, 10, 64)
		if err != nil {
			return nil, p.errorAtCurrent("invalid integer literal")
		}
		return &ast.IntExpr{Value: v}, nil
	}
	if p.match(lexer.Float) {
		v, err := strconv.ParseFloat(p.previous().Lexeme, 64)
		if err != nil {
			return nil, p.errorAtCurrent("invalid float literal")
		}
		return &ast.FloatExpr{Value: v}, nil
	}
	if p.match(lexer.String) {
		return &ast.StringExpr{Value: p.previous().Lexeme}, nil
	}
	if p.match(lexer.Ident) {
		name := p.previous().Lexeme
		if p.match(lexer.LBracket) {
			args := make([]string, 0, 2)
			for {
				t, err := p.parseType()
				if err != nil {
					return nil, err
				}
				args = append(args, string(t))
				if !p.match(lexer.Comma) {
					break
				}
			}
			if _, err := p.consume(lexer.RBracket, "expected ']' after generic args"); err != nil {
				return nil, err
			}
			name = fmt.Sprintf("%s[%s]", name, strings.Join(args, ","))
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
					fields = append(fields, ast.StructLitField{Name: fName.Lexeme, Value: fVal})
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
			return &ast.StructLitExpr{TypeName: name, Fields: fields}, nil
		}
		if strings.Contains(name, "[") {
			return nil, p.errorAtCurrent("generic type expressions must be struct literals")
		}
		return &ast.IdentExpr{Name: name}, nil
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
	if !p.match(lexer.LBracket) {
		return ast.Type(base.Lexeme), nil
	}
	args := make([]string, 0, 2)
	for {
		t, err := p.parseType()
		if err != nil {
			return ast.TypeInvalid, err
		}
		args = append(args, string(t))
		if !p.match(lexer.Comma) {
			break
		}
	}
	if _, err := p.consume(lexer.RBracket, "expected ']' after generic type args"); err != nil {
		return ast.TypeInvalid, err
	}
	return ast.Type(fmt.Sprintf("%s[%s]", base.Lexeme, strings.Join(args, ","))), nil
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
			boundTok, err := p.consume(lexer.Ident, "expected interface name after ':'")
			if err != nil {
				return nil, nil, err
			}
			bounds[tok.Lexeme] = ast.Type(boundTok.Lexeme)
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
		params = append(params, ast.Param{Name: paramName.Lexeme, Type: paramType})
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
	return fmt.Errorf("parse error at %d:%d: %s (got '%s')", tok.Line, tok.Col, msg, tok.Lexeme)
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
