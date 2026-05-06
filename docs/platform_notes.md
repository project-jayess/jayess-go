# Platform Notes

Jayess targets Linux, macOS, and Windows with platform-specific toolchain and
native library behavior.

## Linux

Shared libraries commonly use `.so`. Webview apps may require GTK and WebKitGTK
packages. Linker behavior usually routes through Clang/lld.

## macOS

Shared libraries commonly use `.dylib`. Framework flags such as `-framework
Cocoa` and `-framework WebKit` are used by some bindings.

## Windows

Shared libraries use `.dll`. Native bindings may need import libraries and
Windows-specific linker flags.

## Release Rule

Document target-specific native dependencies and test every packaged target on
that target platform.

## Example Target Commands

```sh
jayess --emit=exe --target=linux-x64 examples/01-basics.js
jayess --emit=exe --target=darwin-arm64 examples/01-basics.js
jayess --emit=exe --target=windows-x64 examples/01-basics.js
```
