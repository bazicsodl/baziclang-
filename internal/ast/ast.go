package ast

import "baziclang/internal/source"

type Type string

const (
	TypeInvalid Type = "invalid"
	TypeAny     Type = "any"
	TypeVoid    Type = "void"
	TypeInt     Type = "int"
	TypeFloat   Type = "float"
	TypeBool    Type = "bool"
	TypeString  Type = "string"
)

type Node interface {
	node()
	Span() source.Span
}

type NodeInfo struct {
	Range source.Span
}

func (n NodeInfo) Span() source.Span { return n.Range }

type Decl interface {
	Node
	decl()
}

type Stmt interface {
	Node
	stmt()
}

type Expr interface {
	Node
	expr()
}

type Program struct {
	NodeInfo
	Imports []ImportRef
	Package *PackageDecl
	Decls   []Decl
}

func (*Program) node() {}

type ImportRef struct {
	OwnerPackageID  string
	Alias           string
	Path            string
	TargetPackageID string
	ExplicitAlias   bool
	BareAllowed     bool
}

type PackageDecl struct {
	NodeInfo
	Name string
}

func (*PackageDecl) node() {}

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
	Type  Type
}

type StructDecl struct {
	NodeInfo
	Public          bool
	PackageID       string
	Name            string
	InternalName    string
	TypeParams      []string
	TypeParamBounds map[string]Type
	Fields          []StructField
}

func (*StructDecl) node() {}
func (*StructDecl) decl() {}

type InterfaceMethod struct {
	Range  source.Span
	Name   string
	Params []Param
	Return Type
}

type InterfaceDecl struct {
	NodeInfo
	Public  bool
	PackageID string
	Name    string
	InternalName string
	Methods []InterfaceMethod
}

func (*InterfaceDecl) node() {}
func (*InterfaceDecl) decl() {}

type ImplDecl struct {
	NodeInfo
	PackageID     string
	StructType    Type
	InterfaceName string
}

func (*ImplDecl) node() {}
func (*ImplDecl) decl() {}

type EnumDecl struct {
	NodeInfo
	Public    bool
	PackageID string
	Name     string
	InternalName string
	Variants []string
}

func (*EnumDecl) node() {}
func (*EnumDecl) decl() {}

type FuncDecl struct {
	NodeInfo
	Public          bool
	PackageID       string
	Name            string
	InternalName    string
	TypeParams      []string
	TypeParamBounds map[string]Type
	Params          []Param
	ReturnType      Type
	Body            *BlockStmt
}

func (*FuncDecl) node() {}
func (*FuncDecl) decl() {}

type Param struct {
	Range source.Span
	Name  string
	Type  Type
}

type GlobalLetDecl struct {
	NodeInfo
	Public bool
	PackageID string
	Name    string
	InternalName string
	Type    Type
	Init    Expr
	IsConst bool
}

func (*GlobalLetDecl) node() {}
func (*GlobalLetDecl) decl() {}

type BlockStmt struct {
	NodeInfo
	Stmts []Stmt
}

func (*BlockStmt) node() {}
func (*BlockStmt) stmt() {}

type LetStmt struct {
	NodeInfo
	Name    string
	Type    Type
	Init    Expr
	IsConst bool
}

func (*LetStmt) node() {}
func (*LetStmt) stmt() {}

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
	Then *BlockStmt
	Else *BlockStmt
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

type WhileStmt struct {
	NodeInfo
	Cond Expr
	Body *BlockStmt
}

func (*WhileStmt) node() {}
func (*WhileStmt) stmt() {}

type MatchArm struct {
	Range   source.Span
	Variant string
	Guard   Expr
	Body    *BlockStmt
}

type MatchStmt struct {
	NodeInfo
	Subject Expr
	Arms    []MatchArm
}

func (*MatchStmt) node() {}
func (*MatchStmt) stmt() {}

type MatchExprArm struct {
	Range   source.Span
	Variant string
	Guard   Expr
	Value   Expr
}

type MatchExpr struct {
	NodeInfo
	Subject      Expr
	Arms         []MatchExprArm
	ResolvedType Type
}

func (*MatchExpr) node() {}
func (*MatchExpr) expr() {}

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

type IdentExpr struct {
	NodeInfo
	Name     string
	Resolved string
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
	Callee   string
	Receiver Expr
	Method   string
	Args     []Expr
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}

type FieldAccessExpr struct {
	NodeInfo
	Object         Expr
	Field          string
	ResolvedGlobal string
}

func (*FieldAccessExpr) node() {}
func (*FieldAccessExpr) expr() {}

type StructLitExpr struct {
	NodeInfo
	TypeName string
	Fields   []StructLitField
}

type StructLitField struct {
	Range source.Span
	Name  string
	Value Expr
}

func (*StructLitExpr) node() {}
func (*StructLitExpr) expr() {}
