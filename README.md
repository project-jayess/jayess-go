# jayess-go

Go compiler skeleton for the Jayess programming language.

## Status

This repository now contains an MVP compiler pipeline:

1. lexing
2. parsing
3. semantic validation
4. lifetime analysis placeholder
5. lowering to a minimal IR
6. LLVM IR text emission

The current source subset is intentionally small:

```javascript
function main(args) {
  const delay = 500;
  var total = 10.5 + 2 * 3;
  print(total);
  print(args[0]);
  sleep(delay);
  var name = readLine("What is your name? ");
  print(name);
  readKey("Press any key to continue");
  return 0;
}
```

Jayess variable declarations are:
- `var` for mutable block-scoped bindings
- `const` for immutable bindings

`let` is not supported.
`public` is not supported. Module visibility is controlled by `export`.
Top-level `private` is not supported. Class privacy uses JavaScript-style `#members`.

Classes are available in MVP form:
- `class Name { ... }`
- `export class Name { ... }`
- `export default class Name { ... }`
- `new Name(...)`

Current class scope is intentionally small:
- constructor
- instance field initializers
- instance methods
- `#private` instance fields and methods
- static fields
- static methods
- `static #private` fields and methods
- `this`
- method calls like `obj.method()`
- single inheritance with `extends`
- `super(...)` in constructors
- `super.method()` in instance/static methods
- `super.property` access for inherited properties

Native wrapper interop is available in a first pass:
- `import { jayess_add } from "./native/math.c";`
- `import { jayess_add as add } from "./native/math.c";`
- `import "./native/math.c";` for side-effect-only linking
- imported native wrapper calls receive boxed Jayess runtime values and return boxed Jayess runtime values
- wrappers should include [jayess_runtime.h](/C:/Users/ncksd/Documents/it/jayess/jayess-go/runtime/jayess_runtime.h)
- C++ wrappers should export C ABI entrypoints with `extern "C"`

Local relative imports are also supported in MVP form:

```javascript
import { add, twice } from "./lib/math.js";

function main(args) {
  var value = add(3, 4);
  print(twice(value));
  return value;
}
```

Imported functions must be exported explicitly:

```javascript
export function add(a, b) {
  return a + b;
}
```

Class example:

```javascript
export class Counter {
  value = 1;
  #secret = 9;

  constructor(step) {
    this.step = step;
  }

  total() {
    return this.value + this.step + this.#secret;
  }
}
```

Current import support is limited to relative file imports:
- `import "./utils.js";`
- `import { add, twice } from "./lib/math.js";`
- `import { add as sum } from "./lib/math.js";`
- `import thing from "./lib/module.js";`
- `import thing, { add as sum } from "./lib/module.js";`
- `import * as ns from "./lib/module.js";`
- `import { add } from "@demo/math";`
- `import thing from "@demo/math";`
- `import * as ns from "@demo/math";`

Current export support is limited to:
- `export function name(...) { ... }`
- `export const NAME = value;`
- `export var name = value;`
- `export default function name(...) { ... }`
- `export default <expression>;`
- `export { local, other as renamed };`
- `export { local as renamed } from "@demo/math";`

Native wrapper example:

```javascript
import { jayess_add, jayess_greet } from "./native/math.c";

function main(args) {
  print(jayess_add(3, 4));
  print(jayess_greet("Kimchi"));
  return 0;
}
```

Example C wrapper:

```c
#include "jayess_runtime.h"

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
  return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
```

## Build

```bash
$env:GOCACHE="$PWD\.gocache"
$env:GOFLAGS="-buildvcs=false"
go build -o build\windows\jayess.exe .\cmd\jayess
```

## Run

```bash
go run ./cmd/jayess --target=host --emit=llvm -o build/hello.ll examples/hello.js
go run ./cmd/jayess --target=host --emit=llvm -o build/import.ll examples/import.js
```

This emits LLVM IR text to `build/hello.ll`.

To build a native executable once `clang` is installed and on `PATH`:

```bash
go run ./cmd/jayess --target=host --emit=exe -o build/hello.exe examples/hello.js
```

On Windows, the default is now native executable output, so this also works:

```bash
.\build\windows\jayess.exe -o .\examples\build\hello.exe .\examples\hello.js
```

That defaults to `build/hello.exe`.

The current executable path is:

1. Jayess source
2. Jayess LLVM IR text
3. `clang`
4. native executable

## Next steps

- replace the text emitter with LLVM bindings or a thin C bridge
- expand the parser toward Jayess's JavaScript-like syntax
- implement package resolution and runtime support
