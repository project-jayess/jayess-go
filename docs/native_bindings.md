# Native Bindings

Native bindings are normal `.js` modules that export a default `bind(...)`
manifest from `"ffi"`.

```js
import { bind } from "ffi";

export const add = () => {};

export default bind({
  sources: ["./math.c"],
  includeDirs: ["./include"],
  libraryDirs: [],
  sharedLibraries: [],
  licenseFiles: ["./LICENSE.math"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "math_add", type: "function" }
  }
});
```

## Manifest Fields

Manifest values are extracted from object, array, and string literals. Supported
fields include `sources`, `includeDirs`, `libraryDirs`, `sharedLibraries`,
`licenseFiles`, `cflags`, `ldflags`, `platforms`, and `exports`.

## Imports

Binding modules support named imports for exported native symbols. Bare,
default, and namespace imports are rejected.
