# Native Interop

Jayess can link native C or C++ code through manually-authored `*.bind.js` files and call exported native functions from Jayess.

## Importing Bindings

Example:

```javascript
import { add } from "./native/math.bind.js";
```

`*.bind.js` files are native binding modules, not normal Jayess source modules.
They are imported only for named native bindings.

Supported:

```javascript
import { add } from "./native/math.bind.js";
```

Not supported:

```javascript
import "./native/math.bind.js";
import native from "./native/math.bind.js";
import * as native from "./native/math.bind.js";
```

`*.bind.js` files declare:
- native source files
- include directories
- compile flags
- link flags
- optional `pkgConfig` package discovery
- optional per-platform source/include/flag overrides
- exported Jayess-visible bindings

The listed native sources are compiled and linked into the final executable through the native toolchain.

## Binding File Format

Example `math.bind.js`:

```js
export default {
  sources: ["./math.c", "./helper.c"],
  includeDirs: ["./include"],
  cflags: ["-DMY_BINDING=1"],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
```

Bindings can also ask Jayess to resolve native flags through `pkg-config`.
Those discovered `--cflags` and `--libs` values are appended to the binding's
native compile/link flags.

```js
export default {
  sources: ["./gtk.c"],
  pkgConfig: ["gtk+-3.0"],
  exports: {
    initNative: { symbol: "jayess_gtk_init", type: "function" }
  }
};
```

Bindings can also provide platform-specific overrides through a `platforms`
object keyed by `linux`, `darwin`, or `windows`. Those arrays are appended on
top of the base `sources`, `includeDirs`, `cflags`, and `ldflags`.

```js
export default {
  sources: ["./webview.cpp"],
  includeDirs: ["../../../../refs/webview/core/include"],
  cflags: ["-std=c++14"],
  ldflags: [],
  platforms: {
    linux: {
      ldflags: ["-lgtk-3", "-lwebkit2gtk-4.1"]
    },
    darwin: {
      ldflags: ["-framework", "Cocoa", "-framework", "WebKit"]
    },
    windows: {
      ldflags: ["-lole32", "-lcomctl32"]
    }
  },
  exports: {
    createWindowNative: { symbol: "jayess_webview_create_window", type: "function" }
  }
};
```

Bindings can also expose imported module values with `type: "value"`. Those are
initialized by calling a zero-argument native getter during module startup.

```js
const f = () => {};
export const version = 0;

export default {
  sources: ["./value.c"],
  exports: {
    version: { symbol: "mylib_version_value", type: "value" }
  }
};
```

```c
jayess_value *mylib_version_value(void) {
  return jayess_value_from_number(7);
}
```

Editor-friendly placeholder exports are allowed so normal JS tooling does not
flag imports from `*.bind.js`:

```js
const f = () => {};
export const add = f;

export default {
  sources: ["./math.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
```

Jayess usage:

```javascript
import { add } from "./math.bind.js";

function main(args) {
  console.log(add(3, 4));
  return 0;
}
```

## C++

For C++ wrappers, expose a C ABI entrypoint:

```cpp
extern "C" jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
  return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
```

## Runtime Headers

Low-level native bindings use:

- [jayess_runtime.h](/home/remote-desktop/Desktop/it/jayess/jayess-go/runtime/jayess_runtime.h)

## Boundary Safety And Ownership

Bindings receive and return boxed `jayess_value *` values. Conversion and
ownership rules matter:

- numeric conversion uses helpers like `jayess_value_to_number(...)` and
  `jayess_value_from_number(...)`
- strings borrowed from `jayess_value_as_string(...)` should only be used during
  the current native call
- long-lived string storage should use `jayess_value_to_string_copy(...)` and
  release memory with `jayess_string_free(...)`
- byte buffers copied out with `jayess_value_to_bytes_copy(...)` or
  `jayess_expect_bytes_copy(...)` must be freed by the binding
- opaque handles can be exposed with `jayess_value_from_native_handle(...)` or
  `jayess_value_from_managed_native_handle(...)`
- managed handles can be closed safely with
  `jayess_value_close_native_handle(...)`

Type validation helpers are available for safer bindings:

- `jayess_expect_object(...)`
- `jayess_expect_bytes_copy(...)`
- `jayess_expect_native_handle(...)`

Bindings can raise Jayess-visible errors directly:

- `jayess_throw_error(...)`
- `jayess_throw_type_error(...)`
- `jayess_throw_named_error(...)`

## Shared Native Sources Across Multiple Bindings

Multiple `*.bind.js` files can reference the same helper source file.

- if two bindings list the same source file path, Jayess deduplicates that
  native source during build
- this is intended for shared helpers such as `shared.c` used by multiple
  binding entrypoints
- shared helpers should live at one stable path and be referenced from each
  binding that needs them
- if the same helper code is copied into different files at different paths,
  Jayess treats those as distinct native sources and normal linker symbol rules
  still apply

In practice:

- share helper implementations by path
- keep exported binding entrypoints in their own source files
- avoid duplicated compiled copies of the same helper under different filenames

## Notes

- Native interop is explicit. Jayess is not a JavaScript engine and does not auto-load Node APIs.
- Manual binding files are the intended bridge to C and C++ libraries.
- Keep bindings small and stable; put library-specific adaptation logic in the binding layer.
