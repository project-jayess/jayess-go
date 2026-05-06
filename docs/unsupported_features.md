# Unsupported Features

Jayess intentionally supports a limited JavaScript-like subset. Unsupported
forms should fail with clear parser or semantic diagnostics.

## Language Restrictions

- `let` is not supported; use `var` or `const`
- top-level `public` and `private` are not supported
- full browser and Node.js globals are not provided
- dynamic package manager behavior is not part of the runtime
- unsupported TypeScript-style syntax is rejected
- unsupported destructuring, class, module, or async forms are rejected

## Recommended Alternatives

Use explicit modules, native bindings, and runtime packages. Keep native library
integration in binding files and keep release assets in distribution manifests
or app distribution inputs.

## Example

```js
// Unsupported:
let count = 1;

// Supported:
var total = 1;
const name = "Jayess";
```
