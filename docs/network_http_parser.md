# picohttpparser

picohttpparser is optional low-level HTTP parser support only. Standard Jayess
HTTP clients and servers should use the internal `http` and `https` runtime
packages, which do not require picohttpparser.

## Purpose

Use an explicit picohttpparser binding only to parse raw HTTP request or
response bytes without the normal server stack. It is useful for low-level
parser packages and protocol tests, not for ordinary application servers.

## Ownership

Input bytes are owned by the caller. Parsed Jayess strings, arrays, and objects
are runtime-owned values. Native wrappers should copy data when it must survive
after the call returns.

## Limits

picohttpparser is a parser, not a networking stack. Socket handling and TLS
belong in internal `tcp`, `http`, `https`, and `tls` packages.

Bindings must declare any redistributable parser assets so app distribution can
package them automatically. Apps that import only internal `http` or `https` do
not need these assets.

## Example Shape

```js
import { parseRequest } from "./native/picohttpparser.js";

function main() {
  const request = parseRequest("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n");
  console.log(request.method);
  console.log(request.path);
  return 0;
}
```
