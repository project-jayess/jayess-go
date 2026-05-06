# LLVM Backend

The LLVM backend emits LLVM IR and coordinates object, library, shared library,
or executable output.

## Responsibilities

- emit functions, blocks, statements, and expressions
- call Jayess runtime helpers for dynamic operations
- emit runtime literals and value construction
- preserve structured exits and cleanup behavior
- support target-specific object and linking flows

## Emission Modes

The CLI supports `--emit=llvm|bc|obj|lib|shared|exe`. Object emission can use
the LLVM C API when built with `jayess_llvmc`; otherwise the compiler can write
temporary IR under `temp/jayess-build` and invoke target tools.

## Toolchain

Tool resolution checks bundled distribution tools, configured toolchain paths,
local LLVM builds, and `PATH`. Distribution builds should include LLVM/Clang/lld
tools and relevant license files.

## Example Commands

```sh
go run ./cmd/jayess --emit=llvm --target=linux-x64 examples/01-basics.js
go run ./cmd/jayess --emit=obj --target=linux-x64 examples/01-basics.js
go run ./cmd/jayess --emit=shared --target=linux-x64 examples/01-basics.js
```
