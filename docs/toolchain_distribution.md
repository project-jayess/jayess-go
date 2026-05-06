# Toolchain Distribution

Toolchain distribution packages Jayess with the compiler tools it needs.

## Contents

A distributable compiler SDK should include:

- the `jayess` compiler executable
- target tool directories such as `tools/<target>/bin`
- LLVM/Clang/lld tools required by compiler modes
- runtime headers and libraries required by native bindings
- license and notice files for shipped tools

## Command Shape

Use `cmd/jayess-dist` or the built `jayess-dist` command with an explicit LLVM
build directory when packaging bundled LLVM tools.

## Verification

Unpack the SDK, run the packaged compiler, compile an example, and confirm the
compiler resolves tools from the package instead of the developer machine.

## Example Layout

```text
jayess-toolchain-linux-x64/
  bin/jayess
  tools/linux-x64/bin/clang
  tools/linux-x64/bin/lld
  licenses/LLVM-LICENSE.txt
```
