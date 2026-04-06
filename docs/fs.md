# File System

Jayess provides a compact native file-system surface through `fs`.

## Read and Write

```javascript
var text = fs.readFile("notes.txt");
var utf8 = fs.readFile("notes.txt", "utf8");
fs.writeFile("notes.txt", "kimchi");
```

Current behavior:

- `fs.readFile(path[, encoding])` returns file contents or `undefined`
- supported encodings are pragmatic text aliases such as `"utf8"` and `"utf-8"`
- `fs.writeFile(path, content)` returns a boolean-like result

## Existence and Metadata

```javascript
var exists = fs.exists("notes.txt");
var stat = fs.stat("notes.txt");
```

`fs.stat(path)` returns `undefined` when the path does not exist. When it exists, the result object includes:

- `path`
- `isDir`
- `isFile`
- `size`
- `mtimeMs`
- `permissions`

## Directories

```javascript
fs.mkdir("build/tmp", { recursive: true });
var entries = fs.readDir("build", { recursive: true });
```

`fs.readDir(...)` entry objects include:

- `name`
- `path`
- `isDir`
- `isFile`
- `size`
- `mtimeMs`
- `permissions`

## Remove and Copy

```javascript
fs.copyFile("a.txt", "b.txt");
fs.copyDir("assets", "backup-assets");
fs.remove("backup-assets", { recursive: true });
fs.rename("a.txt", "b.txt");
```

Notes:

- `fs.remove(path)` removes a single file or an empty directory
- `fs.remove(path, { recursive: true })` removes directory trees recursively
- `fs.copyDir(from, to)` performs a recursive directory copy
