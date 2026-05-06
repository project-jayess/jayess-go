# Compiler Data Structures

Jayess self-hosting code should use a small set of deterministic container
primitives instead of depending on object enumeration details for compiler
internals.

## Vector

`CompilerVector` is an ordered grow-only list with indexed reads and writes. It
is suitable for token streams, AST child lists, diagnostic lists, and work
queues. `Values` returns a copy so callers cannot mutate the vector storage by
accident.

## Table

`CompilerTable` is a string-keyed map with deterministic insertion-order keys.
It is suitable for scopes, symbol tables, module registries, and diagnostic
indexes. Updating an existing key preserves its original key order.

## Record

`CompilerRecord` stores shaped compiler data. A record shape names the allowed
fields, and writes to unknown fields fail. It is suitable for tokens, AST nodes,
type metadata, and other compiler records where accidental field creation would
hide bugs.

## Ownership

Container APIs store Jayess `Value` instances by value. Values returned from
vectors, tables, and records are caller-owned copies of the runtime value handle.
List snapshots and key snapshots return new Go slices, so mutating a snapshot
does not mutate the container.

## Self-Hosting Use

Future Jayess-written compiler components should prefer these containers for
compiler internals instead of depending on object enumeration behavior. Vectors
fit token streams and AST child lists, tables fit scopes and module registries,
and records fit shaped AST nodes, tokens, and type metadata.

## Example Shape

```js
const tokens = CompilerVector();
tokens.push({ kind: "identifier", text: "main" });

const symbols = CompilerTable();
symbols.set("main", { kind: "function" });
```

The exact constructors depend on the Jayess runtime package surface used by a
self-hosted compiler component.
