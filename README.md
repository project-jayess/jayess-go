# jayess-go

Go compiler skeleton for the Jayess programming language.

## Status

This repository now contains an MVP compiler pipeline:

1. lexing
2. parsing
3. semantic validation
4. lifetime analysis placeholder
5. lowering to a minimal IR
6. LLVM IR text emission

The current source subset is intentionally small:

```javascript
function main(args) {
  const delay = 500;
  var total = 10.5 + 2 * 3;
  console.log(total);
  console.log(args[0]);
  sleep(delay);
  var name = readLine("What is your name? ");
  console.log(name);
  readKey("Press any key to continue");
  return 0;
}
```

Documentation index: [docs/index.md](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/index.md)
LLVM/backend details are documented in [docs/llvm-backend.md](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/llvm-backend.md).

Console output is documented in [docs/console.md](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/console.md).
`print(...)` still works, but it is deprecated in favor of `console.log(...)`.
Async functions, Promise helpers, timers, and async file I/O are documented in [docs/async.md](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/async.md).

Jayess variable declarations are:
- `var` for mutable block-scoped bindings
- `const` for immutable bindings

`let` is not supported.
`public` is not supported. Module visibility is controlled by `export`.
Top-level `private` is not supported. Class privacy uses JavaScript-style `#members`.

Classes are available in MVP form:
- `class Name { ... }`
- `export class Name { ... }`
- `export default class Name { ... }`
- `new Name(...)`

Current class scope is intentionally small:
- constructor
- instance field initializers
- instance methods
- `#private` instance fields and methods
- static fields
- static methods
- `static #private` fields and methods
- `this`
- method calls like `obj.method()`
- single inheritance with `extends`
- `super(...)` in constructors
- `super.method()` in instance/static methods
- `super.property` access for inherited properties

Manual native binding interop is available:
- `import { add } from "./native/math.bind.js";`
- `*.bind.js` binding files declare native sources, include dirs, flags, and exported symbols
- `*.bind.js` can also apply per-platform source/include/flag overrides through `platforms.linux`, `platforms.darwin`, and `platforms.windows`
- imported native binding calls receive boxed Jayess runtime values and return boxed Jayess runtime values
- binding implementations should include [jayess_runtime.h](/C:/Users/ncksd/Documents/it/jayess/jayess-go/runtime/jayess_runtime.h)
- C++ wrappers should export C ABI entrypoints with `extern "C"`
- bindings can use public runtime helpers for objects/arrays, byte buffers, and opaque native handles

The repo now also ships a native package example surface through `node_modules`:
- `import { parseRequest, formatResponse } from "@jayess/httpserver";`
- `parseRequest(...)` is backed by PicoHTTPParser through a Jayess native wrapper module
- `formatResponse(...)` currently forwards to the built-in HTTP response formatter

Manual bindings are declared in `*.bind.js`:

```js
const f = () => {};
export const add = f;

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

Platform-specific native flags can be expressed directly in the same binding:

```js
export default {
  sources: ["./webview.cpp"],
  ldflags: [],
  platforms: {
    linux: { ldflags: ["-lgtk-3", "-lwebkit2gtk-4.1"] },
    darwin: { ldflags: ["-framework", "Cocoa", "-framework", "WebKit"] },
    windows: { ldflags: ["-lole32", "-lcomctl32"] }
  },
  exports: {
    createWindowNative: { symbol: "jayess_webview_create_window", type: "function" }
  }
};
```

The named placeholder export keeps editors/formatters/linting happy for imports like:

```js
import { add } from "./math.bind.js";
```

while the compiler still treats `export default` as the source of truth for the native binding metadata.
`*.bind.js` files are native binding modules, not normal Jayess source modules,
so Jayess only supports named imports from them. Bare, default, and namespace
imports are rejected.

Bindings can also expose imported module values with `type: "value"`. Those are
initialized by calling a zero-argument native getter during module startup.

Useful binding-side runtime helpers now include:
- `jayess_value_as_object(...)` / `jayess_value_as_array(...)`
- `jayess_expect_object(...)` / `jayess_expect_array(...)` / `jayess_expect_string(...)`
- `jayess_value_from_bytes_copy(...)` / `jayess_value_to_bytes_copy(...)`
- `jayess_value_to_string_copy(...)`
- `jayess_expect_bytes_copy(...)`
- `jayess_string_free(...)`
- `jayess_bytes_free(...)`
- `jayess_value_from_native_handle(...)` / `jayess_value_as_native_handle(...)`
- `jayess_value_from_managed_native_handle(...)` / `jayess_value_close_native_handle(...)`
- `jayess_expect_native_handle(...)`
- `jayess_throw_error(...)` / `jayess_throw_type_error(...)` / `jayess_throw_named_error(...)`

Recommended binding authoring surface:
- include `jayess_runtime.h`
- write plain C ABI entrypoints like `jayess_value *mylib_add(jayess_value *a, jayess_value *b)`
- use the low-level `jayess_value_*`, `jayess_object_*`, `jayess_array_*`, `jayess_expect_*`, and `jayess_throw_*` helpers directly

Ownership rules for wrapper authors:
- `jayess_value_from_*` returns boxed Jayess values owned by the Jayess runtime
- `jayess_value_as_string(...)` is a borrowed view for immediate use only; do not store it in long-lived native state
- `jayess_value_to_string_copy(...)` returns a fresh heap copy for long-lived wrapper state; free it with `jayess_string_free(...)`
- `jayess_value_to_bytes_copy(...)` returns a fresh heap copy; free it with `jayess_bytes_free(...)` when done
- `jayess_value_from_native_handle(...)` is for unmanaged opaque handles
- `jayess_value_from_managed_native_handle(...)` is for explicitly closable resources; use `jayess_value_close_native_handle(...)` to run the registered finalizer once
- managed native handles become invalid after close, and repeated close calls are safe
- wrapper authors should keep borrowed Jayess pointers only for the duration of the current native call; store copied strings/bytes or managed handles instead

CLI commands currently available:
- `jayess <input.js>` or `jayess compile <input.js>`
- `jayess run <input.js> [args...]`
- `jayess test [path|file.test.js]`
- `jayess init [directory]`

Useful compiler/backend flags:
- `--emit=llvm|bc|obj|lib|shared|exe`
- `--opt=O0|O1|O2|O3|Oz`
- `--cpu=<name>`
- `--feature=<flag>` (repeatable or comma-separated, such as `+sse2` or `-avx`)
- `--reloc=pic|pie|static`
- `--code-model=small|medium|large|kernel`

Examples:
- `jayess --emit=bc --target=linux-x64 main.js`
- `jayess --emit=obj --target=linux-x64 main.js`
- `jayess --emit=lib --target=linux-x64 main.js`
- `jayess --emit=shared --target=linux-x64 main.js`
- `jayess --emit=exe --opt=O2 --cpu=native main.js`
- `jayess --emit=obj --feature=+sse2 --feature=-avx --reloc=pic --code-model=small main.js`

`jayess test` discovers `*.test.js` files, compiles each test for the host target, runs the resulting native executable, and treats exit code `0` as pass.

Local relative imports are also supported in MVP form:

```javascript
import { add, twice } from "./lib/math.js";

function main(args) {
  var value = add(3, 4);
  console.log(twice(value));
  return value;
}
```

Imported functions must be exported explicitly:

```javascript
export function add(a, b) {
  return a + b;
}
```

Class example:

```javascript
export class Counter {
  value = 1;
  #secret = 9;

  constructor(step) {
    this.step = step;
  }

  total() {
    return this.value + this.step + this.#secret;
  }
}
```

Current import support is limited to relative file imports:
- `import "./utils.js";`
- `import { add, twice } from "./lib/math.js";`
- `import { add as sum } from "./lib/math.js";`
- `import thing from "./lib/module.js";`
- `import thing, { add as sum } from "./lib/module.js";`
- `import * as ns from "./lib/module.js";`
- `import { add } from "@demo/math";`
- `import thing from "@demo/math";`
- `import * as ns from "@demo/math";`

Current export support is limited to:
- `export function name(...) { ... }`
- `export const NAME = value;`
- `export var name = value;`
- `export default function name(...) { ... }`
- `export default <expression>;`
- `export { local, other as renamed };`
- `export { local as renamed } from "@demo/math";`
- `export * from "./more.js";`
- `export * as math from "./more.js";`

Native binding import example:

```javascript
import { add, greet } from "./native/math.bind.js";

function main(args) {
  console.log(add(3, 4));
  console.log(greet("Kimchi"));
  return 0;
}
```

Example `math.bind.js`:

```javascript
const f = () => {};
export const add = f;
export const greet = f;

export default {
  sources: ["./math.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "jayess_add", type: "function" },
    greet: { symbol: "jayess_greet", type: "function" }
  }
};
```

Example C binding implementation:

```c
#include "jayess_runtime.h"

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
  return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
```

## Build

```bash
$env:GOCACHE="$PWD\.gocache"
$env:GOFLAGS="-buildvcs=false"
go build -o build\windows\jayess.exe .\cmd\jayess
```

## Run

```bash
go run ./cmd/jayess --target=host --emit=llvm -o build/hello.ll examples/hello.js
go run ./cmd/jayess --target=host --emit=obj -o build/hello.o examples/hello.js
go run ./cmd/jayess --target=host --emit=lib -o build/libhello.a examples/hello.js
go run ./cmd/jayess --target=host --emit=shared -o build/libhello.so examples/hello.js
go run ./cmd/jayess --target=host --emit=llvm -o build/import.ll examples/import.js
```

This emits LLVM IR text to `build/hello.ll`.
Use `--opt=O0`, `O1`, `O2`, `O3`, or `Oz` to control clang optimization for object and executable builds. `O0` is the default.

To build a native executable once `clang` is installed and on `PATH`:

```bash
go run ./cmd/jayess --target=host --emit=exe -o build/hello.exe examples/hello.js
go run ./cmd/jayess --target=host --emit=exe --opt=O2 -o build/hello-opt.exe examples/hello.js
```

Warnings are shown by default. Use `--warnings=none` to suppress them temporarily, or `--warnings=error` to fail the build when warnings are emitted:

```bash
go run ./cmd/jayess --warnings=error --target=host --emit=llvm -o build/hello.ll examples/hello.js
```

When treating warnings as errors, specific warning categories can be allowed during migrations:

```bash
go run ./cmd/jayess --warnings=error --allow-warning=deprecation --target=host --emit=llvm -o build/hello.ll examples/hello.js
```

On Windows, the default is now native executable output, so this also works:

```bash
.\build\windows\jayess.exe -o .\examples\build\hello.exe .\examples\hello.js
```

That defaults to `build/hello.exe`.

The current executable path is:

1. Jayess source
2. Jayess LLVM IR text
3. `clang`
4. native executable
