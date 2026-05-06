# Statements

Jayess supports the statement forms needed for native programs and compiler
workloads.

## Blocks and Conditionals

Blocks create scope. `if` and `else` use runtime truthiness for their condition.

## Loops

Supported loop forms include `while`, `do while`, `for`, `for in`, and `for of`
where the parser, lowering, and backend support the source form. Higher-level
loop forms lower through generalized loop primitives rather than separate
backend designs for each surface syntax.

## Switch and Labels

`switch`, labeled statements, `break`, and `continue` are supported through
structured control-flow lowering. Labels are resolved during semantic analysis.

## Return and Throw

`return` exits the current function. `throw`, `try`, `catch`, and `finally`
lower to abrupt-control paths that preserve cleanup and scope behavior.

## Example

```js
function main() {
  var total = 0;

  for (var i = 0; i < 4; i = i + 1) {
    if (i == 2) {
      continue;
    }
    total = total + i;
  }

  switch (total) {
    case 4:
      console.log("four");
      break;
    default:
      console.log("other");
      break;
  }

  return 0;
}
```
