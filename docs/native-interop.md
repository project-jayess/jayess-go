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

Native function exports may also declare `borrowsArgs: true` when the native
callee only reads Jayess arguments during the current call and does not retain
or consume them after return. That allows the current cleanup slice to release
proven-transient arguments safely after the call boundary. This is
intentionally opt-in and must not be used for native functions that close,
store, or otherwise take ownership of passed values.

The listed native sources are compiled and linked into the final executable through the native toolchain.

This same binding model is already strong enough to wrap platform-style audio
APIs directly. The current test coverage includes a manual SDL3 audio binding
through real upstream `refs/SDL/include` headers plus a small native shim/stub,
covering init, driver queries, default-playback open, format inspection, and
managed handle cleanup.

Managed native-handle finalizers are also now exercised through scope-exit
cleanup of non-escaping dynamic locals on the current lifetime path. The
proved boundary is normal block exit, early return, loop `break` /
`continue`, and exception-path unwinding through `throw` / `try` for
non-escaping dynamic locals in the currently implemented slice:
- block-scoped `let` / `const` dynamic locals
- function-scoped non-loop `var` dynamic locals

The current executable proof also covers nested loop/block/`try` control-flow
so those cleanups are neither skipped nor delivered twice on the tested paths.
That same proof now also covers manual managed-handle close inside an eligible
scope: once a handle is closed explicitly, the later scope-exit cleanup does
not finalize it again.

The current compiler path also cleans up discarded direct Jayess call results
when the callee is proven to return a fresh value. That proof is intentionally
conservative: it covers obviously fresh return expressions such as literals and
fresh scalar computations, now includes discarded `bind` / `call` / `apply`
temporaries on the current proved slice, and does not try to free discarded
results from alias-returning helpers like DOM tree mutators.

That same cleanup path now also covers obviously fresh discarded temporaries at
statement exit for object literals, array literals, template literals, string
concat/dynamic add results, bitwise results, fresh dynamic/bigint comparison
operands/results, fresh conditional/nullish results, comma expressions with
fresh discarded operands/results, plus transient `typeof` and `instanceof`
operands/results on the currently proved slice.

There is also now an opt-in Linux ASAN/LSAN probe for that current cleanup
slice. It recompiles the cleanup-probe package with address sanitizer enabled
and exercises block exit, function-scoped non-loop `var`, borrowed parameter
cleanup, function-scoped `var` redeclaration inside loops, early return,
`throw`, loop `break` / `continue`, nested control flow, and manual close in
one executable. That probe is intended to catch concrete double-free,
use-after-free, and leak regressions on the currently implemented
managed-handle path without claiming the broader unchecked lifetime rows are
fully solved yet. The current probe is now green for that implemented slice,
including the borrowed native-argument boundary above and eligible loop-`var`
redeclaration cleanup, plus discarded fresh
direct Jayess call results plus
fresh-return constructor/function/`bind`-`call`-`apply`/object/array/template/concat/bitwise/comparison/logical/unary-`!`/conditional/nullish/comma/member/index/optional-member/optional-index/`typeof`/`instanceof`
temporaries plus discarded runtime `is`-check and fresh switch
case/discriminant temporaries at statement exit. It also now covers the narrow
scalar `[1, 2].forEach((x) => 0)` path when the callback has zero bound
arguments: the discarded fresh receiver array is released with shallow
container cleanup after iteration. It also now covers discarded scalar
`[1, 2].map((x) => x + 1)` and `[1, 2].filter((x) => x > 0)` with zero-bound-arg
callbacks: the discarded fresh receiver arrays are shallow-cleaned and the
fresh result arrays are reclaimed at statement exit. It also now covers the
narrow scalar discarded `[1, 2].find((x) => false)` path with a zero-bound-arg
callback: the discarded fresh receiver array is shallow-cleaned and the
no-match `undefined` result path is ASAN/LSAN clean. It also now covers the
narrow scalar `[1, 2].forEach(boundOffset.bind(null, 1))` path: one pre-bound
argument uses the direct two-argument callback path instead of boxed arg-array
apply, and the discarded fresh receiver array is ASAN/LSAN clean. That probe
also now covers one-bound scalar `[1, 2].map(boundOffset.bind(null, 1))`,
`[1, 2].filter(boundGreaterThan.bind(null, 0))`, and
`[1, 2].find(boundEquals.bind(null, 9))`: one pre-bound argument uses the
direct two-argument callback path, the discarded fresh receiver arrays are
clean, the `map` / `filter` result arrays are reclaimed at statement exit, and
the no-match `find` path is ASAN/LSAN clean. It also now covers two-bound
scalar `[1, 2].forEach(boundPairSum.bind(null, 1, 2))`,
`[1, 2].map(boundPairSum.bind(null, 1, 2))`,
`[1, 2].map(boundPairSum.bind(null, 10, 10))`,
`[1, 2].filter(boundBetween.bind(null, 0, 3))`, and
`[1, 2].find(boundTripleEquals.bind(null, 4, 5))`: two pre-bound arguments
use the direct three-argument callback path, the discarded fresh receiver
arrays are clean, the `map` / `filter` result arrays are reclaimed at
statement exit, and the no-match `find` path is ASAN/LSAN clean. It also now
covers three-bound scalar `[1, 2].forEach(boundTripleSum.bind(null, 1, 2, 3))`,
`[1, 2].map(boundTripleSum.bind(null, 1, 2, 3))`,
`[1, 2].map(boundTripleSum.bind(null, 10, 10, 10))`,
`[1, 2].filter(boundWindow.bind(null, 0, 1, 3))`, and
`[1, 2].find(boundQuadEquals.bind(null, 3, 4, 5))`: three pre-bound arguments
use the direct four-argument callback path, the discarded fresh receiver
arrays are clean, the `map` / `filter` result arrays are reclaimed at
statement exit, and the no-match `find` path is ASAN/LSAN clean. It also now
covers four-bound scalar `[1, 2].forEach(boundQuadSum.bind(null, 1, 2, 3, 4))`,
`[1, 2].map(boundQuadSum.bind(null, 1, 2, 3, 4))`,
`[1, 2].filter(boundOuterWindow.bind(null, 0, 1, 4, 3))`, and
`[1, 2].find(boundQuintEquals.bind(null, 3, 4, 5, 6))`: four pre-bound
arguments use the direct five-argument callback path, the discarded fresh
receiver arrays are clean, the `map` / `filter` result arrays are reclaimed at
statement exit, and the no-match `find` path is ASAN/LSAN clean. Those
`forEach` rows are still not being used to claim the broader global `9.5`
safety rows, but the proved slice is stronger now: discarded non-immortal
numeric callback results are clean for a direct callback
(` [1, 2].forEach(largeOffset)`), one-/two-/three-bound callbacks
(` [1, 2].forEach(boundOffset.bind(null, 20))`,
` [1, 2].forEach(boundPairSum.bind(null, 10, 10))`,
` [1, 2].forEach(boundTripleSum.bind(null, 10, 10, 10))`), a four-bound callback
(` [1, 2].forEach(boundQuadSum.bind(null, 4, 4, 4, 4))`), a five-bound callback
(` [1, 2].forEach(boundQuintSum.bind(null, 1, 1, 1, 1, 16))`), a six-bound callback
(` [1, 2].forEach(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16))`), a seven-bound callback
(` [1, 2].forEach(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16))`), an eight-bound callback
(` [1, 2].forEach(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16))`), a nine-bound callback
(` [1, 2].forEach(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), a ten-bound callback
(` [1, 2].forEach(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), an eleven-bound callback
(` [1, 2].forEach(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), and a twelve-bound callback
(` [1, 2].forEach(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) under the opt-in
ASAN/LSAN probe.

The discarded `map` slice is stronger than `forEach` here: non-immortal numeric
callback returns are now proven clean for a direct callback
(` [1, 2].map(largeOffset)`), one-/two-/three-bound callbacks
(` [1, 2].map(boundOffset.bind(null, 20))`,
` [1, 2].map(boundPairSum.bind(null, 10, 10))`,
` [1, 2].map(boundTripleSum.bind(null, 10, 10, 10))`), and a four-bound
callback (` [1, 2].map(boundQuadSum.bind(null, 4, 4, 4, 4))`), a
five-bound callback (` [1, 2].map(boundQuintSum.bind(null, 1, 1, 1, 1, 16))`),
and a six-bound callback (` [1, 2].map(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16))`),
plus a seven-bound callback (` [1, 2].map(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16))`),
an eight-bound callback (` [1, 2].map(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16))`),
a nine-bound callback (` [1, 2].map(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16))`),
a ten-bound callback (` [1, 2].map(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`),
an eleven-bound callback (` [1, 2].map(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`),
and a twelve-bound callback (` [1, 2].map(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`)
under the opt-in ASAN/LSAN probe. The bound cases needed a runtime ownership
fix so bound functions now own cloned primitive bound args instead of
borrowing the temporary bind-array wrapper.

The discarded `filter` truthiness-result slice now matches that same non-immortal
coverage: direct (` [1, 2].filter(largeOffset)`), one-bound
(` [1, 2].filter(boundOffset.bind(null, 20))`), two-bound
(` [1, 2].filter(boundPairSum.bind(null, 10, 10))`), three-bound
(` [1, 2].filter(boundTripleSum.bind(null, 10, 10, 10))`), and four-bound
(` [1, 2].filter(boundQuadSum.bind(null, 4, 4, 4, 4))`), a five-bound
(` [1, 2].filter(boundQuintSum.bind(null, 1, 1, 1, 1, 16))`) callback, a six-bound
(` [1, 2].filter(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16))`) callback, a seven-bound
(` [1, 2].filter(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16))`) callback, an eight-bound
(` [1, 2].filter(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16))`) callback, a nine-bound
(` [1, 2].filter(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) callback, a ten-bound
(` [1, 2].filter(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) callback, an eleven-bound
(` [1, 2].filter(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) callback, and a twelve-bound
(` [1, 2].filter(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) callback are now
clean under the opt-in ASAN/LSAN probe. The needed fix was to release the boxed
callback result immediately after truthiness conversion inside the `filter`
helper path.

The discarded `find` slice now covers both the callback-result truthiness path
and alias-returning match results for direct (` [1, 2].find(largeOffset)`),
one-bound (` [1, 2].find(boundOffset.bind(null, 20))`), two-bound
(` [1, 2].find(boundPairSum.bind(null, 10, 10))`), three-bound
(` [1, 2].find(boundTripleSum.bind(null, 10, 10, 10))`), four-bound
(` [1, 2].find(boundQuadSum.bind(null, 4, 4, 4, 4))`), five-bound
(` [1, 2].find(boundQuintSum.bind(null, 1, 1, 1, 1, 16))`), six-bound
(` [1, 2].find(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16))`), seven-bound
(` [1, 2].find(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16))`), eight-bound
(` [1, 2].find(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16))`), nine-bound
(` [1, 2].find(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), ten-bound
(` [1, 2].find(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), an eleven-bound
(` [1, 2].find(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`), and a twelve-bound
(` [1, 2].find(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16))`) callbacks under the
opt-in ASAN/LSAN probe. The needed fix there was the same ownership boundary as
`filter`: release the boxed callback result immediately after truthiness
conversion, while leaving the returned matching element aliased and unowned.

Array callback scaffolding is still outside the broader global claims. The
current probe does not yet claim the full callback ownership surface under
ASAN/LSAN, because broader bound-callback helper paths beyond the currently
proved zero-/one-/two-/three-/four-/five-/six-/seven-/eight-/nine-/ten-/eleven-/twelve-bound scalar slices still need a
deeper model than the current statement-exit cleanup rules. After moving cleanup
into the fast callback-result branches themselves and releasing `filter` / `find`
callback results immediately after truthiness conversion, the remaining hard
ownership gaps in this area are now beyond the currently proved helper shapes
rather than the direct scalar non-immortal `forEach` / `map` / `filter` / `find`
result paths.

Discarded `new Class(...)` is no longer all-or-nothing. Constructors that
return the generated `__self` object or a fresh alternate object/value are now
covered by the ASAN/LSAN probe. Constructors that return existing aliased
objects/values are still outside the broad global lifetime claims, because
constructor results are not globally fresh by default under the current
language semantics.

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

The current manual binding model is already proven against multiple audio-style
native APIs, including a real SDL3-header-backed binding path, a stub-backed
OpenAL-style device/context path, and a real vendored-source miniaudio null-backend
path, plus a real PortAudio-header-backed stream/device shape. That means the interop
surface is not limited to one particular C library shape.

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
