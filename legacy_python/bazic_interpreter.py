"""
BAZIC-Lang Refactored Interpreter
Single-file prototype (lexer, parser, AST, evaluator, REPL)
Added: arrays, object literals, property access and indexing
Module loader (Node-style): import(path) -> exports object
"""

import sys
import re
import json
from pathlib import Path
from collections import namedtuple
from dataclasses import dataclass
from typing import Any, List, Tuple, Optional

# -------------------------
# Tokens / Lexer
# -------------------------
Token = namedtuple("Token", ["type", "val", "pos"])  # pos = (line, col)

TOKEN_SPEC = [
    ("NUMBER",   r"\d+(?:\.\d+)?"),
    ("STRING",   r'"([^"\\]|\\.)*"'),
    ("IDENT",    r"[A-Za-z_][A-Za-z0-9_]*"),
    ("NEWLINE",  r"\n"),
    ("SKIP",     r"[ \t]+"),
    ("COMMENT",  r"//.*"),

    # MULTI-CHAR OPERATORS
    ("OP_MULTI", r"<=|>=|==|!=|&&|\|\||\+\+|--"),

    # SINGLE-CHAR OPERATORS (SAFE)
    ("OP_SINGLE", r"[=+\-*/%<>]"),

    ("LPAREN",   r"\("),
    ("RPAREN",   r"\)"),
    ("LBRACE",   r"\{"),
    ("RBRACE",   r"\}"),
    ("LBRACK",   r"\["),
    ("RBRACK",   r"\]"),
    ("COMMA",    r","),
    ("SEMICOLON",r";"),
    ("COLON",    r":"),
    ("DOT",      r"\."),
]

TOK_REGEX = re.compile("|".join(f"(?P<{t}>{p})" for t, p in TOKEN_SPEC))
KEYWORDS = {"let","fn","if","else","while","return","true","false","null","export"}

class LexerError(Exception): pass


def lex(src: str) -> List[Token]:
    tokens = []
    line = 1
    col = 1
    pos = 0
    L = len(src)
    while pos < L:
        m = TOK_REGEX.match(src, pos)
        if not m:
            raise LexerError(f"Unexpected character {src[pos]!r} at line {line} col {col}")
        kind = m.lastgroup
        txt = m.group(kind)
        if kind == 'NEWLINE':
            tokens.append(Token('NEWLINE','\n',(line,col)))
            line += 1
            col = 1
            pos = m.end()
            continue
        if kind == 'SKIP' or kind == 'COMMENT':
            # update column and continue
            lines = txt.count('\n')
            if lines:
                line += lines
                col = 1
            else:
                col += len(txt)
            pos = m.end()
            continue
        if kind == 'IDENT' and txt in KEYWORDS:
            ttype = txt.upper()
            tokens.append(Token(ttype, txt, (line,col)))
        elif kind == 'NUMBER':
            val = float(txt) if '.' in txt else int(txt)
            tokens.append(Token('NUMBER', val, (line,col)))
        elif kind == 'STRING':
            inner = txt[1:-1]
            inner = inner.encode('utf-8').decode('unicode_escape')
            tokens.append(Token('STRING', inner, (line,col)))
        elif kind in ('OP_MULTI', 'OP_SINGLE'):
            tokens.append(Token('OP', txt, (line,col)))
        else:
            tokens.append(Token(kind, txt, (line,col)))
        col += len(txt)
        pos = m.end()
    tokens.append(Token('EOF', '', (line,col)))
    return tokens

# -------------------------
# AST node dataclasses
# -------------------------
@dataclass
class Node: pass

@dataclass
class Program(Node):
    body: List[Node]

@dataclass
class Let(Node):
    name: str
    expr: Node

@dataclass
class Return(Node):
    expr: Node

@dataclass
class ExprStmt(Node):
    expr: Node

@dataclass
class Block(Node):
    stmts: List[Node]

@dataclass
class If(Node):
    cond: Node
    thenb: Block
    elseb: Optional[Block]

@dataclass
class While(Node):
    cond: Node
    body: Block

@dataclass
class Number(Node):
    val: Any

@dataclass
class String(Node):
    val: str

@dataclass
class Bool(Node):
    val: bool

@dataclass
class Null(Node):
    pass

@dataclass
class Var(Node):
    name: str

@dataclass
class Assign(Node):
    name: str
    expr: Node

@dataclass
class Call(Node):
    name: str
    args: List[Node]

@dataclass
class Unary(Node):
    op: str
    rhs: Node

@dataclass
class Binary(Node):
    op: str
    left: Node
    right: Node

@dataclass
class FuncDef(Node):
    name: Optional[str]
    params: List[str]
    body: Block

@dataclass
class Closure(Node):
    params: List[str]
    body: Block
    env: Any

# New AST types
@dataclass
class ArrayLiteral(Node):
    elems: List[Node]

@dataclass
class ObjectLiteral(Node):
    pairs: List[Tuple[Any, Node]]  # keys (str or Node), value expr

@dataclass
class Index(Node):
    target: Node
    index: Node

@dataclass
class PropertyAccess(Node):
    target: Node
    prop: str

@dataclass
class AssignProp(Node):
    target: Node
    prop: str
    expr: Node

@dataclass
class AssignIndex(Node):
    target: Node
    index: Node
    expr: Node

# Export AST node (for "export name = expr" and "export fn name(...) {...}")
@dataclass
class Export(Node):
    name: str
    expr: Node

# -------------------------
# Parser — recursive descent
# -------------------------
class ParserError(Exception): pass

class Parser:
    def __init__(self, tokens: List[Token]):
        self.toks = tokens
        self.i = 0

    def peek(self) -> Token:
        return self.toks[self.i]

    def peek_n(self, n: int) -> Token:
        idx = self.i + n
        if idx < len(self.toks):
            return self.toks[idx]
        return Token('EOF','',(0,0))

    def next(self) -> Token:
        t = self.peek(); self.i += 1; return t

    def accept(self, *types) -> Optional[Token]:
        if self.peek().type in types:
            return self.next()
        return None

    def expect(self, ttype) -> Token:
        t = self.next()
        if t.type != ttype:
            raise ParserError(f"Expected {ttype}, got {t.type} at {t.pos}")
        return t

    def parse_program(self) -> Program:
        stmts = []
        while self.peek().type != 'EOF':
            if self.peek().type == 'NEWLINE':
                self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next()
        return Program(stmts)

    def parse_stmt(self) -> Node:
        t = self.peek()
        if t.type == 'LET':
            return self.parse_let()
        if t.type == 'FN':
            return self.parse_fndef()
        if t.type == 'IF':
            return self.parse_if()
        if t.type == 'WHILE':
            return self.parse_while()
        if t.type == 'RETURN':
            self.next(); expr = self.parse_expr(); return Return(expr)
        if t.type == 'EXPORT':
            return self.parse_export()
        # expression statement
        expr = self.parse_expr(); return ExprStmt(expr)

    def parse_block(self) -> Block:
        self.expect('LBRACE')
        stmts = []
        while self.peek().type != 'RBRACE':
            if self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next()
        self.expect('RBRACE')
        return Block(stmts)
    
    def parse_let(self) -> Let:
        self.expect('LET')
        name = self.expect('IDENT').val

        # look for '='
        if self.accept('OP') and self.toks[self.i-1].val == '=':
            expr = self.parse_expr()
        else:
            expr = Null()
        return Let(name, expr)


    def parse_fndef(self) -> FuncDef:
        self.expect('FN')
        name = None
        if self.peek().type == 'IDENT':
            name = self.next().val
        self.expect('LPAREN')
        params = []
        if self.peek().type != 'RPAREN':
            while True:
                params.append(self.expect('IDENT').val)
                if not self.accept('COMMA'):
                    break
        self.expect('RPAREN')
        body = self.parse_block()
        return FuncDef(name, params, body)

    def parse_export(self) -> Export:
        # export fn name(...) { ... }
        self.expect('EXPORT')
        if self.peek().type == 'FN':
            f = self.parse_fndef()
            if not f.name:
                raise ParserError("exported function must have a name")
            # return Export where expr is the FuncDef itself
            return Export(f.name, f)
        # export IDENT = expr
        if self.peek().type == 'IDENT':
            name = self.next().val
            if self.peek().type == 'OP' and self.peek().val == '=':
                self.next()
                expr = self.parse_expr()
                return Export(name, expr)
            raise ParserError("Expected '=' after export name")
        # could support `export { a, b }` later
        raise ParserError("Unsupported export form")

    def parse_if(self) -> If:
        self.expect('IF')
        self.expect('LPAREN')
        cond = self.parse_expr()
        self.expect('RPAREN')
        # skip separators
        while self.peek().type in ('NEWLINE','SEMICOLON'):
            self.next()
        thenb = self.parse_block()
        elseb = None
        if self.accept('ELSE'):
            while self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next()
            if self.peek().type == 'IF':
                # else if -> nested if wrapped as block
                elseb = Block([self.parse_if()])
            else:
                elseb = self.parse_block()
        return If(cond, thenb, elseb)

    def parse_while(self) -> While:
        self.expect('WHILE')
        self.expect('LPAREN')
        cond = self.parse_expr()
        self.expect('RPAREN')
        while self.peek().type in ('NEWLINE','SEMICOLON'):
            self.next()
        body = self.parse_block()
        return While(cond, body)

    # Expressions: precedence climbing via functions
    def parse_expr(self) -> Node:
        return self.parse_or()

    def parse_or(self):
        node = self.parse_and()
        while self.accept('OP') and self.toks[self.i-1].val == '||':
            node = Binary('||', node, self.parse_and())
        return node

    def parse_and(self):
        node = self.parse_equality()
        while self.accept('OP') and self.toks[self.i-1].val == '&&':
            node = Binary('&&', node, self.parse_equality())
        return node

    def parse_equality(self):
        node = self.parse_rel()
        while self.peek().type == 'OP' and self.peek().val in ('==', '!='):
            op = self.next().val
            node = Binary(op, node, self.parse_rel())
        return node

    def parse_rel(self):
        node = self.parse_add()
        while self.peek().type == 'OP' and self.peek().val in ('<','>','<=','>='):
            op = self.next().val
            node = Binary(op, node, self.parse_add())
        return node
   
    def parse_add(self):
        node = self.parse_mul()
        while self.peek().type == 'OP' and self.peek().val in ('+','-'):
            op = self.next().val
            node = Binary(op, node, self.parse_mul())
        return node

    def parse_mul(self):
        node = self.parse_unary()
        while self.peek().type == 'OP' and self.peek().val in ('*','/','%'):
            op = self.next().val
            node = Binary(op, node, self.parse_unary())
        return node

    def parse_unary(self):
        if self.accept('OP') and self.toks[self.i-1].val == '-':
            return Unary('-', self.parse_unary())
        # could add `!`/`not` later
        return self.parse_primary()
    
    def parse_primary(self):
        t = self.peek()

        # literals
        if t.type == 'NUMBER':
            self.next()
            node = Number(t.val)
        elif t.type == 'STRING':
            self.next()
            node = String(t.val)
        elif t.type == 'TRUE':
            self.next()
            node = Bool(True)
        elif t.type == 'FALSE':
            self.next()
            node = Bool(False)
        elif t.type == 'NULL':
            self.next()
            node = Null()

        # array literal
        elif t.type == 'LBRACK':
            self.next()
            elems = []
            if self.peek().type != 'RBRACK':
                while True:
                    elems.append(self.parse_expr())
                    if not self.accept('COMMA'):
                        break
            self.expect('RBRACK')
            node = ArrayLiteral(elems)

        # object literal OR block
        elif t.type == 'LBRACE':
            # object literal lookahead
            if self.peek_n(1).type in ('IDENT','STRING') and self.peek_n(2).type == 'COLON':
                self.next()
                pairs = []
                if self.peek().type != 'RBRACE':
                    while True:
                        keytok = self.next()
                        if keytok.type in ('STRING','IDENT'):
                            key = keytok.val
                        else:
                            raise ParserError("Invalid object key")
                        self.expect('COLON')
                        val = self.parse_expr()
                        pairs.append((key, val))
                        if not self.accept('COMMA'):
                            break
                self.expect('RBRACE')
                node = ObjectLiteral(pairs)
            else:
                node = self.parse_block()

        # identifiers / calls / assignment
        elif t.type == 'IDENT':
            name = self.next().val

            # assignment
            if self.peek().type == 'OP' and self.peek().val == '=':
                self.next()
                return Assign(name, self.parse_expr())

            # call
            if self.peek().type == 'LPAREN':
                self.next()
                args = []
                if self.peek().type != 'RPAREN':
                    while True:
                        args.append(self.parse_expr())
                        if not self.accept('COMMA'):
                            break
                self.expect('RPAREN')
                node = Call(name, args)
            else:
                node = Var(name)

        # (expr)
        elif t.type == 'LPAREN':
            self.next()
            node = self.parse_expr()
            self.expect('RPAREN')

        else:
            raise ParserError(f"Unexpected token {t.type}")

        # POSTFIX ACCESS
        while True:
            if self.accept('DOT'):
                prop = self.expect('IDENT').val
                node = PropertyAccess(node, prop)
                continue
            if self.accept('LBRACK'):
                idx = self.parse_expr()
                self.expect('RBRACK')
                node = Index(node, idx)
                continue
            break

        # POSTFIX ASSIGNMENT
        if self.peek().type == 'OP' and self.peek().val == '=':
            self.next()
            value = self.parse_expr()

            if isinstance(node, Var):
                return Assign(node.name, value)
            if isinstance(node, PropertyAccess):
                return AssignProp(node.target, node.prop, value)
            if isinstance(node, Index):
                return AssignIndex(node.target, node.index, value)

            raise ParserError("Invalid assignment target")

        return node

# -------------------------
# Runtime / evaluator
# -------------------------
class RuntimeError_(Exception): pass

class ReturnSignal(Exception):
    def __init__(self, value):
        self.value = value

class Env:
    def __init__(self, parent=None):
        self.map = {}
        self.parent = parent
    def define(self, name, val): self.map[name] = val
    def get(self, name):
        if name in self.map: return self.map[name]
        if self.parent: return self.parent.get(name)
        raise RuntimeError_(f"Undefined variable '{name}'")
    def set(self, name, val):
        if name in self.map:
            self.map[name] = val; return
        if self.parent:
            self.parent.set(name, val); return
        raise RuntimeError_(f"Undefined variable '{name}'")

# Helper

def is_truthy(v):
    return not (v is None or v is False)

# Builtin helper wrapper
@dataclass
class Builtin:
    fn: Any
    name: str

# -------------------------
# MODULE LOADER (Option A)
# -------------------------
MODULE_CACHE = {}  # absolute_path -> exports dict

def _resolve_module_path(spec: str, importer: Optional[str] = None) -> Path:
    """
    Resolve module spec to an absolute Path.
    If spec is relative (starts with '.'), resolve against importer (a file path), else against CWD.
    Add .baz extension if missing.
    """
    spec = str(spec)
    p = Path(spec)
    if not p.suffix:
        p = p.with_suffix('.baz')
    if p.is_absolute():
        return p.resolve()
    if spec.startswith('.'):
        base = Path(importer).parent if importer else Path('.').resolve()
        return (base / p).resolve()
    # treat as relative to cwd or allow a modules directory later
    return Path(spec).resolve()

def load_module_by_path(path: str, importer: Optional[str] = None):
    """
    Load a module from a resolved filesystem path. Returns an exports dict.
    Uses MODULE_CACHE to avoid re-evaluation.
    """
    path_obj = _resolve_module_path(path, importer)
    key = str(path_obj)
    if key in MODULE_CACHE:
        return MODULE_CACHE[key]

    if not path_obj.exists():
        raise FileNotFoundError(f"Module not found: {path_obj}")

    src = path_obj.read_text(encoding='utf-8')

    # create fresh module env (no parent)
    module_env = Env(parent=None)
    # load standard library into module_env
    load_std(module_env)

    # provide a place to collect exports
    module_env.map['__exports__'] = {}

    # ensure module-level convenience functions
    module_env.define('print', Builtin(lambda *a: print(*a), 'print'))
    module_env.define('input', Builtin(lambda msg="": input(msg), 'input'))

    # Evaluate module
    toks = lex(src)
    p = Parser(toks)
    ast = p.parse_program()
    eval_node(ast, module_env)

    exports = module_env.map.get('__exports__', {})
    # cache
    MODULE_CACHE[key] = exports
    return exports

# -------------------------
# Standard library loader
# -------------------------
def load_std(env: Env):
    import json as _json
    import time as _time
    import math as _math
    import hashlib as _hashlib
    import copy as _copy
    from pathlib import Path as _Path
    import os as _os

    # CORE
    core = {
        "len": Builtin(lambda x: len(x), "core.len"),
        "type": Builtin(lambda x: str(type(x).__name__), "core.type"),
        "range": Builtin(lambda n: list(range(int(n))), "core.range"),
    }

    # MATH
    math_obj = {
        "pi": _math.pi,
        "abs": Builtin(lambda x: abs(x), "math.abs"),
        "sqrt": Builtin(lambda x: x**0.5, "math.sqrt"),
        "floor": Builtin(lambda x: int(x // 1), "math.floor"),
        "ceil": Builtin(lambda x: int(-(-x // 1)), "math.ceil"),
        "random": Builtin(lambda: __import__("random").random(), "math.random"),
        "sin": Builtin(lambda x: _math.sin(x), "math.sin"),
        "cos": Builtin(lambda x: _math.cos(x), "math.cos"),
        "tan": Builtin(lambda x: _math.tan(x), "math.tan"),
        "pow": Builtin(lambda a,b: a**b, "math.pow"),
        "round": Builtin(lambda x: round(x), "math.round"),
    }

    # STRING
    string_obj = {
        "lower": Builtin(lambda s: str(s).lower(), "string.lower"),
        "upper": Builtin(lambda s: str(s).upper(), "string.upper"),
        "trim": Builtin(lambda s: str(s).strip(), "string.trim"),
        "split": Builtin(lambda s, sep: str(s).split(sep), "string.split"),
        "replace": Builtin(lambda s, a, b: str(s).replace(a, b), "string.replace"),
        "starts_with": Builtin(lambda s, x: str(s).startswith(x), "string.starts_with"),
        "ends_with": Builtin(lambda s, x: str(s).endswith(x), "string.ends_with"),
        "startsWith": Builtin(lambda s,p: str(s).startswith(p), "string.startsWith"),
        "endsWith": Builtin(lambda s,p: str(s).endswith(p), "string.endsWith"),
    }

    # ARRAY
    array_obj = {
        "push": Builtin(lambda a, v: (a.append(v) or a), "array.push"),
        "pop": Builtin(lambda a: a.pop(), "array.pop"),
        "join": Builtin(lambda a, sep: sep.join(str(x) for x in a), "array.join"),
        "map": Builtin(lambda arr, fn: [fn(x) if callable(fn) else None for x in arr], "array.map"),
        "filter": Builtin(lambda arr, fn: [x for x in arr if fn(x)], "array.filter"),
        "find": Builtin(lambda arr, fn: next((x for x in arr if fn(x)), None), "array.find"),
        "includes": Builtin(lambda a, v: v in a, "array.includes"),
        # reduce implemented straightforwardly
        "reduce": Builtin(
            lambda arr, fn, init=None: (
                (lambda acc: ( (lambda _acc: [ (_acc := fn(_acc, x)) for x in arr ] or _acc ) (init if init is not None else arr[0]) ))
            )(None),
            "array.reduce"
        )
    }

    # JSON
    json_obj = {
        "stringify": Builtin(lambda x: _json.dumps(x), "json.stringify"),
        "parse": Builtin(lambda s: _json.loads(s), "json.parse"),
    }

    # TIME
    time_obj = {
        "now": Builtin(lambda: int(_time.time()), "time.now"),
        "sleep": Builtin(lambda s: _time.sleep(float(s)), "time.sleep"),
        "format": Builtin(lambda ts: _time.strftime("%Y-%m-%dT%H:%M:%S", _time.localtime(int(ts))), "time.format"),
    }

    # CONSOLE / DEBUG
    console = {"log": Builtin(lambda *a: print(*a), "console.log")}
    debug = {"dump": Builtin(lambda x: print("DEBUG:", repr(x)), "debug.dump")}

    # SAFE FILESYSTEM (limited/sandboxed-ish)
    def _safe_read(p):
        p = str(p)
        if ".." in p: raise Exception("unsafe path")
        return _Path(p).read_text(encoding='utf-8')
    def _safe_write(p, d):
        p = str(p)
        if ".." in p: raise Exception("unsafe path")
        _Path(p).write_text(str(d), encoding='utf-8'); return True
    fs = {
        "read": Builtin(_safe_read, "fs.read"),
        "write": Builtin(_safe_write, "fs.write"),
        "exists": Builtin(lambda p: _Path(str(p)).exists(), "fs.exists"),
        "list": Builtin(lambda p=".": [str(x) for x in _Path(str(p)).iterdir()], "fs.list"),
    }

    # CRYPTO
    crypto = {
        "sha256": Builtin(lambda s: _hashlib.sha256(str(s).encode('utf-8')).hexdigest(), "crypto.sha256"),
    }

    # UTILS
    utils = {
        "clone": Builtin(lambda x: _copy.copy(x), "utils.clone"),
        "deepclone": Builtin(lambda x: _copy.deepcopy(x), "utils.deepclone"),
        "assert": Builtin(lambda cond, msg="Assertion failed": None if cond else (_ for _ in ()).throw(Exception(msg)), "utils.assert"),
    }

    # IMPORT builtin (Node-style)
    def _builtin_import(path_str):
        # resolve relative to cwd (or let load_module_by_path handle importer)
        return load_module_by_path(str(path_str), importer=str(Path('.').resolve()))

    # Build std and register
    std = {
        "core": core,
        "math": math_obj,
        "string": string_obj,
        "array": array_obj,
        "json": json_obj,
        "time": time_obj,
        "console": console,
        "debug": debug,
        "fs": fs,
        "crypto": crypto,
        "utils": utils,
        "import": Builtin(lambda p: _builtin_import(p), "import"),
    }

    env.define("std", std)

    # Optional shortcuts (also safe to re-define later)
    env.define("print", Builtin(lambda *a: print(*a), "print"))
    env.define("input", Builtin(lambda msg="": input(msg), "input"))
    env.define("eval", Builtin(lambda x: eval_text(x, env), "eval"))

# -------------------------
# Evaluate
# -------------------------
def eval_text(expr, env):
    # wrap into a program so parser accepts it
    src = expr + "\n"
    toks = lex(src)
    p = Parser(toks)
    ast = p.parse_program()
    return eval_node(ast, env)

def eval_node(node: Node, env: Env):
    if isinstance(node, Program):
        out = None
        for s in node.body:
            out = eval_node(s, env)
        return out

    if isinstance(node, Let):
        val = eval_node(node.expr, env)
        env.define(node.name, val)
        return val

    if isinstance(node, FuncDef):
        closure = ("closure", node.params, node.body, env)
        if node.name:
            env.define(node.name, closure)
        return closure

    if isinstance(node, Block):
        local = Env(parent=env)
        out = None
        for s in node.stmts:
            out = eval_node(s, local)
        return out

    if isinstance(node, If):
        if is_truthy(eval_node(node.cond, env)):
            return eval_node(node.thenb, env)
        if node.elseb:
            return eval_node(node.elseb, env)
        return None

    if isinstance(node, While):
        last = None
        while is_truthy(eval_node(node.cond, env)):
            last = eval_node(node.body, env)
        return last

    if isinstance(node, Return):
        val = eval_node(node.expr, env)
        raise ReturnSignal(val)

    if isinstance(node, ExprStmt):
        return eval_node(node.expr, env)

    if isinstance(node, Number):
        return node.val
    if isinstance(node, String):
        return node.val
    if isinstance(node, Bool):
        return node.val
    if isinstance(node, Null):
        return None
    if isinstance(node, Var):
        return env.get(node.name)
    if isinstance(node, Assign):
        val = eval_node(node.expr, env)
        env.set(node.name, val)
        return val

    if isinstance(node, ArrayLiteral):
        return [eval_node(e, env) for e in node.elems]
    
    if isinstance(node, ObjectLiteral):
        out = {}
        for key, vexpr in node.pairs:
            out[key] = eval_node(vexpr, env)
        return out

    if isinstance(node, Export):
        # evaluate export expression, register in both env and module exports
        val = None
        # If expr is a FuncDef, evaluate it to create closure and define
        if isinstance(node.expr, FuncDef):
            val = eval_node(node.expr, env)
            # The function was already defined into env (FuncDef sets env.define if name present)
        else:
            val = eval_node(node.expr, env)
            env.define(node.name, val)

        # register in exports if module exports present
        # Prefer direct map access to avoid raising if __exports__ not defined
        if '__exports__' in env.map:
            env.map['__exports__'][node.name] = val
        else:
            # try to find in parent chain
            cur = env
            while cur:
                if '__exports__' in cur.map:
                    cur.map['__exports__'][node.name] = val
                    break
                cur = cur.parent
        return val

    if isinstance(node, AssignProp):
        obj = eval_node(node.target, env)
        val = eval_node(node.expr, env)

        if not isinstance(obj, dict):
            raise RuntimeError_("Cannot set property on non-object")

        obj[node.prop] = val
        return val

    if isinstance(node, AssignIndex):
        obj = eval_node(node.target, env)
        idx = eval_node(node.index, env)
        val = eval_node(node.expr, env)

        try:
            obj[idx] = val
        except Exception as e:
            raise RuntimeError_(f"Index assignment error: {e}")

        return val

    if isinstance(node, Index):
        targ = eval_node(node.target, env)
        idx = eval_node(node.index, env)
        try:
            return targ[idx]
        except Exception as e:
            raise RuntimeError_(f"Indexing error: {e}")

    if isinstance(node, PropertyAccess):
        targ = eval_node(node.target, env)
        prop = node.prop
        # support dicts and objects
        if isinstance(targ, dict):
            if prop in targ:
                return targ[prop]
            raise RuntimeError_(f"Property '{prop}' not found")
        # for tuples representing closures, allow access? not supported
        raise RuntimeError_(f"Property access not supported on type {type(targ)}")

    # ---- CALL ----
    if isinstance(node, Call):
        # Special builtin: import("path")
        if node.name == "import":
            if len(node.args) != 1:
                raise RuntimeError_("import() takes exactly 1 argument")
            path = eval_node(node.args[0], env)
            return load_module_by_path(path)

        # Special builtin: print(...)
        if node.name == "print":
            vals = [eval_node(a, env) for a in node.args]
            print(*vals)
            return None

        # Fetch function
        callee = env.get(node.name)

        # Closure call
        if isinstance(callee, tuple) and callee[0] == "closure":
            _, params, body, closure_env = callee

            if len(params) != len(node.args):
                raise RuntimeError_("Argument count mismatch")

            local = Env(parent=closure_env)
            for p, a in zip(params, node.args):
                local.define(p, eval_node(a, env))

            try:
                eval_node(body, local)
                return None
            except ReturnSignal as r:
                return r.value

        # Builtin function call
        if isinstance(callee, Builtin):
            args = [eval_node(a, env) for a in node.args]
            return callee.fn(*args)

        # Not a function
        raise RuntimeError_(f"Not a function: {node.name}")

    if isinstance(node, Unary):
        if node.op == '-':
            return -eval_node(node.rhs, env)
        raise RuntimeError_(f"Unknown unary op {node.op}")

    if isinstance(node, Binary):
        a = eval_node(node.left, env)
        b = eval_node(node.right, env)
        op = node.op
        # protect against division by zero
        if op == '/' and b == 0:
            raise RuntimeError_("Division by zero")
        if op == '%' and b == 0:
            raise RuntimeError_("Modulo by zero")
        if op == '+': return a + b
        if op == '-': return a - b
        if op == '*': return a * b
        if op == '/': return a / b
        if op == '%': return a % b
        if op == '==': return a == b
        if op == '!=': return a != b
        if op == '<': return a < b
        if op == '>': return a > b
        if op == '<=': return a <= b
        if op == '>=': return a >= b
        if op == '&&': return a if not is_truthy(a) else b
        if op == '||': return a if is_truthy(a) else b
        raise RuntimeError_(f"Unknown binary op {op}")

    raise RuntimeError_(f"Unhandled AST node: {node}")

# -------------------------
# Top-level helpers / REPL
# -------------------------

SAMPLE = r"""
print("\033[96m                                      ==               ")
print("                                ==          ==          ")
print("                             ==                ==       ")
print("                          ==                      ==    ")
print("                       ==                            == \033[0m")

print("\033[93m                            WELCOME TO BAZIC-LANG\033[0m")

print("\033[96m                       ==                            ==    ")
print("                          ==                      ==       ")
print("                             ==                ==          ")
print("                                ==          ==             ")
print("                                      ==                 \033[0m")

print("\033[92m                     ====================================\033[0m")
print(" ")
print("\033[95m                       A lightweight scripting language\033[0m")
print(" ")
print("\033[92m                     ====================================\033[0m")
print(" ")
print("\033[94m                               Author: Churchill\033[0m")
print(" ")
print("\033[92m                     ====================================\033[0m")
print(" ")
print("\033[93m                              Interpreter: BAZIC\033[0m")
print(" ")
print("\033[92m                     ====================================\033[0m")
print(" ")

"""

def run(src: str, env: Env):
    toks = lex(src)
    p = Parser(toks)
    ast = p.parse_program()
    return eval_node(ast, env)

def repl():
    print("BAZIC-Lang REPL. :quit to exit")
    env = Env()
    env = Env()
    load_std(env)
    # register builtin print as a wrapper
    env.define('print', Builtin(lambda *a: print(*a), 'print'))
    env.define('input', Builtin(lambda msg="": input(msg), 'input'))
    env.define('eval', Builtin(lambda x: eval_text(x, env), 'eval'))
    while True:
        try:
            src = input('bazic> ')
        except (KeyboardInterrupt, EOFError):
            print(); break
        if src.strip() == ':quit': break
        if src.strip() == '': continue
        # allow multiline blocks
        while src.count('{') > src.count('}'):
            more = input('... ')
            src += '\n' + more
        try:
            out = run(src, env)
            if out is not None:
                print(repr(out))
        except Exception as e:
            print('Error:', e)

def run_file(path: str):
    env = Env()
    env = Env()
    load_std(env)
    env.define('print', Builtin(lambda *a: print(*a), 'print'))
    env.define('input', Builtin(lambda msg="": input(msg), 'input'))
    env.define('eval', Builtin(lambda x: eval_text(x, env), 'eval'))
    src = Path(path).read_text(encoding='utf-8')
    try:
        run(src, env)
    except Exception as e:
        print('Runtime error:', e)

# Module loader scaffold (keeps existing structure, simple)
MODULES_DIRNAME = 'bazic_modules'
LOCKFILE = '.bazic_lock.json'

def load_module(name, importer_path='.', project_root='.'):
    # Backwards compatibility wrapper for earlier API (name -> module name in lockfile)
    project_root = Path(project_root)
    lock_path = project_root / LOCKFILE
    if not lock_path.exists():
        raise FileNotFoundError('No lockfile found')
    lock = json.loads(lock_path.read_text(encoding='utf-8'))
    if name not in lock:
        raise KeyError(f'Package {name} not installed')
    ver = lock[name]
    pkg_dir = project_root / MODULES_DIRNAME / f"{name}@{ver}"
    manifest = pkg_dir / 'bazic.json'
    if not manifest.exists(): raise FileNotFoundError('Package missing bazic.json')
    meta = json.loads(manifest.read_text(encoding='utf-8'))
    entry = pkg_dir / meta.get('main','index.baz')
    src = entry.read_text(encoding='utf-8')
    module_env = Env()
    module_env.define('print', Builtin(lambda *a: print(*a), 'print'))
    run(src, module_env)
    return module_env

# Entry
if __name__ == '__main__':
    if len(sys.argv) > 1:
        run_file(sys.argv[1])
    else:
        print('Running SAMPLE...\n')
        env = Env()
        env = Env()
        load_std(env)
        env.define('print', Builtin(lambda *a: print(*a), 'print'))
        env.define('input', Builtin(lambda msg="": input(msg), 'input'))
        env.define('eval', Builtin(lambda x: eval_text(x, env), 'eval'))
        try:
            run(SAMPLE, env)
        except Exception as e:
            print('Runtime error:', e)
        repl()
