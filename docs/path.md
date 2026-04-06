# Path

Jayess provides cross-platform path helpers through `path`.

## Helpers

```javascript
var joined = path.join("tmp", "note.txt");
var normalized = path.normalize("tmp/./note.txt");
var resolved = path.resolve("tmp", "..", "note.txt");
var relative = path.relative("tmp", path.join("tmp", "nested", "note.txt"));
```

Supported helpers:

- `path.join(...parts)`
- `path.normalize(path)`
- `path.resolve(...parts)`
- `path.relative(from, to)`
- `path.parse(path)`
- `path.format(parts)`
- `path.isAbsolute(path)`
- `path.basename(path)`
- `path.dirname(path)`
- `path.extname(path)`

## Platform Constants

```javascript
console.log(path.sep);
console.log(path.delimiter);
```

- `path.sep` returns the current platform path separator
- `path.delimiter` returns the current platform list delimiter

## Parse / Format

```javascript
var parts = path.parse("tmp/note.txt");
console.log(parts.dir);
console.log(path.format(parts));
```

`path.parse(path)` returns:

- `root`
- `dir`
- `base`
- `ext`
- `name`
