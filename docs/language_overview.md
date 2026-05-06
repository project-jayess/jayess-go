# Language Overview

Jayess is a JavaScript-like native programming language compiled by the Go
hosted Jayess compiler. The language keeps familiar JavaScript syntax where it
maps cleanly to predictable native output, but it deliberately avoids the full
dynamic browser/Node runtime model.

## Goals

- compile `.js` Jayess sources into LLVM IR, objects, libraries, or executables
- keep syntax familiar for JavaScript developers
- provide explicit native binding support for C and C++ libraries
- package applications with the runtime assets they need
- keep the language small enough for reliable native compilation

## Current Shape

Jayess supports variables, functions, closures, objects, arrays, classes,
modules, control flow, exceptions, and native bindings. The compiler pipeline is
implemented in Go and is organized around lexing, parsing, semantic analysis,
lowering, and LLVM backend emission.

## Example

```js
import { add } from "./native/math.js";

function main(args) {
  const first = 40;
  var total = add(first, 2);
  console.log(total);
  return 0;
}
```

## Non-Goals

Jayess is not trying to be a browser JavaScript runtime or a Node.js clone. It
does not require a package manager or runtime JavaScript executable. External
native libraries are expected to be installed or supplied by the developer and
declared through bindings.
