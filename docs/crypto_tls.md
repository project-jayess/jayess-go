# Crypto and TLS

Crypto and TLS support is expected to use OpenSSL or another explicit native
binding supplied by the developer.

## Setup

Install the target platform's OpenSSL headers and libraries or provide a
project-local vendor layout. Keep version and license files with the native
library assets.

## Binding

Expose only the C ABI entrypoints needed by Jayess code. Use Jayess runtime
helpers to convert strings, byte arrays, objects, and errors across the native
boundary.

## Distribution

Package OpenSSL shared libraries when the executable requires them. Include the
OpenSSL license and notices in the distribution.

## Example Shape

```js
import { sha256Hex } from "./native/openssl.js";

function main() {
  const digest = sha256Hex("hello");
  console.log(digest);
  return 0;
}
```
