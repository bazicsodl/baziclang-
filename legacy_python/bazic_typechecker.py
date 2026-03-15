
# bazic_typechecker.py - skeleton for future gradual typing
# Strategy: parse annotations on let/fn, propagate simple types for literals and arrays.
# This file provides a checker function that walks AST and reports obvious mismatches.
from typing import Any

def check_program(ast, env_types=None):
    # Placeholder: return empty list of errors for now
    return []

# Example usage (to be integrated with parser AST): call check_program(ast)
