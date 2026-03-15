package lexer

import "testing"

func FuzzTokenize(f *testing.F) {
	f.Add("fn main(): void { println(\"hello\"); }")
	f.Add("struct User { id: int name: string }")
	f.Add("match x { _ => 1 }")
	f.Fuzz(func(t *testing.T, input string) {
		l := New(input)
		_, _ = l.Tokenize()
	})
}
