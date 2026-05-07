# Mongoose Embedded Server

Mongoose is optional alternative embedded-server support only. Standard Jayess
HTTP servers should use the internal `http` runtime package, which is built from
Jayess-owned Go helpers and does not require Mongoose to be installed or shipped.

Use a Mongoose binding only when an application explicitly wants Mongoose API or
behavior rather than the normal Jayess HTTP server API.

## Lifecycle

The binding should provide explicit create, listen, poll or run, and shutdown
operations. Server handles should be managed native handles with deterministic
close behavior.

## Routing

Keep Jayess routing APIs small and explicit. Convert request method, path,
headers, and body into Jayess values before invoking user handlers.

## Shutdown

Server shutdown should release native resources and stop accepting new work
before the application exits.

Bindings must declare any redistributable Mongoose wrapper assets so app
distribution can package them automatically. Apps that import only internal
`http` do not need these assets.

## Example Shape

```js
import { createServer } from "http";

function main() {
  const server = createServer((request, response) => {
    response.statusCode = 200;
    response.end("ok");
  });
  server.listen(8080, "127.0.0.1");
  return 0;
}
```
