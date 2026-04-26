# File System

Jayess provides a compact native file-system surface through `fs`.

## Read and Write

```javascript
var text = fs.readFile("notes.txt");
var utf8 = fs.readFile("notes.txt", "utf8");
var asyncText = await fs.readFileAsync("notes.txt", "utf8");
fs.writeFile("notes.txt", "kimchi");
await fs.writeFileAsync("notes.txt", "async kimchi");
```

Current behavior:

- `fs.readFile(path[, encoding])` returns file contents or `undefined`
- `fs.readFileAsync(path[, encoding])` returns a promise-like value resolved with file contents or `undefined`
- supported encodings are pragmatic text aliases such as `"utf8"` and `"utf-8"`
- `fs.writeFile(path, content)` returns a boolean-like result
- `fs.writeFileAsync(path, content)` returns a promise-like value resolved with the write result

Async file behavior is documented in [Async Runtime](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/async.md).

## Streams

Jayess supports a small Node-like stream construction surface:

```javascript
var writer = fs.createWriteStream("notes.txt");
writer.write("kim");
writer.write("chi");
writer.on("finish", () => {
  console.log("written");
  return 0;
});
writer.on("finish", () => {
  console.log("also written");
  return 0;
});
writer.once("finish", () => {
  console.log("written once");
  return 0;
});
console.log(writer.listenerCount("finish"));
console.log(writer.eventNames());
writer.end();

var reader = fs.createReadStream("notes.txt");
console.log(reader.read(3)); // kim
console.log(reader.read(3)); // chi
console.log(reader.read()); // null at end of file
reader.close();

fs.createReadStream("notes.txt")
  .on("data", (chunk) => {
    console.log(chunk);
    return 0;
  })
  .on("end", () => {
    console.log("done");
    return 0;
  });

fs.createReadStream("notes.txt").pipe(fs.createWriteStream("copy.txt"));

var missing = fs.createReadStream("missing.txt");
missing.on("error", (err) => {
  console.log(err.message);
  return 0;
});

var bytes = new Uint8Array(2);
bytes[0] = 65;
bytes[1] = 66;
fs.createWriteStream("bytes.bin").write(bytes);
var readBack = fs.createReadStream("bytes.bin").readBytes(2);
console.log(readBack[0], readBack[1]);
console.log(readBack.includes(66));
console.log(readBack.slice(0, 1).length);
console.log(readBack.toString());

var textBytes = Uint8Array.fromString("hello");
fs.createWriteStream("hello.bin").write(textBytes);
console.log(fs.createReadStream("hello.bin").readBytes(5).toString());
console.log(Uint8Array.fromString("4142", "hex").toString("utf8")); // AB
console.log(Uint8Array.fromString("AB").toString("hex")); // 4142
console.log(Uint8Array.fromString("a2ltY2hp", "base64").toString()); // kimchi
console.log(Uint8Array.fromString("kimchi").toString("base64")); // a2ltY2hp
console.log(Uint8Array.concat(Uint8Array.fromString("kim"), Uint8Array.fromString("chi")).toString());
console.log(Uint8Array.fromString("jay").concat(Uint8Array.fromString("ess")).toString());
console.log(Uint8Array.fromString("kimchi").equals(Uint8Array.fromString("kimchi")));
console.log(Uint8Array.equals(Uint8Array.fromString("kimchi"), Uint8Array.fromString("kimchi")));
console.log(Uint8Array.compare(Uint8Array.fromString("kim"), Uint8Array.fromString("kimchi"))); // -1
console.log(Uint8Array.fromString("kimchi").compare(Uint8Array.fromString("kim"))); // 1
console.log(Uint8Array.fromString("kimchi").indexOf(Uint8Array.fromString("chi"))); // 3
console.log(Uint8Array.fromString("kimchi").startsWith(Uint8Array.fromString("kim"))); // true
console.log(Uint8Array.fromString("kimchi").endsWith(105)); // true
var bytes = new Uint8Array(6);
bytes.set(Uint8Array.fromString("kim"), 1);
bytes.copyWithin(4, 1, 3);
console.log(bytes.toString("hex")); // 006b696d6b69
var buffer = new ArrayBuffer(4);
var view = new DataView(buffer);
view.setUint16(0, 4660, false);
view.setUint16(2, 4660, true);
console.log(new Uint8Array(buffer).toString("hex")); // 12343412
view.setInt8(0, -1);
console.log(view.getInt8(0)); // -1
view.setFloat32(0, 1.5, false);
console.log(new Uint8Array(buffer).toString("hex")); // 3fc000003412
```

Supported:

- `fs.createReadStream(path)` returns a `ReadStream` object
- `readStream.read()` returns the next text chunk, `null` at EOF, or `undefined` on error/closed stream
- `readStream.read(size)` reads up to `size` bytes
- `readStream.readBytes(size)` reads up to `size` bytes and returns a `Uint8Array`, `null` at EOF, or `undefined` on error/closed stream
- `readStream.close()` closes the stream
- `readStream.destroy()` is an alias for `close()`
- `readStream.on("data", callback)` reads chunks synchronously and calls `callback(chunk)`
- `readStream.on("end", callback)` registers an end callback; if the stream has already ended, it runs immediately
- `readStream.on("error", callback)` registers an error callback; if the stream already errored, it runs immediately
- `readStream.once("data", callback)` reads one chunk synchronously and calls `callback(chunk)` once
- `readStream.once("end", callback)` and `readStream.once("error", callback)` register single-use callbacks
- `readStream.off(event)` and `readStream.removeListener(event)` remove all callbacks for that event
- `readStream.off(event, callback)` and `readStream.removeListener(event, callback)` remove one matching callback for that event
- `readStream.listenerCount(event)` returns the number of persistent and one-shot callbacks for that event
- `readStream.eventNames()` returns event names with active callbacks
- `readStream.pipe(writeStream)` copies all remaining chunks to a write stream and ends the destination
- `readStream.readableEnded` is `true` after EOF or close
- `fs.createWriteStream(path)` returns a `WriteStream` object
- `writeStream.write(value)` writes text or `Uint8Array` bytes and returns `true` or `false`
- `writeStream.end()` flushes and closes the stream
- `writeStream.close()` and `writeStream.destroy()` are aliases for `end()`
- `writeStream.on("error", callback)` registers an error callback; if the stream already errored, it runs immediately
- `writeStream.on("finish", callback)` registers a completion callback; if the stream already ended successfully, it runs immediately
- `writeStream.once("error", callback)` and `writeStream.once("finish", callback)` register single-use callbacks
- `writeStream.off(event)` and `writeStream.removeListener(event)` remove all callbacks for that event
- `writeStream.off(event, callback)` and `writeStream.removeListener(event, callback)` remove one matching callback for that event
- `writeStream.listenerCount(event)` returns the number of persistent and one-shot callbacks for that event
- `writeStream.eventNames()` returns event names with active callbacks
- `writeStream.writableEnded` is `true` after `end()`
- `stream.closed` is `true` after the stream is closed explicitly or by `end()`
- `stream.errored` is `true` after an open/read/write/flush error
- `stream.error` contains the last stream error object, or `null`
- `fs.symlink(target, path)` creates a symbolic link and returns a boxed boolean-like result
- `fs.watch(path)` creates an async-oriented watcher for a file or directory path
- `fs.watchSync(path)` creates the same watcher type for explicit synchronous polling usage
- `watcher.poll()` returns `null` if nothing changed, or a change object and emits `"change"` if metadata changed
- `watcher.pollAsync(timeoutMs)` returns a promise-like value that resolves with the next change object, or `null` when the timeout expires
- `watcher.on("change", callback)` and `watcher.once("change", callback)` observe detected changes
- `watcher.on("close", callback)` and `watcher.once("close", callback)` observe watcher shutdown
- `watcher.close()` and `watcher.destroy()` stop the watcher
- `watcher.listenerCount(event)` and `watcher.eventNames()` expose active watcher listeners
- watcher objects expose `path`, `exists`, `isDir`, `isFile`, `size`, `mtimeMs`, `closed`, `errored`, and `error`

Current boundary:

- Streams are synchronous native file streams.
- `on("data", ...)` and `pipe(...)` drain synchronously in the current runtime.
- Multiple listeners per event are supported, and listeners run in registration order.
- Backpressure, `drain`, async event scheduling, and general EventEmitter semantics are not implemented yet.
- `fs.watch(path)` and `fs.watchSync(path)` are polling-based: changes are detected when `poll()` or `pollAsync(...)` checks the path, and they currently compare existence, type, size, and modification time.

## Existence and Metadata

```javascript
var exists = fs.exists("notes.txt");
var stat = fs.stat("notes.txt");
```

`fs.stat(path)` returns `undefined` when the path does not exist. When it exists, the result object includes:

- `path`
- `isDir`
- `isFile`
- `size`
- `mtimeMs`
- `permissions`

## Directories

```javascript
fs.mkdir("build/tmp", { recursive: true });
var entries = fs.readDir("build", { recursive: true });
```

`fs.readDir(...)` entry objects include:

- `name`
- `path`
- `isDir`
- `isFile`
- `size`
- `mtimeMs`
- `permissions`

## Remove and Copy

```javascript
fs.copyFile("a.txt", "b.txt");
fs.copyDir("assets", "backup-assets");
fs.remove("backup-assets", { recursive: true });
fs.rename("a.txt", "b.txt");
```

Notes:

- `fs.remove(path)` removes a single file or an empty directory
- `fs.remove(path, { recursive: true })` removes directory trees recursively
- `fs.copyDir(from, to)` performs a recursive directory copy
