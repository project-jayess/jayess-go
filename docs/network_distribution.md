# Network Distribution

Standard Jayess networking uses internal runtime packages. Apps that import
`http`, `https`, `tcp`, `udp`, `dns`, `tls`, or `stream` should not require end
users to install libcurl, Mongoose, picohttpparser, OpenSSL, or libuv.

## Internal Packages

Internal networking packages are linked through Jayess-owned runtime helpers.
The distributor may include Jayess runtime metadata files when an internal
package needs them, but it must not copy external native libraries for normal
HTTP, HTTPS, TCP, UDP, DNS, TLS, or stream imports.

## Optional Native Bindings

Package external runtime libraries only when the application explicitly imports
an optional native binding. Examples include libcurl for advanced transfer
features, Mongoose as an alternative embedded server, or picohttpparser as a
low-level parser package.

Optional bindings must declare all redistributable native assets so Jayess can
package them automatically. The end user should only need the distribution
produced by the compiler.

## Licenses

Include license and notice files for every copied native library when optional
native bindings are imported. No external native-library notices are needed for
standard internal networking alone.

## Smoke Test

Run a packaged HTTP client or server from inside the final distribution folder.
The smoke test should pass without system-installed networking libraries.

## Example Layout

```text
dist/internal-http-client/
  bin/http-client
  runtime/os_cli_runtime.json

dist/advanced-curl-client/
  bin/curl-client
  lib/libcurl.so
  licenses/LICENSE.curl
```
