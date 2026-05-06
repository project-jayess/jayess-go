# picohttpparser

picohttpparser support is for low-level HTTP parsing where a full server stack
is not needed.

## Purpose

Use it to parse HTTP request or response bytes into Jayess objects. It is useful
inside higher-level server packages or tests.

## Ownership

Input bytes are owned by the caller. Parsed Jayess strings, arrays, and objects
are runtime-owned values. Native wrappers should copy data when it must survive
after the call returns.

## Limits

picohttpparser is a parser, not a networking stack. Socket handling and TLS
belong in other packages.

## Example Shape

```js
import { parseRequest } from "@jayess/httpserver";

function main() {
  const request = parseRequest("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n");
  console.log(request.method);
  console.log(request.path);
  return 0;
}
```
