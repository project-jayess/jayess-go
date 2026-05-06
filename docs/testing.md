# Testing

Tests live under focused Go packages and the top-level `test/` package for
cross-package integration behavior.

## Strategy

- keep unit tests close to the package behavior being verified
- put compiler integration tests under `test/`
- keep generated temporary files under `temp/`
- add executable smoke tests for CLI, lowering, backend, binding, and
  distribution behavior

## Release Verification

Run the package test set from a clean checkout and then test the packaged
compiler or app distribution. Release verification should include compiling at
least one example from the distribution itself.

## Example Commands

```sh
go test ./cmd/jayess ./cmd/jayess-dist ./lexer ./parser ./semantic ./runtime ./test
go run ./cmd/jayess --emit=llvm examples/01-basics.js
```
