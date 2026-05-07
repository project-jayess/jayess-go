# Webview Distribution

`@jayess/webview` should be treated as a first-party Jayess package owned by
the compiler and runtime distribution.

Normal use should not require:

- installing a separate Jayess package outside the compiler/runtime flow
- shipping unrelated third-party GUI libraries as app-owned payload

The current ownership split is:

- package surface: `@jayess/webview`
- vendored package source: `stdlib/@jayess/webview`
- runtime support: `runtime/webview`
- build and packaging model: `webview/` plus `appdist/`

The runtime host contract itself is documented in `docs/webview_runtime.md`.
Backend/runtime packaging expectations are documented in
`docs/webview_backend_distribution.md`.

Jayess applications may still depend on platform host capabilities such as
system webview support. Those requirements should be documented as platform or
build-machine prerequisites, not as separate Jayess package installation steps.

Packaged GUI apps should load their own HTML, CSS, script, and static assets
from the produced app output. They must not depend on developer-machine asset
paths.
