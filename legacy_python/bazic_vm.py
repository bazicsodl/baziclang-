
# bazic_vm.py - Prototype bytecode VM for a tiny subset
from typing import List, Any, Tuple

# Opcodes: PUSH_CONST, ADD, SUB, MUL, DIV, PRINT, HALT
PUSH = "PUSH"; ADD = "ADD"; SUB = "SUB"; MUL = "MUL"; DIV = "DIV"; PRINT = "PRINT"; HALT = "HALT"

class VM:
    def __init__(self, code: List[Tuple[str, Any]]):
        self.code = code
        self.stack = []
        self.ip = 0

    def run(self):
        while self.ip < len(self.code):
            op, arg = self.code[self.ip]
            #print("ip", self.ip, "op", op, "arg", arg, "stack", self.stack)
            if op == PUSH:
                self.stack.append(arg)
            elif op == ADD:
                b = self.stack.pop(); a = self.stack.pop(); self.stack.append(a + b)
            elif op == SUB:
                b = self.stack.pop(); a = self.stack.pop(); self.stack.append(a - b)
            elif op == MUL:
                b = self.stack.pop(); a = self.stack.pop(); self.stack.append(a * b)
            elif op == DIV:
                b = self.stack.pop(); a = self.stack.pop(); self.stack.append(a / b)
            elif op == PRINT:
                val = self.stack.pop(); print(val)
            elif op == HALT:
                break
            else:
                raise RuntimeError(f"Unknown opcode {op}")
            self.ip += 1

def assemble_expression(expr: str):
    # VERY naive assembler for expressions like "1 2 + print"
    toks = expr.split()
    code = []
    for t in toks:
        if t.isdigit():
            code.append((PUSH, int(t)))
        elif t.replace('.','',1).isdigit():
            code.append((PUSH, float(t)))
        elif t == '+': code.append((ADD, None))
        elif t == '-': code.append((SUB, None))
        elif t == '*': code.append((MUL, None))
        elif t == '/': code.append((DIV, None))
        elif t == 'print': code.append((PRINT, None))
        else:
            raise RuntimeError("Unknown token in assemble_expression: " + t)
    code.append((HALT, None))
    return code

if __name__ == '__main__':
    code = assemble_expression("3 4 + print")
    vm = VM(code); vm.run()
