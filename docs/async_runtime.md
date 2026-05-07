# Async Runtime

Jayess async scheduling is implemented with compiler-owned Go runtime helpers.
The supported timer and microtask paths do not require libuv, Node.js, or a
user-installed native event loop.

## Event Loop

The runtime owns a shared `EventLoop` service. CLI programs, HTTP servers,
streams, child processes, and future async services should schedule work through
that service instead of creating independent process-wide loops.

Filesystem, child-process, TCP, UDP, HTTP, HTTPS, DNS, stream, timer, and
microtask APIs should route through Jayess runtime services. Core scheduling and
I/O must not depend on libuv unless an application explicitly imports an
optional libuv binding package.

The loop has two generalized queues:

- microtasks, used by `queueMicrotask` and promise-style follow-up work
- timers, used by `setTimeout` and `setInterval`

Microtasks run before due timers. Microtasks queued by a timer run before the
next due timer is processed.

## Timers

`setTimeout(callback, delay)` schedules a one-shot timer and returns a handle.
`setInterval(callback, delay)` schedules a repeated timer and returns a handle.
`clearTimeout(handle)` and `clearInterval(handle)` cancel scheduled handles.

Delay values are runtime durations. Negative delays are normalized to zero.

## Distribution

Internal scheduling does not add runtime assets or shared libraries. Optional
libuv bindings can still exist for packages that explicitly import them, but
Jayess timer and microtask builtins should not require libuv in an end-user app
package.

Apps that use normal filesystem, process, TCP, UDP, timer, or microtask APIs
should package only Jayess-owned runtime assets. They should not copy `libuv`
shared libraries or require a user-installed event-loop library.

## Example

```js
const handle = setTimeout(() => {
	print("timer fired");
}, 10);

queueMicrotask(() => {
	print("microtask first");
});

clearTimeout(handle);
```
