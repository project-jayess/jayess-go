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

Jayess does not currently emit DWARF or other platform-native debug metadata
into LLVM IR. That means native debuggers do not yet get full source-level
stepping directly from emitted Jayess IR.

The backend does preserve debug-friendly source information in two practical
ways today:

- emitted LLVM IR includes source comments for lowered functions, for example:
  - `; source function main at 10:1`
  - `; debug frame main (10:1)`
- runtime call-frame labels include the Jayess function name plus line/column
  when available, and uncaught exception stack traces print those labels

That gives current crash/debug workflows a reasonable mapping back to Jayess
source even without full DWARF support:

- emitted IR stays readable during backend inspection
- runtime stack traces show `function (line:col)` entries
- no-opt (`O0`) builds are regression-tested to preserve those stack locations

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

Other current platform notes:

- cross-target object builds are covered for major targets, but linked native
  executable coverage is only proven on Linux in this repository today
- Jayess does not emit a custom linker script or platform-specific CRT startup
  path; it relies on the selected clang target triple and host toolchain
- path handling, file permissions, and networking semantics above the LLVM
  layer still depend on the host runtime and platform C library behavior
