# Runtime Utilities

Jayess stream, Buffer, URL, query, and util helpers are internal runtime
utilities. They do not require native parser, buffer, or formatting libraries.

## Streams

Streams use the shared `IOStream` model:

- readable streams
- writable streams
- duplex streams
- transform streams
- pipe
- backpressure metadata through high-water marks

The same stream model is used by filesystem, HTTP bodies, child-process output,
compression, TCP sockets, and UDP packet bridges.

## Buffer

Buffer helpers provide byte storage and binary operations:

- create
- UTF-8 string conversion
- slice
- copy
- little-endian `uint16` reads and writes
- `Uint8Array` byte views
- read and write streams

## URL And Query

URL helpers use Go-owned parsing and encoding:

- parse and format URLs
- parse and stringify query strings
- query escaping and unescaping
- file URL/path conversion

## Util

`util.format` and `util.inspect` provide small deterministic formatting helpers
for runtime diagnostics and CLI output. They are intentionally compact and do
not attempt to clone every Node.js formatting edge case.

## Example

```js
const raw = Buffer.fromString("hello", "utf8");
const input = Buffer.createReadStream(raw);
const compressed = compression.createCompressStream("gzip", input);

const parsed = url.parse("https://example.com/?q=jayess");
const text = util.format("host=%s", parsed.host);
```
