# Diagnostics

Diagnostics describe lexer, parser, semantic, resolver, lowering, backend, and
toolchain problems in a deterministic user-facing format.

## Fields

A diagnostic should include severity, source span, message, and optional notes.
Project diagnostics should include file and module context when available.

## Ordering

Diagnostics should be sorted deterministically, especially for multi-file
projects. Stable ordering makes tests, editor integrations, and releases easier
to verify.

## Recovery

Recoverable diagnostics allow the compiler to report multiple independent
errors in one run. Stages should stop only when continuing would produce
misleading follow-up errors or unsafe output.

## Example Shape

```text
error: cannot assign to const binding
  at src/main.js:2:1
note: binding was declared const here
  at src/main.js:1:7
```
