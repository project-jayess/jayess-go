# Async Runtime

Jayess supports a JavaScript-like async surface, but it is implemented as a native runtime feature, not as the JavaScript event loop.

The current model is useful for promise-style control flow, timers, and basic async file I/O. It is not yet a full native suspend/resume async system.

## Async Functions

```javascript
async function loadValue(): number {
  return 10;
}

var value = await loadValue();
console.log(value);
```

Supported forms:

- `async function name(...) { ... }`
- `async function(...) { ... }`
- `async (...) => expression`
- `async value => expression`

Inside an async function:

- `return value` becomes a fulfilled promise-like value.
- `throw value` becomes a rejected promise-like value.
- The function body currently runs synchronously until it returns or throws.

## Await

```javascript
var value = await Promise.resolve("kimchi");

try {
  await Promise.reject(new Error("bad"));
} catch (err) {
  console.log(err.message);
}
```

`await` behavior:

- Awaiting a fulfilled promise-like value returns its value.
- Awaiting a rejected promise-like value throws the rejection reason through Jayess runtime exceptions.
- Awaiting a non-promise value returns the value unchanged.
- Await drains the runtime task queue until the awaited promise-like value settles.

Current limitation:

- `await` does not yet lower the function into a compiler-generated suspend/resume state machine.

## Promise API

Supported static helpers:

- `Promise.resolve(value)`
- `Promise.reject(reason)`
- `Promise.all(values)`
- `Promise.race(values)`
- `Promise.allSettled(values)`
- `Promise.any(values)`

Supported instance methods:

- `promise.then(onFulfilled, onRejected)`
- `promise.catch(onRejected)`
- `promise.finally(onFinally)`

Examples:

```javascript
var all = await Promise.all([
  Promise.resolve("a"),
  Promise.resolve("b")
]);

var settled = await Promise.allSettled([
  Promise.resolve("ok"),
  Promise.reject("no")
]);

console.log(settled[0].status, settled[0].value);
console.log(settled[1].status, settled[1].reason);
```

`Promise.any(values)` fulfills with the first fulfilled input. If all inputs reject, it rejects with an `AggregateError`.

```javascript
try {
  await Promise.any([
    Promise.reject("first"),
    Promise.reject("second")
  ]);
} catch (err) {
  console.log(err.name);
  console.log(err.errors[0], err.errors[1]);
}
```

## AggregateError

```javascript
var err = new AggregateError(["a", "b"], "everything failed");
console.log(err.name);
console.log(err.message);
console.log(err.errors[0]);
```

Supported properties:

- `name`
- `message`
- `errors`

Supported methods:

- `toString()`

## Timers

```javascript
var value = await timers.sleep(10, "ready");
console.log(value);
```

Timer APIs are documented in [Timers](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/timers.md).

Preferred surface:

- `timers.sleep(delayMs, value?)`
- `timers.setTimeout(callback, delayMs)`
- `timers.clearTimeout(id)`

Runtime ordering:

- Ready promise callbacks run before timer tasks when the runtime drains the queue.
- The runtime drains queued tasks before program exit.

## Async File I/O

```javascript
await fs.writeFileAsync("notes.txt", "kimchi");
var text = await fs.readFileAsync("notes.txt", "utf8");
console.log(text);
```

Supported:

- `fs.readFileAsync(path)`
- `fs.readFileAsync(path, encoding)`
- `fs.writeFileAsync(path, content)`

Current implementation:

- Async file tasks resolve through the Jayess runtime queue.
- File reads and writes run through a fixed background worker pool.
- The current worker pool size is exposed as `process.threadPoolSize()`.

## Current Boundary

The async runtime is intentionally pragmatic at this stage.

Implemented:

- Promise-like values
- Microtask-style promise callback queue
- Timers
- Basic async file read/write
- Rejection through `try / catch`

Not implemented yet:

- Compiler-generated async suspend/resume state machines
- User-created threads and shared-memory threading APIs
- Runtime-configurable worker pool sizing
- Event-loop phases beyond the current task queue
- Async socket I/O and basic async HTTP client requests on the worker-pool runtime
- Full JavaScript Promise specification edge cases
