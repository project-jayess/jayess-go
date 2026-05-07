# External Library Inventory

This inventory tracks Jayess features that still rely on external libraries,
SDKs, or compiler tools. The policy is that Jayess should package
redistributable runtime dependencies required by imported code. End users should
not have to install native libraries separately after receiving a Jayess-built
app package.

## Internal Runtime Features

These features are implemented by Jayess runtime/compiler code and should not
require a third-party native library in the app package:

| Area | Current backing |
| --- | --- |
| Process, stdio, exit codes | Go runtime helpers |
| Filesystem and file streams | Go runtime helpers |
| Child process spawn/exec | Go runtime helpers |
| Terminal detection | Go runtime helpers |
| HTTP server/client basics | Go `net/http` runtime helpers |
| HTTP server event surface | Jayess HTTP event layer over Go `net/http` |
| Crypto random bytes, hashes, HMAC, AES-GCM, Ed25519, RSA-OAEP, key PEM, certificate parsing, secure compare | Go crypto runtime helpers |
| TLS and HTTPS facades | Go `crypto/tls`, `crypto/x509`, and `net/http` runtime helpers |
| gzip/deflate compression | Go `compress/gzip` and `compress/flate` runtime helpers |
| DNS lookup, reverse lookup, resolver config, IP parsing | Go `net.Resolver` and `net.ParseIP` runtime helpers |
| TCP and UDP socket helpers | Go `net` runtime helpers |
| Event loop, timers, microtasks | Go runtime scheduling helpers |
| Worker message passing and shared integer memory | Go goroutine and synchronization runtime helpers |
| Streams, Buffer, URL, query, util formatting | Go runtime helpers |
| Source/path/compiler data structures | Go runtime helpers |
| Simple key/value storage | Go runtime helpers with JSON-backed persistence |
| Asset lookup, WAV metadata, PCM parsing and mixing | Go runtime helpers |

## Compiler Toolchain Dependencies

| Dependency | Used for | Packaging status | Notes |
| --- | --- | --- | --- |
| LLVM tools | IR assembly, object emission, target support | Jayess toolchain distribution should bundle required tools | `dist` copies LLVM tools from a configured LLVM build directory. |
| Clang | Fallback compile/link driver | Jayess toolchain distribution should bundle it when required | Used when internal LLVM/lld paths are unavailable. |
| lld | Linking | Jayess toolchain distribution should bundle it when required | Internal `lldc` support needs linked lld libraries at compiler build time. |
| LLVM C API / libLLVM | Internal object emission with `jayess_llvmc` | Compiler distribution should include redistributable runtime libraries | Build-time cgo link currently points at `refs/llvm-project/build/lib` or equivalent. |
| Platform SDKs | Final platform linking | Build-machine requirement when non-redistributable | Apple SDK and Microsoft SDK files cannot generally be bundled as Jayess assets. |

## Optional Native Binding Libraries

These are not required for core Jayess programs, but apps importing the related
bindings must declare and package redistributable runtime assets.

| Area | External dependency | Current model | Packaging expectation |
| --- | --- | --- | --- |
| SQLite | SQLite source or `sqlite3` shared library | Native binding/package model | Vendor source or package shared library plus license. |
| libcurl networking | libcurl headers/source/library | Native binding/package model | Package libcurl and TLS backend libraries if dynamically linked. |
| TLS/crypto via native packages | OpenSSL or platform TLS libraries when used by a binding | Native binding/package model | Package redistributable shared libraries; document platform trust store assumptions. |
| GLFW graphics | GLFW, OpenGL/platform libraries | Native binding/package model | Package redistributable GLFW libraries; platform graphics/system libraries may be build-machine/system requirements. |
| raylib graphics | raylib source or shared library | Native binding/package model | Vendor source or package shared library plus license. |
| GTK UI | GTK and related platform packages | Native binding/package model | Large dependency set; prefer declared package metadata and platform-specific docs. |
| Webview apps | GTK/WebKitGTK on Linux, WebView2/Win32 on Windows, WebKit on macOS | Native binding/package model | Redistributable runtime pieces should be packaged; platform frameworks may remain system-provided. |
| Audio | SDL audio, OpenAL, miniaudio, PortAudio, or platform-native APIs | Native binding/package model | Package selected redistributable backend and license. |
| libuv | libuv source or library | Native binding/package model | Vendor source or package shared library plus license. |
| picohttpparser | picohttpparser source | Native binding/package model for low-level parsing | Not needed for the internal HTTP server. Vendor source if the low-level parser package is imported. |

Optional non-Node packages such as SQLite, GUI, graphics, and audio stay in the
native package/binding flow. They should not be treated as hidden core runtime
requirements. When an application imports one of these packages, the package
metadata or binding manifest must declare any redistributable libraries and
license files so app distribution can copy them automatically.

## Node-Like APIs To Internalize

These are available in Node.js as built-in capabilities or as implementation
details shipped with Node. Jayess should prefer internal runtime
implementations or compiler-managed packaged dependencies instead of expecting
developers or end users to install native libraries separately.

| Priority | Node-like area | Current Jayess risk | Preferred Jayess direction |
| --- | --- | --- | --- |
| High | `crypto` hashing, HMAC, random bytes, signing, verification, symmetric encryption | Common crypto plus AES-GCM, Ed25519, RSA-OAEP, key PEM import/export, and certificate parsing are internal | Keep supported crypto internal; only use packaged OpenSSL or platform crypto for algorithms that cannot be internalized. |
| High | `tls` and `https` | Internal facades exist for certificates, trust stores, ALPN, client config, and HTTPS server/client setup | Keep HTTPS/TLS compiler-owned and package any redistributable TLS backend automatically if a future platform requires one. |
| High | `zlib`/compression including gzip, deflate, Brotli | gzip/deflate are internal; Brotli is explicitly unsupported and must not package an external backend silently | Keep gzip/deflate internal; add Brotli only as compiler-owned runtime support in a future checklist item. |
| High | `dns` | lookup, reverse lookup, resolver config, and IP parsing are internal | Keep DNS on Go-owned runtime helpers; do not require c-ares or resolver command-line tools. |
| High | `net` TCP and `dgram` UDP | TCP/UDP client, server, bind, send, receive, stream, timeout, and close helpers are internal | Keep socket support on Go-owned runtime helpers rather than libuv as an app dependency. |
| High | Event loop/timers/microtasks | Internal deterministic scheduler exists for timer and microtask builtins | Keep event loop/timer scheduling compiler-owned; libuv remains optional explicit binding support only. |
| Medium | Worker threads | Worker creation, message passing, cleanup, and small shared-memory atomics are internal | Keep workers on Jayess-owned runtime support; external thread libraries remain optional explicit bindings only. |
| Medium | Streams | Shared stream primitives and cross-runtime stream tests are internal | Continue internal stream runtime so HTTP, fs, child process, compression, TCP, and UDP share one stream model. |
| Medium | URL/query/string utilities | URL/query helpers are internal | Keep internal; do not use native URL/parser libraries. |
| Medium | Buffer and binary data | Buffer helpers are internal | Keep internal; do not rely on native buffer libraries. |
| Low | HTTP parsing | picohttpparser is available as a binding model, but Node-like HTTP server no longer needs it | Keep picohttpparser optional for low-level users; internal HTTP should not require it. |
| Low | SQLite/database packages | Internal `storage` covers simple dependency-free key/value persistence; SQLite remains optional for SQL compatibility | Keep SQLite as optional package/native binding with strict packaging metadata. |
| Low | GUI/graphics/audio packages | Internal asset/WAV/PCM helpers cover simple processing without device playback | Keep device playback and graphics as optional native packages; package redistributable libraries automatically when imported. |

## Reference Directories

`refs/` contains read-only reference projects and external source checkouts.
Those files are not production Jayess source and should not be copied into app
or compiler distributions unless a packaging task explicitly vendors a
redistributable artifact with licenses.

`old_version/` is also not production source and should remain untouched during
normal implementation work.

## Packaging Rules

- Imported Jayess packages and native bindings are the source of truth for
  runtime assets.
- Internal Node-like runtime imports should not add `.so`, `.dll`, or `.dylib`
  assets unless a future platform-specific backend explicitly declares a
  redistributable runtime asset.
- Shared libraries required at runtime must be declared in binding manifests or
  package metadata.
- Runtime assets that require licenses should declare `licenseFiles` or
  equivalent metadata.
- Build-only system requirements should be marked as build-only and documented.
- End-user packages should include all redistributable dependencies needed by
  imported code.
- Platform SDKs and system frameworks that cannot be redistributed may remain
  build-machine or platform requirements, but they must be documented clearly.

## Platform SDK Boundaries

Platform SDKs are build-machine requirements when redistribution is not allowed.
Examples include Apple SDKs, Microsoft SDKs, and system frameworks that must be
present on the target platform. These are not end-user install steps for a
Jayess-built package. The compiler and distribution docs should describe them as
requirements for building or targeting a platform, not as runtime libraries that
the receiving user has to install manually.

## Highest-Risk External Areas

1. GTK/WebKit/webview dependency closure, because it can pull many platform
   packages and licenses.
2. TLS stacks, because libcurl/OpenSSL/platform trust stores vary by target.
3. Graphics stacks, because OpenGL/DirectX/Metal and windowing dependencies are
   platform-specific.
4. Compiler toolchain packaging, because LLVM/Clang/lld must be bundled
   consistently for release builds.
