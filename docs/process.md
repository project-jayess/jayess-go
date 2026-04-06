# Process

Jayess exposes a small native-process surface through `process`.

## Helpers

```javascript
console.log(process.cwd());
console.log(process.argv());
console.log(process.env("HOME"));
console.log(process.platform());
console.log(process.arch());
```

Supported helpers:

- `process.cwd()`
- `process.argv()`
- `process.env(name)`
- `process.platform()`
- `process.arch()`
- `process.exit(code)`

## Notes

- `process.argv()` returns the runtime argument array
- `process.platform()` returns values such as `"windows"`, `"linux"`, or `"darwin"`
- `process.arch()` returns values such as `"x64"` or `"arm64"`
