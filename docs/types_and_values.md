# Types and Values

Jayess values are represented by the runtime value model used by compiled code
and native bindings.

## Primitive Values

Supported primitive value categories include nullish values, booleans, numbers,
BigInts where enabled by lowering/backend support, strings, functions, native
handles, objects, and arrays.

## Truthiness

Conditionals and logical operators use Jayess runtime truthiness. Nullish values
and false booleans are false. Numeric zero and empty strings follow the current
runtime truthiness rules used by lowering and backend runtime calls.

## Equality

The compiler lowers equality through type-aware and runtime-backed operations.
Primitive equality is handled directly where possible. Object, array, function,
and native handle equality is identity based unless a runtime operation defines a
specific comparison.

## Native Boundary

Native binding functions receive and return boxed Jayess runtime values. Binding
authors should use helpers from `runtime/jayess_runtime.h` to inspect and create
values.

## Example

```js
function main() {
  const name = "Jayess";
  var count = 1;
  const user = { name: name, count: count };

  if (user.count == 1) {
    console.log(user.name);
  }

  return 0;
}
```
