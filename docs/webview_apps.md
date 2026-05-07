# Webview Apps

Webview support allows native desktop apps with platform webview dependencies.
The first-party toolkit layout proposal lives in `docs/webview_toolkit.md`.
The Jayess-facing import path should be `@jayess/webview` rather than a local
binding import when using the first-party package surface.
Focused API and distribution notes live in `docs/webview_api.md` and
`docs/webview_distribution.md`.

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
import { createWindow, run } from "@jayess/webview";

function main() {
  const window = createWindow("Jayess", "https://example.com");
  run(window);
  return 0;
}
```
