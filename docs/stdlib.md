# Standard Library

This page summarizes the standard library surface currently wired into the Jayess compiler/runtime.

## Console

Jayess supports:

- `console.log(...)`
- `console.warn(...)`
- `console.error(...)`

`print(...)` still works, but it is deprecated.

See [Console](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/console.md).

## Process, Path, and File System

Supported process helpers:

- `process.cwd()`
- `process.argv()`
- `process.env(name)`
- `process.platform()`
- `process.arch()`
- `process.exit(code)`

Supported path helpers:

- `path.join(...parts)`
- `path.normalize(path)`
- `path.resolve(...parts)`
- `path.relative(from, to)`
- `path.parse(path)`
- `path.format(parts)`
- `path.sep`
- `path.delimiter`
- `path.isAbsolute(path)`
- `path.basename(path)`
- `path.dirname(path)`
- `path.extname(path)`

Supported file helpers:

- `fs.readFile(path)`
- `fs.readFile(path, encoding)`
- `fs.writeFile(path, content)`
- `fs.exists(path)`
- `fs.readDir(path[, options])`
- `fs.stat(path)`
- `fs.mkdir(path[, options])`
- `fs.remove(path)`
- `fs.copyFile(from, to)`
- `fs.copyDir(from, to)`
- `fs.rename(from, to)`

Notes:

- `fs.readFile(path)` returns file contents or `undefined`
- `fs.readFile(path, encoding)` currently supports text encodings like `utf8`
- `fs.writeFile(path, content)` returns a boxed boolean-like result
- `fs.mkdir(path, { recursive: true })` creates parent directories when needed
- `fs.readDir(path, { recursive: true })` walks nested directories
- `fs.readDir(path)` returns entry objects with `name`, `path`, `isDir`, `isFile`, and `size`
- `fs.readDir(path)` entry objects also include `mtimeMs` and `permissions` where available
- `fs.stat(path)` returns file metadata with `path`, `isDir`, `isFile`, `size`, `mtimeMs`, and `permissions`
- `fs.remove(path)` distinguishes files and directories
- `fs.remove(path, { recursive: true })` removes directory trees recursively
- `fs.copyDir(from, to)` copies directory trees recursively
- `path.parse(path)` returns an object with `root`, `dir`, `base`, `ext`, and `name`

See also:

- [File System](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/fs.md)
- [Path](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/path.md)
- [Process](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/process.md)

## Collections

### Map

Supported:

- `new Map()`
- `map.set(key, value)`
- `map.get(key)`
- `map.has(key)`
- `map.delete(key)`
- `map.clear()`
- `map.keys()`
- `map.values()`
- `map.entries()`
- `map.size`

### Set

Supported:

- `new Set()`
- `set.add(value)`
- `set.has(value)`
- `set.delete(value)`
- `set.clear()`
- `set.values()`
- `set.entries()`
- `set.size`

### Iteration

`for...of` works with:

- arrays
- `Map`
- `Set`

The current implementation uses a pragmatic iterable bridge rather than the full JavaScript iterator protocol.

## Date

Supported:

- `Date.now()`
- `new Date()`
- `new Date(value)`
- `date.getTime()`
- `date.toString()`
- `date.toISOString()`

## JSON

Supported:

- `JSON.stringify(value)`
- `JSON.parse(text)`

Current note:

- `JSON.parse(...)` currently returns `undefined` on parse failure instead of throwing

## RegExp

Supported:

- `new RegExp(pattern[, flags])`
- `re.source`
- `re.flags`
- `re.test(text)`

String regex helpers:

- `text.match(re)`
- `text.search(re)`
- `text.replace(re, replacement)`
- `text.split(re)`

Current note:

- this is a pragmatic regex slice, not full ECMAScript regex parity

## Math

Supported:

- `Math.floor`
- `Math.ceil`
- `Math.round`
- `Math.min`
- `Math.max`
- `Math.abs`
- `Math.pow`
- `Math.sqrt`
- `Math.random`

## Number

Supported:

- `Number.isNaN`
- `Number.isFinite`

## String

Supported instance helpers:

- `length`
- `includes`
- `startsWith`
- `endsWith`
- `slice`
- `trim`
- `toUpperCase`
- `toLowerCase`
- `split`

Supported static helper:

- `String.fromCharCode`

## Array

Supported:

- `length`
- `push`
- `pop`
- `shift`
- `unshift`
- `slice`
- `includes`
- `join`
- `map`
- `filter`
- `find`
- `forEach`

Supported static helpers:

- `Array.isArray`
- `Array.from`
- `Array.of`

## Object

Supported:

- `Object.keys`
- `Object.values`
- `Object.entries`
- `Object.assign`
- `Object.hasOwn`
- `Object.fromEntries`
