# Workers

Jayess workers are implemented with Go-owned runtime helpers. Supported worker
features do not require an external thread pool, libuv, Node.js, or user
installed native libraries.

## Runtime Model

A worker has:

- an inbox for messages sent by the parent
- an outbox for messages sent back to the parent
- message callbacks registered by the parent
- explicit cleanup through close/termination
- last-error state for handler failures

Workers are backed by Go goroutines in the current runtime. The API is a
Jayess-owned abstraction so the compiler can preserve the same surface if the
execution backend changes later.

## Shared Memory

The current shared-memory boundary is intentionally small:

- fixed-size integer slots
- synchronized `atomicLoad`
- synchronized `atomicStore`
- range errors for invalid indexes

This is not a full Node.js `SharedArrayBuffer` clone. Byte-level shared buffers,
wait/notify operations, and cross-process shared memory are future work.

## Example

```js
const thread = worker.thread("./worker.js");
worker.onMessage(thread, (message) => {
	print(message);
});

worker.postMessage(thread, { ready: true });

const memory = worker.sharedMemory(2);
worker.atomicStore(memory, 0, 1);
const value = worker.atomicLoad(memory, 0);
```

## Limitations

Workers are Jayess runtime workers, not operating-system processes. They do not
share arbitrary object graphs, file descriptors, or native handles. Messages
should be values that the runtime can copy or safely transfer.
