# Self-Hosting

Self-hosting means compiling future Jayess compiler components written in
Jayess with the Go-hosted compiler.

## Readiness Areas

- reliable parsing of larger programs and modules
- compiler data structures for tokens, ASTs, symbols, and diagnostics
- file, path, and source text APIs
- recoverable diagnostics
- stable module loading
- Jayess-callable backend/toolchain APIs
- runtime services that avoid Go-only internals
- performance baselines for large source files

## Milestone Tests

Self-hosting milestone tests live under `test/`. They should verify that small
Jayess-written compiler utilities can be compiled by the Go-hosted compiler.

## Remaining Risk

Full self-hosting still requires sustained compiler feature parity and practical
performance on real compiler-sized projects.

## Example Milestone Utility

```js
export function countTokens(source) {
  const tokens = lex(source);
  return tokens.length;
}
```

A milestone test can compile a small utility like this with the Go-hosted
compiler before larger compiler components are attempted.
