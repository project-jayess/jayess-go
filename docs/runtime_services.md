# Runtime Services

Runtime services expose compiler-useful APIs to Jayess code without requiring
Go internals.

## Source and Path APIs

Source file, source text, and path helpers support compiler tools that need
deterministic file loading, line/column tracking, and relative import handling.

## Compiler Data Structures

Compiler vectors, tables, and records provide deterministic containers suitable
for tokens, AST nodes, symbol tables, module registries, and diagnostics.

## Tool Services

Compiler tool services describe runtime assets and packaging metadata needed by
Jayess-built command-line utilities.

## Example Use

```js
function main() {
  const source = readSourceFile("./main.js");
  const text = source.text;
  console.log(text.length);
  return 0;
}
```

The exact package import depends on the runtime service surface exposed by the
compiler distribution.
