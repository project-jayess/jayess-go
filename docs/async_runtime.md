# Async Runtime

Async runtime integration is intended for evented native APIs such as libuv.

## Scope

The compiler can expose async/runtime services where the corresponding package
and native library are available. Jayess does not require a Node.js event loop.

## Native Libraries

When a package uses libuv or another native event loop, developers must install
or provide the library, declare it in bindings, and package runtime libraries
with the built application.

## Design Rule

Async behavior should be exposed through stable Jayess packages, not hidden
process-wide globals.

## Example Shape

```js
import { runLoop, setTimer } from "./native/uv.js";

function main() {
  setTimer(100, () => {
    console.log("timer fired");
  });
  runLoop();
  return 0;
}
```
