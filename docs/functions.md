# Functions

Functions are first-class Jayess values that can be declared, passed, called,
and returned.

## Declarations and Calls

```js
function add(a, b) {
  return a + b;
}

const value = add(1, 2);
```

Calls evaluate the callee and arguments before invoking the function or native
binding wrapper. The backend emits direct calls when possible and runtime calls
when dynamic behavior is required.

## Closures

Closures capture referenced outer variables through compiler-managed capture
records. Escape and lifetime analysis prepare the capture layout before backend
emission.

## Returns

`return` lowers through a generalized return path so expressions, cleanup, and
abrupt control flow are handled consistently.

## Current Limits

Jayess supports a practical function subset. Unsupported JavaScript function
features should produce explicit parser or semantic diagnostics rather than
silently changing behavior.
