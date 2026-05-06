# Runtime Values

Compiled Jayess programs use a shared runtime value model for primitives,
objects, arrays, functions, strings, bytes, and native handles.

## Helpers

Native bindings should include `runtime/jayess_runtime.h` and use helpers such
as:

- `jayess_value_as_object`
- `jayess_expect_object`
- `jayess_expect_array`
- `jayess_value_to_string_copy`
- `jayess_value_from_bytes_copy`
- `jayess_value_from_native_handle`
- `jayess_value_from_managed_native_handle`
- `jayess_throw_error`

## Ownership

Boxed `jayess_value` instances are owned by the Jayess runtime. Native code can
borrow pointers during a call, but long-lived native state should copy strings
or bytes and use native handles for external resources.

See `docs/runtime_ownership.md` for detailed ownership rules.

## Example Native Return

```c
#include "jayess_runtime.h"

jayess_value *native_name(void) {
  return jayess_value_from_string("jayess");
}
```
