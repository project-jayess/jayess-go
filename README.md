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
- `import { add } from "./native/math.js";`
- binding files are normal `.js` files that import `bind` from `"ffi"` and `export default bind(...)`
- binding files declare native sources, include dirs, flags, and exported symbols
- binding files can also apply per-platform source/include/flag overrides through `platforms.linux`, `platforms.darwin`, and `platforms.windows`
- imported native binding calls receive boxed Jayess runtime values and return boxed Jayess runtime values
- binding implementations should include [jayess_runtime.h](/C:/Users/ncksd/Documents/it/jayess/jayess-go/runtime/jayess_runtime.h)
- C++ wrappers should export C ABI entrypoints with `extern "C"`
- bindings can use public runtime helpers for objects/arrays, byte buffers, and opaque native handles

The repo now also ships a native package example surface through `node_modules`:
- `import { parseRequest, formatResponse } from "@jayess/httpserver";`
- `parseRequest(...)` is backed by PicoHTTPParser through a Jayess native wrapper module
- `formatResponse(...)` currently forwards to the built-in HTTP response formatter

Manual bindings are declared in normal `.js` files by exporting `bind(...)` as
the default export:

```js
import { bind } from "ffi"

const f = () => {};
export const add = f;

export default bind({
  sources: ["./math.c", "./helper.c"],
  includeDirs: ["./include"],
  libraryDirs: ["./vendor/lib"],
  sharedLibraries: ["mylib", "./vendor/libhelper.so"],
  licenseFiles: ["./vendor/LICENSE.helper"],
  cflags: ["-DMY_BINDING=1"],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
});
```

Platform-specific native flags can be expressed directly in the same binding:

```js
import { bind } from "ffi";

export default bind({
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
});
```

The named placeholder export keeps editors/formatters/linting happy for imports like:

```js
import { add } from "./math.js";
```

while the compiler treats `export default bind(...)` as the source of truth for
native binding metadata. A `.js` file without that default export remains a
normal Jayess source module. Binding modules support named imports for exported
native symbols; bare, default, and namespace imports are rejected.
The object passed to `bind(...)` is extracted as a literal binding manifest:
`sources`, `includeDirs`, `libraryDirs`, `sharedLibraries`, `cflags`,
`ldflags`, `licenseFiles`, `platforms`, and `exports` must be declared with
object, array, and string literals. `sharedLibraries` can name a library such as
`"sqlite3"` (linked as `-lsqlite3`) or point at a prebuilt shared library file
such as `"./vendor/lib/libsqlite3.so"`. `licenseFiles` lists project-provided
license or notice files that should be copied into app distributions.
When a module imports one of these `.js` binding files, the resolver parses the
target file, validates the requested named imports against `exports`, and feeds
the extracted manifest into native build planning.

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

Useful compiler/backend flags:
- `--emit=llvm|bc|obj|lib|shared|exe`
- `--opt=O0|O1|O2|O3|Oz`
- `--cpu=<name>`
- `--feature=<flag>` (repeatable or comma-separated, such as `+sse2` or `-avx`)
- `--reloc=pic|pie|static`
- `--code-model=small|medium|large|kernel`

Examples:
- `jayess --emit=llvm --target=linux-x64 main.js`
- `jayess --emit=obj --target=linux-x64 main.js`
- `jayess --emit=shared --target=linux-x64 main.js`

Current backend support includes LLVM IR, object-file, and shared-library
emission from LLVM IR.
Jayess also reserves `import ... from "llvm"` as the Jayess-facing compiler
construction package. Its current model exposes API groups for contexts,
modules, builders, types, values, targets, object emission, and linking, backed
by the internal LLVM C API and lld shim work. The Go package now includes a
minimal IR builder model for modules, functions, basic blocks, `i32`/`void`
types, integer constants, and return instructions. This is the public surface
intended to let future Jayess code build compiler pieces without importing Go
internals.
The CLI maps `--emit=obj` to LLVM C API object emission when the compiler is
built with `-tags jayess_llvmc`; otherwise it writes temporary IR under
`temp/jayess-build` and invokes a target-specific `clang -c` command.
The selected target controls default shared-library naming and linker mode:
Linux emits `.so` with `-shared`, macOS emits `.dylib` with `-dynamiclib`,
and Windows emits `.dll` with `-shared`. The CLI maps `--emit=shared` to a
target-specific `clang` command unless both internal LLVM object emission and
internal lld linking are enabled. External tools are resolved from
`JAYESS_TOOLCHAIN`, `tools/<target>/bin` beside the `jayess` executable,
working-directory `tools/<target>/bin`, built LLVM directories under
`refs/llvm-project`, then `PATH`.
Missing tools are reported before temporary build files are written.
Builds made with `-tags jayess_llvmc` can use the LLVM C API through cgo to emit
the object file internally for `--emit=obj` and `--emit=shared`. When an `lld`
C++ shim is linked in with `-tags jayess_lld` as well, Jayess
routes final shared-library linking through `lld::lldMain` in process instead of
an external `clang` driver. That build mode expects lld/LLVM libraries from a
built `refs/llvm-project` tree or equivalent linker/library paths. A cloned
LLVM source tree alone is not enough because LLVM generates headers such as
`llvm/Config/abi-breaking.h` during its CMake configure step.

Distribution packages are built with the separate `jayess-dist` helper:

```bash
go run ./cmd/jayess-dist --platform=linux-x64 --version=0.1.0
```

The package is written under `dist/<platform>/jayess-<version>-<platform>/`.
Its layout keeps `jayess` at the package root and copies bundled LLVM tools into
`tools/bin`, which is one of the compiler's automatic toolchain search
locations. By default the package builder tries to include `clang`,
`clang++`, `lld`, `ld.lld`, `llvm-as`, and `llc` from
`refs/llvm-project/build/bin`, plus runtime LLVM libraries such as `libLLVM`
from `refs/llvm-project/build/lib`. LLVM, Clang, lld, and LLVM third-party
notice files are copied into `licenses/`. Missing tools are reported as packaging
errors by default so release packages do not accidentally ship without Clang or
lld. Use `--strict-tools=false` only for local partial-package checks. Release
packages should be produced after LLVM has been configured with Clang and lld
and built with the needed tool targets.

Bundled LLVM, Clang, and lld cover Jayess IR assembly, object emission, and
linking mechanics, but they do not replace every vendor platform SDK:

- Linux targets expect the target sysroot, C runtime startup files, system
  libraries, and native package headers/libraries to come from the build
  machine, a configured cross sysroot, or explicit binding paths.
- macOS targets require Apple's SDK and platform libraries from Xcode or the
  Command Line Tools for real system-framework linking. Jayess can select the
  macOS target and produce `.dylib` link commands, but it does not redistribute
  the Apple SDK.
- Windows targets require a Windows C runtime/import-library environment, such
  as MSVC/Windows SDK libraries or a MinGW-w64 sysroot, for final executable or
  DLL links. Jayess packages may include LLVM tools, but they do not bundle
  Microsoft SDK files.
- Native binding manifests should declare project-owned include/library paths
  through `includeDirs`, `libraryDirs`, `sharedLibraries`, and `licenseFiles`.
  Platform SDK files should remain external unless the license explicitly allows
  redistribution in the app distribution.

The compiler SDK does not bundle SDL, GLFW, raylib, curl, GTK, or other native
package refs by default. Developers bind those libraries from their own project
or installed SDKs. When an application is packaged, Jayess can copy the bound
runtime shared libraries into that application's distribution folder.

Binding-owned distribution inputs should stay with the project, for example:

```text
my-app/
  native/
    mylib.js
    include/
    lib/
      libmylib.so
```

For a full local toolchain package, configure LLVM with both Clang and lld:

```bash
cmake -S refs/llvm-project/llvm -B refs/llvm-project/build \
  -DLLVM_ENABLE_PROJECTS="clang;lld" \
  -DLLVM_TARGETS_TO_BUILD="X86;AArch64" \
  -DCMAKE_BUILD_TYPE=Release
cmake --build refs/llvm-project/build --target clang clang++ lld llvm-as llc
```

The dist helper also writes a compressed artifact beside the package directory:
Linux/macOS packages use `.tar.gz`, Windows packages use `.zip`, and each
archive gets a `.sha256` checksum file.

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
import { add, greet } from "./native/math.js";

function main(args) {
  console.log(add(3, 4));
  console.log(greet("Kimchi"));
  return 0;
}
```

Example `math.js` native binding module:

```javascript
import { bind } from "ffi";

const f = () => {};
export const add = f;
export const greet = f;

export default bind({
  sources: ["./math.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "jayess_add", type: "function" },
    greet: { symbol: "jayess_greet", type: "function" }
  }
});
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

## Build distributable Jayess SDK

Build the compiler and package it with the bundled toolchain under `dist/`:

```bash
go run ./cmd/jayess-dist --platform=linux-x64 --version=0.1.0
```

The output is:

```text
dist/linux-x64/jayess-0.1.0-linux-x64/
  jayess
  README.txt
  licenses/
  tools/
    bin/
    lib/
dist/linux-x64/jayess-0.1.0-linux-x64.tar.gz
dist/linux-x64/jayess-0.1.0-linux-x64.tar.gz.sha256
```

Use this command for release packages so Jayess ships as one SDK containing the
compiler, bundled LLVM/Clang/lld tools, LLVM runtime libraries, and license
notices.

## Build distributable Jayess apps

Applications that use native binding packages may need runtime shared libraries
beside the executable. Jayess models this as an app distribution step: static
libraries are linked into the executable, while `.so`, `.dylib`, and `.dll`
runtime libraries are copied into the app output directory. Build an app
distribution from an already-built executable with either CLI form:

```bash
go run ./cmd/jayess compile --target=linux-x64 --emit=dist --executable build/my-app -o dist/my-app src/main.js
go run ./cmd/jayess package --target=linux-x64 --executable build/my-app -o dist/my-app src/main.js
```

The app distribution layout is:

```text
dist/my-app/
  my-app
  libglfw.so
  libcurl.so
  licenses/
    LICENSE.glfw
    LICENSE.curl
```

On Windows, required DLLs are copied beside `my-app.exe`. This keeps the built
app runnable without asking users to install the native package libraries
separately. The runtime asset resolver uses the native binding build plan:
`libraryDirs` tells Jayess where to search, and `sharedLibraries` tells it which
shared libraries must be shipped with the executable. Static libraries passed
through `ldflags` or non-shared archive paths are treated as link-time inputs and
are not copied into the app distribution. Path-style shared library entries such
as `"./vendor/lib/libhelper.so"` are resolved relative to the binding module and
copied directly. Named shared libraries such as `"glfw"` are searched in
`libraryDirs` using the target platform's shared-library names. Missing runtime
shared libraries are reported as diagnostics and the app distribution is not
created. Files listed in `licenseFiles` are copied into `licenses/` beside the
packaged app.
This is how projects that bind SDL, GLFW, raylib, curl, GTK, or other native
libraries ship everything needed by the end user without requiring separate
library installation.

## Run

```bash
go run ./cmd/jayess --target=host --emit=llvm -o build/hello.ll examples/hello.js
go run ./cmd/jayess --target=host --emit=shared -o build/libhello.so examples/hello.js
go run ./cmd/jayess --target=host --emit=llvm -o build/import.ll examples/import.js
```

`--emit=llvm` emits LLVM IR text to `build/hello.ll`.
`--emit=shared` builds the shared library at `build/libhello.so`.
Use `--opt=O0`, `O1`, `O2`, `O3`, or `Oz` to carry optimization intent through the command surface. `O0` is the default.

The CLI parses `bc`, `obj`, `lib`, and `exe` emit modes for compatibility with
the older command surface, but this Go backend currently executes only `llvm`
and shared-library emission.

The current CLI path is:

1. Jayess source
2. parser validation
3. MVP LLVM IR lowering
4. LLVM IR output or shared-library toolchain execution
