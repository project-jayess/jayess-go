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
- `fs.remove(path)` distinguishes files and directories
- `fs.remove(path, { recursive: true })` removes directory trees recursively
- `fs.copyDir(from, to)` copies directory trees recursively
- `path.parse(path)` returns an object with `root`, `dir`, `base`, `ext`, and `name`

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

`dns.lookup(host)` returns an object with `host`, `address`, and `family`, or `undefined` when resolution fails.
`dns.lookupAll(host)` returns an array of address records, or `undefined` when resolution fails.
`dns.reverse(address)` returns a hostname string, or `undefined` when reverse lookup fails.

Supported net helpers:

- `net.isIP(input)`
- `net.connect({ host, port })`
- `net.listen({ host, port })`

Supported http helpers:

- `http.parseRequest(input)`
- `http.formatRequest(parts)`
- `http.parseResponse(input)`
- `http.formatResponse(parts)`
- `http.request(options)`
- `http.get(input)`
- `http.requestAsync(options)`
- `http.getAsync(input)`

`net.isIP(input)` returns `4`, `6`, or `0`.
`net.connect({ host, port })` returns a blocking TCP socket object.
`net.listen({ host, port })` returns a blocking TCP server object.
`http.parseRequest(...)` and `http.parseResponse(...)` return parsed HTTP message objects.
`http.formatRequest(...)` and `http.formatResponse(...)` build HTTP/1.x message text.
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
Jayess exposes an HTTPS client surface through `https.get(...)`, `https.request(...)`, `https.getStream(...)`, `https.requestStream(...)`, `https.getStreamAsync(...)`, `https.requestStreamAsync(...)`, `https.getAsync(...)`, and `https.requestAsync(...)`, with `rejectUnauthorized` available for development/test certificate bypass. `https.request(...)` and `https.requestAsync(...)` support custom methods and request bodies through the same TLS-backed transport, and the stream variants expose `response.bodyStream` for incremental reads once headers are available.
The runtime also exposes `tls.isAvailable()`, `tls.backend()`, `tls.connect(...)`, `https.isAvailable()`, and `https.backend()` so Jayess code can branch on current TLS/HTTPS capability.
Socket objects support `read`, `readAsync`, `readBytes`, `write`, `writeAsync`, `end`, `close`, `destroy`, `setNoDelay`, `setKeepAlive`, `setTimeout`, `on`, `once`, `off`, `listenerCount`, `eventNames`, `readable`, `writable`, `timeout`, `localAddress`, `localPort`, `remoteFamily`, `localFamily`, `bytesRead`, `bytesWritten`, `address`, `remote`, `protocol`, `alpnProtocol`, `alpnProtocols`, and `getPeerCertificate()`.
Server objects support `accept`, `acceptAsync`, `close`, `address`, `setTimeout`, `timeout`, `connectionsAccepted`, `on`, `once`, `off`, `listenerCount`, and `eventNames`. `server.address()` returns `{ address, port, family }`.

Jayess now exposes a low-level TLS socket path through `tls.connect(...)`, backed by SChannel on Windows and OpenSSL on non-Windows builds, returning a normal `Socket` object with `secure`, `authorized`, `backend`, `protocol`, `alpnProtocol`, `alpnProtocols`, and `getPeerCertificate()`. `tls.connect(...)` accepts optional `alpnProtocols` as a string or array of protocol strings, plus trust options like `serverName`, `caFile`, `caPath`, and `trustSystem`. The peer-certificate helper exposes `subject`, `issuer`, `subjectCN`, `issuerCN`, `serialNumber`, `validFrom`, `validTo`, `subjectAltNames`, `backend`, and `authorized`. HTTPS now runs on top of that TLS transport instead of a separate WinHTTP-only client path, passes through the same TLS trust options, and currently pins ALPN to `http/1.1`. Custom CA file/path trust configuration now works on both backends; on SChannel, Jayess performs explicit post-handshake certificate validation against the custom trust collection, while `trustSystem: false` disables system-root fallback.

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
