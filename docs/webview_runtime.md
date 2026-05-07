# Webview Runtime

`runtime/webview` is the Jayess-owned internal host layer behind
`@jayess/webview`.

The current internal runtime slice is intentionally small and generalized:

- window lifecycle
- content mounting
- event delivery
- open and save dialog flow
- packaged runtime metadata

The internal host model is implemented in Go and does not require:

- a separately installed Jayess package
- a separately installed helper service
- shipping unrelated third-party GUI libraries as app-owned payload

## Host Model

The current host model tracks:

- windows with title, size, and lifecycle state
- mounted content for embedded HTML, CSS, script, and generated documents
- queued events for lifecycle, host messages, dialog results, and file drops
- pending dialog requests and completed dialog results

This model is designed so a later platform-specific backend can satisfy the
same contract without changing the Jayess-facing package surface.

## Packaging

The runtime ships package-owned metadata through `runtime/webview_runtime.json`.
That runtime asset is copied through normal app distribution when
`@jayess/webview` is imported.

Packaged apps should therefore rely on:

- the produced Jayess app bundle
- the Jayess-owned webview runtime asset
- documented platform prerequisites only

They should not depend on developer-machine paths or an additional GUI library
payload controlled by the application itself.
