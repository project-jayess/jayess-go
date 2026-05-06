# Modules

Jayess modules use JavaScript-like `import` and `export` syntax with compiler
controlled resolution.

## Imports

Relative imports load local Jayess files or native binding files.

```js
import { add } from "./native/math.js";
```

Reserved package imports such as `"ffi"` and `"llvm"` are handled by compiler
packages.

## Exports

Modules can export declarations and named values. Binding modules export a
default `bind(...)` manifest and named placeholder symbols for editor-friendly
imports.

## Resolution

The resolver builds a deterministic module graph, validates import/export names,
and reports diagnostics with file and symbol context.
