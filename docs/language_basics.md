# Language Basics

Jayess source files use `.js` syntax, but only the supported Jayess subset is
accepted.

## Source Files

Files are parsed as modules when they use `import` or `export`. Relative imports
resolve through the project loader and should include stable paths.

## Comments and Literals

Jayess accepts JavaScript-style comments and common literal forms used by the
supported parser: numbers, strings, booleans, `null`, arrays, objects, and
BigInt literals where the lowering/backend supports them.

## Variables

Use `const` for immutable bindings and `var` for mutable block-scoped bindings.
`let` is intentionally unsupported.

```js
const start = 1;
var total = start + 2;
total = total + 1;
```

## Scope

Blocks, functions, classes, and modules create scopes. Closures capture values
through compiler-managed closure state. Module visibility is controlled with
`export`, not top-level `public` or `private`.
