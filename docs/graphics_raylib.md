# raylib Graphics

raylib support is expected to use native package or binding integration.

## Setup

Install or vendor raylib headers and libraries for each target platform. Declare
include paths, library paths, shared libraries, and license files in the binding
manifest.

## Assets

Keep textures, sounds, fonts, and other game assets in project folders that are
included by app distribution.

## Example Shape

Expose a small Jayess API for window creation, drawing, input polling, and
shutdown. Keep native raylib resources behind managed handles where needed.

```js
import { initWindow, beginDrawing, clearBackground, endDrawing, closeWindow } from "./native/raylib.js";

function main() {
  initWindow(800, 450, "Jayess");
  beginDrawing();
  clearBackground(255, 255, 255);
  endDrawing();
  closeWindow();
  return 0;
}
```
