# Network Examples

Network examples should stay small and show the distribution contract clearly.
Use Jayess internal packages for normal applications. Optional native transport
bindings are for advanced cases only and must declare their own packaged assets.

## HTTP Client Shape

```js
import { get } from "https";

function main() {
  const response = get("https://example.com");
  console.log(response.status);
  return 0;
}
```

## Embedded Server Shape

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

## DNS Shape

```js
import { lookup } from "dns";

function main() {
  const result = lookup("example.com");
  console.log(result.address);
  return 0;
}
```

## TCP Shape

```js
import { connect } from "tcp";

function main() {
  const socket = connect("example.com", 80);
  socket.write("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n");
  console.log(socket.read());
  socket.close();
  return 0;
}
```

## UDP Shape

```js
import { createSocket } from "udp";

function main() {
  const socket = createSocket("udp4");
  socket.send("ping", "127.0.0.1", 5353);
  socket.close();
  return 0;
}
```

## Stream Shape

```js
import { PassThrough } from "stream";

function main() {
  const stream = new PassThrough();
  stream.write("hello");
  console.log(stream.read());
  return 0;
}
```

Optional packages such as libcurl, Mongoose, and picohttpparser may still exist,
but they are explicit advanced bindings. They are not part of the standard
internal networking path.
