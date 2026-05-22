package lexer

import "testing"

func TestBlockComments(t *testing.T) {
	src := "let x = 1; /* block\ncomment */ let y = 2; // line\nlet z = 3;"
	tokens, err := New(src).Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kinds := []TokenKind{}
	for _, tok := range tokens {
		kinds = append(kinds, tok.Kind)
	}
	want := []TokenKind{KwLet, Ident, Equal, Int, Semicolon, KwLet, Ident, Equal, Int, Semicolon, KwLet, Ident, Equal, Int, Semicolon, EOF}
	if len(kinds) != len(want) {
		t.Fatalf("token count mismatch: got %d want %d", len(kinds), len(want))
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("token %d: got %s want %s", i, kinds[i], want[i])
		}
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	_, err := New("/* no end").Tokenize()
	if err == nil {
		t.Fatalf("expected error for unterminated block comment")
	}
}

func TestStringEscapes(t *testing.T) {
	src := "let s = \"a\\\\b\\n\\t\\r\\\"\";"
	tokens, err := New(src).Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, tok := range tokens {
		if tok.Kind == String {
			found = true
			if tok.Lexeme != "a\\b\n\t\r\"" {
				t.Fatalf("unexpected string value: %q", tok.Lexeme)
			}
		}
	}
	if !found {
		t.Fatalf("expected string token")
	}
}

func TestInvalidStringEscape(t *testing.T) {
	_, err := New("let s = \"\\q\";").Tokenize()
	if err == nil {
		t.Fatalf("expected error for invalid escape")
	}
}

func TestPackageKeyword(t *testing.T) {
	tokens, err := New("package main;").Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 3 {
		t.Fatalf("unexpected token count: %d", len(tokens))
	}
	if tokens[0].Kind != KwPackage {
		t.Fatalf("expected first token to be package keyword, got %s", tokens[0].Kind)
	}
	if tokens[1].Kind != Ident || tokens[1].Lexeme != "main" {
		t.Fatalf("expected package name token, got %s %q", tokens[1].Kind, tokens[1].Lexeme)
	}
}

func TestPubKeyword(t *testing.T) {
	tokens, err := New("pub fn helper(): int { return 1; }").Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 2 || tokens[0].Kind != KwPub || tokens[1].Kind != KwFn {
		t.Fatalf("expected pub fn prefix tokens, got %#v", tokens[:min(2, len(tokens))])
	}
}

func TestImportAliasKeyword(t *testing.T) {
	tokens, err := New("import \"util\" as tools;").Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 4 {
		t.Fatalf("unexpected token count: %d", len(tokens))
	}
	if tokens[0].Kind != KwImport || tokens[1].Kind != String || tokens[2].Kind != KwAs || tokens[3].Kind != Ident {
		t.Fatalf("unexpected import alias tokens: %#v", tokens[:min(4, len(tokens))])
	}
	if tokens[3].Lexeme != "tools" {
		t.Fatalf("expected alias identifier 'tools', got %q", tokens[3].Lexeme)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
