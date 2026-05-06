# Mongoose Embedded Server

Mongoose can be exposed as an embedded web server through native bindings.

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

## Example Shape

```js
import { createServer } from "./native/mongoose.js";

function main() {
  const server = createServer("127.0.0.1:8080", (request) => {
    return { status: 200, body: "ok" };
  });
  server.run();
  return 0;
}
```
