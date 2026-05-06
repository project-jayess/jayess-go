# Semantic Analysis

Semantic analysis validates the meaning of parsed Jayess programs before
lowering and backend emission.

## Checks

- declarations and duplicate names
- import/export availability
- assignment targets
- constructable values
- control-flow labels and jump validity
- module and native binding imports
- unsupported language forms that parsed successfully

## Symbols

Symbol tables track names across blocks, functions, classes, and modules.
Deterministic lookup behavior is important for reproducible diagnostics and
compiler-sized projects.

## Diagnostics

Semantic diagnostics should include file, span, message, and useful context. The
stage should collect multiple independent errors when possible.

## Example Check

```js
const value = 1;
value = 2;
```

Semantic analysis should reject this assignment because `value` was declared
with `const`.
