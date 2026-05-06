# Release Checklist

A complete feature checklist does not by itself mean the repository is ready to
release. Release from a clean, reviewed state.

## Steps

1. Confirm `jayess-feature-checklist.md` has no unexpected unchecked release
   blockers.
2. Review and commit intended source, test, and documentation changes.
3. Remove generated `temp/` artifacts from the release commit.
4. Run the Go package test suite from a clean checkout.
5. Build the Jayess compiler distribution.
6. Verify the distribution contains expected tools, runtime files, and licenses.
7. Compile a Jayess example using the packaged compiler.
8. Build an app distribution and run it from inside the package folder.
9. Confirm third-party native library licenses are included.

## Release Rule

Do not tag a release from a dirty workspace unless the dirty files are exactly
the release artifacts being published and are documented separately.

## Example Verification

```sh
git status --short
go test ./...
./dist/jayess-toolchain-linux-x64/bin/jayess --emit=exe examples/01-basics.js
```
