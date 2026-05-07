# libcurl Networking

libcurl is optional advanced transport support. Standard Jayess HTTP and HTTPS
applications should use the internal `http` and `https` runtime packages, which
do not require libcurl to be installed or shipped with the app.

## When To Use libcurl

Use an explicit libcurl binding only when an application needs behavior outside
the internal runtime transport, such as:

- proxy features beyond the internal HTTP client
- FTP or non-HTTP transfer protocols
- cookie jar parity with curl
- custom TLS backend behavior
- HTTP/2 or HTTP/3 extras not exposed by the internal runtime

## Binding Model

If libcurl is imported, expose a small Jayess API for transfers, status codes,
headers, bodies, and errors. Keep native curl handles behind managed native
handles so they can be closed deterministically.

Bindings must declare any redistributable libcurl and TLS backend runtime
assets so app distribution can package them automatically. This is not required
for apps that import only internal `http` or `https`.

## Errors

Convert curl failures into Jayess errors or result objects with stable error
codes. Do not leak raw native pointers into user code.

## Example Shape

```js
function main() {
  const response = http.request("https://example.com");
  console.log(response.status);
  console.log(response.body);
  return 0;
}
```
