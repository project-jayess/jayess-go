# Native Library Setup

Jayess does not ship arbitrary third-party native libraries by default. The
developer is responsible for installing or vendoring the libraries used by a
binding.

## Options

- install system development packages
- keep headers and libraries under a project `vendor/` directory
- build a third-party library in `temp/` and copy the release artifacts into a
  project-controlled location

## Binding Inputs

Point binding manifests at the headers, sources, library directories, shared
libraries, and license files that belong to the project.

## Reproducibility

Document native library versions in the project docs or release notes so another
developer can rebuild the same app distribution.

## Example Layout

```text
vendor/
  sqlite/
    include/sqlite3.h
    lib/libsqlite3.so
    LICENSE
src/
  native/sqlite.js
```
