# Timers

Jayess supports timer tasks through the `timers` standard-library namespace.

The namespaced API is preferred for new code. The global timer functions remain available as compatibility aliases.

## Promise Sleep

```javascript
var value = await timers.sleep(10, "ready");
console.log(value);

await timers.sleep(100);
```

Supported:

- `timers.sleep(delayMs)` returns a promise that fulfills with `undefined` after the delay.
- `timers.sleep(delayMs, value)` returns a promise that fulfills with `value` after the delay.

`timers.sleep` is useful for deterministic async control flow and for testing Promise combinators:

```javascript
var winner = await Promise.race([
  timers.sleep(25, "slow"),
  timers.sleep(0, "fast")
]);

console.log(winner); // fast
```

`timers.sleep` timers are not cancellable. Use `timers.setTimeout` when code needs a cancellable timer id.

## Cancellable Timers

```javascript
var id = timers.setTimeout(() => {
  console.log("later");
  return 0;
}, 100);

timers.clearTimeout(id);
```

Supported:

- `timers.setTimeout(callback, delayMs)` schedules a timer callback and returns a numeric timer id.
- `timers.clearTimeout(id)` cancels a queued timer callback if it has not run yet.

Timer callbacks must be functions. The callback is invoked with `undefined`.

## Compatibility Globals

The following global functions still work:

- `setTimeout(callback, delayMs)`
- `clearTimeout(id)`
- `sleepAsync(delayMs)`
- `sleepAsync(delayMs, value)`

Prefer `timers.setTimeout`, `timers.clearTimeout`, and `timers.sleep` in new code.

The blocking `sleep(milliseconds)` builtin is separate. It blocks execution and does not return a promise.

## Runtime Ordering

- Ready promise callbacks run before timer tasks when the runtime drains the queue.
- Timer tasks run when their due time has passed.
- The runtime drains queued tasks before program exit.
- `timers.sleep(0, value)` still fulfills asynchronously through the runtime queue.
