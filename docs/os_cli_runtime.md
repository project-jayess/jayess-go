# OS and CLI Runtime Services

Jayess exposes OS-facing services through declared standard-library surfaces and
small Go runtime helpers. These helpers are intended for CLI programs, build
tools, and future self-hosting work without depending on Go-only internals.

## Process and Stdio

The process runtime tracks arguments, environment variables, current working
directory, process id, platform, high-resolution elapsed time, and explicit exit
code state. Stdin, stdout, and stderr are stream objects.

```js
function main() {
  const input = process.stdin.read();
  process.stdout.write("read " + input.length + " bytes\n");
  process.stderr.write("done\n");
  process.exit(0);
}
```

## Filesystem

The filesystem helpers cover common CLI and compiler tasks: reading, writing,
appending, copying, renaming, deleting, stat metadata, existence checks,
directory creation, directory listing, recursive walking, chmod, symlinks where
the platform allows them, polling-based watch events, and file streams.

```js
function main() {
  fs.mkdirp("dist");
  fs.writeFile("dist/message.txt", "hello\n");
  fs.appendFile("dist/message.txt", "from Jayess\n");

  const input = fs.createReadStream("dist/message.txt");
  const output = fs.createWriteStream("dist/copy.txt");
  stream.pipe(input, output);
}
```

## Child Processes

Child process helpers provide spawn and exec behavior with captured stdout,
captured stderr, stdin input, exit status, signal delivery, and cleanup.

```js
function main() {
  const result = childProcess.exec("jayess", ["--version"]);
  process.stdout.write(result.stdout);
  return result.exitCode;
}
```

## Terminal Handling

The terminal surface reports whether a file is a terminal, terminal dimensions
from the environment, and basic color support detection.

```js
function main() {
  if (terminal.supportsColor(process.stdout)) {
    process.stdout.write("\x1b[32mready\x1b[0m\n");
  } else {
    process.stdout.write("ready\n");
  }
}
```

These services are declaration-driven at the compiler surface and backed by
focused Go runtime helpers so tests can validate behavior independently from
backend code generation.
