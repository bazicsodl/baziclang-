
# bazic_interpreter_full.py
# BAZIC-Lang full prototype: arrays, dict literals, stdlib, simple require, and scaffolds.
import sys, re, json
from pathlib import Path
from collections import namedtuple
from dataclasses import dataclass
from typing import Any, List, Optional, Tuple

Token = namedtuple("Token", ["type","val","pos"])

TOKEN_SPEC = [
    ("NUMBER",   r"\d+(?:\.\d+)?"),
    ("STRING",   r'"([^"\\\\]|\\\\.)*"'),
    ("IDENT",    r"[A-Za-z_][A-Za-z0-9_]*"),
    ("NEWLINE",  r"\\n"),
    ("SKIP",     r"[ \\t]+"),
    ("COMMENT",  r"//.*"),
    ("OP_MULTI", r"<=|>=|==|!=|&&|\\|\\||\\+\\+|--"),
    ("OP_SINGLE", r"[=+\\-*/%<>]"),
    ("LPAREN",   r"\\("),
    ("RPAREN",   r"\\)"),
    ("LBRACE",   r"\\{"),
    ("RBRACE",   r"\\}"),
    ("LBRACKET", r"\\["),
    ("RBRACKET", r"\\]"),
    ("COMMA",    r","),
    ("SEMICOLON",r";"),
    ("COLON",    r":"),
    ("DOT",      r"\\."),
]

TOK_REGEX = re.compile("|".join(f"(?P<{t}>{p})" for t,p in TOKEN_SPEC))
KEYWORDS = {"let","fn","if","else","while","return","true","false","null"}

class LexerError(Exception): pass

def lex(src: str):
    tokens = []
    line = 1; col = 1; pos = 0; L = len(src)
    while pos < L:
        m = TOK_REGEX.match(src, pos)
        if not m:
            raise LexerError(f"Unexpected {src[pos]!r} at {line}:{col}")
        kind = m.lastgroup; txt = m.group(kind)
        if kind == 'NEWLINE':
            tokens.append(Token('NEWLINE','\\n',(line,col))); line += 1; col = 1; pos = m.end(); continue
        if kind in ('SKIP','COMMENT'):
            lines = txt.count('\\n')
            if lines:
                line += lines; col = 1
            else:
                col += len(txt)
            pos = m.end(); continue
        if kind == 'IDENT' and txt in KEYWORDS:
            tokens.append(Token(txt.upper(), txt, (line,col)))
        elif kind == 'NUMBER':
            val = float(txt) if '.' in txt else int(txt); tokens.append(Token('NUMBER',val,(line,col)))
        elif kind == 'STRING':
            inner = txt[1:-1]; inner = inner.encode('utf-8').decode('unicode_escape'); tokens.append(Token('STRING',inner,(line,col)))
        elif kind in ('OP_MULTI','OP_SINGLE'):
            tokens.append(Token('OP', txt, (line,col)))
        else:
            tokens.append(Token(kind, txt, (line,col)))
        col += len(txt); pos = m.end()
    tokens.append(Token('EOF','',(line,col))); return tokens

# AST nodes
@dataclass
class Node: pass
@dataclass
class Program(Node): body: List[Node]=None
@dataclass
class Let(Node): name: str=None; expr: Node=None
@dataclass
class Return(Node): expr: Node=None
@dataclass
class ExprStmt(Node): expr: Node=None
@dataclass
class Block(Node): stmts: List[Node]=None
@dataclass
class If(Node): cond: Node=None; thenb: Block=None; elseb: Optional[Block]=None
@dataclass
class While(Node): cond: Node=None; body: Block=None
@dataclass
class Number(Node): val: Any=None
@dataclass
class String(Node): val: str=None
@dataclass
class Bool(Node): val: bool=None
@dataclass
class Null(Node): pass
@dataclass
class Var(Node): name: str=None
@dataclass
class Assign(Node): name: str=None; expr: Node=None
@dataclass
class Call(Node): name: str=None; args: List[Node]=None
@dataclass
class Unary(Node): op: str=None; rhs: Node=None
@dataclass
class Binary(Node): op: str=None; left: Node=None; right: Node=None
@dataclass
class FuncDef(Node): name: Optional[str]=None; params: List[str]=None; body: Block=None
@dataclass
class Array(Node): items: List[Node]=None
@dataclass
class DictLiteral(Node): items: List[Tuple[Any, Any]] = None
@dataclass
class Index(Node): target: Node=None; index: Node=None

class ParserError(Exception): pass

class Parser:
    def __init__(self,tokens): self.toks = tokens; self.i = 0
    def peek(self): return self.toks[self.i]
    def next(self): t = self.peek(); self.i += 1; return t
    def accept(self,*types):
        if self.peek().type in types: return self.next()
        return None
    def expect(self, ttype):
        t = self.next()
        if t.type != ttype:
            raise ParserError(f"Expected {ttype} got {t.type} at {t.pos}")
        return t

    def parse_program(self):
        stmts = []
        while self.peek().type != 'EOF':
            if self.peek().type == 'NEWLINE': self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        return Program(stmts)

    def parse_stmt(self):
        t = self.peek()
        if t.type == 'LET': return self.parse_let()
        if t.type == 'FN': return self.parse_fndef()
        if t.type == 'IF': return self.parse_if()
        if t.type == 'WHILE': return self.parse_while()
        if t.type == 'RETURN': self.next(); return Return(self.parse_expr())
        return ExprStmt(self.parse_expr())

    def parse_block(self):
        self.expect('LBRACE'); stmts = []
        while self.peek().type != 'RBRACE':
            if self.peek().type in ('NEWLINE','SEMICOLON'): self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        self.expect('RBRACE'); return Block(stmts)

    def parse_let(self):
        self.expect('LET'); name = self.expect('IDENT').val
        if self.accept('OP') and self.toks[self.i-1].val == '=':
            expr = self.parse_expr()
        else:
            expr = Null()
        return Let(name, expr)

    def parse_fndef(self):
        self.expect('FN'); name = None
        if self.peek().type == 'IDENT': name = self.next().val
        self.expect('LPAREN'); params = []
        if self.peek().type != 'RPAREN':
            while True:
                params.append(self.expect('IDENT').val)
                if not self.accept('COMMA'): break
        self.expect('RPAREN'); body = self.parse_block(); return FuncDef(name, params, body)

    def parse_if(self):
        self.expect('IF'); self.expect('LPAREN'); cond = self.parse_expr(); self.expect('RPAREN')
        while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        thenb = self.parse_block(); elseb = None
        if self.accept('ELSE'):
            while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
            if self.peek().type == 'IF':
                elseb = Block([self.parse_if()])
            else:
                elseb = self.parse_block()
        return If(cond, thenb, elseb)

    def parse_while(self):
        self.expect('WHILE'); self.expect('LPAREN'); cond = self.parse_expr(); self.expect('RPAREN')
        while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        body = self.parse_block(); return While(cond, body)

    # expr precedence
    def parse_expr(self): return self.parse_or()
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
        while self.peek().type == 'OP' and self.peek().val in ('==','!='):
            op = self.next().val; node = Binary(op, node, self.parse_rel())
        return node
    def parse_rel(self):
        node = self.parse_add()
        while self.peek().type == 'OP' and self.peek().val in ('<','>','<=','>='):
            op = self.next().val; node = Binary(op, node, self.parse_add())
        return node
    def parse_add(self):
        node = self.parse_mul()
        while self.peek().type == 'OP' and self.peek().val in ('+','-'):
            op = self.next().val; node = Binary(op, node, self.parse_mul())
        return node
    def parse_mul(self):
        node = self.parse_unary()
        while self.peek().type == 'OP' and self.peek().val in ('*','/','%'):
            op = self.next().val; node = Binary(op, node, self.parse_unary())
        return node
    def parse_unary(self):
        if self.accept('OP') and self.toks[self.i-1].val == '-': return Unary('-', self.parse_unary())
        return self.parse_primary()

    def parse_primary(self):
        t = self.peek()
        if t.type == 'NUMBER': self.next(); return Number(t.val)
        if t.type == 'STRING': self.next(); return String(t.val)
        if t.type == 'TRUE': self.next(); return Bool(True)
        if t.type == 'FALSE': self.next(); return Bool(False)
        if t.type == 'NULL': self.next(); return Null()

        if t.type == 'IDENT':
            name = self.next().val
            if self.peek().type == 'OP' and self.peek().val == '=':
                self.next(); return Assign(name, self.parse_expr())
            if self.peek().type == 'LPAREN':
                self.next(); args = []
                if self.peek().type != 'RPAREN':
                    while True:
                        args.append(self.parse_expr())
                        if not self.accept('COMMA'): break
                self.expect('RPAREN'); node = Call(name, args)
            else:
                node = Var(name)
            # indexing
            while self.peek().type == 'LBRACKET':
                self.next()
                idx = self.parse_expr(); self.expect('RBRACKET')
                node = Index(node, idx)
            return node

        if t.type == 'LPAREN':
            self.next(); node = self.parse_expr(); self.expect('RPAREN'); return node

        # array literal
        if t.type == 'LBRACKET':
            self.next(); items = []
            if self.peek().type != 'RBRACKET':
                while True:
                    items.append(self.parse_expr())
                    if not self.accept('COMMA'): break
            self.expect('RBRACKET'); return Array(items)

        # dict literal (only valid in expression position)
        if t.type == 'LBRACE':
            self.next(); pairs = []
            if self.peek().type != 'RBRACE':
                while True:
                    # key can be STRING or IDENT or NUMBER
                    keyt = self.peek()
                    if keyt.type == 'STRING': key = self.next().val
                    elif keyt.type == 'IDENT': key = self.next().val
                    elif keyt.type == 'NUMBER': key = self.next().val
                    else: raise ParserError(f"Invalid dict key at {keyt.pos}")
                    self.expect('COLON')
                    val = self.parse_expr()
                    pairs.append((key,val))
                    if not self.accept('COMMA'): break
            self.expect('RBRACE'); return DictLiteral(pairs)

        raise ParserError(f"Unexpected token {t.type} {t.val} at {t.pos}")

# Runtime
class RuntimeError_(Exception): pass
class ReturnSignal(Exception):
    def __init__(self,value): self.value = value

class Env:
    def __init__(self,parent=None):
        self.map = {}; self.parent = parent
    def define(self,name,val): self.map[name]=val
    def get(self,name):
        if name in self.map: return self.map[name]
        if self.parent: return self.parent.get(name)
        raise RuntimeError_(f"Undefined variable '{name}'")
    def set(self,name,val):
        if name in self.map: self.map[name]=val; return
        if self.parent: self.parent.set(name,val); return
        raise RuntimeError_(f"Undefined variable '{name}'")

def is_truthy(v): return not (v is None or v is False)

@dataclass
class Builtin:
    fn: Any; name: str

def eval_text(expr, env):
    src = expr + "\\n"; toks = lex(src); p = Parser(toks); ast = p.parse_program(); return eval_node(ast, env)

def eval_node(node, env: Env):
    if isinstance(node, Program):
        out = None
        for s in node.body: out = eval_node(s, env)
        return out
    if isinstance(node, Let):
        val = eval_node(node.expr, env); env.define(node.name, val); return val
    if isinstance(node, FuncDef):
        closure = ("closure", node.params, node.body, env)
        if node.name: env.define(node.name, closure)
        return closure
    if isinstance(node, Block):
        local = Env(parent=env); out = None
        for s in node.stmts: out = eval_node(s, local)
        return out
    if isinstance(node, If):
        if is_truthy(eval_node(node.cond, env)): return eval_node(node.thenb, env)
        if node.elseb: return eval_node(node.elseb, env)
        return None
    if isinstance(node, While):
        last = None
        while is_truthy(eval_node(node.cond, env)): last = eval_node(node.body, env)
        return last
    if isinstance(node, Return):
        val = eval_node(node.expr, env); raise ReturnSignal(val)
    if isinstance(node, ExprStmt):
        return eval_node(node.expr, env)
    if isinstance(node, Number): return node.val
    if isinstance(node, String): return node.val
    if isinstance(node, Bool): return node.val
    if isinstance(node, Null): return None
    if isinstance(node, Var): return env.get(node.name)
    if isinstance(node, Assign):
        val = eval_node(node.expr, env); env.set(node.name, val); return val
    if isinstance(node, Array): return [eval_node(it, env) for it in node.items]
    if isinstance(node, DictLiteral):
        d = {}
        for k_expr, v_expr in node.items:
            # keys: if k_expr is a literal, evaluate it; else if it's a string/ident it's fine
            if isinstance(k_expr, str):
                key = k_expr
            else:
                key = eval_node(k_expr, env)
            d[key] = eval_node(v_expr, env)
        return d
    if isinstance(node, Index):
        target = eval_node(node.target, env); idx = eval_node(node.index, env)
        try:
            return target[idx]
        except Exception as e:
            raise RuntimeError_(f"Index error: {e}")
    if isinstance(node, Call):
        if node.name == 'print':
            vals = [eval_node(a, env) for a in node.args]; print(*vals); return None
        callee = env.get(node.name)
        if isinstance(callee, tuple) and callee[0] == 'closure':
            _, params, body, closure_env = callee
            if len(params) != len(node.args): raise RuntimeError_("Argument count mismatch")
            local = Env(parent=closure_env)
            for p,a in zip(params, node.args): local.define(p, eval_node(a, env))
            try:
                eval_node(body, local); return None
            except ReturnSignal as r: return r.value
        if isinstance(callee, Builtin):
            args = [eval_node(a, env) for a in node.args]; return callee.fn(*args)
        raise RuntimeError_(f"Not a function: {node.name}")
    if isinstance(node, Unary):
        if node.op == '-': return -eval_node(node.rhs, env)
        raise RuntimeError_(f"Unknown unary op {node.op}")
    if isinstance(node, Binary):
        a = eval_node(node.left, env); b = eval_node(node.right, env); op = node.op
        if op == '/' and b == 0: raise RuntimeError_("Division by zero")
        if op == '%' and b == 0: raise RuntimeError_("Modulo by zero")
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
    raise RuntimeError_(f"Unhandled node: {node}")

# Simple module loader (improved)
MODULES_DIRNAME = 'bazic_modules'
LOCKFILE = '.bazic_lock.json'

def simple_require(path: str, importer_path='.') -> dict:
    p = Path(importer_path) / path
    if not p.exists():
        p2 = p.with_suffix('.baz')
        if p2.exists(): p = p2
        else: raise FileNotFoundError(f"Module not found: {path}")
    src = p.read_text(encoding='utf-8')
    module_env = Env()
    register_stdlib(module_env)
    run(src, module_env)
    return module_env.map

# Standard library
def register_stdlib(env: Env):
    env.define('print', Builtin(lambda *a: print(*a), 'print'))
    env.define('input', Builtin(lambda msg='': input(msg), 'input'))
    env.define('len', Builtin(lambda x: len(x), 'len'))
    env.define('range', Builtin(lambda a,b=None: list(range(a)) if b is None else list(range(a,b)), 'range'))
    env.define('push', Builtin(lambda arr,v: (arr.append(v), None)[1], 'push'))
    env.define('pop', Builtin(lambda arr: arr.pop(), 'pop'))
    env.define('keys', Builtin(lambda d: list(d.keys()), 'keys'))
    env.define('values', Builtin(lambda d: list(d.values()), 'values'))
    env.define('require', Builtin(lambda p: simple_require(p), 'require'))
    env.define('str', Builtin(lambda x: str(x), 'str'))
    env.define('int', Builtin(lambda x: int(x), 'int'))
    env.define('float', Builtin(lambda x: float(x), 'float'))
    env.define('json_dump', Builtin(lambda x: json.dumps(x), 'json_dump'))
    env.define('json_load', Builtin(lambda s: json.loads(s), 'json_load'))

def run(src: str, env: Env):
    toks = lex(src); p = Parser(toks); ast = p.parse_program(); return eval_node(ast, env)

def repl():
    print("BAZIC-Lang Full REPL. :quit to exit")
    env = Env(); register_stdlib(env); env.define('eval', Builtin(lambda x: eval_text(x, env), 'eval'))
    while True:
        try: src = input('bazic> ')
        except (KeyboardInterrupt, EOFError): print(); break
        if src.strip() == ':quit': break
        if src.strip() == '': continue
        while src.count('{') > src.count('}'):
            more = input('... '); src += '\\n' + more
        try:
            out = run(src, env)
            if out is not None: print(repr(out))
        except Exception as e:
            print('Error:', e)

if __name__ == '__main__':
    if len(sys.argv) > 1:
        path = sys.argv[1]; env = Env(); register_stdlib(env)
        src = Path(path).read_text(encoding='utf-8')
        try: run(src, env)
        except Exception as e: print('Runtime error:', e)
    else:
        SAMPLE = r"""
let a = [1,2,3]
let d = {"hello": "world", count: 5}
print(len(a))
push(a, 9)
print(a[3])
print(d["hello"])
let m = require("example_module.baz")
print("module keys:", keys(m))
"""
        env = Env(); register_stdlib(env)
        try: run(SAMPLE, env)
        except Exception as e: print('Runtime error:', e)
        repl()
