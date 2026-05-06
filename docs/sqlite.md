# SQLite

SQLite support is provided through native package or binding integration.

## Setup

Developers must provide the SQLite development headers and library for their
target platform unless a project chooses to vendor them through its own binding
layout.

## Linking

Declare SQLite native sources or shared libraries in a binding manifest. Include
library directories, shared libraries, and license files as needed for the
application distribution.

## Distribution

If the built executable depends on a SQLite shared library, package that runtime
library and the SQLite license/notice files with the app distribution.

## Example Shape

```js
import { open } from "./native/sqlite.js";

function main() {
  const db = open("./app.db");
  db.exec("create table if not exists notes (text value)");
  return 0;
}
```
