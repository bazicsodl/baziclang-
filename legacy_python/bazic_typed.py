# bazic_typed.py
# BAZIC-Lang prototype with a simple gradual/static type checker + interpreter
# Run: python bazic_typed.py

import sys, re
from collections import namedtuple

Token = namedtuple("Token", ["type", "val"])

# ---------- Lexer ----------
TOKEN_SPEC = [
    ('NUMBER',   r'\d+(\.\d+)?'),
    ('IDENT',    r'[A-Za-z_][A-Za-z0-9_]*'),
    ('STRING',   r'"([^"\\]|\\.)*"'),
    ('SKIP',     r'[ \t]+'),
    ('NEWLINE',  r'\n'),
    ('COMMENT',  r'//.*'),
    ('OP',       r'==|!=|<=|>=|&&|\|\||[-+*/%<>]=?|=|:'),
    ('LPAREN',   r'\('),
    ('RPAREN',   r'\)'),
    ('LBRACE',   r'\{'),
    ('RBRACE',   r'\}'),
    ('COMMA',    r','),
    ('SEMICOLON',r';'),
]
TOK_REGEX = re.compile('|'.join('(?P<%s>%s)' % pair for pair in TOKEN_SPEC))
KEYWORDS = {'let','fn','if','else','while','return','print','true','false','null','not'}

def lex(src):
    pos = 0
    line = 1
    tokens = []
    while pos < len(src):
        m = TOK_REGEX.match(src, pos)
        if not m:
            raise SyntaxError(f"Unexpected character: {src[pos]!r} at line {line}")
        kind = m.lastgroup
        txt = m.group(kind)
        pos = m.end()
        if kind == 'NUMBER':
            if '.' in txt:
                val = float(txt)
            else:
                val = int(txt)
            tokens.append(Token('NUMBER', val))
        elif kind == 'IDENT':
            if txt in KEYWORDS:
                tokens.append(Token(txt.upper(), txt))
            else:
                tokens.append(Token('IDENT', txt))
        elif kind == 'STRING':
            inner = txt[1:-1]
            inner = inner.encode('utf-8').decode('unicode_escape')
            tokens.append(Token('STRING', inner))
        elif kind == 'SKIP' or kind == 'COMMENT':
            continue
        elif kind == 'NEWLINE':
            tokens.append(Token('NEWLINE', '\n')); line += 1
        elif kind == 'OP':
            # Treat ':' as COLON for type annotations
            if txt == ':':
                tokens.append(Token('COLON', txt))
            elif txt == '=':
                tokens.append(Token('ASSIGN', txt))
            else:
                opmap = {'&&':'AND','||':'OR','==':'EQ','!=':'NE','<=':'LE','>=':'GE'}
                if txt in opmap:
                    tokens.append(Token(opmap[txt], txt))
                else:
                    tokens.append(Token('OP', txt))
        else:
            tokens.append(Token(kind, txt))
    tokens.append(Token('EOF', ''))
    return tokens

# ---------- AST nodes (simple tuples) ----------
# We'll annotate AST nodes with type annotation fields where appropriate

# ---------- Parser ----------
class Parser:
    def __init__(self, tokens):
        self.tokens = tokens; self.i = 0
    def peek(self): return self.tokens[self.i]
    def next(self): t = self.peek(); self.i += 1; return t
    def accept(self, *types):
        if self.peek().type in types: return self.next()
        return None
    def expect(self, type_):
        t = self.next()
        if t.type != type_: raise SyntaxError(f"Expected {type_}, got {t.type}")
        return t

    def parse_program(self):
        stmts = []
        while self.peek().type != 'EOF':
            if self.peek().type == 'NEWLINE': self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        return ('program', stmts)

    def parse_stmt(self):
        t = self.peek()
        if t.type == 'LET': return self.parse_let()
        if t.type == 'FN': return self.parse_fndef_stmt()
        if t.type == 'IF': return self.parse_if()
        if t.type == 'WHILE': return self.parse_while()
        if t.type == 'RETURN':
            self.next(); expr = self.parse_expr(); return ('return', expr)
        expr = self.parse_expr()
        return ('expr', expr)

    def parse_block(self):
        self.expect('LBRACE')
        stmts = []
        while self.peek().type != 'RBRACE':
            if self.peek().type == 'NEWLINE': self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'): self.next()
        self.expect('RBRACE')
        return ('block', stmts)

    def parse_type(self):
        # simple type names only: int, float, string, bool, any
        if self.peek().type == 'IDENT':
            name = self.next().val
            return name
        raise SyntaxError("Expected type name")

    def parse_let(self):
        self.expect('LET')
        name = self.expect('IDENT').val
        ann = None
        if self.accept('COLON'):
            ann = self.parse_type()
        if self.accept('ASSIGN'):
            expr = self.parse_expr()
        else:
            expr = ('null', None)
        return ('let', name, ann, expr)

    def parse_fndef_stmt(self):
        node = self.parse_fn_expr_or_decl()
        return node

    def parse_fn_expr_or_decl(self):
        # fn name?(params): ret_type? { body }
        self.expect('FN')
        name = None
        if self.peek().type == 'IDENT':
            name = self.next().val
        self.expect('LPAREN')
        params = []
        if self.peek().type != 'RPAREN':
            while True:
                pname = self.expect('IDENT').val
                ptype = None
                if self.accept('COLON'):
                    ptype = self.parse_type()
                params.append( (pname, ptype) )
                if not self.accept('COMMA'): break
        self.expect('RPAREN')
        rettype = None
        if self.accept('COLON'):
            rettype = self.parse_type()
        body = self.parse_block()
        return ('fn', name, params, rettype, body)

    def parse_if(self):
        self.expect('IF'); self.expect('LPAREN'); cond = self.parse_expr(); self.expect('RPAREN')
        then_block = self.parse_block(); else_block = None
        if self.accept('ELSE'):
            if self.peek().type == 'IF':
                else_block = ('block', [self.parse_if()])
            else:
                else_block = self.parse_block()
        return ('if', cond, then_block, else_block)

    def parse_while(self):
        self.expect('WHILE'); self.expect('LPAREN'); cond = self.parse_expr(); self.expect('RPAREN')
        body = self.parse_block(); return ('while', cond, body)

    # expressions
    def parse_expr(self): return self.parse_or()
    def parse_or(self):
        node = self.parse_and()
        while self.accept('OR'):
            right = self.parse_and(); node = ('or', node, right)
        return node
    def parse_and(self):
        node = self.parse_equality()
        while self.accept('AND'):
            right = self.parse_equality(); node = ('and', node, right)
        return node
    def parse_equality(self):
        node = self.parse_relational()
        while True:
            if self.accept('EQ'):
                right = self.parse_relational(); node = ('==', node, right)
            elif self.accept('NE'):
                right = self.parse_relational(); node = ('!=', node, right)
            else: break
        return node
    def parse_relational(self):
        node = self.parse_add()
        while True:
            t = self.peek()
            if t.type == 'OP' and t.val in ('<','>','<=','>='):
                op = self.next().val; right = self.parse_add(); node = (op, node, right)
            else: break
        return node
    def parse_add(self):
        node = self.parse_mul()
        while True:
            if self.accept('OP') and self.tokens[self.i-1].val in ('+','-'):
                op = self.tokens[self.i-1].val; right = self.parse_mul(); node = (op, node, right)
            else: break
        return node
    def parse_mul(self):
        node = self.parse_unary()
        while True:
            if self.accept('OP') and self.tokens[self.i-1].val in ('*','/','%'):
                op = self.tokens[self.i-1].val; right = self.parse_unary(); node = (op, node, right)
            else: break
        return node
    def parse_unary(self):
        if self.accept('OP') and self.tokens[self.i-1].val == '-':
            val = self.parse_unary(); return ('neg', val)
        if self.accept('IDENT') and self.tokens[self.i-1].val == 'not':
            val = self.parse_unary(); return ('not', val)
        return self.parse_primary()
    def parse_primary(self):
        t = self.peek()
        if t.type == 'NUMBER':
            self.next(); return ('number', t.val)
        if t.type == 'STRING':
            self.next(); return ('string', t.val)
        if t.type == 'TRUE':
            self.next(); return ('bool', True)
        if t.type == 'FALSE':
            self.next(); return ('bool', False)
        if t.type == 'NULL':
            self.next(); return ('null', None)
        if t.type == 'FN':
            return self.parse_fn_expr_or_decl()
        if t.type == 'IDENT':
            name = self.next().val
            # assignment
            if self.peek().type == 'ASSIGN':
                self.next(); expr = self.parse_expr(); return ('assign', name, expr)
            # call
            if self.peek().type == 'LPAREN':
                self.next()
                args = []
                if self.peek().type != 'RPAREN':
                    while True:
                        args.append(self.parse_expr())
                        if not self.accept('COMMA'): break
                self.expect('RPAREN')
                return ('call', name, args)
            return ('var', name)
        if t.type == 'LPAREN':
            self.next(); node = self.parse_expr(); self.expect('RPAREN'); return node
        raise SyntaxError(f"Unexpected token in expression: {t.type} {t.val}")

# ---------- Runtime types / helpers ----------
PRIMITIVE_TYPES = {'int','float','string','bool','any'}
def type_of_literal(val):
    if isinstance(val, int): return 'int'
    if isinstance(val, float): return 'float'
    if isinstance(val, str): return 'string'
    if isinstance(val, bool): return 'bool'
    if val is None: return 'any'
    return 'any'

def types_compatible(expected, actual):
    # 'any' is compatible with everything; ints compatible with floats via coercion
    if expected is None or expected == 'any': return True
    if actual is None or actual == 'any': return True
    if expected == actual: return True
    # numeric coercion: int -> float allowed when expected float
    if expected == 'float' and actual == 'int': return True
    return False

# ---------- Function / Builtin classes ----------
class Function:
    def __init__(self, params, body, closure_env, name=None, ann_ret=None):
        self.params = params  # list of (pname, ptype)
        self.body = body
        self.closure_env = closure_env
        self.name = name
        self.ann_ret = ann_ret
    def __repr__(self):
        n = self.name or ''
        ps = ', '.join([f"{p}:{t}" if t else p for p,t in self.params])
        r = f":{self.ann_ret}" if self.ann_ret else ""
        return f"<fn {n}({ps}){r}>"

class Builtin:
    def __init__(self, fn, name):
        self.fn = fn; self.name = name
    def __repr__(self): return f"<builtin {self.name}>"

# ---------- Environment ----------
class Env:
    def __init__(self, parent=None):
        self.map = {}; self.parent = parent
    def get(self, name):
        if name in self.map: return self.map[name]
        if self.parent: return self.parent.get(name)
        raise NameError(f"Undefined variable '{name}'")
    def set(self, name, val):
        if name in self.map:
            self.map[name] = val; return
        if self.parent and self.parent.has(name):
            self.parent.set(name, val); return
        self.map[name] = val
    def define(self, name, val):
        self.map[name] = val
    def has(self, name):
        if name in self.map: return True
        if self.parent: return self.parent.has(name)
        return False

# ---------- Static Type Checker (simple, annotation-driven) ----------
class TypeErrorStatic(Exception):
    pass

def type_check_program(ast):
    # type environment maps variable name -> annotated type or inferred literal type
    type_env = {}
    # function type registry for checking calls: name -> Function signature (params, ret)
    fn_signatures = {}

    def tc_node(node, env_types):
        tp = node[0]
        if tp == 'program':
            for s in node[1]:
                tc_node(s, env_types)
            return None
        if tp == 'let':
            _, name, ann, expr = node
            expr_t = tc_node(expr, env_types)
            final_t = ann or expr_t or 'any'
            if ann and not types_compatible(ann, expr_t):
                raise TypeErrorStatic(f"Type mismatch for '{name}': declared {ann}, got {expr_t}")
            env_types[name] = final_t
            return None
        if tp == 'fn':
            _, name, params, rettype, body = node
            # register signature before checking body (for recursion)
            sig_params = [ptype or 'any' for (_, ptype) in params]
            fn_signatures[name or '<anon>'] = (sig_params, rettype or 'any')
            # when checking the body, extend env with params
            nested = dict(env_types)
            for (pname, ptype) in params:
                nested[pname] = ptype or 'any'
            # allow recursion name bound inside body as function type if named
            if name:
                nested[name] = ('fn', tuple(sig_params), rettype or 'any')
            tc_node(body, nested)
            return ('fn', tuple(sig_params), rettype or 'any')
        if tp == 'block':
            _, stmts = node
            nested = dict(env_types)
            for s in stmts:
                tc_node(s, nested)
            return None
        if tp == 'return':
            expr_t = tc_node(node[1], env_types)
            return expr_t
        if tp == 'expr':
            return tc_node(node[1], env_types)
        if tp == 'number':
            return type_of_literal(node[1])
        if tp == 'string':
            return 'string'
        if tp == 'bool':
            return 'bool'
        if tp == 'null':
            return 'any'
        if tp == 'var':
            name = node[1]
            return env_types.get(name, 'any')
        if tp == 'assign':
            _, name, expr = node
            val_t = tc_node(expr, env_types)
            # if var already annotated, check
            if name in env_types:
                if not types_compatible(env_types[name], val_t):
                    raise TypeErrorStatic(f"Type mismatch on assignment to '{name}': {env_types[name]} vs {val_t}")
            env_types[name] = val_t
            return val_t
        if tp == 'call':
            _, name, args = node
            # check args
            arg_types = [tc_node(a, env_types) for a in args]
            # try to find function signature
            if name in fn_signatures:
                sig_params, rettype = fn_signatures[name]
                # len check
                if len(sig_params) != len(arg_types):
                    raise TypeErrorStatic(f"Arity mismatch calling {name}: expected {len(sig_params)}, got {len(arg_types)}")
                for expected, actual in zip(sig_params, arg_types):
                    if not types_compatible(expected, actual):
                        raise TypeErrorStatic(f"Argument type mismatch calling {name}: expected {expected}, got {actual}")
                return rettype or 'any'
            # builtins: 'print' -> any
            if name == 'print': return 'any'
            # unknown call: be permissive -> any
            return 'any'
        if tp == 'neg':
            t = tc_node(node[1], env_types)
            if t not in ('int','float','any'):
                raise TypeErrorStatic(f"Unary - applied to non-numeric type {t}")
            return t
        if tp == 'not':
            t = tc_node(node[1], env_types)
            if t not in ('bool','any'):
                raise TypeErrorStatic(f"not applied to non-bool type {t}")
            return 'bool'
        if tp == 'and' or tp == 'or':
            a = tc_node(node[1], env_types)
            b = tc_node(node[2], env_types)
            if a not in ('bool','any') or b not in ('bool','any'):
                raise TypeErrorStatic(f"Logical op on non-bool types {a},{b}")
            return 'bool'
        if tp in ('+','-','*','/','%'):
            l = tc_node(node[1], env_types)
            r = tc_node(node[2], env_types)
            if l not in ('int','float','any') or r not in ('int','float','any'):
                raise TypeErrorStatic(f"Arithmetic on non-numeric types {l},{r}")
            # if either float -> float
            if l == 'float' or r == 'float': return 'float'
            return 'int'
        if tp in ('==','!=','<','>','<=','>='):
            l = tc_node(node[1], env_types); r = tc_node(node[2], env_types)
            return 'bool'
        raise TypeErrorStatic(f"Type checker cannot handle node {tp}")

    tc_node(ast, type_env)

# ---------- Interpreter / Evaluator ----------
class ReturnSignal(Exception):
    def __init__(self, value): self.value = value

def is_truthy(v):
    return not (v is None or v is False)

def eval_node(node, env):
    tp = node[0]
    if tp == 'program':
        out = None
        for s in node[1]:
            out = eval_node(s, env)
        return out
    if tp == 'let':
        _, name, ann, expr = node
        val = eval_node(expr, env)
        env.define(name, val)
        return val
    if tp == 'fn':
        _, name, params, rettype, body = node
        func = Function(params, body, env, name=name, ann_ret=rettype)
        if name:
            env.define(name, func)
        return func
    if tp == 'block':
        _, stmts = node
        local = Env(parent=env); out = None
        for s in stmts:
            out = eval_node(s, local)
        return out
    if tp == 'if':
        _, cond, thenb, elseb = node
        if is_truthy(eval_node(cond, env)): return eval_node(thenb, env)
        elif elseb: return eval_node(elseb, env)
        return None
    if tp == 'while':
        _, cond, body = node
        last = None
        while is_truthy(eval_node(cond, env)):
            last = eval_node(body, env)
        return last
    if tp == 'return':
        val = eval_node(node[1], env); raise ReturnSignal(val)
    if tp == 'expr': return eval_node(node[1], env)
    if tp == 'number': return node[1]
    if tp == 'string': return node[1]
    if tp == 'bool': return node[1]
    if tp == 'null': return None
    if tp == 'var': return env.get(node[1])
    if tp == 'assign':
        _, name, expr = node
        val = eval_node(expr, env); env.set(name, val); return val
    if tp == 'call':
        _, name, args = node
        if name == 'print':
            vals = [eval_node(a, env) for a in args]; print(*vals); return None
        fn = env.get(name)
        if isinstance(fn, Function):
            params = fn.params; body = fn.body; closure = fn.closure_env
            if len(params) != len(args):
                raise TypeError("Argument count mismatch")
            local = Env(parent=closure)
            for (pname, _), a in zip(params, args):
                local.define(pname, eval_node(a, env))
            try:
                eval_node(body, local); return None
            except ReturnSignal as r: return r.value
        if isinstance(fn, Builtin):
            vals = [eval_node(a, env) for a in args]; return fn.fn(*vals)
        raise TypeError(f"Not a function: {name}")
    if tp == 'neg': return -eval_node(node[1], env)
    if tp == 'not': return not is_truthy(eval_node(node[1], env))
    if tp == 'and':
        a = eval_node(node[1], env); 
        if not is_truthy(a): return a
        return eval_node(node[2], env)
    if tp == 'or':
        a = eval_node(node[1], env)
        if is_truthy(a): return a
        return eval_node(node[2], env)
    if tp in ('+','-','*','/','%','==','!=','<','>','<=','>='):
        left = eval_node(node[1], env); right = eval_node(node[2], env)
        if tp == '+': return left + right
        if tp == '-': return left - right
        if tp == '*': return left * right
        if tp == '/': return left / right
        if tp == '%': return left % right
        if tp == '==': return left == right
        if tp == '!=': return left != right
        if tp == '<': return left < right
        if tp == '>': return left > right
        if tp == '<=': return left <= right
        if tp == '>=': return left >= right
    raise RuntimeError(f"Unknown node: {tp}")

# ---------- Helpers / REPL / entry ----------
def run_src(src, env, do_typecheck=True):
    toks = lex(src); parser = Parser(toks); ast = parser.parse_program()
    if do_typecheck:
        try:
            type_check_program(ast)
        except TypeErrorStatic as e:
            raise TypeError(f"Static type error: {e}")
    return eval_node(ast, env)

def repl():
    print("BAZIC-Lang (typed) REPL. Type :quit to exit.")
    env = Env(); env.define('print', Builtin(lambda *a: print(*a), 'print'))
    while True:
        try:
            src = input('bazic> ')
        except (EOFError, KeyboardInterrupt):
            print(); break
        if src.strip() in ('',':quit'): continue
        open_braces = src.count('{') - src.count('}')
        while open_braces > 0:
            try: more = input('... ')
            except (EOFError, KeyboardInterrupt): more = ''
            src += '\n' + more; open_braces = src.count('{') - src.count('}')
        try:
            out = run_src(src, env, do_typecheck=True)
            if out is not None: print(repr(out))
        except Exception as e:
            print("Error:", e)

def run_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        src = f.read()
    env = Env(); env.define('print', Builtin(lambda *a: print(*a), 'print'))
    try:
        run_src(src, env, do_typecheck=True)
    except Exception as e:
        print("Runtime/static error:", e)

# ---------- Demo ----------
DEMO = r'''
// correct annotated code
let x: int = 10
let y: float = 2.5
let z = x + 5
print("z", z)

fn add(a: int, b: int): int {
  return a + b
}

print(add(2,3))

// wrong: will be caught by static checker
let s: string = "hello"
// let bad: int = "oops"   // uncomment to see static error

// function missing annotation but inferred from body (limited)
fn inc(n: int): int {
  return n + 1
}
print(inc(10))

// gradual typing examples
let dyn   = 5        // no annotation -> any by default
let adyn: any = "hi" // explicit any disables checks
'''

# ---------- Run demo or REPL ----------
if __name__ == '__main__':
    if len(sys.argv) > 1:
        run_file(sys.argv[1])
    else:
        print("Running typed demo...")
        run_src(DEMO, Env(), do_typecheck=True)
        repl()
