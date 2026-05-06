# Network Distribution

Network packages often depend on native shared libraries.

## Required Assets

Package all runtime libraries required by the selected bindings, such as curl,
OpenSSL, Mongoose wrapper libraries, or platform TLS dependencies when they are
not provided by the operating system.

## Licenses

Include license and notice files for every copied native library.

## Smoke Test

Run a packaged HTTP client or server from inside the final distribution folder
to ensure no undeclared library path is required.

## Example Layout

```text
dist/http-client/
  bin/http-client
  lib/libcurl.so
  lib/libssl.so
  licenses/LICENSE.curl
  licenses/LICENSE.openssl
```
