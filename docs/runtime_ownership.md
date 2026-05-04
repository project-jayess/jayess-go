# Runtime Ownership Rules

This document defines the ownership terms used by public native binding helpers
that pass `jayess_value *` values across the Jayess runtime boundary.

## Ownership Terms

- Owned value: a `jayess_value *` managed by the Jayess runtime. Wrapper code must
  not manually free it.
- Borrowed value: a `jayess_value *` or view that is valid only during the current
  native call. Wrapper code must not store it in long-lived native state.
- Copied buffer: a string or byte buffer allocated for wrapper code. The matching
  free helper must release it.
- Retained value: a value preserved by the runtime because it was stored in a
  container, closure environment, global/module state, or managed native handle.
- Closed value: a managed native handle whose finalizer has run. Closing again is
  safe, but using the closed handle must report an error.

## `jayess_value *` Helpers

| Helper family | Ownership rule |
| --- | --- |
| `jayess_value_from_*` | Returns a runtime-owned `jayess_value *`. Wrapper code must not free it manually. |
| `jayess_value_from_bytes_copy(...)` | Copies bytes into a runtime-owned `jayess_value *`. The caller may release or reuse the source buffer after the call. |
| `jayess_value_as_object(...)` | Returns a borrowed object view from a `jayess_value *`; it must not outlive the current native call. |
| `jayess_value_as_array(...)` | Returns a borrowed array view from a `jayess_value *`; it must not outlive the current native call. |
| `jayess_value_as_string(...)` | Returns a borrowed string view valid only during the current native call. Use `jayess_value_to_string_copy(...)` for long-lived native state. |
| `jayess_value_to_string_copy(...)` | Returns a copied string buffer owned by wrapper code. Release it with `jayess_string_free(...)`. |
| `jayess_value_to_bytes_copy(...)` | Returns a copied byte buffer owned by wrapper code. Release it with `jayess_bytes_free(...)`. |
| `jayess_value_from_native_handle(...)` | Returns a runtime-owned wrapper for an unmanaged opaque handle. Jayess does not close the underlying handle. |
| `jayess_value_as_native_handle(...)` | Returns a borrowed native handle pointer. Wrapper code must validate that managed handles are not closed before use. |
| `jayess_value_from_managed_native_handle(...)` | Returns a runtime-owned managed handle value. The registered finalizer is retained and may run at most once. |
| `jayess_value_close_native_handle(...)` | Closes a managed native handle value. Repeated close is safe; later use of the closed handle must report an error. |

## String and Byte Buffers

Borrowed string and byte views are valid only during the current native call.
Native wrappers that need long-lived state must call a `*_copy(...)` helper and
release the copied buffer with `jayess_string_free(...)` or
`jayess_bytes_free(...)`.

## Retention Boundaries

Values stored in objects, arrays, closure environments, globals, module state,
or managed native handles must be retained or otherwise preserved by the runtime
for as long as the owner can reach them. Replacing a stored value must release the
previous retained value safely.

## Containers

Object property insertion and array element insertion must retain or otherwise
preserve the stored value so object and array references remain valid after the
storing scope exits. Replacing an object property or array element must release
the previous retained value after the new value is preserved.

Removing a value from an object or array must release only the container's reference. Other aliases, containers, closures, globals, module state, or native
handles that still reference the same value must keep it valid.

## Dynamic Objects

Dynamic object values own an ordered property table. Property keys are canonical
Jayess property strings produced from source property names or computed keys.
Writing a property preserves the inserted value before replacing any previous
slot value. Deleting a property removes only the object's reference and preserves
the validity of other aliases to the same value.

Object enumeration uses insertion order for currently present properties. Updating
an existing property must not move it in enumeration order. Object spread copies
the current enumerable property values into the target object, and object rest
copies every enumerable property except the excluded keys.

## Dynamic Arrays

Dynamic array values own indexed slots plus an ordinary named-property table.
Numeric canonical property keys address array slots; non-index property keys use
the named-property table. The `length` property reflects the current slot count.

Writing past the current length grows the slot table and leaves unassigned slots
as holes that read as `undefined` when materialized for iteration or spread.
Reducing `length` truncates slots and releases only the array references to the
removed values. Growing `length` creates holes without retaining new values.

Array `for...in` enumeration yields present numeric indexes first, followed by
named properties in insertion order. Array `for...of`, array spread, and array
rest materialize holes as `undefined` values so lowering has deterministic
runtime behavior.

## Closure Environments

Closure environments must retain captured values for as long as any closure can
reach the environment. Closure environment cleanup must release captured values
exactly once after the environment becomes unreachable.

## Invalid Use Prevention

Jayess-managed values must have one runtime owner for cleanup purposes. Retain
and release operations must be balanced so a value finalizer can run at most once,
preventing double-free of managed values.

Values reachable from locals, containers, closure environments, globals, module
state, or native handles must stay valid until the last retaining owner releases them.
Borrowed pointers and views must not be used after the current native call.

Freed or closed runtime values must not be reused silently. Any operation that
uses a closed managed native handle or otherwise invalid runtime value must report a runtime error or compiler diagnostic instead of dereferencing stale storage.

Compiler lifetime metadata, runtime retain/release ownership, and native binding
contracts must agree on pointer and reference validity across every boundary.

## Native Binding Safety

Native wrappers must not store borrowed Jayess pointers beyond the current call.
Wrappers that need long-lived native state must copy strings or bytes with the runtime copy helpers and release those copies with the matching free helper.

Managed native handles become invalid after close. Repeated close on a managed native handle is safe and must not run the finalizer more than once. Using a closed managed native handle must report a runtime error.

Native finalizers must run at most once for each managed native handle, whether
the handle is closed explicitly or finalized by runtime cleanup.
