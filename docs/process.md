# Process

Jayess exposes a small native-process surface through `process`.

## Helpers

```javascript
console.log(process.cwd());
console.log(process.argv());
console.log(process.env("HOME"));
console.log(process.platform());
console.log(process.arch());
console.log(process.threadPoolSize());
process.onSignal("SIGTERM", (event) => {
  console.log(event.signal);
});
```

Supported helpers:

- `process.cwd()`
- `process.argv()`
- `process.env(name)`
- `process.platform()`
- `process.arch()`
- `process.threadPoolSize()`
- `process.exit(code)`
- `process.onSignal(signal, callback)`
- `process.onceSignal(signal, callback)`
- `process.offSignal(signal[, callback])`
- `process.raise(signal)`

## Notes

- `process.argv()` returns the runtime argument array
- `process.platform()` returns values such as `"windows"`, `"linux"`, or `"darwin"`
- `process.arch()` returns values such as `"x64"` or `"arm64"`
- `process.threadPoolSize()` returns the fixed native async file I/O worker count
- signal callbacks currently receive an event object like `{ signal, number }`
- signal delivery is dispatched from normal runtime polling, not directly inside the OS signal handler
