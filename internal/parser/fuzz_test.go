package parser

import (
	"testing"

	"baziclang/internal/lexer"
)

func FuzzParseProgram(f *testing.F) {
	f.Add("fn main(): void { println(\"hello\"); }")
	f.Add("import \"std\"; fn add(a: int, b: int): int { return a + b; }")
	f.Add("struct Box[T] { value: T } enum Role { Admin, User }")
	f.Fuzz(func(t *testing.T, input string) {
		l := lexer.New(input)
		toks, err := l.Tokenize()
		if err != nil {
			return
		}
		p := New(toks)
		_, _ = p.ParseProgram()
	})
}
