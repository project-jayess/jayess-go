# Internal HTTP Server

Jayess provides an HTTP server/runtime model backed by Go's standard `net/http`
package. It does not require libcurl, Mongoose, picohttpparser, or another
external native library for basic HTTP server use.

## Server

```js
import "http";

function main() {
  const server = http.createServer((req, res) => {
    const request = http.requestObject(req);
    const response = http.responseObject(res);
    http.status(response, 200);
    http.writeBody(response, "hello " + http.readBody(request));
    return http.headers(request);
  });
  return server;
}
```

The runtime server supports request method, URL/path, headers, request body
reading, response status, response headers, and response body writing.

## Server Events

The internal server exposes a Node-style event API:

```js
function main() {
  const server = http.createServer();

  server.on("request", (req, res) => {
    http.status(res, 200);
    http.writeBody(res, "ok");
  });

  server.once("listening", () => {
    process.stdout.write("listening\n");
  });

  server.on("error", (err) => {
    process.stderr.write(String(err));
  });

  server.listen("127.0.0.1:3000");
  return server;
}
```

Supported event names are `request`, `connection`, `listening`, `close`,
`error`, `clientError`, `checkContinue`, `checkExpectation`, `connect`,
`upgrade`, and `dropRequest`. Supported event methods are `on`, `addListener`,
`once`, `off`, `removeListener`, `removeAllListeners`, `emit`, `eventNames`,
and `listenerCount`.

## Client Helpers

```js
function main() {
  const request = http.withTimeout(http.request("http://127.0.0.1:3000"), 1000);
  const kept = http.keepAlive(request);
  return http.streamBody(kept);
}
```

## Compiler Integration

Direct `http.*` calls lower to `jayess_http_*` LLVM runtime symbols. The HTTP
server implementation is internal runtime code, so app distribution does not
need to ship a separate external HTTP library for this feature.

The HTTP server example is available at `examples/17-http-server.js`.
