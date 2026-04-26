# Networking Foundation

Jayess currently includes URL, query-string, HTTP message, basic HTTP/HTTPS client, DNS, and TCP helpers as the first networking-oriented standard library surface.

These helpers now include blocking HTTP and HTTPS client requests, plus blocking `http.createServer(...)` and `https.createServer(...)` server surfaces. The `http` helpers remain the shared protocol/message layer that the higher-level `https`, `tls`, `net`, `dgram`, and `dns` APIs build on.

## URL

Supported helpers:

- `url.parse(input)`
- `url.format(parts)`

`url.parse(input)` returns an object with:

- `href`
- `protocol`
- `host`
- `pathname`
- `query`
- `hash`
- `queryObject`

Example:

```js
var parsed = url.parse("https://example.com/api?q=kimchi&lang=en#top");
console.log(parsed.protocol);      // https:
console.log(parsed.host);          // example.com
console.log(parsed.pathname);      // /api
console.log(parsed.queryObject.q); // kimchi
```

`url.format(parts)` accepts an object with `protocol`, `host`, `pathname`, `query`, and `hash` fields.

```js
console.log(url.format({
  protocol: "https:",
  host: "example.com",
  pathname: "/api",
  query: "q=kimchi",
  hash: "top"
}));
```

Current boundary:

- This is a small Jayess URL helper, not full WHATWG URL parity.
- User/password, port-specific helpers, origin normalization, and relative URL resolution are not implemented yet.

## Query String

Supported helpers:

- `querystring.parse(input)`
- `querystring.stringify(parts)`

Example:

```js
var values = querystring.parse("name=kimchi%20man&spicy=10");
console.log(values.name);  // kimchi man
console.log(values.spicy); // 10

console.log(querystring.stringify({ name: "kimchi man" }));
```

Current boundary:

- Percent decoding and encoding are supported.
- Repeated keys, nested values, arrays, and configurable separators are not implemented yet.

## HTTP

Supported helpers:

- `http.parseRequest(input)`
- `http.formatRequest(parts)`
- `http.parseResponse(input)`
- `http.formatResponse(parts)`
- `http.createServer(handler)`
- `http.request(options)`
- `http.requestStream(options)`
- `http.requestStreamAsync(options)`
- `http.get(input)`
- `http.getStream(input)`
- `http.getStreamAsync(input)`
- `http.requestAsync(options)`
- `http.getAsync(input)`

Example:

```js
var requestText = http.formatRequest({
  method: "POST",
  path: "/submit",
  version: "HTTP/1.1",
  headers: { Host: "example.com" },
  body: "kimchi=1"
});

var request = http.parseRequest(requestText);
console.log(request.method);
console.log(request.headers.Host);

var responseText = http.formatResponse({
  version: "HTTP/1.1",
  status: 200,
  reason: "OK",
  headers: { "Content-Type": "text/plain" },
  body: "done"
});

var response = http.parseResponse(responseText);
console.log(response.status);
console.log(response.body);

var server = http.createServer((req, res) => {
  res.statusCode = 200;
  res.setHeader("Content-Type", "text/plain");
  res.end("hello");
});
server.listen(8080, "127.0.0.1");

var live = http.get({ host: "example.com", port: 80, path: "/" });
console.log(live.status);

var fromUrl = http.get("http://example.com:8080/hello?name=kimchi");
console.log(fromUrl.status);

var streamed = http.getStream("http://example.com:8080/large");
console.log(streamed.status);
console.log(streamed.bodyStream.read(5));
streamed.bodyStream.close();

var asyncStreamed = await http.getStreamAsync("http://example.com:8080/large");
console.log(asyncStreamed.status);
console.log(asyncStreamed.bodyStream.read(5));
asyncStreamed.bodyStream.close();

var asyncResponse = await http.getAsync("http://example.com/api");
console.log(asyncResponse.status);

var maybeTimedOut = http.get({ host: "example.com", port: 80, path: "/", timeout: 250 });
console.log(maybeTimedOut);
```

Current boundary:

- These helpers only build and parse HTTP/1.x text messages.
- `http.request(options)` performs a blocking plain-HTTP request and returns a parsed response object.
- `http.requestStream(options)` performs a blocking plain-HTTP request and returns after response headers are available, with `response.bodyStream` exposed as a live readable stream.
- `http.requestStreamAsync(options)` returns a promise-like value resolved after response headers are available, with the same streamed response shape as `http.requestStream(options)`.
- `http.get(input)` performs a blocking plain-HTTP `GET` request from either a URL string or `{ host, port, path, headers? }`.
- `http.getStream(input)` performs a blocking plain-HTTP `GET` and returns the same streamed response shape as `http.requestStream(options)`.
- `http.getStreamAsync(input)` returns a promise-like value resolved with the same streamed response shape as `http.getStream(input)`.
- `http.requestAsync(options)` returns a promise-like value resolved with the same response shape as `http.request(options)`.
- `http.getAsync(input)` returns a promise-like value resolved with the same response shape as `http.get(input)`.
- `http.request({ url, ... })` supports a URL string source with explicit method/body/headers overrides.
- The HTTP client now fills in `Host`, `Connection: close`, and `Content-Length` for non-empty bodies when those headers are not supplied explicitly.
- The HTTP request options accept `timeout` in milliseconds and currently return `undefined` on timeout or transport failure.
- The HTTP client now follows plain-HTTP redirects for `301`, `302`, `303`, `307`, and `308`, with `maxRedirects` defaulting to `5`.
- Final HTTP responses now expose `redirected`, `redirectCount`, and `url`.
- Final HTTP responses also expose `ok` and `statusText`.
- Final HTTP responses also expose `bodyBytes` as a `Uint8Array`.
- Streamed HTTP responses expose `bodyStream` with `read(size?)`, `readBytes(size?)`, `on(...)`, `once(...)`, `off(...)`, `listenerCount(...)`, `eventNames()`, `pipe(...)`, `close()`, `destroy()`, `readableEnded`, `closed`, `errored`, and `error`.
- `http.parseResponse(...)`, `http.request(...)`, `http.get(...)`, `http.requestAsync(...)`, and `http.getAsync(...)` now decode `Transfer-Encoding: chunked` response bodies.
- The HTTP client now stops reading once a full response is available from `Content-Length` or chunked framing instead of always waiting for socket close.
- `http.requestStream(...)` and `http.getStream(...)` now return before the entire body is buffered, so user code can decide whether to read, pipe, or close the body stream.
- `http.createServer(handler)` currently provides a blocking plain-HTTP server path with `listen(port[, host])` and `close()`.
- `https.createServer(options, handler)` currently provides a blocking HTTPS server path with the same `listen(port[, host])` and `close()` methods.
- request handlers receive `req` with `method`, `url`, `path`, `headers`, and `body`.
- response objects currently support `statusCode`, `setHeader(name, value)`, `write(chunk)`, and `end(chunk?)`.
- Async HTTP currently runs on the runtime worker pool around blocking socket operations, not a true nonblocking transport.
- Redirect following is currently limited to plain `http://...` targets and root-relative `Location` headers.
- Chunked request encoding, trailers, and compression are not implemented yet.
- The current HTTP/HTTPS server path now supports sequential keep-alive requests on the same connection when the client keeps it open. Responses fall back to chunked transfer encoding when needed so the connection can remain reusable.
- `https.createServer(...)` currently uses the OpenSSL-backed TLS path on non-Windows builds and expects PEM file paths in `options.cert` and `options.key`.
- Server-side HTTPS is not implemented on Windows yet.

## HTTPS

Supported helpers:

- `https.createServer(options, handler)`
- `https.get(input)`
- `https.request(input)`
- `https.requestStream(input)`
- `https.requestStreamAsync(input)`
- `https.getAsync(input)`
- `https.requestAsync(input)`
- `https.getStream(input)`
- `https.getStreamAsync(input)`
- `https.isAvailable()`
- `https.backend()`

Example:

```js
var insecure = false;
console.log(https.isAvailable());
console.log(https.backend());

var response = https.get({
  url: "https://example.com/hello",
  rejectUnauthorized: insecure
});
console.log(response.status);
console.log(response.body);

var fromRequest = https.request({
  url: "https://example.com/hello",
  rejectUnauthorized: insecure
});
console.log(fromRequest.status);

var streamed = https.getStream({
  url: "https://example.com/large",
  rejectUnauthorized: insecure
});
console.log(streamed.status);
console.log(streamed.bodyStream.read(5));
streamed.bodyStream.close();

var asyncStreamed = await https.getStreamAsync({
  url: "https://example.com/large",
  rejectUnauthorized: insecure
});
console.log(asyncStreamed.status);
console.log(asyncStreamed.bodyStream.read(5));
asyncStreamed.bodyStream.close();

var asyncResponse = await https.getAsync({
  url: "https://example.com/hello",
  rejectUnauthorized: insecure
});
console.log(asyncResponse.body);

var server = https.createServer({
  cert: "./server-cert.pem",
  key: "./server-key.pem",
}, (req, res) => {
  res.statusCode = 200;
  res.end("secure");
});
server.listen(8443, "127.0.0.1");
```

Current boundary:

- HTTPS is now implemented through the Jayess TLS transport layer instead of a separate WinHTTP-only request backend.
- `https.get(...)` and `https.getAsync(...)` perform blocking/worker-backed HTTPS GET requests and return the same response shape as `http.get(...)`.
- `https.request(...)` and `https.requestAsync(...)` support request bodies and custom methods through the same TLS-backed transport path.
- `https.getStream(...)` and `https.requestStream(...)` perform blocking HTTPS requests and return after response headers are available, with `response.bodyStream` exposed for incremental reads.
- `https.getStreamAsync(...)` and `https.requestStreamAsync(...)` resolve a promise once response headers are available and return the same streamed response shape as the sync stream variants.
- `https.isAvailable()` reports whether the current native runtime has HTTPS support.
- `https.backend()` reports the active TLS backend name, currently `schannel` on Windows and `openssl` on non-Windows builds.
- HTTPS responses expose `status`, `reason`, `statusText`, `ok`, `headers`, `body`, `bodyBytes`, `redirected`, `redirectCount`, and `url`.
- `rejectUnauthorized` defaults to `true`. Setting it to `false` skips certificate validation checks for development/testing use.
- `https.createServer(...)` accepts PEM file paths in `cert` and `key`, and request handlers receive the same `(req, res)` shape as `http.createServer(...)`.
- HTTPS also passes through TLS trust options like `serverName`, `caFile`, `caPath`, and `trustSystem`.
- HTTPS currently pins ALPN to `http/1.1` because the Jayess HTTP client is still HTTP/1.x only.
- HTTPS redirects are now handled by the Jayess-owned redirect loop, including `maxRedirects`.
- `maxRedirects: 0` disables redirect following and returns the first 30x response directly.
- Non-Windows HTTPS now depends on an OpenSSL-backed TLS build/runtime.
- Custom CA file/path trust configuration now works on both backends.
- On the Schannel backend, Jayess validates the peer certificate against a custom trust collection built from `caFile` / `caPath`, optionally combined with system trust when `trustSystem` is left enabled.
- The current HTTPS server path is blocking and now supports sequential keep-alive requests on the same connection when the client keeps it open.

## TLS

Supported helpers:

- `tls.isAvailable()`
- `tls.backend()`
- `tls.connect(options)`
- `tls.createServer(options, handler)`

Example:

```js
if (tls.isAvailable()) {
  console.log(tls.backend());
}

var socket = tls.connect({ host: "example.com", port: 443 });
if (socket) {
  console.log(socket.secure, socket.backend, socket.protocol, socket.alpnProtocol);
  var cert = socket.getPeerCertificate();
  console.log(cert.subjectCN, cert.issuerCN, cert.serialNumber);
  console.log(cert.validFrom, cert.validTo);
  console.log(cert.subjectAltNames);
  socket.close();
}

var server = tls.createServer({
  cert: "./server-cert.pem",
  key: "./server-key.pem",
}, (socket) => {
  var text = socket.read();
  socket.write("pong");
  socket.close();
});
server.listen(8443, "127.0.0.1");
```

Current boundary:

- `tls.isAvailable()` reports whether a native TLS/HTTPS backend is present in the current build/runtime.
- `tls.backend()` returns the backend name, currently `schannel` on Windows and `openssl` on non-Windows builds.
- `tls.connect(...)` performs a real native TCP+TLS handshake and returns a stream-like `Socket` object with the normal socket lifecycle methods.
- `tls.createServer(...)` performs server-side TLS handshakes and passes accepted secure sockets to the handler.
- `tls.connect(...)` also accepts `alpnProtocols` as either a string or an array of protocol strings.
- `tls.connect(...)` also accepts trust-related options: `serverName`, `caFile`, `caPath`, and `trustSystem`.
- `tls.createServer(...)` currently accepts PEM file paths in `cert` and `key`.
- TLS sockets currently expose `secure`, `authorized`, `backend`, `protocol`, `alpnProtocol`, `alpnProtocols`, and `getPeerCertificate()` in addition to the standard socket properties.
- `socket.getPeerCertificate()` returns an object with `subject`, `issuer`, `subjectCN`, `issuerCN`, `serialNumber`, `validFrom`, `validTo`, `subjectAltNames`, `backend`, and `authorized`, or `undefined` when no peer certificate is available.
- `rejectUnauthorized` defaults to `true`. Setting it to `false` disables certificate rejection for development/testing use.
- `socket.protocol` reports the negotiated TLS protocol version, and `socket.alpnProtocol` reports the negotiated ALPN protocol when one is selected.
- On the OpenSSL backend, `caFile`/`caPath` can provide custom trust roots and `trustSystem: false` disables default system trust roots.
- On the Schannel backend, `caFile`/`caPath` load custom trust roots for post-handshake certificate validation and `trustSystem: false` disables fallback to system trust.
- `https.*` now uses this low-level TLS socket transport instead of a separate request backend.
- `tls.createServer(...)` is currently implemented on non-Windows builds only.

## DNS

Supported helpers:

- `dns.lookup(host)`
- `dns.lookupAll(host)`
- `dns.reverse(address)`

`dns.lookup(host)` resolves a hostname using the platform resolver. It returns an object with `host`, `address`, and `family`, or `undefined` when the hostname cannot be resolved.
`dns.lookupAll(host)` resolves all available IPv4/IPv6 records and returns an array of `{ host, address, family }` objects, or `undefined` when resolution fails.
`dns.reverse(address)` resolves an IPv4 or IPv6 address back to a hostname, or returns `undefined` when the input is not an IP address or no reverse name is available.

## UDP

Supported helpers:

- `net.createDatagramSocket(options)`

Datagram socket methods:

- `address()`
- `send(value, port, host)`
- `receive(size?)`
- `setBroadcast(enabled)`
- `joinGroup(group, interfaceAddress?)`
- `leaveGroup(group, interfaceAddress?)`
- `setMulticastInterface(interfaceAddress)`
- `setMulticastLoopback(enabled)`
- `setTimeout(ms)`
- `close()`

Example:

```js
var receiver = net.createDatagramSocket({ host: "0.0.0.0", port: 9999, type: "udp4" });
var sender = net.createDatagramSocket({ host: "0.0.0.0", port: 0, type: "udp4" });

sender.setBroadcast(true);
sender.send("hello", 9999, "255.255.255.255");

receiver.joinGroup("239.255.0.1", "127.0.0.1");
sender.setMulticastInterface("127.0.0.1");
sender.setMulticastLoopback(true);
sender.send("hello", 9999, "239.255.0.1");
```

Current boundary:

- UDP sockets support send/receive, local bind, timeout, broadcast, and IPv4 multicast group join/leave.
- `receive(...)` returns `{ data, bytes, address, port, family }`.
- The current multicast helpers are IPv4-oriented and use explicit interface addresses such as `127.0.0.1` for loopback tests.

Example:

```js
var result = dns.lookup("localhost");
console.log(result.host);
console.log(result.address);
console.log(result.family); // 4 or 6

var records = dns.lookupAll("localhost");
console.log(records[0].address);

console.log(dns.reverse("127.0.0.1"));
```

Current boundary:

- `dns.lookup(...)` is synchronous.
- `dns.lookupAll(...)` is synchronous.
- `dns.reverse(...)` is synchronous.
- Resolver options and async DNS APIs are not implemented yet.

## Net

Supported helpers:

- `net.isIP(input)`
- `net.connect({ host, port })`
- `net.listen({ host, port })`

`net.isIP(input)` returns `4` for IPv4 input, `6` for IPv6 input, and `0` when the input is not a valid IP address.
`net.connect({ host, port })` opens a blocking TCP client connection and returns a socket object.
`net.listen({ host, port })` opens a blocking TCP server socket and returns a server object.

Example:

```js
console.log(net.isIP("127.0.0.1")); // 4
console.log(net.isIP("::1"));       // 6
console.log(net.isIP("kimchi"));    // 0

var socket = net.connect({ host: "127.0.0.1", port: 8080 });
socket.on("close", () => {
  console.log("closed");
  return 0;
});
console.log(socket.readable, socket.writable);
console.log(socket.localAddress, socket.localPort);
console.log(socket.remoteFamily, socket.localFamily);
console.log(socket.address().address, socket.address().port, socket.address().family);
console.log(socket.remote().address, socket.remote().port, socket.remote().family);
socket.setNoDelay(true);
socket.setKeepAlive(true);
socket.setTimeout(250);
console.log(socket.timeout);
console.log(socket.bytesRead, socket.bytesWritten);
socket.write("ping");
console.log(socket.read(4));
console.log(socket.bytesRead, socket.bytesWritten);
console.log(await socket.writeAsync("pong"));
console.log(await socket.readAsync(4));
socket.end();
console.log(socket.readable, socket.writable);

var server = net.listen({ host: "127.0.0.1", port: 8080 });
server.setTimeout(400);
console.log(server.timeout);
console.log(server.address().address, server.address().port, server.address().family);
var client = await server.acceptAsync();
console.log(server.connectionsAccepted);
console.log(client.read(4));
client.end();
server.close();
```

Current boundary:

- `socket.read(size?)` returns text, `null` at EOF, or `undefined` on error/closed socket.
- `socket.readAsync(size?)` returns a promise-like value resolved with the same result shape as `socket.read(size?)`.
- `socket.readBytes(size?)` returns `Uint8Array`, `null` at EOF, or `undefined` on error/closed socket.
- `socket.write(value)` accepts text or `Uint8Array` and returns `true` or `false`.
- `socket.writeAsync(value)` returns a promise-like value resolved with the same result shape as `socket.write(value)`.
- `socket.end()`, `socket.close()`, and `socket.destroy()` close the socket.
- `socket.setNoDelay(enabled)` configures `TCP_NODELAY` and returns the socket.
- `socket.setKeepAlive(enabled)` configures `SO_KEEPALIVE` and returns the socket.
- `socket.setTimeout(ms)` configures blocking socket receive/send deadlines and returns the socket.
- `socket.readable` and `socket.writable` track whether the socket is still open for blocking reads and writes.
- `socket.timeout` reflects the configured timeout in milliseconds.
- `socket.localAddress` and `socket.localPort` expose the bound local endpoint.
- `socket.remoteFamily` and `socket.localFamily` expose the address family as `4` or `6`.
- `socket.bytesRead` and `socket.bytesWritten` track successful socket I/O byte counts.
- `socket.address()` returns `{ address, port, family }` for the local endpoint.
- `socket.remote()` returns `{ address, port, family }` for the peer endpoint.
- `socket.on("close", callback)` and `socket.once("close", callback)` are supported.
- `socket.on("error", callback)` and `socket.once("error", callback)` are supported.
- `socket.off(...)`, `socket.listenerCount(event)`, and `socket.eventNames()` are supported.
- `server.accept()` blocks until one client connects and returns a `Socket`.
- `server.acceptAsync()` returns a promise-like value resolved with an accepted `Socket`.
- `server.connectionsAccepted` tracks how many accepted sockets have been returned.
- `server.setTimeout(ms)` configures the blocking accept timeout and returns the server.
- `server.timeout` reflects the configured timeout in milliseconds.
- `server.close()` closes the listening socket.
- `server.address()` returns `{ address, port, family }`.
- `server.on("close", callback)` and `server.on("error", callback)` are supported.
- Socket I/O is blocking in this first pass.
- Nonblocking accept loops, connection events, TLS, UDP, and HTTP layering are not implemented yet.

## Planned Networking APIs

Useful next layers:

- evented `Socket`/`Server` integration with the runtime scheduler
- streaming HTTP bodies and nonblocking `http.request(...)` on top of scheduler-backed TCP streams.
- `tls.connect(...)` once TLS linkage and certificate validation are designed.
- `dgram.createSocket(...)` for UDP.
