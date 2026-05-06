# Expressions

Jayess supports a JavaScript-like expression surface where the compiler can
lower it into predictable runtime or LLVM backend operations.

## Operators

Supported operators include arithmetic, comparison, equality, logical,
assignment, update, member access, indexing, function calls, conditional
expressions, comma expressions, `typeof`, `delete`, `in`, `instanceof`, and
nullish coalescing where implemented by lowering/backend support.

## Precedence

Jayess follows JavaScript-style precedence for the supported subset. Parentheses
are recommended when mixing assignment, logical, conditional, and comma
expressions.

## Assignment

Assignments require a valid assignment target: an identifier, member expression,
index expression, or supported destructuring target. Invalid assignment targets
are rejected during semantic analysis.

## Conversion

The compiler does not implement full JavaScript coercion. Operations either use
known primitive behavior or runtime helpers that define Jayess-specific
conversion and error rules.

## Example

```js
function main() {
  var total = 1 + 2 * 3;
  total = total + 4;

  const ok = total >= 10 && total != 0;
  const label = ok ? "ready" : "blocked";
  console.log(label);

  return 0;
}
```
