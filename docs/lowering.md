# Lowering

Lowering rewrites high-level Jayess AST forms into simpler core forms that the
backend can emit consistently.

## Strategy

Lowering should use generalized mechanisms rather than one-off handling for
every syntax case. For example, `for in` and `for of` should lower through the
same loop primitives used by `while`-style control flow where possible.

## Responsibilities

- normalize return values and abrupt control flow
- preserve side-effect order
- lower logical, nullish, comparison, and update expressions
- normalize loop, switch, label, try/catch/finally, and throw behavior
- restore scopes and cleanup paths consistently

## Contract

After lowering, the backend should see fewer special cases. If a feature needs
backend support, lowering should still expose it through shared core operations
instead of duplicating syntax-specific emission rules.

## Example Rewrite

```js
for (var i = 0; i < count; i = i + 1) {
  total = total + i;
}
```

Lowering can normalize this into initialization plus a core loop with condition,
body, and step operations. Backend emission should not need a separate design for
every surface loop syntax.
