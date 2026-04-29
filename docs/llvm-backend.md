# LLVM Backend

This document describes the current Jayess LLVM backend contract as it exists in
this repository today. It is intentionally about the generated IR and native
toolchain boundary, not just the language surface.

## Entry and calling convention

Jayess uses the default C calling convention in emitted LLVM IR.

The generated entrypoints are:

- source `main(...)` lowers to `define double @jayess_user_main(...)`
- the executable entrypoint is `define i32 @main(i32 %argc, ptr %argv)`

The wrapper `@main` is responsible for:

- calling `@jayess_init_globals()`
- building Jayess argv values with `@jayess_make_args(...)` when needed
- calling `@jayess_user_main(...)`
- running queued async work via `@jayess_run_microtasks()`
- shutting down runtime state via `@jayess_runtime_shutdown()`
- reporting uncaught runtime exceptions with `@jayess_report_uncaught_exception()`

Non-entry Jayess functions are lowered as pointer-based runtime functions:

- parameters are passed as `ptr`
- return values are passed as `ptr`

That keeps the language/runtime boundary uniform for boxed Jayess values.

## Data layout assumptions

Jayess currently emits a target triple but does not emit an explicit
`target datalayout` string. The backend relies on LLVM/clang inferring layout
from the selected target triple.

On the LLVM side:

- boxed Jayess values are opaque `ptr`
- objects, arrays, functions, handles, and runtime-managed values stay opaque
- runtime structure layout is defined in the C runtime, not in emitted LLVM IR

The only special-case ABI path today is source `main`, which returns a `double`
so the entry wrapper can convert it to the native process exit code with
`fptosi ... to i32`.

## Exceptions and errors

Jayess exceptions are runtime-managed, not LLVM EH-managed.

Generated IR uses runtime helpers such as:

- `@jayess_throw(...)`
- `@jayess_has_exception()`
- `@jayess_take_exception()`
- `@jayess_report_uncaught_exception()`
- `@jayess_push_call_frame(...)`
- `@jayess_pop_call_frame()`

The backend does not currently use LLVM exception handling constructs such as:

- `invoke`
- `landingpad`
- `personality`

Instead, exception state is tracked by the runtime and checked explicitly in
generated control flow.

## Debug information and source mapping

Jayess now emits LLVM debug metadata for lowered functions when source
line/column information is present in the IR module.

Current emitted metadata includes:

- `source_filename = ...`
- `!llvm.dbg.cu`
- `!DICompileUnit`
- `!DIFile`
- `!DISubprogram`
- `!DILocation` on emitted call sites inside lowered Jayess functions

On the native-build side, object/executable/shared-library compilation now
passes `-g` when the input LLVM IR carries that metadata, so platform-native
debug sections are retained instead of being dropped during clang codegen.

The current proven boundary is:

- emitted LLVM IR contains stable source comments for lowered functions
- emitted LLVM IR contains LLVM debug metadata for Jayess functions
- Linux object-file builds carry real DWARF entries that `llvm-dwarfdump`
  can inspect for:
  - compile unit
  - source filename
  - lowered function symbols and Jayess function names
- runtime call-frame labels still include Jayess function name plus line/column
  for uncaught exception stack reporting

That gives current crash/debug workflows both:

- readable emitted IR for backend inspection
- native DWARF/object metadata for debugger and toolchain inspection where
  supported

## Native interoperability boundary

Manual native bindings from `*.bind.js` are compiled by clang and linked with:

- Jayess-generated LLVM IR
- `runtime/jayess_runtime.c`
- any listed native sources and libraries

Binding implementations should include `jayess_runtime.h` and use the low-level
boxed-value/runtime helpers directly.

## Toolchain interoperability

Current backend tests verify that emitted IR can be consumed by common LLVM
tools on supported environments:

- `llvm-as`
- `opt`
- `llc`

Jayess also supports direct emission of:

- LLVM IR text (`--emit=llvm`)
- LLVM bitcode (`--emit=bc`)
- object files (`--emit=obj`)
- native executables (`--emit=exe`)

## Per-platform linker and runtime quirks

Jayess currently relies on the system clang driver for final native linkage, so
some platform behavior is intentionally encoded at the toolchain boundary rather
than in emitted LLVM IR.

Current built-in system link assumptions are:

- Linux and Darwin executable builds add:
  - `-lssl`
  - `-lcrypto`
  - `-lz`
  - `-lm`
- Windows executable builds add:
  - `-lws2_32`
  - `-lwinhttp`
  - `-lsecur32`
  - `-lcrypt32`
  - `-lbcrypt`

Manual native binding link flags from `*.bind.js` are appended after those
platform defaults.

Those default runtime/link flags are also covered by backend argument tests for
Linux, Darwin, and Windows target triples, so the platform-specific linkage
contract is explicit rather than only implicit in toolchain code.

Other current platform notes:

- cross-target object builds are covered for the current named target set:
  - `linux-x64`
  - `linux-arm64`
  - `darwin-x64`
  - `darwin-arm64`
  - `windows-x64`
- linked native executable coverage is still only proven on Linux in this
  repository today
- current executable proof matrix is intentionally split:
  - proven executable target: `linux-x64`
  - proven cross-target object emission only: `darwin-x64`, `darwin-arm64`,
    `windows-x64`, `linux-arm64`
- when cross-target executable builds fail because the host toolchain lacks the
  target SDK or C runtime headers, Jayess reports that boundary explicitly as a
  native toolchain error instead of only surfacing raw clang output
  - darwin targets now point at the missing Apple SDK/sysroot boundary and call
    out `xcrun` / `SDKROOT` style setup explicitly
  - windows targets now point at the missing Windows SDK plus C runtime
    boundary and call out MSVC/`clang-cl` or MinGW-style sysroot setup
  - other cross-target libc failures still point at the missing target sysroot
    boundary explicitly
- Jayess does not emit a custom linker script or platform-specific CRT startup
  path; it relies on the selected clang target triple and host toolchain
- path handling, file permissions, and networking semantics above the LLVM
  layer still depend on the host runtime and platform C library behavior
