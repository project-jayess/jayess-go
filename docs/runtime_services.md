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

## Scheduling And I/O Services

The shared runtime service layer owns timers, microtasks, filesystem access,
process helpers, child-process spawning, TCP, UDP, DNS, HTTP, HTTPS, and streams.
Backends should lower those standard library calls to Jayess runtime symbols,
not to libuv or another user-installed event-loop library.

Optional native bindings may still expose libuv for experiments, but those
bindings are separate from the core runtime service path.

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
