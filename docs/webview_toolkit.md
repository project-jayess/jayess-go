# Webview Toolkit Layout

This document proposes the first package and file layout for the Jayess-owned
`@jayess/webview` toolkit work.

The goal is to keep the public Jayess package stable while isolating host and
platform details behind small internal layers. The first implementation should
prefer generalized mechanisms such as window lifecycle, content mounting, event
dispatch, and asset packaging instead of growing a large per-widget surface.

## Public Ownership

`@jayess/webview` should be treated as a first-party Jayess package.

From the Jayess user perspective:

- it ships with the Jayess toolchain or standard package set
- it does not require separate package installation outside the Jayess flow
- it should compile and package through normal Jayess app distribution

From the implementation perspective:

- the public package remains Jayess-owned
- host and platform code can stay internal
- higher-level logic can move into Jayess over time

## Package Split

The current repo already has a Go `webview/` package for binding and platform
planning. Keep that package focused on build planning and platform capability
modeling. Do not turn it into a mixed frontend, runtime, and packaging file set.

Proposed package split:

- `webview/`
  - Go package for binding model, platform support, packaging metadata, and host capability planning
- `runtime/webview/`
  - runtime-facing support for handles, host calls, lifecycle dispatch, and asset mount state
- `stdlib/@jayess/webview/`
  - Jayess-facing package sources for app API, window API, content API, and event API
- `test/webview_*`
  - focused tests for build planning, packaging, lifecycle, and API surface
- `docs/webview_*`
  - focused docs split by app usage, toolkit layout, and packaging behavior

## Go File Layout

Keep the Go side split by one responsibility per file. A reasonable first layout
for the existing `webview/` package is:

- `webview/model.go`
  - package-level API kinds, binding module shape, and validation entrypoints
- `webview/platform.go`
  - per-platform backend and linker metadata
- `webview/plan.go`
  - build and packaging planning helpers only
- `webview/diagnostics.go`
  - small focused diagnostics builders for unsupported or incomplete platform cases
- `webview/assets.go`
  - app asset packaging model only

If runtime support grows, keep it separate:

- `runtime/webview/handles.go`
- `runtime/webview/lifecycle.go`
- `runtime/webview/content.go`
- `runtime/webview/events.go`
- `runtime/webview/assets.go`

## Jayess Package Layout

The public Jayess package should stay small at first and expose generalized app
primitives instead of many widget-specific features.

Suggested source layout:

- `stdlib/@jayess/webview/index.js`
  - public exports only
- `stdlib/@jayess/webview/app.js`
  - app startup and shutdown entrypoints
- `stdlib/@jayess/webview/window.js`
  - create window, title, size, close, show
- `stdlib/@jayess/webview/content.js`
  - mount content, navigate, load asset entrypoint
- `stdlib/@jayess/webview/events.js`
  - event subscription and dispatch helpers
- `stdlib/@jayess/webview/drop.js`
  - drag-and-drop event model
- `stdlib/@jayess/webview/dialogs.js`
  - file open and save dialogs

Keep each file small. Avoid one large `webview.js` with all toolkit behavior.

## First API Slice

The first public API should cover:

- app lifecycle
- single-window creation
- title and size
- mounted content entrypoint
- host-to-Jayess event dispatch
- file drop
- file open and save dialogs

Do not start with a giant widget set. Let the initial toolkit prove:

- package ownership
- cross-platform startup
- event loop integration
- asset packaging
- consistent diagnostics

## Tests

Keep tests focused and layer-specific:

- `test/webview_platform_test.go`
  - platform capability model
- `test/webview_binding_model_test.go`
  - binding and plan validation
- `test/webview_assets_test.go`
  - asset packaging model
- `test/webview_runtime_lifecycle_test.go`
  - runtime lifecycle behavior
- `test/webview_runtime_events_test.go`
  - event and drag-drop behavior

Prefer small deterministic tests over large end-to-end suites for each step.

## Migration Direction

The first implementation can rely on internal native host code where required,
but the long-term direction should allow more of the package logic to move into
Jayess itself as self-hosting improves.

That means:

- keep the public package surface stable
- keep native host calls narrow
- keep packaging behavior explicit
- avoid exposing raw platform APIs as the primary Jayess surface
