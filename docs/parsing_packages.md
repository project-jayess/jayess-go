# HTML, XML, and CSS Parsing Packages

Jayess includes built-in parsing package support for web text formats.

## Purpose

These packages are intended for native Jayess tools that need lightweight
inspection or transformation of HTML, XML, or CSS without embedding a browser
runtime.

## Usage

Import the package exposed by the compiler/runtime package surface and call the
documented parsing helpers for the format. Keep parsed data ownership within
Jayess values unless a native binding explicitly documents another contract.

## Distribution

Built-in parsing packages should not require end users to install separate
native libraries unless the package implementation explicitly uses one.

## Example Shape

```js
import { parseHTML } from "html";

function main() {
  const doc = parseHTML("<main><h1>Jayess</h1></main>");
  console.log(doc.root.name);
  return 0;
}
```
