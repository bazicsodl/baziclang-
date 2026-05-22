package mir

import (
	"baziclang/internal/source"
	baztypes "baziclang/internal/types"
)

type Node interface {
	Span() source.Span
	node()
}

type NodeInfo struct {
	Range source.Span
}

func (n NodeInfo) Span() source.Span { return n.Range }

type Program struct {
	NodeInfo
	Decls []Decl
}

func (*Program) node() {}

type Decl interface {
	Node
	decl()
}

type ImportDecl struct {
	NodeInfo
	Path          string
	Alias         string
	ExplicitAlias bool
}

func (*ImportDecl) node() {}
func (*ImportDecl) decl() {}

type StructField struct {
	Range source.Span
	Name  string
	Type  baztypes.Type
}

type StructDecl struct {
	NodeInfo
	Name            string
	TypeParams      []string
	TypeParamBounds map[string]baztypes.Type
	Fields          []StructField
}

func (*StructDecl) node() {}
func (*StructDecl) decl() {}

type InterfaceMethod struct {
	Range  source.Span
	Name   string
	Params []Param
	Return baztypes.Type
}

type InterfaceDecl struct {
	NodeInfo
	Name    string
	Methods []InterfaceMethod
}

func (*InterfaceDecl) node() {}
func (*InterfaceDecl) decl() {}

type ImplDecl struct {
	NodeInfo
	StructType    baztypes.Type
	InterfaceName string
}

func (*ImplDecl) node() {}
func (*ImplDecl) decl() {}

type EnumDecl struct {
	NodeInfo
	Name     string
	Variants []string
}

func (*EnumDecl) node() {}
func (*EnumDecl) decl() {}

type Param struct {
	Range source.Span
	Name  string
	Type  baztypes.Type
}

type FuncDecl struct {
	NodeInfo
	Name            string
	TypeParams      []string
	TypeParamBounds map[string]baztypes.Type
	Params          []Param
	ReturnType      baztypes.Type
	Body            *Block
	CFG             *CFG
}

func (*FuncDecl) node() {}
func (*FuncDecl) decl() {}

type GlobalLetDecl struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Init    Expr
	IsConst bool
}

func (*GlobalLetDecl) node() {}
func (*GlobalLetDecl) decl() {}

type Block struct {
	NodeInfo
	Stmts []Stmt
}

func (*Block) node() {}
func (*Block) stmt() {}

type CFG struct {
	Entry  string
	Blocks []*BasicBlock
}

type BasicBlock struct {
	NodeInfo
	Name   string
	Instrs []Stmt
	Term   Terminator
}

type Terminator interface {
	Node
	terminator()
}

type ReturnTerminator struct {
	NodeInfo
	Value Expr
}

func (*ReturnTerminator) node()       {}
func (*ReturnTerminator) terminator() {}

type JumpTerminator struct {
	NodeInfo
	Target string
}

func (*JumpTerminator) node()       {}
func (*JumpTerminator) terminator() {}

type CondTerminator struct {
	NodeInfo
	Cond Expr
	Then string
	Else string
}

func (*CondTerminator) node()       {}
func (*CondTerminator) terminator() {}

type MatchTerminatorArm struct {
	Range   source.Span
	Variant string
	Guard   Expr
	Target  string
}

type MatchTerminator struct {
	NodeInfo
	Subject Expr
	Arms    []MatchTerminatorArm
	Default string
}

func (*MatchTerminator) node()       {}
func (*MatchTerminator) terminator() {}

type Stmt interface {
	Node
	stmt()
}

type LetStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Init    Expr
	IsConst bool
}

func (*LetStmt) node() {}
func (*LetStmt) stmt() {}

type ConstStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Value   Expr
	IsConst bool
}

func (*ConstStmt) node() {}
func (*ConstStmt) stmt() {}

type CopyStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Source  *IdentExpr
	IsConst bool
}

func (*CopyStmt) node() {}
func (*CopyStmt) stmt() {}

type UnaryOpStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Op      string
	Right   Expr
	IsConst bool
}

func (*UnaryOpStmt) node() {}
func (*UnaryOpStmt) stmt() {}

type BinaryOpStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Left    Expr
	Op      string
	Right   Expr
	IsConst bool
}

func (*BinaryOpStmt) node() {}
func (*BinaryOpStmt) stmt() {}

type CallStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Func    string
	Args    []Expr
	IsConst bool
}

func (*CallStmt) node() {}
func (*CallStmt) stmt() {}

type FieldAccessStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Object  Expr
	Field   string
	IsConst bool
}

func (*FieldAccessStmt) node() {}
func (*FieldAccessStmt) stmt() {}

type StructLitStmt struct {
	NodeInfo
	Name     string
	Type     baztypes.Type
	TypeName string
	Fields   []StructLitField
	IsConst  bool
}

func (*StructLitStmt) node() {}
func (*StructLitStmt) stmt() {}

type MatchValueStmt struct {
	NodeInfo
	Name    string
	Type    baztypes.Type
	Subject Expr
	Arms    []MatchExprArm
	IsConst bool
}

func (*MatchValueStmt) node() {}
func (*MatchValueStmt) stmt() {}

type AssignStmt struct {
	NodeInfo
	Target Expr
	Value  Expr
}

func (*AssignStmt) node() {}
func (*AssignStmt) stmt() {}

type IfStmt struct {
	NodeInfo
	Cond Expr
	Then *Block
	Else *Block
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

type WhileStmt struct {
	NodeInfo
	Cond Expr
	Body *Block
}

func (*WhileStmt) node() {}
func (*WhileStmt) stmt() {}

type MatchArm struct {
	Range   source.Span
	Variant string
	Guard   Expr
	Body    *Block
}

type MatchStmt struct {
	NodeInfo
	Subject Expr
	Arms    []MatchArm
}

func (*MatchStmt) node() {}
func (*MatchStmt) stmt() {}

type ReturnStmt struct {
	NodeInfo
	Value Expr
}

func (*ReturnStmt) node() {}
func (*ReturnStmt) stmt() {}

type ExprStmt struct {
	NodeInfo
	Expr Expr
}

func (*ExprStmt) node() {}
func (*ExprStmt) stmt() {}

type Expr interface {
	Node
	expr()
}

type IdentExpr struct {
	NodeInfo
	Name string
}

func (*IdentExpr) node() {}
func (*IdentExpr) expr() {}

type IntExpr struct {
	NodeInfo
	Value int64
}

func (*IntExpr) node() {}
func (*IntExpr) expr() {}

type FloatExpr struct {
	NodeInfo
	Value float64
}

func (*FloatExpr) node() {}
func (*FloatExpr) expr() {}

type BoolExpr struct {
	NodeInfo
	Value bool
}

func (*BoolExpr) node() {}
func (*BoolExpr) expr() {}

type StringExpr struct {
	NodeInfo
	Value string
}

func (*StringExpr) node() {}
func (*StringExpr) expr() {}

type NilExpr struct {
	NodeInfo
}

func (*NilExpr) node() {}
func (*NilExpr) expr() {}

type UnaryExpr struct {
	NodeInfo
	Op    string
	Right Expr
}

func (*UnaryExpr) node() {}
func (*UnaryExpr) expr() {}

type BinaryExpr struct {
	NodeInfo
	Left  Expr
	Op    string
	Right Expr
}

func (*BinaryExpr) node() {}
func (*BinaryExpr) expr() {}

type CallExpr struct {
	NodeInfo
	Func string
	Args []Expr
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}

type FieldAccessExpr struct {
	NodeInfo
	Object Expr
	Field  string
}

func (*FieldAccessExpr) node() {}
func (*FieldAccessExpr) expr() {}

type StructLitField struct {
	Range source.Span
	Name  string
	Value Expr
}

type StructLitExpr struct {
	NodeInfo
	TypeName string
	Fields   []StructLitField
}

func (*StructLitExpr) node() {}
func (*StructLitExpr) expr() {}

type MatchExprArm struct {
	Range   source.Span
	Variant string
	Guard   Expr
	Value   Expr
}

type MatchExpr struct {
	NodeInfo
	Subject Expr
	Arms    []MatchExprArm
	Type    baztypes.Type
}

func (*MatchExpr) node() {}
func (*MatchExpr) expr() {}
