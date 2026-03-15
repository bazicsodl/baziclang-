package lexer

import (
	"fmt"
	"unicode"
)

type TokenKind string

const (
	EOF    TokenKind = "EOF"
	Ident  TokenKind = "IDENT"
	Int    TokenKind = "INT"
	Float  TokenKind = "FLOAT"
	String TokenKind = "STRING"

	KwImport    TokenKind = "import"
	KwStruct    TokenKind = "struct"
	KwEnum      TokenKind = "enum"
	KwInterface TokenKind = "interface"
	KwImpl      TokenKind = "impl"
	KwFn        TokenKind = "fn"
	KwLet       TokenKind = "let"
	KwConst     TokenKind = "const"
	KwIf        TokenKind = "if"
	KwElse      TokenKind = "else"
	KwWhile     TokenKind = "while"
	KwMatch     TokenKind = "match"
	KwReturn    TokenKind = "return"
	KwTrue      TokenKind = "true"
	KwFalse     TokenKind = "false"
	KwNil       TokenKind = "nil"

	LParen    TokenKind = "("
	RParen    TokenKind = ")"
	LBrace    TokenKind = "{"
	RBrace    TokenKind = "}"
	LBracket  TokenKind = "["
	RBracket  TokenKind = "]"
	Comma     TokenKind = ","
	Semicolon TokenKind = ";"
	Colon     TokenKind = ":"
	Dot       TokenKind = "."

	Plus      TokenKind = "+"
	Minus     TokenKind = "-"
	Star      TokenKind = "*"
	Slash     TokenKind = "/"
	Percent   TokenKind = "%"
	Bang      TokenKind = "!"
	Equal     TokenKind = "="
	EqEq      TokenKind = "=="
	NotEq     TokenKind = "!="
	Less      TokenKind = "<"
	LessEq    TokenKind = "<="
	Greater   TokenKind = ">"
	GreaterEq TokenKind = ">="
	AndAnd    TokenKind = "&&"
	OrOr      TokenKind = "||"
)

type Token struct {
	Kind   TokenKind
	Lexeme string
	Line   int
	Col    int
}

type Lexer struct {
	src  []rune
	pos  int
	line int
	col  int
	last TokenKind
	parenDepth   int
	bracketDepth int
}

func New(input string) *Lexer {
	return &Lexer{src: []rune(input), line: 1, col: 1}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	tokens := make([]Token, 0, len(l.src)/2)
	for !l.isAtEnd() {
		ch := l.peek()
		if unicode.IsSpace(ch) {
			sawNewline := l.consumeWhitespace()
			if sawNewline && l.canInsertSemicolon() {
				tokens = append(tokens, Token{Kind: Semicolon, Lexeme: ";", Line: l.line, Col: l.col})
				l.last = Semicolon
				continue
			}
			continue
		}
		if ch == '/' && l.peekNext() == '/' {
			l.consumeLineComment()
			continue
		}
		if ch == '/' && l.peekNext() == '*' {
			if err := l.consumeBlockComment(); err != nil {
				return nil, err
			}
			continue
		}
		startLine, startCol := l.line, l.col
		if unicode.IsLetter(ch) || ch == '_' {
			ident := l.consumeIdent()
			kind := keywordOrIdent(ident)
			tokens = append(tokens, Token{Kind: kind, Lexeme: ident, Line: startLine, Col: startCol})
			l.last = kind
			continue
		}
		if unicode.IsDigit(ch) {
			kind, lit := l.consumeNumber()
			tokens = append(tokens, Token{Kind: kind, Lexeme: lit, Line: startLine, Col: startCol})
			l.last = kind
			continue
		}
		switch ch {
		case '"':
			str, err := l.consumeString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: String, Lexeme: str, Line: startLine, Col: startCol})
			l.last = String
		case '(':
			l.advance()
			tokens = append(tokens, Token{Kind: LParen, Lexeme: "(", Line: startLine, Col: startCol})
			l.last = LParen
			l.parenDepth++
		case ')':
			l.advance()
			tokens = append(tokens, Token{Kind: RParen, Lexeme: ")", Line: startLine, Col: startCol})
			l.last = RParen
			if l.parenDepth > 0 {
				l.parenDepth--
			}
		case '{':
			l.advance()
			tokens = append(tokens, Token{Kind: LBrace, Lexeme: "{", Line: startLine, Col: startCol})
			l.last = LBrace
		case '}':
			l.advance()
			tokens = append(tokens, Token{Kind: RBrace, Lexeme: "}", Line: startLine, Col: startCol})
			l.last = RBrace
		case '[':
			l.advance()
			tokens = append(tokens, Token{Kind: LBracket, Lexeme: "[", Line: startLine, Col: startCol})
			l.last = LBracket
			l.bracketDepth++
		case ']':
			l.advance()
			tokens = append(tokens, Token{Kind: RBracket, Lexeme: "]", Line: startLine, Col: startCol})
			l.last = RBracket
			if l.bracketDepth > 0 {
				l.bracketDepth--
			}
		case ',':
			l.advance()
			tokens = append(tokens, Token{Kind: Comma, Lexeme: ",", Line: startLine, Col: startCol})
			l.last = Comma
		case ';':
			l.advance()
			tokens = append(tokens, Token{Kind: Semicolon, Lexeme: ";", Line: startLine, Col: startCol})
			l.last = Semicolon
		case ':':
			l.advance()
			tokens = append(tokens, Token{Kind: Colon, Lexeme: ":", Line: startLine, Col: startCol})
			l.last = Colon
		case '.':
			l.advance()
			tokens = append(tokens, Token{Kind: Dot, Lexeme: ".", Line: startLine, Col: startCol})
			l.last = Dot
		case '+':
			l.advance()
			tokens = append(tokens, Token{Kind: Plus, Lexeme: "+", Line: startLine, Col: startCol})
			l.last = Plus
		case '-':
			l.advance()
			tokens = append(tokens, Token{Kind: Minus, Lexeme: "-", Line: startLine, Col: startCol})
			l.last = Minus
		case '*':
			l.advance()
			tokens = append(tokens, Token{Kind: Star, Lexeme: "*", Line: startLine, Col: startCol})
			l.last = Star
		case '%':
			l.advance()
			tokens = append(tokens, Token{Kind: Percent, Lexeme: "%", Line: startLine, Col: startCol})
			l.last = Percent
		case '/':
			l.advance()
			tokens = append(tokens, Token{Kind: Slash, Lexeme: "/", Line: startLine, Col: startCol})
			l.last = Slash
		case '!':
			l.advance()
			if l.match('=') {
				tokens = append(tokens, Token{Kind: NotEq, Lexeme: "!=", Line: startLine, Col: startCol})
				l.last = NotEq
			} else {
				tokens = append(tokens, Token{Kind: Bang, Lexeme: "!", Line: startLine, Col: startCol})
				l.last = Bang
			}
		case '=':
			l.advance()
			if l.match('=') {
				tokens = append(tokens, Token{Kind: EqEq, Lexeme: "==", Line: startLine, Col: startCol})
				l.last = EqEq
			} else {
				tokens = append(tokens, Token{Kind: Equal, Lexeme: "=", Line: startLine, Col: startCol})
				l.last = Equal
			}
		case '<':
			l.advance()
			if l.match('=') {
				tokens = append(tokens, Token{Kind: LessEq, Lexeme: "<=", Line: startLine, Col: startCol})
				l.last = LessEq
			} else {
				tokens = append(tokens, Token{Kind: Less, Lexeme: "<", Line: startLine, Col: startCol})
				l.last = Less
			}
		case '>':
			l.advance()
			if l.match('=') {
				tokens = append(tokens, Token{Kind: GreaterEq, Lexeme: ">=", Line: startLine, Col: startCol})
				l.last = GreaterEq
			} else {
				tokens = append(tokens, Token{Kind: Greater, Lexeme: ">", Line: startLine, Col: startCol})
				l.last = Greater
			}
		case '&':
			l.advance()
			if !l.match('&') {
				return nil, l.errAt(startLine, startCol, "expected '&' after '&'")
			}
			tokens = append(tokens, Token{Kind: AndAnd, Lexeme: "&&", Line: startLine, Col: startCol})
			l.last = AndAnd
		case '|':
			l.advance()
			if !l.match('|') {
				return nil, l.errAt(startLine, startCol, "expected '|' after '|'")
			}
			tokens = append(tokens, Token{Kind: OrOr, Lexeme: "||", Line: startLine, Col: startCol})
			l.last = OrOr
		default:
			return nil, l.errAt(startLine, startCol, fmt.Sprintf("unexpected character '%c'", ch))
		}
	}
	if l.canInsertSemicolon() && l.last != Semicolon {
		tokens = append(tokens, Token{Kind: Semicolon, Lexeme: ";", Line: l.line, Col: l.col})
	}
	tokens = append(tokens, Token{Kind: EOF, Lexeme: "", Line: l.line, Col: l.col})
	return tokens, nil
}

func keywordOrIdent(s string) TokenKind {
	switch s {
	case "import":
		return KwImport
	case "struct":
		return KwStruct
	case "enum":
		return KwEnum
	case "interface":
		return KwInterface
	case "impl":
		return KwImpl
	case "fn":
		return KwFn
	case "let":
		return KwLet
	case "const":
		return KwConst
	case "if":
		return KwIf
	case "else":
		return KwElse
	case "while":
		return KwWhile
	case "match":
		return KwMatch
	case "return":
		return KwReturn
	case "true":
		return KwTrue
	case "false":
		return KwFalse
	case "nil":
		return KwNil
	default:
		return Ident
	}
}

func (l *Lexer) isAtEnd() bool { return l.pos >= len(l.src) }
func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	return l.src[l.pos]
}
func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}
func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() || l.src[l.pos] != expected {
		return false
	}
	l.advance()
	return true
}

func (l *Lexer) consumeWhitespace() bool {
	sawNewline := false
	for !l.isAtEnd() && unicode.IsSpace(l.peek()) {
		if l.peek() == '\r' {
			sawNewline = true
			l.advance()
			if l.peek() == '\n' {
				l.advance()
			}
			continue
		}
		if l.peek() == '\n' {
			sawNewline = true
		}
		l.advance()
	}
	return sawNewline
}

func (l *Lexer) consumeLineComment() {
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}
}

func (l *Lexer) canInsertSemicolon() bool {
	if l.parenDepth > 0 || l.bracketDepth > 0 {
		return false
	}
	switch l.last {
	case Ident, Int, Float, String, KwTrue, KwFalse, KwNil, RParen, RBracket, RBrace, KwReturn:
		return true
	default:
		return false
	}
}

func (l *Lexer) consumeBlockComment() error {
	startLine, startCol := l.line, l.col
	l.advance()
	l.advance()
	for !l.isAtEnd() {
		if l.peek() == '*' && l.peekNext() == '/' {
			l.advance()
			l.advance()
			return nil
		}
		l.advance()
	}
	return l.errAt(startLine, startCol, "unterminated block comment")
}

func (l *Lexer) consumeIdent() string {
	start := l.pos
	for !l.isAtEnd() {
		ch := l.peek()
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			l.advance()
			continue
		}
		break
	}
	return string(l.src[start:l.pos])
}

func (l *Lexer) consumeNumber() (TokenKind, string) {
	start := l.pos
	kind := Int
	for !l.isAtEnd() && unicode.IsDigit(l.peek()) {
		l.advance()
	}
	if !l.isAtEnd() && l.peek() == '.' && unicode.IsDigit(l.peekNext()) {
		kind = Float
		l.advance()
		for !l.isAtEnd() && unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}
	return kind, string(l.src[start:l.pos])
}

func (l *Lexer) consumeString() (string, error) {
	startLine, startCol := l.line, l.col
	l.advance()
	var out []rune
	for !l.isAtEnd() {
		ch := l.peek()
		if ch == '\n' {
			return "", l.errAt(startLine, startCol, "unterminated string literal")
		}
		if ch == '"' {
			l.advance()
			return string(out), nil
		}
		if ch == '\\' {
			l.advance()
			if l.isAtEnd() {
				return "", l.errAt(startLine, startCol, "unterminated string literal")
			}
			esc := l.peek()
			switch esc {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case '\\':
				out = append(out, '\\')
			case '"':
				out = append(out, '"')
			default:
				return "", l.errAt(startLine, startCol, fmt.Sprintf("invalid escape '\\%c'", esc))
			}
			l.advance()
			continue
		}
		out = append(out, ch)
		l.advance()
	}
	return "", l.errAt(startLine, startCol, "unterminated string literal")
}

func (l *Lexer) errAt(line, col int, msg string) error {
	return fmt.Errorf("lex error at %d:%d: %s", line, col, msg)
}
