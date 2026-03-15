package ast

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
}

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
	Decls []Decl
}

func (*Program) node() {}

type ImportDecl struct {
	Path string
}

func (*ImportDecl) node() {}
func (*ImportDecl) decl() {}

type StructField struct {
	Name string
	Type Type
}

type StructDecl struct {
	Name       string
	TypeParams []string
	TypeParamBounds map[string]Type
	Fields     []StructField
}

func (*StructDecl) node() {}
func (*StructDecl) decl() {}

type InterfaceMethod struct {
	Name   string
	Params []Param
	Return Type
}

type InterfaceDecl struct {
	Name    string
	Methods []InterfaceMethod
}

func (*InterfaceDecl) node() {}
func (*InterfaceDecl) decl() {}

type ImplDecl struct {
	StructType    Type
	InterfaceName string
}

func (*ImplDecl) node() {}
func (*ImplDecl) decl() {}

type EnumDecl struct {
	Name     string
	Variants []string
}

func (*EnumDecl) node() {}
func (*EnumDecl) decl() {}

type FuncDecl struct {
	Name       string
	TypeParams []string
	TypeParamBounds map[string]Type
	Params     []Param
	ReturnType Type
	Body       *BlockStmt
}

func (*FuncDecl) node() {}
func (*FuncDecl) decl() {}

type Param struct {
	Name string
	Type Type
}

type GlobalLetDecl struct {
	Name string
	Type Type
	Init Expr
	IsConst bool
}

func (*GlobalLetDecl) node() {}
func (*GlobalLetDecl) decl() {}

type BlockStmt struct {
	Stmts []Stmt
}

func (*BlockStmt) node() {}
func (*BlockStmt) stmt() {}

type LetStmt struct {
	Name string
	Type Type
	Init Expr
	IsConst bool
}

func (*LetStmt) node() {}
func (*LetStmt) stmt() {}

type AssignStmt struct {
	Target Expr
	Value  Expr
}

func (*AssignStmt) node() {}
func (*AssignStmt) stmt() {}

type IfStmt struct {
	Cond Expr
	Then *BlockStmt
	Else *BlockStmt
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
}

func (*WhileStmt) node() {}
func (*WhileStmt) stmt() {}

type MatchArm struct {
	Variant string
	Guard   Expr
	Body    *BlockStmt
}

type MatchStmt struct {
	Subject Expr
	Arms    []MatchArm
}

func (*MatchStmt) node() {}
func (*MatchStmt) stmt() {}

type MatchExprArm struct {
	Variant string
	Guard   Expr
	Value   Expr
}

type MatchExpr struct {
	Subject      Expr
	Arms         []MatchExprArm
	ResolvedType Type
}

func (*MatchExpr) node() {}
func (*MatchExpr) expr() {}

type ReturnStmt struct {
	Value Expr
}

func (*ReturnStmt) node() {}
func (*ReturnStmt) stmt() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) node() {}
func (*ExprStmt) stmt() {}

type IdentExpr struct {
	Name string
}

func (*IdentExpr) node() {}
func (*IdentExpr) expr() {}

type IntExpr struct {
	Value int64
}

func (*IntExpr) node() {}
func (*IntExpr) expr() {}

type FloatExpr struct {
	Value float64
}

func (*FloatExpr) node() {}
func (*FloatExpr) expr() {}

type BoolExpr struct {
	Value bool
}

func (*BoolExpr) node() {}
func (*BoolExpr) expr() {}

type StringExpr struct {
	Value string
}

func (*StringExpr) node() {}
func (*StringExpr) expr() {}

type NilExpr struct{}

func (*NilExpr) node() {}
func (*NilExpr) expr() {}

type UnaryExpr struct {
	Op    string
	Right Expr
}

func (*UnaryExpr) node() {}
func (*UnaryExpr) expr() {}

type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (*BinaryExpr) node() {}
func (*BinaryExpr) expr() {}

type CallExpr struct {
	Callee   string
	Receiver Expr
	Method   string
	Args     []Expr
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}

type FieldAccessExpr struct {
	Object Expr
	Field  string
}

func (*FieldAccessExpr) node() {}
func (*FieldAccessExpr) expr() {}

type StructLitExpr struct {
	TypeName string
	Fields   []StructLitField
}

type StructLitField struct {
	Name  string
	Value Expr
}

func (*StructLitExpr) node() {}
func (*StructLitExpr) expr() {}
