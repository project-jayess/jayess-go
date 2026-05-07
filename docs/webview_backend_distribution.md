# Webview Backend Distribution

`@jayess/webview` is first-party from the Jayess perspective. Users should not
need to install a separate Jayess webview package after a Jayess application is
compiled.

Backend/runtime handling is platform-specific:

- Windows: redistributable WebView2 runtime pieces can be packaged through the
  Jayess app distribution flow when they are available to the build.
- macOS: the runtime uses system frameworks such as `Cocoa.framework` and
  `WebKit.framework`.
- Linux: the runtime uses system GUI/webview prerequisites such as GTK and
  WebKitGTK.

This means Jayess owns:

- the package surface
- the runtime contract
- app distribution planning for redistributable backend assets
- documentation of unavoidable system prerequisites

This does not mean every platform backend is fully self-contained inside a pure
Go binary. Where redistribution is supported, Jayess app distribution should
copy the needed backend runtime pieces. Where redistribution is not supported or
not practical, the system prerequisites must be documented clearly.
