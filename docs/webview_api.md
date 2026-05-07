# Webview API

`@jayess/webview` is the first-party Jayess package for the internal webview
toolkit direction.

The first API slice stays intentionally small:

- app lifecycle
- window creation and shutdown
- content mounting
- event delivery
- file open and save dialogs
- source-file and asset drag-and-drop

The main surface should stay generalized. Jayess code should work with mounted
content trees and event delivery rather than raw per-platform widget APIs.

Raw host escape hatches may exist, but they are secondary to the main package
surface and should remain explicit.

## Initial Non-Goals

The first package slice does not try to cover:

- a large native widget catalog
- per-platform styling APIs
- platform-specific menu systems
- every browser or DOM API
- a broad retained-mode component framework

Those can be added later only after the package, runtime, and packaging model
are stable.
