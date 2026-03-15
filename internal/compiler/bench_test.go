package compiler

import "testing"

const benchSource = `
struct User {
    name: string;
    age: int;
}

enum Status { Ok, Err }

fn label(u: User): string {
    return u.name;
}

fn main(): void {
    let u = User { name: "A", age: 1 };
    let s = Ok;
    match s { Ok: { println(label(u)); } Err: { println("bad"); } }
}
`

func BenchmarkCompileToGoSimple(b *testing.B) {
    for i := 0; i < b.N; i++ {
        if _, err := CompileToGo(benchSource); err != nil {
            b.Fatalf("compile failed: %v", err)
        }
    }
}

const benchGeneric = `
struct Box[T] { value: T; }

fn id[T](v: T): T { return v; }

fn main(): void {
    let b = Box[int] { value: 7 };
    println(id(b.value));
}
`

func BenchmarkCompileToGoGenerics(b *testing.B) {
    for i := 0; i < b.N; i++ {
        if _, err := CompileToGo(benchGeneric); err != nil {
            b.Fatalf("compile failed: %v", err)
        }
    }
}
