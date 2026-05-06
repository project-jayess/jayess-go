# Performance

Jayess compiler performance should be measured on compiler-sized source inputs,
not only tiny examples.

## Baselines

Performance baselines should cover lexing, parsing, semantic analysis, lowering,
module resolution, diagnostic collection, and backend emission.

## Commands

Use Go tests and benchmarks under `test/` for repeatable measurements. Generated
large fixtures should be placed under `temp/` rather than committed unless they
are intentionally small reviewable test files.

## Risks

Avoid quadratic behavior in module graph traversal, symbol lookup, AST walks,
diagnostic sorting, and string/source-position handling.

## Example Commands

```sh
go test ./test -run Performance
go test ./test -bench .
```
