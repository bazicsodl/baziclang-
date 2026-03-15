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
