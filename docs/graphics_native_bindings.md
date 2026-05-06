# Graphics Native Bindings

GLFW, SDL, and similar graphics libraries should be integrated as developer
provided native bindings.

## Expectations

- install or vendor the native library
- expose a C ABI wrapper for Jayess
- declare headers, library paths, shared libraries, and licenses in `bind(...)`
- package runtime libraries and assets with the app distribution

## Design Rule

Do not assume the compiler ships these libraries by default. The compiler should
organize and package the binding outputs, while the project owns the dependency.

## Example Import

```js
import { createWindow, pollEvents } from "./native/glfw.js";

function main() {
  const window = createWindow(800, 600, "Jayess");
  pollEvents(window);
  return 0;
}
```
