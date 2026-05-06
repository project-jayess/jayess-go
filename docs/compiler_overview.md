# Compiler Overview

The Jayess compiler is organized as small Go packages that mirror compiler
stages and runtime support.

## Pipeline

1. `lexer` converts source text into tokens with source positions.
2. `parser` builds the AST from tokens and reports syntax diagnostics.
3. `semantic` checks declarations, imports, assignment targets, and language
   restrictions.
4. `lifetime` and `escape` prepare closure and lifetime information.
5. `lowering` rewrites high-level forms into simpler core forms.
6. `llvmbackend` emits LLVM IR and runtime calls.
7. `llvmc`, `lldc`, and `tooling` provide object emission and linking helpers.

## Supporting Packages

`ast` defines syntax nodes. `resolver` handles project and module loading.
`runtime` contains the runtime value model and services exposed to compiled
programs. `binding` extracts and validates native binding manifests. `dist` and
`appdist` build distributable compiler and application layouts.

## CLI

The main compiler entry point is `cmd/jayess`. Distribution packaging lives in
`cmd/jayess-dist`.

## Example Commands

```sh
go run ./cmd/jayess --emit=llvm examples/01-basics.js
go run ./cmd/jayess --emit=obj --target=linux-x64 examples/01-basics.js
go run ./cmd/jayess-dist --help
```
