# Console

Jayess provides a JavaScript-like `console` surface for text output.

## Methods

```javascript
console.log("hello", 1, true);
console.warn("warning");
console.error("error");
```

- `console.log(...)` writes to standard output.
- `console.warn(...)` writes to standard error.
- `console.error(...)` writes to standard error.

Arguments are printed space-separated, similar to JavaScript console output.

## Print Deprecation

`print(...)` still works for compatibility, but it is deprecated.

Prefer:

```javascript
console.log("hello");
```

instead of:

```javascript
print("hello");
```

The compiler now emits a warning when `print(...)` is used.
