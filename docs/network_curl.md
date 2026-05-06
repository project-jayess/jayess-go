# libcurl Networking

libcurl support is expected to be provided through native package or binding
integration.

## Setup

Install libcurl headers and libraries for the target platform or provide them in
a project vendor layout.

## API Shape

Expose a small Jayess API for transfers, status codes, headers, bodies, and
errors. Keep native curl handles behind managed native handles so they can be
closed deterministically.

## Errors

Convert curl failures into Jayess errors or result objects with stable error
codes. Do not leak raw native pointers into user code.

## Example Shape

```js
import { get } from "./native/curl.js";

function main() {
  const response = get("https://example.com");
  console.log(response.status);
  console.log(response.body);
  return 0;
}
```
