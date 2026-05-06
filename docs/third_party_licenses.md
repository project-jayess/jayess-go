# Third-Party Licenses

Developers are responsible for including licenses for native libraries and tools
they ship with an application.

## Binding Licenses

Use `licenseFiles` in binding manifests to list license or notice files that
belong with native binding assets.

## Toolchain Licenses

Compiler distributions that include LLVM, Clang, lld, or related tools must
include the corresponding license files.

## Release Rule

If a binary distribution contains a third-party binary, source bundle, or copied
runtime library, include its license and notices in the distribution.

## Example Manifest Field

```js
export default bind({
  sharedLibraries: ["./vendor/lib/libsqlite3.so"],
  licenseFiles: ["./vendor/LICENSE.sqlite"],
  exports: {}
});
```
