# Webview Apps

Webview support allows native desktop apps with platform webview dependencies.

## Platform Dependencies

Linux commonly needs GTK/WebKitGTK libraries, macOS uses Cocoa/WebKit
frameworks, and Windows uses WebView or system UI libraries depending on the
binding implementation.

## Binding

Declare platform-specific flags in `platforms.linux`, `platforms.darwin`, and
`platforms.windows` inside the binding manifest.

## Packaging

Package only the runtime libraries that are not guaranteed by the target system.
Document system package requirements when platform frameworks are expected to be
installed separately.

## Example Shape

```js
import { createWindow, run } from "./native/webview.js";

function main() {
  const window = createWindow("Jayess", "https://example.com");
  run(window);
  return 0;
}
```
