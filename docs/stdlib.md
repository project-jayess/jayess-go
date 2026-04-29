# Standard Library

This page summarizes the standard library surface currently wired into the Jayess compiler/runtime.

## Console

Jayess supports:

- `console.log(...)`
- `console.warn(...)`
- `console.error(...)`

`print(...)` still works, but it is deprecated.

See [Console](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/console.md).

## Process, Path, and File System

Supported process helpers:

- `process.cwd()`
- `process.argv()`
- `process.env(name)`
- `process.platform()`
- `process.arch()`
- `process.threadPoolSize()`
- `process.exit(code)`
- `process.onSignal(signal, callback)`
- `process.onceSignal(signal, callback)`
- `process.offSignal(signal[, callback])`
- `process.raise(signal)`

## Runtime Compilation

- `compile(source, outputPath?)`
- `compileFile(inputPath, outputPathOrOptions?)`

`compile(...)` compiles Jayess source text into a native executable artifact. It is not `eval`: it writes source to a temporary file, invokes the external Jayess compiler, and returns an object with `ok`, `output`, `status`, `stdout`, `stderr`, and `error`.

```js
var result = compile("function main() { return 0; }", "build/generated");
console.log(result.ok);
console.log(result.output);
console.log(result.stdout);
console.log(result.stderr);
console.log(result.error);
```

The second argument can also be an options object:

```js
var result = compile("function main() { return 0; }", {
  output: "build/generated",
  target: "windows-x64",
  emit: "exe",
  warnings: "error"
});
```

Runtime compilation uses the `JAYESS_COMPILER` environment variable when set, otherwise it runs `jayess` from `PATH`. The runtime launches the compiler directly with argv-style arguments and captures stdout/stderr. The compiled executable is not run automatically.

Use `compileFile(...)` when the source already exists on disk:

```js
var result = compileFile("src/main.js", {
  output: "build/main",
  emit: "exe"
});
```

`compileFile(...)` passes the real input path to the compiler, which preserves file-based diagnostics and import/package resolution.
Supported option values are intentionally narrow: `emit` can be `"exe"` or `"llvm"`, `warnings` can be `"default"`, `"none"`, or `"error"`, and `target` may contain only letters, digits, `.`, `_`, and `-`.

Supported path helpers:

- `path.join(...parts)`
- `path.normalize(path)`
- `path.resolve(...parts)`
- `path.relative(from, to)`
- `path.parse(path)`
- `path.format(parts)`
- `path.sep`
- `path.delimiter`
- `path.isAbsolute(path)`
- `path.basename(path)`
- `path.dirname(path)`
- `path.extname(path)`

Supported file helpers:

- `fs.readFile(path)`
- `fs.readFile(path, encoding)`
- `fs.readFileAsync(path)`
- `fs.readFileAsync(path, encoding)`
- `fs.writeFile(path, content)`
- `fs.exists(path)`
- `fs.readDir(path[, options])`
- `fs.stat(path)`
- `fs.mkdir(path[, options])`
- `fs.remove(path)`
- `fs.copyFile(from, to)`
- `fs.copyDir(from, to)`
- `fs.rename(from, to)`
- `fs.symlink(target, path)`
- `fs.watch(path)`
- `fs.watchSync(path)`

Notes:

- `fs.readFile(path)` returns file contents or `undefined`
- `fs.readFile(path, encoding)` currently supports text encodings like `utf8`
- `fs.readFileAsync(path[, encoding])` returns a promise-like value and schedules the file read through the Jayess task queue
- `fs.writeFile(path, content)` returns a boxed boolean-like result
- `fs.mkdir(path, { recursive: true })` creates parent directories when needed
- `fs.readDir(path, { recursive: true })` walks nested directories
- `fs.readDir(path)` returns entry objects with `name`, `path`, `isDir`, `isFile`, and `size`
- `fs.readDir(path)` entry objects also include `mtimeMs` and `permissions` where available
- `fs.stat(path)` returns file metadata with `path`, `isDir`, `isFile`, `size`, `mtimeMs`, and `permissions`
- `permissions` is normalized by host family: POSIX hosts return a 9-character `rwxrwxrwx`-style string, while Windows currently returns the coarse capability string `"rwx"`
- `fs.remove(path)` distinguishes files and directories
- `fs.remove(path, { recursive: true })` removes directory trees recursively
- `fs.copyDir(from, to)` copies directory trees recursively
- `fs.symlink(target, path)` creates a symbolic link when the host platform permits it
- `fs.watch(path)` creates a polling-based watcher that works for both files and directories and is intended for async use through `pollAsync(timeoutMs)`
- `fs.watchSync(path)` creates the same watcher type for explicit synchronous polling with `poll()`
- `path.parse(path)` returns an object with `root`, `dir`, `base`, `ext`, and `name`
- drive-root paths like `C:/tmp/nested/file.txt` are parsed and normalized consistently even on non-Windows hosts

Supported compression helpers:

- `compression.gzip(value)`
- `compression.gunzip(value)`
- `compression.deflate(value)`
- `compression.inflate(value)`
- `compression.brotli(value)`
- `compression.unbrotli(value)`
- `compression.createGzipStream()`
- `compression.createGunzipStream()`
- `compression.createDeflateStream()`
- `compression.createInflateStream()`
- `compression.createBrotliStream()`
- `compression.createUnbrotliStream()`

Compression notes:

- the direct helpers return compressed or decompressed byte values, with invalid compressed input returning `undefined`
- the stream constructors currently expose synchronous transform-style stream objects
- those compression stream objects are also usable as duplex read/write objects in the current runtime by writing input, calling `end()`, and then reading the transformed output back with `read()` or `readBytes(...)`
- `pipe(...)` through compression streams is proven, and the current file/compression stream model now also exposes synchronous backpressure state through `write(...)`, `writableLength`, `writableHighWaterMark`, `writableNeedDrain`, and `drain`
- `drain` is still synchronous in the current runtime; asynchronous scheduling and TCP-level backpressure are separate boundaries

See also:

- [File System](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/fs.md)
- [Path](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/path.md)
- [Process](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/process.md)

## Networking Foundation

Supported URL helpers:

- `url.parse(input)`
- `url.format(parts)`

Supported query-string helpers:

- `querystring.parse(input)`
- `querystring.stringify(parts)`

Supported DNS helpers:

- `dns.lookup(host)`
- `dns.lookupAll(host)`
- `dns.reverse(address)`
- `dns.setResolver(options)`
- `dns.clearResolver()`

`dns.lookup(host)` returns an object with `host`, `address`, and `family`, or `undefined` when resolution fails.
`dns.lookupAll(host)` returns an array of address records, or `undefined` when resolution fails.
`dns.reverse(address)` returns a hostname string, or `undefined` when reverse lookup fails.
`dns.setResolver({ hosts, reverse })` installs a synchronous override table ahead of the platform resolver, where `hosts` maps hostnames to one address string or an array of address strings and `reverse` maps address strings back to hostnames.
That override layer gives Jayess a deterministic, host-independent DNS/IP helper path for tests and applications that need the same lookup behavior across Linux, macOS, and Windows, even though full socket and HTTP runtime coverage is still only exercised on Linux here.
`dns.clearResolver()` removes that override table and returns the runtime to the platform resolver only.

Supported net helpers:

- `net.isIP(input)`
- `net.connect({ host, port })`
- `net.listen({ host, port })`

Supported http helpers:

- `http.parseRequest(input)`
- `http.formatRequest(parts)`
- `http.parseResponse(input)`
- `http.formatResponse(parts)`
- `http.createServer(handler)`
- `https.createServer(options, handler)`
- `http.request(options)`
- `http.get(input)`
- `http.requestAsync(options)`
- `http.getAsync(input)`

`net.isIP(input)` returns `4`, `6`, or `0`.
`net.connect({ host, port })` returns a blocking TCP socket object.
`net.listen({ host, port })` returns a blocking TCP server object.
`net.createDatagramSocket({ host, port, type, timeout?, broadcast? })` returns a UDP datagram socket object.
`http.parseRequest(...)` and `http.parseResponse(...)` return parsed HTTP message objects.
`http.formatRequest(...)` and `http.formatResponse(...)` build HTTP/1.x message text.
`http.createServer(handler)` creates a blocking plain-HTTP server object with `listen(port[, host])` and `close()`. `https.createServer(options, handler)` creates the same public server shape over TLS on non-Windows builds, currently loading PEM file paths from `options.cert` and `options.key`. In both cases the handler receives `(req, res)`, where `req` currently exposes `method`, `url`, `path`, `headers`, and `body`, and `res` exposes `statusCode`, `setHeader(name, value)`, `write(chunk)`, and `end(chunk?)`.
`http.request(...)` and `http.get(...)` perform blocking plain-HTTP client requests and return parsed response objects, including URL-string input such as `http.get("http://host:8080/path")` and `http.request({ url: "http://host/path", ... })`.
`http.requestAsync(...)` and `http.getAsync(...)` return promise-like values resolved with the same response shape on the current worker-pool async runtime.
The client synthesizes `Host`, `Connection: close`, and `Content-Length` for non-empty bodies when they are missing.
The HTTP request options also accept `timeout` in milliseconds; timeout or other transport failure currently yields `undefined`.
The HTTP client follows plain-HTTP redirects for `301`, `302`, `303`, `307`, and `308`, with `maxRedirects` defaulting to `5`.
Final HTTP responses expose `redirected`, `redirectCount`, and `url`.
Final HTTP responses also expose `ok` and `statusText`.
Final HTTP responses also expose `bodyBytes` as a `Uint8Array`.
`http.requestStream(...)` and `http.getStream(...)` expose `response.bodyStream` for incremental blocking reads instead of forcing full-body buffering up front. `http.requestStreamAsync(...)` and `http.getStreamAsync(...)` resolve a promise once headers are available and return the same streamed response shape.
Chunked HTTP response bodies are decoded automatically when `Transfer-Encoding: chunked` is present.
The client now finishes once a full `Content-Length` or chunked response has arrived rather than always waiting for connection close.
The current HTTP and HTTPS server path now supports sequential keep-alive requests on the same connection when the client keeps it open.
Jayess exposes an HTTPS surface through `https.createServer(...)`, `https.get(...)`, `https.request(...)`, `https.getStream(...)`, `https.requestStream(...)`, `https.getStreamAsync(...)`, `https.requestStreamAsync(...)`, `https.getAsync(...)`, and `https.requestAsync(...)`, with `rejectUnauthorized` available for development/test certificate bypass on the client side. `https.request(...)` and `https.requestAsync(...)` support custom methods and request bodies through the same TLS-backed transport, and the stream variants expose `response.bodyStream` for incremental reads once headers are available. `https.createServer(...)` currently provides a blocking one-request-per-connection server path on non-Windows builds.
The runtime also exposes `tls.isAvailable()`, `tls.backend()`, `tls.connect(...)`, `tls.createServer(...)`, `https.isAvailable()`, and `https.backend()` so Jayess code can branch on current TLS/HTTPS capability.
Socket objects support `read`, `readAsync`, `readBytes`, `write`, `writeAsync`, `end`, `close`, `destroy`, `setNoDelay`, `setKeepAlive`, `setTimeout`, `on`, `once`, `off`, `listenerCount`, `eventNames`, `readable`, `writable`, `writableLength`, `writableHighWaterMark`, `writableNeedDrain`, `timeout`, `localAddress`, `localPort`, `remoteFamily`, `localFamily`, `bytesRead`, `bytesWritten`, `address`, `remote`, `protocol`, `alpnProtocol`, `alpnProtocols`, and `getPeerCertificate()`.
Server objects support `accept`, `acceptAsync`, `close`, `address`, `setTimeout`, `timeout`, `connectionsAccepted`, `on`, `once`, `off`, `listenerCount`, and `eventNames`. `server.address()` returns `{ address, port, family }`.
Datagram sockets support `address`, `send`, `receive`, `setBroadcast`, `joinGroup`, `leaveGroup`, `setMulticastInterface`, `setMulticastLoopback`, `setTimeout`, `close`, `broadcast`, `multicastInterface`, and `multicastLoopback`.

Jayess now also ships an in-tree native HTTP parsing package:

- `@jayess/httpserver`
- `parseRequest(text)`
- `parseResponse(text)`
- `parseRequestIncremental(text, lastLen)`
- `parseResponseIncremental(text, lastLen)`
- `decodeChunked(textOrBytes)`
- `formatRequest(parts)`
- `formatResponse(parts)`

`@jayess/httpserver.parseRequest(...)` and `parseResponse(...)` are backed by PicoHTTPParser through the Jayess native binding path. Request parsing returns `method`, `path`, `query`, `httpVersion`, `headers`, and `body`; response parsing returns `status`, `reason`, `httpVersion`, `headers`, and `body`. Malformed parse input raises `SyntaxError` instead of silently returning `undefined`.
`parseRequestIncremental(...)` and `parseResponseIncremental(...)` return `{ complete, consumed, message }`, where `message` is present only once a full message has been parsed. `decodeChunked(...)` accepts either HTTP chunked body text or a `Uint8Array` and returns `{ complete, body, remaining }`.
`@jayess/httpserver.formatRequest(...)` and `formatResponse(...)` forward to the built-in `http.formatRequest(...)` and `http.formatResponse(...)` helpers so package consumers can pair native request/response parsing with the existing formatter surface.

Jayess also ships an in-tree Mongoose binding package:

- `@jayess/mongoose`
- `createManager()`
- `listenServer(manager, url)`
- `listenTlsServer(manager, url, certPath, keyPath)`
- `pollManager(manager, timeoutMs)`
- `nextEvent(manager)`
- `reply(connection, status, headers, body)`
- `upgradeWebSocket(event)`
- `sendWebSocket(connection, data)`
- `closeConnection(connection)`
- `freeManager(manager)`
- `createRouter()`
- `addRoute(router, method, path, handler)`
- `get(router, path, handler)`
- `post(router, path, handler)`
- `put(router, path, handler)`
- `del(router, path, handler)`
- `all(router, path, handler)`
- `createEmbeddedApp(files, fallbackPath?)`
- `serveEmbeddedApp(event, urlPrefix, app[, fallbackPath])`
- `serveStatic(event, urlPrefix, rootDir)`
- `startChunked(event, status, headers)`
- `writeChunk(stream, chunk)`
- `endChunked(stream[, finalChunk])`
- `dispatch(router, event)`

`@jayess/mongoose` is a manual `*.bind.js` package layered over a small C shim and vendored sources from [refs/mongoose](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/mongoose). The current package exposes a polling server-manager model rather than native callbacks into Jayess: `pollManager(...)` drives Mongoose, `nextEvent(...)` returns the next queued event as a Jayess object, and `reply(...)` sends an HTTP response for an accepted connection. Manager and connection values are represented as native handles and can be shut down cleanly through `freeManager(...)` and `closeConnection(...)`. `listenServer(...)` binds plain HTTP listeners, and `listenTlsServer(...)` binds HTTPS listeners using certificate/key PEM files and Mongoose's built-in TLS mode. For package-level routing, `createRouter()` plus `get(...)`, `post(...)`, `put(...)`, `del(...)`, `all(...)`, and `dispatch(...)` provide dispatch helpers on top of that polling model, and routed Jayess handlers can safely capture and reuse Jayess state across multiple native events. `serveStatic(event, urlPrefix, rootDir)` serves matching GET requests directly through the native binding and returns a boolean indicating whether the request was handled. For embedded web-app style serving, `createEmbeddedApp(...)` and `serveEmbeddedApp(...)` let Jayess code prepare in-memory files and SPA fallbacks without touching the filesystem, which is useful for webview-style integrations; the current package can serve HTML/JS assets directly from memory and fall back to an embedded `index.html` for SPA-style routes. For streamed responses, `startChunked(...)` opens a chunked HTTP response, `writeChunk(...)` appends body chunks, and `endChunked(...)` terminates the stream and closes the response cleanly. For WebSocket routes, HTTP upgrade requests can be turned into WebSocket connections with `upgradeWebSocket(event)`, after which `nextEvent(...)` yields `wsOpen` and `wsMessage` events and `sendWebSocket(connection, data)` sends text frames. Invalid TLS file paths and invalid WebSocket upgrades propagate as normal Jayess errors, so they can be caught with `try/catch` and surfaced through standard diagnostics.

Jayess also ships an in-tree SQLite binding package:

- `@jayess/sqlite`
- `open(path)`
- `close(db)`
- `exec(db, sql)`
- `prepare(db, sql)`
- `finalize(stmt)`
- `reset(stmt)`
- `clearBindings(stmt)`
- `bindNull(stmt, index)`
- `bindInteger(stmt, index, value)`
- `bindFloat(stmt, index, value)`
- `bindString(stmt, index, value)`
- `bindBlob(stmt, index, value)`
- `get(stmt)`
- `getArray(stmt)`
- `all(stmt)`
- `allArray(stmt)`
- `run(stmt)`
- `changes(db)`
- `lastInsertRowId(db)`
- `busyTimeout(db, milliseconds)`
- `begin(db)`
- `commit(db)`
- `rollback(db)`
- `query(db, sql)`
- `queryArray(db, sql)`
- `pragma(db, sql)`

`@jayess/sqlite` is a manual `*.bind.js` package layered over a small C shim and the vendored SQLite amalgamation in [refs/sqlite](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/sqlite). Database values are represented as managed `SQLiteDatabase` native handles, and prepared statements are represented as managed `SQLiteStatement` handles. `get(stmt)` returns the next row as an object keyed by column name, while `getArray(stmt)` returns the next row as an array indexed by column position; both return `null` once the statement is done. `all(...)` and `allArray(...)` provide row-iteration helpers on top of that step path. Bound strings and blobs are copied into SQLite before execution, and text/blob values read from rows are copied back into normal Jayess values, so they remain valid after the statement is finalized and the database is closed. Invalid SQL and other SQLite failures propagate as `SQLiteError`, and using finalized or closed handles afterward raises `TypeError` through the normal native-handle safety path.

Jayess also ships an in-tree libcurl binding package:

- `@jayess/curl`
- `createEasy()`
- `configure(handle, options)`
- `perform(handle)`
- `performStream(handle, onChunk)`
- `cleanup(handle)`
- `createMulti()`
- `addHandle(multi, handle)`
- `performMulti(multi)`
- `cleanupMulti(multi)`
- `request(options)`
- `requestStream(options, onChunk)`
- `requestMulti(optionsList)`
- `requestAsync(options)`
- `requestMultiAsync(optionsList)`

`@jayess/curl` is a manual `*.bind.js` package layered over a small C shim, vendored curl headers from [refs/curl](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/curl), and the host libcurl runtime on Linux. The normal package path already compiles against those vendored headers through `includeDirs: ["../../../../refs/curl/include"]`, while still linking to the host shared libcurl runtime on the tested Linux path. Easy handles are represented as managed `CurlEasy` native handles and can be closed explicitly with `cleanup(...)`. `configure(...)` currently supports `url`, `method`, `headers`, `body`, `followRedirects`, `timeoutMs`, `insecure`, `outputPath`, `cookie`, and `proxy`. `perform(...)` returns an object with `status`, `headers`, `body`, and `path`; when `outputPath` is provided, the response body is written directly to disk and `path` points at the written file. `performStream(...)` and `requestStream(...)` keep the same blocking transfer model but stream response body chunks into a Jayess callback as libcurl receives them, then return a final object with `status`, `headers`, `chunks`, and no buffered `body`. The package also exposes a managed `CurlMulti` handle for blocking multi-transfer batches: `createMulti(...)` allocates the multi handle, `addHandle(...)` adds configured easy handles, `performMulti(...)` drives all currently-added transfers through libcurl’s multi interface and returns an array of response objects in handle order, `cleanupMulti(...)` closes the multi handle, and `requestMulti(...)` is the convenience wrapper that configures, performs, and cleans up a whole batch in one call. On top of that blocking/native surface, `requestAsync(...)` and `requestMultiAsync(...)` now run the existing request helpers on a Jayess worker and resolve a Promise back on the main scheduler, so timers and other Jayess async work can continue while the transfer is in flight. The current smoke coverage proves plain HTTP, HTTPS with `insecure: true`, redirect following, timeout errors, uploads, cookie submission, direct download-to-file, proxy routing, streaming body delivery, blocking multi batches, and Promise-based async request/multi wrappers through this package. Transfer failures and configuration problems surface as `CurlError`, and missing libcurl header or link inputs produce explicit native build diagnostics.

Jayess also ships an in-tree libuv binding package:

- `@jayess/libuv`
- `createLoop()`
- `run(loop)`
- `runOnce(loop)`
- `stop(loop)`
- `closeLoop(loop)`
- `scheduleStop(loop, timeoutMs)`
- `scheduleCallback(loop, timeoutMs, callback)`
- `readFile(loop, path, callback)`
- `watchSignal(loop, signal, callback)`
- `closeSignalWatcher(watcher)`
- `watchPath(loop, path, callback)`
- `closePathWatcher(watcher)`
- `spawnProcess(loop, file, args, callback)`
- `closeProcess(process)`
- `createUDP(loop)`
- `bindUDP(socket, host, port)`
- `recvUDP(socket, callback)`
- `sendUDP(socket, host, port, data)`
- `closeUDP(socket)`
- `createTCPServer(loop)`
- `listenTCP(server, host, port, callback)`
- `acceptTCP(server)`
- `closeTCPServer(server)`
- `createTCPClient(loop)`
- `connectTCP(client, host, port, callback)`
- `readTCP(client, callback)`
- `writeTCP(client, data)`
- `closeTCPClient(client)`
- `now(loop)`

`@jayess/libuv` is a manual `*.bind.js` package layered over a small C shim and vendored libuv sources from [refs/libuv](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/libuv). On Linux, the binding now builds directly from the upstream Unix/Linux source set instead of relying on a preinstalled `libuv.so`, using the normal pthread/dl/rt link flags. Loop values are represented as managed `UVLoop` native handles and can be closed explicitly with `closeLoop(...)`; using a closed loop afterward raises `TypeError` through the normal native-handle safety path. The current surface focuses on loop lifecycle plus a narrow callback/filesystem/signal/watcher/process/network bridge: `scheduleStop(...)` installs a temporary native timer that stops the loop after a delay, `scheduleCallback(...)` installs a temporary native timer that schedules a normal Jayess callback back onto the Jayess scheduler, `readFile(...)` performs an async libuv file read and completes with a single callback result object `{ ok, data, error }`, `watchSignal(...)` registers a libuv signal watcher on the loop and invokes the Jayess callback with names like `SIGUSR1`, `closeSignalWatcher(...)` closes that watcher explicitly, `watchPath(...)` registers a libuv filesystem event watcher and invokes the Jayess callback with `{ ok, path, eventType, error }`, `closePathWatcher(...)` closes that watcher explicitly, `spawnProcess(...)` starts a subprocess on the uv loop and invokes the callback as `(result, process)` where `result` includes `exitStatus` and `signal`, `closeProcess(...)` closes that process handle explicitly, `createUDP(...)` creates a managed datagram socket, `bindUDP(...)` binds it, `recvUDP(...)` installs a receive callback that yields `{ ok, data, address, port, error }`, `sendUDP(...)` sends a datagram, `closeUDP(...)` closes the socket explicitly, `createTCPServer(...)` creates a managed TCP server, `listenTCP(...)` binds and listens, `acceptTCP(...)` returns the next accepted managed client handle from the listen callback path, `createTCPClient(...)` creates an outbound TCP client handle, `connectTCP(...)` reports connection completion as `{ ok, error }`, `readTCP(...)` yields `{ ok, data, error }`, `writeTCP(...)` sends a string payload, and `closeTCPServer(...)` / `closeTCPClient(...)` close those handles explicitly. `run(...)` and `runOnce(...)` cooperate with the Jayess scheduler instead of blocking it completely, so pending Promise microtasks, `timers.setTimeout(...)` callbacks, libuv fs completions, libuv signal deliveries, libuv path-watch events, libuv process exits, libuv UDP datagrams, and libuv TCP events can continue to make progress while the libuv loop is running on the same thread. The current ownership model is deliberately narrow: the Jayess code that creates the loop also drives and closes it on the same thread, and there is not yet any cross-thread callback or worker handoff surface for libuv handles. Missing libuv build inputs still produce explicit native build diagnostics, and native loop/setup failures surface as `LibUVError`.

Jayess also ships an in-tree HTML parsing package:

- `@jayess/html`
- `tokenizeHtml(input)`
- `parseHtml(input)`
- `parseHtmlFragment(input)`
- `serializeHtml(node)`
- `serializeHtmlWithOptions(node, options)`
- `createElement(tagName, attributes, children)`
- `createText(value)`
- `createComment(value)`
- `setAttribute(node, name, value)`
- `removeAttribute(node, name)`
- `appendChild(node, child)`
- `removeChild(node, index)`
- `replaceChild(node, index, child)`
- `cloneNode(node)`
- `walkDepthFirst(node)`
- `findByTag(node, tagName)`
- `matchesSelector(node, selector)`
- `querySelectorAll(node, selector)`

`@jayess/html` is a small native `*.bind.js` package implemented directly in the repo without external parser dependencies. The current surface is intentionally narrow and HTML-only: `tokenizeHtml(...)` returns a token array, while `parseHtml(...)` returns a document node and `parseHtmlFragment(...)` returns a fragment node. The current proven token slice is separate from parsing and covers doctype tokens, start tags, end tags, text nodes, comments, attributes, boolean attributes, self-closing tags, and token spans. Parsed trees build plain Jayess object/array AST values with node types like `document`, `fragment`, `doctype`, `element`, `text`, and `comment`. Parsed nodes now also carry a `span` object with `start` and `end` positions, and each position currently exposes `offset`, `line`, and `column`, so Jayess programs can correlate parsed structure back to the original source text. Element nodes currently expose `tagName`, `attributes`, `attributeOrder`, `children`, and `selfClosing`, so Jayess programs can inspect and reuse the tree directly. The proved parser slice covers standard tag syntax, document doctypes, nested elements, self-closing tags, string and boolean attributes, text nodes, and HTML comments, and `serializeHtml(...)` can turn that AST back into an HTML string. `serializeHtmlWithOptions(node, { comments: false })` is also proven and strips comment nodes during serialization while leaving the AST untouched, `serializeHtmlWithOptions(node, { pretty: true })` emits a readable indented layout for nested element trees, and `serializeHtmlWithOptions(node, { minify: true })` explicitly selects the compact serialization path. The package now also exposes a small AST manipulation layer: nodes can be created programmatically, attributes can be added or removed, children can be appended/removed/replaced, trees can be cloned, a depth-first traversal order can be materialized with `walkDepthFirst(...)`, and simple node queries can be done with `findByTag(...)`. On top of that, the current selector layer supports `matchesSelector(...)` and `querySelectorAll(...)` for a narrow but useful subset: tag selectors, `#id`, `.class`, attribute selectors like `[disabled]` and `[id=a]`, descendant combinators, child combinators, and basic pseudo-classes through the tree-query path. The currently proved pseudo-class subset is `:first-child`, `:last-child`, and `:empty`. The current backend coverage proves those helpers preserve structure correctly on the HTML AST values produced by the parser, that source spans line up with concrete offsets/line/column expectations on parsed nodes, and that the selector engine works against real parsed trees. The current backend coverage also proves the parser can be fed from `fs.readFile(...)`, that malformed input such as an unterminated comment raises `HTMLParseError`, that recoverable malformed tree shapes like `<div><span>x</div>` or truncated input like `<div><span>x` are handled without crashing by returning usable partial trees, and that the `fs.readFile(...) -> parseHtml(...)` ownership path stays leak-free under the current Linux ASAN/LSAN parser probe. XML and CSS stay intentionally strict on the currently proved path and raise parse errors instead of attempting partial-tree recovery.

Jayess also ships an in-tree XML parsing package:

- `@jayess/xml`
- `tokenizeXml(input)`
- `parseXml(input)`
- `serializeXml(node)`
- `serializeXmlWithOptions(node, options)`

`@jayess/xml` is a small native `*.bind.js` package implemented directly in the repo without external parser dependencies. The current proven surface is intentionally strict and document-oriented: `tokenizeXml(...)` returns a token array, `parseXml(...)` returns a document node, and `serializeXml(...)` turns that AST back into an XML string. The token and AST slice currently covers processing instructions, comments, elements, quoted attributes, text nodes, CDATA sections, self-closing tags, and end tags, with `span.start/end.{offset,line,column}` attached across the parsed tree. The parser enforces strict well-formed XML rules on the proved path: mismatched closing tags, missing root elements, unexpected top-level text, unquoted attributes, and unterminated comments or processing instructions raise `XMLParseError`. Element nodes follow the same broad object shape as the HTML package where practical, with `tagName`, `attributes`, `attributeOrder`, `children`, and `selfClosing`, while processing instructions expose `target` and `data`, and CDATA nodes expose `value`. The current backend coverage also proves a basic namespace annotation layer on parsed XML trees: element nodes now expose `prefix`, `localName`, and resolved `namespaceURI`, while `attributeDetails[name]` exposes the same metadata for attributes, including default namespaces on elements, prefixed element namespaces, prefixed attribute namespaces, and the XMLNS declaration namespace URI. The current backend coverage proves the package can be driven from `fs.readFile(...)`, preserves tree structure and node order, supports nested elements, round-trips a real XML document through `serializeXml(...)`, strips comment nodes when asked through `serializeXmlWithOptions(node, { comments: false })`, emits indented nested-element formatting through `serializeXmlWithOptions(node, { pretty: true })`, explicitly selects the compact path through `serializeXmlWithOptions(node, { minify: true })`, and keeps the `fs.readFile(...) -> parseXml(...)` ownership path leak-free under the current Linux ASAN/LSAN parser probe.

Jayess also ships an in-tree CSS parsing package:

- `@jayess/css`
- `tokenizeCss(input)`
- `parseCss(input)`
- `serializeCss(node)`
- `serializeCssWithOptions(node, options)`

`@jayess/css` is a small native `*.bind.js` package implemented directly in the repo without external parser dependencies. The current proven surface is intentionally narrow and stylesheet-oriented: `tokenizeCss(...)` returns a token array, `parseCss(...)` returns a stylesheet AST, and `serializeCss(...)` turns that AST back into a CSS string. The current token slice covers comments, identifiers, punctuation, strings, numbers, dimensions such as `1.5rem` or `8px`, generic delimiters, and `atKeyword` tokens. The parsed AST currently uses `stylesheet`, `rule`, `declaration`, `comment`, and `atRule` nodes. Rule nodes expose `selector`, `selectorTokens`, and ordered `declarations`; declaration nodes expose `property`, `value`, and `valueParts`; at-rule nodes expose `name`, `prelude`, `preludeTokens`, and nested `rules`; and nodes carry `span.start/end.{offset,line,column}` in the same style as the HTML and XML packages. The proved parser slice covers ordinary stylesheet rules, selector tokenization, declaration blocks, raw value parsing for identifiers/strings/dimensions, comment parsing, rule-order preservation, `@import` statements, and block `@media` rules containing nested ordinary style rules. The current backend coverage also proves the stylesheet can be read from `fs.readFile(...)`, serialized back to CSS, comments can be stripped during serialization through `serializeCssWithOptions(node, { comments: false })`, indented formatting can be requested through `serializeCssWithOptions(node, { pretty: true })`, `serializeCssWithOptions(node, { minify: true })` explicitly selects the compact serialization path, unsupported at-rules such as `@supports` still fail clearly with `CSSParseError` instead of being silently misparsed, and the `fs.readFile(...) -> parseCss(...)` ownership path stays leak-free under the current Linux ASAN/LSAN parser probe.

The current backend and compiler coverage also proves the HTML, XML, and CSS parser packages can cross an ordinary Jayess module boundary cleanly: each package can be imported into a local user module, wrapped there, and then consumed from another file without losing its native binding surface.

The current backend coverage also includes a large-input sanity path for all three parser packages: synthetic HTML, XML, and CSS files with 3000 repeated elements or rules are parsed successfully and complete within the normal executable test budget on the tested Linux path. This is a bounded runtime sanity proof, not a full benchmark suite.

The current backend coverage also proves that parser span coordinates use the same 1-based line/column convention as Jayess compiler diagnostics. HTML, XML, and CSS parsed nodes are compared against mirrored compile-time error locations so those coordinate systems stay aligned at the user-facing reporting level.

Jayess also ships an in-tree OpenSSL binding package:

- `@jayess/openssl`
- `randomBytes(length)`
- `version()`
- `supportsHash(algorithm)`
- `supportsCipher(algorithm)`
- `hash(algorithm, value)`
- `hmac(algorithm, key, value)`
- `encrypt(options)`
- `decrypt(options)`
- `generateKeyPair(options)`
- `publicEncrypt(options)`
- `privateDecrypt(options)`
- `sign(options)`
- `verify(options)`
- `tlsAvailable()`
- `tlsBackend()`
- `tlsConnect(options)`
- `tlsCreateServer(options, handler)`

`@jayess/openssl` is a manual `*.bind.js` package layered over a small C shim, vendored OpenSSL headers from [refs/openssl](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/openssl), and the host OpenSSL libraries on the tested Linux path. The normal package build now compiles the OpenSSL shim with those vendored headers without leaking them into the Jayess runtime build, so the package can pin its own OpenSSL header surface while still linking against the host `libssl` / `libcrypto` on this machine. `randomBytes(...)` returns a byte array, `version()` reports the linked OpenSSL version string, and `supportsHash(...)` / `supportsCipher(...)` let callers gate feature use against the actual host build. `hash(...)` and `hmac(...)` return lowercase hexadecimal digests, and `encrypt(...)` / `decrypt(...)` target the same AES-GCM option shape used by Jayess' built-in crypto helpers. The package also exposes RSA keypair generation plus OAEP public-key encryption and RSA-PSS signing helpers through `generateKeyPair(...)`, `publicEncrypt(...)`, `privateDecrypt(...)`, `sign(...)`, and `verify(...)`. Generated keys are exposed as normal Jayess objects with PEM text, not as native key handles, and peer certificates returned from TLS sockets are copied into plain Jayess objects, so key/certificate values remain usable after the originating native operation or socket has been released. For explicit TLS use, `tlsAvailable()` and `tlsBackend()` report package-visible capability, `tlsConnect(...)` wraps the OpenSSL-backed client path with support for `caFile`, `caPath`, `trustSystem`, `serverName`, and `alpnProtocols`, and `tlsCreateServer(...)` wraps the OpenSSL-backed server path using PEM `cert` and `key` files. Invalid digest/HMAC algorithms propagate as `OpenSSLError`, while authenticated decrypt failures return `undefined` rather than silently producing corrupted plaintext.

Jayess also ships an in-tree GLFW binding package:

- `@jayess/glfw`
- `init()`
- `terminate()`
- `createWindow(width, height, title)`
- `createOpenGLWindow(width, height, title)`
- `destroyWindow(window)`
- `pollEvents()`
- `swapBuffers(window)`
- `makeContextCurrent(window)`
- `isContextCurrent(window)`
- `swapInterval(interval)`
- `getProcAddress(name)`
- `hasProcAddress(name)`
- `windowShouldClose(window)`
- `getTime()`
- `setTime(value)`
- `getWindowSize(window)`
- `setWindowSize(window, width, height)`
- `getFramebufferSize(window)`
- `isJoystickPresent(joystick)`
- `isJoystickGamepad(joystick)`
- `getJoystickName(joystick)`
- `getGamepadName(joystick)`
- `getGamepadButton(joystick, button)`

`@jayess/glfw` is a manual `*.bind.js` package layered over a small C shim and vendored GLFW sources from [refs/glfw](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/glfw). For the tested Linux path it builds against the GLFW null platform backend, so `init()`, `createWindow(...)`, `createOpenGLWindow(...)`, `pollEvents()`, `swapBuffers(...)`, `makeContextCurrent(...)`, `isContextCurrent(...)`, `swapInterval(...)`, `getProcAddress(...)`, `hasProcAddress(...)`, `getTime()`, `setTime(...)`, `getWindowSize(...)`, `setWindowSize(...)`, `getFramebufferSize(...)`, `setWindowFullscreen(...)`, `setWindowWindowed(...)`, joystick/gamepad presence/name/button queries, and `destroyWindow(...)` work without relying on a system `libglfw`. The package also exposes callback registration for keyboard, mouse button, cursor position, and scroll events. On the vendored null backend, those callback paths are currently proven through GLFW's own callback registration plus synthetic event injection helpers: `simulateKeyEvent(...)`, `simulateMouseButtonEvent(...)`, `simulateCursorPosEvent(...)`, and `simulateScrollEvent(...)`. The current lifecycle proof also keeps a Jayess worker alive while the GLFW window loop is active, so worker message passing can coexist with the current GLFW poll/swap flow. That same compiled executable path now also proves `@jayess/audio` playback can coexist with the GLFW app loop when the audio package is rebound to the local Cubeb stub fixture: the GLFW render/poll path, a Jayess worker round trip, and a live audio playback callback loop all make progress in one process. A separate executable proof now also shows that a manual image-decoder binding can coexist with the GLFW rendering path in the same process: a vendored null-backend GLFW OpenGL window can be created, polled, and swapped while a tiny `stb_image`-based binding loads real image metadata from disk. The OpenGL path now also proves procedure lookup for active contexts through `getProcAddress(...)` and `hasProcAddress(...)`, returning a BigInt pointer value or `undefined` for the raw lookup and a boolean result for the capability check. The Vulkan path is now proven one step further: `isVulkanSupported()`, `getRequiredVulkanInstanceExtensions()`, `getVulkanInstanceProcAddress(...)`, and `createVulkanSurface(window, instance)` all work on the tested null backend when a minimal Vulkan loader is present. The current executable proof uses a local fake `libvulkan.so.1` that advertises `VK_KHR_surface` plus `VK_EXT_headless_surface`, then successfully creates a headless Vulkan window surface for a real GLFW no-API window. Separately, the vendored binding now has cross-target object-build coverage for `windows-x64`, `linux-x64`, and `darwin-arm64`, which proves the package build model and per-platform link-flag handling across those targets even though full macOS/Windows runtime execution is not yet covered here.
Window values are represented as managed `GLFWwindow` native handles. `destroyWindow(...)` closes the managed handle, and using that handle afterward should raise a `TypeError` through the normal native-handle safety path.

Jayess also ships an in-tree raylib binding package:

- `@jayess/raylib`
- `setTraceLogLevel(level)`
- `setTraceLogCallback(callback)`
- `clearTraceLogCallback()`
- `emitTraceLog(level, message)`
- `setConfigFlags(flags)`
- `initWindow(width, height, title)`
- `closeWindow()`
- `isWindowReady()`
- `windowShouldClose()`
- `setWindowTitle(title)`
- `setWindowSize(width, height)`
- `getScreenWidth()`
- `getScreenHeight()`
- `beginDrawing()`
- `endDrawing()`
- `clearBackground(color)`
- `drawText(text, x, y, size, color)`
- `drawCircle(x, y, radius, color)`
- `genImageColor(width, height, color)`
- `loadImage(path)`
- `loadImageFromBytes(type, bytes)`
- `unloadImage(image)`
- `isImageReady(image)`
- `getImageWidth(image)`
- `getImageHeight(image)`
- `loadTexture(path)`
- `loadTextureFromImage(image)`
- `unloadTexture(texture)`
- `isTextureReady(texture)`
- `getTextureWidth(texture)`
- `getTextureHeight(texture)`
- `drawTexture(texture, x, y, color)`
- `isKeyPressed(key)`
- `isKeyDown(key)`
- `isMouseButtonDown(button)`
- `getMouseX()`
- `getMouseY()`
- `getMousePosition()`
- `isGamepadAvailable(gamepad)`
- `getGamepadAxisCount(gamepad)`
- `isGamepadButtonDown(gamepad, button)`
- `getGamepadName(gamepad)`
- `setTargetFPS(fps)`
- `getFrameTime()`
- `getTime()`
- `setRandomSeed(seed)`
- `getRandomValue(min, max)`

`@jayess/raylib` is a manual `*.bind.js` package layered over a small C shim and vendored raylib sources from [refs/raylib](/home/remote-desktop/Desktop/it/jayess/jayess-go/refs/raylib). The current package targets raylib's `PLATFORM_MEMORY` software framebuffer backend, so it can initialize and query the software window, run a basic draw loop, render basic shapes/textures/text, query keyboard/mouse/gamepad state, create generated images, upload textures from images, and shut down without depending on an OS window system. Colors are passed as plain Jayess objects like `{ r, g, b, a }`, mouse positions are returned as `{ x, y }`, gamepad names return a Jayess string or `undefined` when no device name is available, and image/texture values are managed native handles. `unloadImage(...)` and `unloadTexture(...)` close those handles; using them afterward should raise a `TypeError` through the normal native-handle safety path. Failed image/texture loads surface as `RaylibError` values that can be handled with normal Jayess `try/catch`. On the tested memory-backend path, `setWindowSize(...)` is proven through the package’s own query surface: after `initWindow(...)`, the package tracks the logical window size it most recently set and exposes that through `getScreenWidth()` / `getScreenHeight()`, so the size-setting path is stable even on this software backend. The same boundary now covers a logical fullscreen/windowed transition too: `setWindowFullscreen()` switches the package into a documented fullscreen mode and reports a logical fullscreen size through the same query surface, while `setWindowWindowed(width, height)` restores windowed logical dimensions. On the tested `PLATFORM_MEMORY` path, the fallback logical fullscreen size is `1920x1080`, not a negotiated OS display mode. The package also has a narrow but real asset-loading path for portable PPM images: `loadImage(path)` can load a `.ppm` file from disk, `loadImageFromBytes(".ppm", bytes)` can load the same format from a `Uint8Array`, and those image handles can be turned into textures through `loadTextureFromImage(...)`. It also exposes a narrow same-thread callback shim for trace-style events: `setTraceLogCallback(...)` stores a Jayess callback, `emitTraceLog(...)` invokes it with `(level, message)`, and `clearTraceLogCallback()` unregisters it again. The current smoke coverage proves that this callback can safely retain a Jayess closure after the original scope exits and that no further callback runs occur after `clearTraceLogCallback()`, while still keeping the claim bounded to this shim-managed callback path rather than general raylib backend events. That same smoke coverage also proves the PPM asset path through an on-disk `tiny.ppm` asset built with `path.join(...)` plus a bytes-based load of the same file, alongside the existing proof that a draw loop can coexist with Jayess timers/`await` on this backend and that the same compiled executable can keep a stub-backed `@jayess/audio` playback stream active while the raylib render loop is running.

Jayess also ships an in-tree audio binding package:

- `@jayess/audio`
- `createContext(name, backendName?)`
- `backendId(context)`
- `maxChannelCount(context)`
- `listOutputDevices(context)`
- `listInputDevices(context)`
- `preferredSampleRate(context)`
- `minLatency(context, options)`
- `createPlaybackStream(context, options)`
- `startPlaybackStream(stream)`
- `pausePlaybackStream(stream)`
- `stopPlaybackStream(stream)`
- `submitPlaybackSamples(stream, samples)`
- `playbackStats(stream)`
- `closePlaybackStream(stream)`
- `nextStreamEvent(stream)`
- `createCaptureStream(context, options)`
- `startCaptureStream(stream)`
- `stopCaptureStream(stream)`
- `readCapturedSamples(stream, frames)`
- `captureStats(stream)`
- `closeCaptureStream(stream)`
- `loadWav(path)`
- `loadOgg(path)`
- `loadMp3(path)`
- `loadFlac(path)`
- `destroyContext(context)`

`@jayess/audio` is a manual `*.bind.js` package layered over a small C shim and the Cubeb C API. Context values are managed `CubebContext` native handles, playback streams are managed `CubebPlaybackStream` handles, and capture streams are managed `CubebCaptureStream` handles. Both stream types use a mutex-protected native float queue internally. The current host-package smoke path proves a compiled Jayess executable can link the audio binding, create a context, read `backendId(...)` and `maxChannelCount(...)`, enumerate output and input devices through `listOutputDevices(...)` and `listInputDevices(...)`, and destroy the context cleanly on this host. Device enumeration returns plain Jayess objects with copied string metadata plus basic type/state/rate/channel information, so callers do not hold Cubeb-owned device pointers past the native call. A dedicated explicit stub-backed executable proof also now covers `preferredSampleRate(...)`, `minLatency(...)`, `createPlaybackStream(...)`, `startPlaybackStream(...)`, `pausePlaybackStream(...)`, `stopPlaybackStream(...)`, `submitPlaybackSamples(...)`, `playbackStats(...)`, `closePlaybackStream(...)`, `nextStreamEvent(...)`, `createCaptureStream(...)`, `startCaptureStream(...)`, `stopCaptureStream(...)`, `readCapturedSamples(...)`, `captureStats(...)`, `closeCaptureStream(...)`, `loadWav(...)`, `loadOgg(...)`, `loadMp3(...)`, and `loadFlac(...)`, including live callback-driven playback consumption, creating streams at the queried minimum-latency frame size, submitting additional PCM frames while playback is already running, reporting underruns when a playback stream is started without queued audio, surfacing a queued `error` event and `lastState: "error"` on a simulated device-loss path in the stub fixture, polling captured input frames back out of a running capture stream, decoding vendored OGG/MP3/FLAC sample assets through `miniaudio`, and keeping audio activity active while a Jayess worker thread is also running. `nextStreamEvent(...)` is the current safe audio-callback bridge: native Cubeb callbacks are queued as plain event records like `started`, `stopped`, `underrun`, and `error`, and Jayess polls them explicitly instead of running JS directly on the audio callback thread. Playback samples can now be supplied either as plain Jayess number arrays, `Float32Array` values for `format: "f32"`, or `Uint8Array` byte buffers containing native-endian signed 16-bit interleaved PCM for `format: "s16"`. Captured samples are currently exposed back to Jayess as `Float32Array` values. `loadWav(...)` currently supports RIFF/WAVE PCM16 and IEEE float32 files, while `loadOgg(...)`, `loadMp3(...)`, and `loadFlac(...)` decode through vendored `miniaudio`; all four loaders return decoded samples as a Jayess `Float32Array` plus metadata such as `sampleRate`, `channels`, `frames`, `format`, and `sourceFormat`. Separately from the packaged Cubeb surface, the manual binding model is now also proven against SDL3-style, OpenAL-style, and PortAudio-style audio APIs through dedicated header-backed binding tests, and against a real vendored miniaudio null-backend path that initializes a context, enumerates the built-in null playback device, opens it, and starts/stops it through a manual binding. On machines without Cubeb development libraries installed, native builds still fail with an explicit Cubeb-related link diagnostic instead of a generic unresolved linker error.

Jayess also ships an in-tree GTK binding package:

- `@jayess/gtk`
- `init()`
- `createWindow()`
- `createLabel(text)`
- `createButton(text)`
- `createTextInput()`
- `createImage(path)`
- `createDrawingArea()`
- `createBox(vertical, spacing)`
- `setTitle(window, title)`
- `setText(widget, text)`
- `addChild(parent, child)`
- `connectSignal(widget, signal, callback)`
- `emitSignal(widget, signal)`
- `queueDraw(widget)`
- `show(window)`
- `hide(window)`
- `pollEvents()`
- `runMainLoop()`
- `quitMainLoop()`
- `destroyWindow(window)`

`@jayess/gtk` is a manual `*.bind.js` package layered over a small C shim and the GTK C API. Window and widget values are represented as managed `GtkWidget` native handles. The current binding also supports target-specific native link-flag selection through `platforms.linux`, `platforms.darwin`, and `platforms.windows` in `gtk.bind.js`, and the binding model now supports `pkgConfig: [...]` discovery as well, so GTK header/include paths and platform link flags can be expressed either explicitly or through `pkg-config` when needed. The current Linux path is proven through both an explicit-include executable build and a rewritten real-package executable build driven by fake `pkg-config` discovery, and the vendored package also has cross-target object-build coverage for `windows-x64`, `linux-x64`, and `darwin-arm64`, which proves the binding build model across those targets even though only the Linux executable path is exercised here. The current explicit stub-backed package proof exercises `init()`, `createWindow()`, `createLabel(...)`, `createButton(...)`, `createTextInput()`, `createImage(...)`, `createDrawingArea()`, `createBox(...)`, `setTitle(...)`, `setText(...)`, `addChild(...)`, `connectSignal(...)`, `emitSignal(...)`, `queueDraw(...)`, `show(...)`, `hide(...)`, `pollEvents()`, `runMainLoop()`, `quitMainLoop()`, and `destroyWindow(...)`, including button `clicked`, entry `changed`, drawing-area `draw`, and window `destroy` signal delivery. That same proof now covers image-widget asset loading from a file path through `createImage(...)`, text-widget rendering paths through labels, buttons, and text inputs, and a minimal custom-drawing path through `createDrawingArea(...)` plus `queueDraw(...)`. Child widget handles are invalidated when their parent window is destroyed, so post-close use continues to raise `TypeError`. On machines without GTK development libraries installed, native builds fail with an explicit GTK header or link diagnostic instead of a generic compiler/linker failure.

Jayess also ships an in-tree webview binding package:

- `@jayess/webview`
- `createWindow(debug?)`
- `destroyWindow(view)`
- `setTitle(view, title)`
- `setSize(view, width, height)`
- `show(view)`
- `hide(view)`
- `setHtml(view, html)`
- `loadFile(view, path)`
- `navigate(view, url)`
- `initJs(view, source)`
- `evalJs(view, source)`
- `bind(view, name)`
- `unbind(view, name)`
- `nextBindingEvent(view)`
- `returnBinding(view, id, status, result)`
- `run(view)`
- `terminate(view)`

`@jayess/webview` is a manual `*.bind.js` package layered over a small C++ shim and the upstream `webview` C API. Webview values are represented as managed `Webview` native handles. The package sources can be compiled and linked through the normal binding path, including explicit local include/source overrides in a temp workspace, and post-close use follows the normal native-handle `TypeError` safety path. The vendored package also has cross-target object-build coverage for `windows-x64`, `linux-x64`, and `darwin-arm64`, which proves the binding build model across those targets even though only stub-backed executable paths are exercised here. The current explicit package proof covers `createWindow(...)`, `destroyWindow(...)`, `setTitle(...)`, `setSize(...)`, `show(...)`, `hide(...)`, `setHtml(...)`, `loadFile(...)`, `navigate(...)`, `initJs(...)`, `evalJs(...)`, `run(...)`, and `terminate(...)` against a local stub backend, so the binding surface, window/content lifecycle, basic window visibility toggling, local-file loading, event-loop entry/termination, and navigation smoke path are exercised even when GTK/WebKit development packages are not installed on the host. The package also now exposes a queue-based JS bridge model: `bind(view, name)` exposes a named JS-callable binding, `nextBindingEvent(view)` returns the next queued `{ name, id, request }` event from the native side, `returnBinding(view, id, status, result)` replies to that invocation with a JSON result string, and `unbind(view, name)` removes the binding again. Because this bridge is queue-based, it does not retain Jayess callback closures inside native webview code; the current bridge proof covers repeated event delivery before unbind, no further event delivery after unbind, and duplicate-bind error propagation against the stub backend. The stub-backed integration coverage also proves the package can work alongside Jayess filesystem/path APIs by loading HTML from a file path constructed through `path.join(...)`, feeding that content into `setHtml(...)`, and opening the same file through `loadFile(...)`. For embedded app content, the current proved path is through `@jayess/mongoose`: `createEmbeddedApp(...)` plus `serveEmbeddedApp(...)` can serve HTML/JS assets directly from memory, which is sufficient for a webview-targeted embedded-app content model. That content-serving path is proven separately from the webview package itself. Same-process coexistence between a stub-backed webview package import and the built-in/native HTTP server path is now also proven through one compiled executable: the reduced coexistence test imports `@jayess/webview`, starts a built-in `http.createServer(...)`, serves a real response, and shuts down cleanly without constructing a live webview window. The package does coexist with the current worker/thread model in a compiled executable: a `worker.create(...)` round trip can run while a stub-backed webview is created, titled, terminated, and driven through `run(...)` on the main thread. The current host-app proof also shows a stub-backed webview can coexist with a GLFW-backed host window path in the same compiled executable, using the vendored GLFW null-platform backend for the tested Linux path. On Linux, the default binding still expects a WebKitGTK-backed `webview` build path and will report explicit missing GTK/WebKit header or link diagnostics when those development dependencies are not installed.


Jayess now exposes a low-level TLS socket path through `tls.connect(...)` and `tls.createServer(...)`, backed by SChannel on Windows for clients and OpenSSL on non-Windows builds for the currently implemented server path, returning normal secure `Socket` objects with `secure`, `authorized`, `backend`, `protocol`, `alpnProtocol`, `alpnProtocols`, and `getPeerCertificate()`. `tls.connect(...)` accepts optional `alpnProtocols` as a string or array of protocol strings, plus trust options like `serverName`, `caFile`, `caPath`, and `trustSystem`. `tls.createServer(...)` currently accepts PEM file paths in `cert` and `key` and passes accepted secure sockets to the handler. The peer-certificate helper exposes `subject`, `issuer`, `subjectCN`, `issuerCN`, `serialNumber`, `validFrom`, `validTo`, `subjectAltNames`, `backend`, and `authorized`. HTTPS now runs on top of that TLS transport instead of a separate WinHTTP-only client path, passes through the same TLS trust options, and currently pins ALPN to `http/1.1`. Custom CA file/path trust configuration now works on both backends; on SChannel, Jayess performs explicit post-handshake certificate validation against the custom trust collection, while `trustSystem: false` disables system-root fallback.

See [Networking Foundation](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/networking.md).

## Collections

## Error Objects

Supported constructors:

- `new Error(message)`
- `new TypeError(message)`

Supported properties and helpers:

- `error.name`
- `error.message`
- `error.toString()`

Current boundary:

- Error objects are ordinary Jayess objects with standard fields.
- Stack traces and JavaScript exception subclass parity are not implemented yet.

## Binary Data

Supported constructors:

- `new ArrayBuffer(length)`
- `new Uint8Array(buffer)`
- `new Uint8Array(length)`
- `new DataView(buffer)`
- `Uint8Array.fromString(text, encoding?)`
- `Uint8Array.concat(...chunks)`
- `Uint8Array.equals(left, right)`
- `Uint8Array.compare(left, right)`

Supported `Uint8Array` behavior:

- numeric index get/set
- `length`
- `byteLength`
- `buffer`
- `fill(value)`
- `includes(value)`
- `indexOf(valueOrBytes)`
- `startsWith(valueOrBytes)`
- `endsWith(valueOrBytes)`
- `set(source, offset?)`
- `copyWithin(target, start, end?)`
- `slice(start, end?)`
- `concat(...chunks)`
- `equals(other)`
- `compare(other)`
- `toString()`
- file streams can write `Uint8Array` values and read bytes with `readStream.readBytes(size)`

Assignments to `Uint8Array` indexes are clamped to an unsigned byte range.
`Uint8Array.fromString(text, encoding?)` creates bytes from text. Supported encodings are `"utf8"`, `"utf-8"`, `"text"`, `"hex"`, and `"base64"`.
`Uint8Array.compare(left, right)` and `bytes.compare(other)` compare bytes lexicographically and return `-1`, `0`, or `1`.
`bytes.indexOf(valueOrBytes)`, `bytes.startsWith(valueOrBytes)`, and `bytes.endsWith(valueOrBytes)` accept either a numeric byte value or another `Uint8Array`.
`bytes.set(source, offset?)` mutates bytes from another `Uint8Array` or array-like value. `bytes.copyWithin(target, start, end?)` copies bytes inside the same buffer and returns the receiver.
`Uint8Array.toString(encoding?)` decodes bytes into a Jayess string. Supported encodings are `"utf8"`, `"utf-8"`, `"text"`, `"hex"`, and `"base64"`.

Supported `DataView` behavior:

- `byteLength`
- `buffer`
- `getUint8(offset)`
- `setUint8(offset, value)`
- `getInt8(offset)`
- `setInt8(offset, value)`
- `getUint16(offset, littleEndian?)`
- `setUint16(offset, value, littleEndian?)`
- `getInt16(offset, littleEndian?)`
- `setInt16(offset, value, littleEndian?)`
- `getUint32(offset, littleEndian?)`
- `setUint32(offset, value, littleEndian?)`
- `getInt32(offset, littleEndian?)`
- `setInt32(offset, value, littleEndian?)`
- `getFloat32(offset, littleEndian?)`
- `setFloat32(offset, value, littleEndian?)`
- `getFloat64(offset, littleEndian?)`
- `setFloat64(offset, value, littleEndian?)`

`DataView` reads and writes the same underlying `ArrayBuffer` bytes that `Uint8Array` views expose. Multi-byte operations default to big-endian when `littleEndian` is omitted or false.

Current boundary:

- This is a first binary-data surface, not full TypedArray/DataView parity.
- Numeric typed arrays beyond `Uint8Array` are not implemented yet.

### Map

Supported:

- `new Map()`
- `map.set(key, value)`
- `map.get(key)`
- `map.has(key)`
- `map.delete(key)`
- `map.clear()`
- `map.keys()`
- `map.values()`
- `map.entries()`
- `map.size`

### Set

Supported:

- `new Set()`
- `set.add(value)`
- `set.has(value)`
- `set.delete(value)`
- `set.clear()`
- `set.values()`
- `set.entries()`
- `set.size`

### Iteration

`for...of` works with:

- arrays
- `Map`
- `Set`
- `Uint8Array`

Supported iterator helpers:

- `Iterator.from(value)`
- `iterator.next()`

`iterator.next()` returns an object with `value` and `done`.

Current boundary:

- The current implementation uses a pragmatic iterable bridge rather than the full JavaScript iterator protocol.
- `function*` and `yield` are not implemented yet.

## Promise and Await

Promise, `await`, and async file I/O are documented in [Async Runtime](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/async.md). Timer APIs are documented in [Timers](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/timers.md).

Supported highlights:

- `Promise.resolve(value)`
- `Promise.reject(reason)`
- `Promise.all(values)`
- `Promise.race(values)`
- `Promise.allSettled(values)`
- `Promise.any(values)`
- `promise.then(onFulfilled, onRejected)`
- `promise.catch(onRejected)`
- `promise.finally(onFinally)`
- `await value`
- `async function name(...) { ... }`
- `new AggregateError(errors, message)`

## Timers

Supported highlights:

- `timers.sleep(delayMs)`
- `timers.sleep(delayMs, value)`
- `timers.setTimeout(callback, delayMs)`
- `timers.clearTimeout(id)`

Compatibility globals:

- `setTimeout(callback, delayMs)`
- `clearTimeout(id)`
- `sleepAsync(delayMs, value)`

## Date

Supported:

- `Date.now()`
- `new Date()`
- `new Date(value)`
- `date.getTime()`
- `date.toString()`
- `date.toISOString()`

## JSON

Supported:

- `JSON.stringify(value)`
- `JSON.parse(text)`

Current note:

- `JSON.parse(...)` currently returns `undefined` on parse failure instead of throwing

## RegExp

Supported:

- `new RegExp(pattern[, flags])`
- `re.source`
- `re.flags`
- `re.test(text)`

String regex helpers:

- `text.match(re)`
- `text.search(re)`
- `text.replace(re, replacement)`
- `text.split(re)`

Current note:

- this is a pragmatic regex slice, not full ECMAScript regex parity

## Math

Supported:

- `Math.floor`
- `Math.ceil`
- `Math.round`
- `Math.min`
- `Math.max`
- `Math.abs`
- `Math.pow`
- `Math.sqrt`
- `Math.random`

## Number

Supported:

- `Number.isNaN`
- `Number.isFinite`

## String

Supported instance helpers:

- `length`
- `includes`
- `startsWith`
- `endsWith`
- `slice`
- `trim`
- `toUpperCase`
- `toLowerCase`
- `split`

Supported static helper:

- `String.fromCharCode`

## Array

Supported:

- `length`
- `push`
- `pop`
- `shift`
- `unshift`
- `slice`
- `includes`
- `join`
- `map`
- `filter`
- `find`
- `forEach`

Supported static helpers:

- `Array.isArray`
- `Array.from`
- `Array.of`

## Object

Supported:

- `Object.keys`
- `Object.values`
- `Object.entries`
- `Object.assign`
- `Object.hasOwn`
- `Object.fromEntries`
