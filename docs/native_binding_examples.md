# Native Binding Examples

This is a minimal binding shape for a C function.

## Jayess Binding File

```js
import { bind } from "ffi";

export const add = () => {};

export default bind({
  sources: ["./math.c"],
  exports: {
    add: { symbol: "math_add", type: "function" }
  }
});
```

## C Wrapper

```c
#include "jayess_runtime.h"

jayess_value *math_add(jayess_value *a, jayess_value *b) {
  int left = jayess_expect_int(a);
  int right = jayess_expect_int(b);
  return jayess_value_from_int(left + right);
}
```

## Shipping

If the wrapper depends on a shared library, include that shared library and its
license in the app distribution.
