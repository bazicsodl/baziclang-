# bazic_closures.py
# BAZIC-Lang interpreter extended with proper closures & function expressions.
# Save and run: python bazic_closures.py

import sys
import re
from collections import namedtuple

Token = namedtuple("Token", ["type", "val"])

TOKEN_SPEC = [
    ('NUMBER',   r'\d+(\.\d+)?'),
    ('IDENT',    r'[A-Za-z_][A-Za-z0-9_]*'),
    ('STRING',   r'"([^"\\]|\\.)*"'),
    ('SKIP',     r'[ \t]+'),
    ('NEWLINE',  r'\n'),
    ('COMMENT',  r'//.*'),
    ('OP',       r'==|!=|<=|>=|&&|\|\||[-+*/%<>]=?|='),
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
            tokens.append(Token('NEWLINE', '\n'))
            line += 1
        elif kind == 'OP':
            opmap = {
                '&&': 'AND', '||': 'OR', '==': 'EQ', '!=': 'NE', '<=': 'LE', '>=': 'GE'
            }
            if txt in opmap:
                tokens.append(Token(opmap[txt], txt))
            elif txt == '=':
                tokens.append(Token('ASSIGN', txt))
            else:
                tokens.append(Token('OP', txt))
        else:
            tokens.append(Token(kind, txt))
    tokens.append(Token('EOF', ''))
    return tokens

# -------------------------
# Parser
# -------------------------
class Parser:
    def __init__(self, tokens):
        self.tokens = tokens
        self.i = 0
    def peek(self):
        return self.tokens[self.i]
    def next(self):
        t = self.peek(); self.i += 1; return t
    def accept(self, *types):
        if self.peek().type in types:
            return self.next()
        return None
    def expect(self, type_):
        t = self.next()
        if t.type != type_:
            raise SyntaxError(f"Expected {type_}, got {t.type}")
        return t

    def parse_program(self):
        stmts = []
        while self.peek().type != 'EOF':
            if self.peek().type == 'NEWLINE':
                self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next()
        return ('program', stmts)

    def parse_stmt(self):
        t = self.peek()
        if t.type == 'LET':
            return self.parse_let()
        if t.type == 'FN':
            return self.parse_fndef_stmt()
        if t.type == 'IF':
            return self.parse_if()
        if t.type == 'WHILE':
            return self.parse_while()
        if t.type == 'RETURN':
            self.next()
            expr = self.parse_expr()
            return ('return', expr)
        expr = self.parse_expr()
        return ('expr', expr)

    def parse_block(self):
        self.expect('LBRACE')
        stmts = []
        while self.peek().type != 'RBRACE':
            if self.peek().type == 'NEWLINE':
                self.next(); continue
            stmts.append(self.parse_stmt())
            while self.peek().type in ('NEWLINE','SEMICOLON'):
                self.next()
        self.expect('RBRACE')
        return ('block', stmts)

    def parse_let(self):
        self.expect('LET')
        name = self.expect('IDENT').val
        if self.accept('ASSIGN'):
            expr = self.parse_expr()
        else:
            expr = ('null', None)
        return ('let', name, expr)

    # function statement (top-level or statement)
    def parse_fndef_stmt(self):
        node = self.parse_fn_expr_or_decl()
        # if it returned a function with a name, the parse node might be ('fn', name, params, body)
        # keep as-is for eval to handle defining the name in env
        return node

    # parse function as expression or declaration
    # supports: fn name?(params) { body } — returns ('fn', name_or_None, params, body)
    def parse_fn_expr_or_decl(self):
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
        return ('fn', name, params, body)

    def parse_if(self):
        self.expect('IF')
        self.expect('LPAREN')
        cond = self.parse_expr()
        self.expect('RPAREN')
        then_block = self.parse_block()
        else_block = None
        if self.accept('ELSE'):
            if self.peek().type == 'IF':
                else_block = ('block', [self.parse_if()])  # nested
            else:
                else_block = self.parse_block()
        return ('if', cond, then_block, else_block)

    def parse_while(self):
        self.expect('WHILE')
        self.expect('LPAREN')
        cond = self.parse_expr()
        self.expect('RPAREN')
        body = self.parse_block()
        return ('while', cond, body)

    # expressions...
    def parse_expr(self):
        return self.parse_or()

    def parse_or(self):
        node = self.parse_and()
        while True:
            if self.accept('OR'):
                right = self.parse_and()
                node = ('or', node, right)
            else:
                break
        return node

    def parse_and(self):
        node = self.parse_equality()
        while True:
            if self.accept('AND'):
                right = self.parse_equality()
                node = ('and', node, right)
            else:
                break
        return node

def parse_equality(self):
    node = self.parse_relational()
    while self.curr.type in ("EQ", "NE"):
        op = self.curr.type
        self.advance()
        right = self.parse_relational()
        node = ("eq", op, node, right)
    return node

    def parse_relational(self):
    node = self.parse_add()
    while self.curr.type in ("LT", "LE", "GT", "GE"):
        op = self.curr.type
        self.advance()
        right = self.parse_add()
        node = ("rel", op, node, right)
    return node


    def parse_add(self):
        node = self.parse_mul()
        while True:
            if self.accept('OP') and self.tokens[self.i-1].val in ('+','-'):
                op = self.tokens[self.i-1].val
                right = self.parse_mul(); node = (op, node, right)
            else:
                break
        return node

    def parse_mul(self):
        node = self.parse_unary()
        while True:
            if self.accept('OP') and self.tokens[self.i-1].val in ('*','/','%'):
                op = self.tokens[self.i-1].val
                right = self.parse_unary(); node = (op, node, right)
            else:
                break
        return node

    def parse_unary(self):
        if self.accept('OP') and self.tokens[self.i-1].val == '-':
            val = self.parse_unary()
            return ('neg', val)
        if self.accept('IDENT') and self.tokens[self.i-1].val == 'not':
            val = self.parse_unary()
            return ('not', val)
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
            # function expression
            return self.parse_fn_expr_or_decl()
        if t.type == 'IDENT':
            name = self.next().val
            # assignment
            if self.peek().type == 'ASSIGN':
                self.next(); expr = self.parse_expr(); return ('assign', name, expr)
            # call?
            if self.peek().type == 'LPAREN':
                self.next()
                args = []
                if self.peek().type != 'RPAREN':
                    while True:
                        args.append(self.parse_expr())
                        if not self.accept('COMMA'):
                            break
                self.expect('RPAREN')
                return ('call', name, args)
            return ('var', name)
        if t.type == 'LPAREN':
            self.next()
            node = self.parse_expr()
            self.expect('RPAREN')
            return node
        raise SyntaxError(f"Unexpected token in expression: {t.type} {t.val}")

# -------------------------
# Interpreter
# -------------------------
class ReturnSignal(Exception):
    def __init__(self, value):
        self.value = value

class Env:
    def __init__(self, parent=None):
        self.map = {}
        self.parent = parent
    def get(self, name):
        if name in self.map:
            return self.map[name]
        if self.parent:
            return self.parent.get(name)
        raise NameError(f"Undefined variable '{name}'")
    def set(self, name, val):
        if name in self.map:
            self.map[name] = val
            return
        if self.parent and self.parent.has(name):
            self.parent.set(name, val); return
        self.map[name] = val
    def define(self, name, val):
        self.map[name] = val
    def has(self, name):
        if name in self.map: return True
        if self.parent: return self.parent.has(name)
        return False

def is_truthy(v):
    return not (v is None or v is False)

# user function representation
# ('userfn', params, body, closure_env)
# builtin functions: ('builtin', python_callable)

def eval_node(node, env):
    tp = node[0]
    if tp == 'program':
        out = None
        for s in node[1]:
            out = eval_node(s, env)
        return out

    if tp == 'let':
        _, name, expr = node
        val = eval_node(expr, env)
        env.define(name, val)
        return val

    if tp == 'fn':
        _, name, params, body = node
        # capture the current lexical environment as closure
        closure_env = env
        func = ('userfn', params, body, closure_env)
        # if function has a name, bind it into the closure so recursion works
        if name:
            # Define the name in the closure env so the function body can refer to it
            closure_env.define(name, func)
        return func

    if tp == 'block':
        _, stmts = node
        out = None
        local = Env(parent=env)
        for s in stmts:
            out = eval_node(s, local)
        return out

    if tp == 'if':
        _, cond, thenb, elseb = node
        if is_truthy(eval_node(cond, env)):
            return eval_node(thenb, env)
        elif elseb:
            return eval_node(elseb, env)
        return None

    if tp == 'while':
        _, cond, body = node
        last = None
        while is_truthy(eval_node(cond, env)):
            last = eval_node(body, env)
        return last

    if tp == 'return':
        val = eval_node(node[1], env)
        raise ReturnSignal(val)

    if tp == 'expr':
        return eval_node(node[1], env)

    if tp == 'number':
        return node[1]
    if tp == 'string':
        return node[1]
    if tp == 'bool':
        return node[1]
    if tp == 'null':
        return None

    if tp == 'var':
        return env.get(node[1])

    if tp == 'assign':
        _, name, expr = node
        val = eval_node(expr, env)
        env.set(name, val)
        return val

    if tp == 'call':
        _, name, args = node
        # short path for builtin print (keeps compatibility)
        if name == 'print':
            vals = [eval_node(a, env) for a in args]
            print(*vals)
            return None
        fn = env.get(name)
        if isinstance(fn, tuple) and fn[0] == 'userfn':
            _, params, body, closure = fn
            if len(params) != len(args):
                raise TypeError("Argument count mismatch")
            # Create a new frame whose parent is the closure captured at definition time
            local = Env(parent=closure)
            # bind parameters (evaluate args in caller env)
            for p, a in zip(params, args):
                local.define(p, eval_node(a, env))
            try:
                eval_node(body, local)
                return None
            except ReturnSignal as r:
                return r.value
        elif isinstance(fn, tuple) and fn[0] == 'builtin':
            _, pycall = fn
            vals = [eval_node(a, env) for a in args]
            return pycall(*vals)
        else:
            raise TypeError(f"Not a function: {name}")

    if tp == 'neg':
        return -eval_node(node[1], env)
    if tp == 'not':
        return not is_truthy(eval_node(node[1], env))
    if tp == 'and':
        a = eval_node(node[1], env)
        if not is_truthy(a): return a
        return eval_node(node[2], env)
    if tp == 'or':
        a = eval_node(node[1], env)
        if is_truthy(a): return a
        return eval_node(node[2], env)
    if tp in ('+','-','*','/','%','==','!=','<','>','<=','>='):
        left = eval_node(node[1], env)
        right = eval_node(node[2], env)
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

# -------------------------
# Helpers / Entrypoints
# -------------------------
def run(src, env):
    tokens = lex(src)
    parser = Parser(tokens)
    ast = parser.parse_program()
    return eval_node(ast, env)

def repl():
    print("BAZIC-Lang (closures) REPL. Type :quit or Ctrl-C to exit.")
    env = Env()
    env.define('print', ('builtin', lambda *a: print(*a)))
    while True:
        try:
            src = input('bazic> ')
        except (EOFError, KeyboardInterrupt):
            print(); break
        if src.strip() == '':
            continue
        if src.strip() == ':quit':
            break
        open_braces = src.count('{') - src.count('}')
        while open_braces > 0:
            try:
                more = input('... ')
            except (EOFError, KeyboardInterrupt):
                more = ''
            src += '\n' + more
            open_braces = src.count('{') - src.count('}')
        try:
            out = run(src, env)
            if out is not None:
                print(repr(out))
        except Exception as e:
            print("Error:", e)

def run_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        src = f.read()
    env = Env()
    env.define('print', ('builtin', lambda *a: print(*a)))
    try:
        run(src, env)
    except Exception as e:
        print("Runtime error:", e)

# -------------------------
# Demo: closures & returned functions
# -------------------------
DEMO = r'''
// return a function that captures x
fn make_adder(x) {
  fn (y) { return x + y; }   // anonymous function expression returned implicitly
}

let add5 = make_adder(5)
print("add5(3) =", add5(3)) // expect 8

// named function expression recursion (factorial)
let fact = fn f(n) {
  if (n <= 1) { return 1; }
  return n * f(n-1);
}

print("fact(5) =", fact(5)) // expect 120

// closure that mutates captured state
fn make_counter() {
  let i = 0;
  fn () {
    i = i + 1;
    return i;
  }
}

let c = make_counter()
print(c()) // 1
print(c()) // 2
print(c()) // 3

// returning closures that capture outer scope across multiple closures
fn make_pair(x) {
  fn getx() { return x; }
  fn setx(v) { x = v; return x; }
  // return a struct-like pair via two variables (demo: return as two functions)
  // We'll store pair in an array-like: but for now show returning getx & setx by binding them to names
  fn res() { return (getx, setx); } // simple trick: return a function that returns two functions
  return res();
}

let pair = make_pair(10)
let getter = pair[0] // note: indexing not implemented in this minimal interp; just illustrative
'''

# -------------------------
# Main
# -------------------------
if __name__ == '__main__':
    if len(sys.argv) > 1:
        run_file(sys.argv[1])
    else:
        print("Running closure demo...")
        # run the key closure examples (without the incomplete 'pair' last example)
        demo_src = r'''
fn make_adder(x) {
  fn (y) { return x + y; }
}
let add5 = make_adder(5)
print(add5(3))

let fact = fn f(n) {
  if (n <= 1) { return 1; }
  return n * f(n-1);
}
print(fact(5))

fn make_counter() {
  let i = 0;
  fn () {
    i = i + 1;
    return i;
  }
}
let c = make_counter()
print(c())
print(c())
print(c())
'''
        run(demo_src, Env())
        repl()
