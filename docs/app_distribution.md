# App Distribution

App distribution packages a compiled Jayess executable with the runtime assets
it needs.

## Contents

A distribution can include:

- the executable
- Jayess runtime assets required by compiled code
- native binding shared libraries
- project-provided data files
- third-party license and notice files

## Native Assets

Imported packages and native bindings are the source of truth for distribution
assets. If the program imports a package or binding, app distribution planning
must collect every redistributable runtime asset required by that import.

Bindings should declare shared libraries, runtime assets, helper assets, and
license files so distribution planning can collect them. If the executable is
dynamically linked, the final folder must include the needed runtime libraries.

## Package Metadata

Imported Jayess packages that need runtime files can provide
`jayess.package.json` at the package root:

```json
{
  "runtimeAssets": [
    { "path": "data/schema.json", "outputName": "assets/schema.json" }
  ],
  "helperAssets": [
    { "path": "bin/helper", "outputName": "helpers/helper" }
  ],
  "licenseFiles": ["LICENSE"]
}
```

Asset paths are resolved relative to the imported package root, not the caller's
working directory. Duplicate assets required by multiple imports are copied
once.

## Redistributable vs Build-Only Dependencies

Redistributable runtime assets must be copied into the app distribution with
their license or notice files. Runtime assets can set `requiresLicense` in
package metadata to make missing license metadata a distribution diagnostic.

Platform SDKs and system frameworks that cannot be redistributed should be
marked as build-only metadata and documented as build-machine requirements. They
should not become end-user installation steps after the Jayess app package has
been produced.

## Verification

Build the distribution from a clean checkout, run the packaged executable from
inside the distribution folder, and verify it does not depend on undeclared
local build paths or separate end-user native library installation.

## Example Layout

```text
dist/my-app/
  bin/my-app
  lib/libsqlite3.so
  assets/logo.png
  licenses/LICENSE.sqlite
```
