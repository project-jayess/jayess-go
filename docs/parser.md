# Parser

The parser builds Jayess AST nodes from the lexer token stream.

## AST

AST nodes live in the `ast` package. Parser files are organized by language
area, such as classes, modules, functions, bindings, object literals, and
control flow.

## Recovery

Parser recovery should collect useful diagnostics instead of stopping at the
first syntax error when practical. Recovery must keep the AST deterministic so
later stages can decide whether compilation can continue.

## Unsupported Syntax

Unsupported JavaScript features should be rejected explicitly. The parser should
prefer focused diagnostics over accepting syntax that later stages cannot lower
or emit.

## Example AST Shape

```js
function main() {
  return 0;
}
```

The parser should produce a program containing a function declaration whose body
contains one return statement with an integer literal expression.
