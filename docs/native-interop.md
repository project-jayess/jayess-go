# Native Interop

Jayess can link native wrapper code written in C or C++ and call exported wrapper functions from Jayess.

## Importing Native Sources

Examples:

```javascript
import { jayess_add, jayess_scale as scale } from "./native/math.c";
import "./native/math.c";
```

Supported native source extensions are resolved by the loader. Native wrappers are linked into the final executable through the native toolchain.

## Wrapper Style

Wrappers should use the Jayess runtime API and exchange boxed Jayess values.

Example C wrapper:

```c
#include "jayess_runtime.h"

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
  return jayess_value_from_number(
    jayess_value_to_number(a) + jayess_value_to_number(b)
  );
}
```

Jayess usage:

```javascript
import { jayess_add } from "./native/math.c";

function main(args) {
  console.log(jayess_add(3, 4));
  return 0;
}
```

## C++

For C++ wrappers, expose a C ABI entrypoint:

```cpp
extern "C" jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
  return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
```

## Runtime Header

Wrappers should include:

- [jayess_runtime.h](/C:/Users/ncksd/Documents/it/jayess/jayess-go/runtime/jayess_runtime.h)

That header exposes helpers for:

- boxing numbers, strings, booleans, arrays, and objects
- converting boxed values
- interacting with the Jayess runtime

## Notes

- Native interop is explicit. Jayess is not a JavaScript engine and does not auto-load Node APIs.
- Native wrappers are the intended bridge to C and C++ libraries.
- Keep wrappers small and stable; put library-specific adaptation logic in the wrapper layer.
