# Jayess feature checklist

## 1. Core language

### 1.1 Lexical / syntax

- [ ] single-line comments (`//`)
- [ ] block comments (`/* */`)
- [ ] identifiers
- [ ] keywords / reserved words
- [ ] semicolon handling
- [ ] operator precedence
- [ ] grouping with parentheses
- [ ] comma expressions if supported

### 1.2 Literals

- [ ] number literals
- [ ] string literals
- [ ] boolean literals
- [ ] `null`
- [ ] `undefined` or Jayess equivalent
- [ ] bigint literals
- [ ] object literals
- [ ] array literals
- [ ] template strings

### 1.3 Variables and bindings

- [ ] `var` declarations
- [ ] `const` declarations
- [ ] block scope
- [ ] lexical scope
- [ ] shadowing
- [ ] declaration without hoisting
- [ ] no use before declaration
- [ ] destructuring declarations
- [ ] default values in declarations

### 1.4 Operators

- [ ] arithmetic operators
- [ ] comparison operators
- [ ] logical operators
- [ ] assignment operators
- [ ] bitwise operators
- [ ] unary operators
- [ ] ternary operator
- [ ] optional chaining
- [ ] nullish coalescing
- [ ] `typeof`
- [ ] `instanceof`

### 1.5 Control flow

- [ ] `if`
- [ ] `else`
- [ ] `switch`
- [ ] `for`
- [ ] `while`
- [ ] `do while`
- [ ] `for...of`
- [ ] `break`
- [ ] `continue`
- [ ] labeled statements if supported
- [ ] `return`
- [ ] `throw`
- [ ] `try`
- [ ] `catch`
- [ ] `finally`

---

## 2. Functions

### 2.1 Basic functions

- [ ] function declarations
- [ ] function expressions
- [ ] arrow functions
- [ ] anonymous functions
- [ ] nested functions
- [ ] recursion
- [ ] first-class functions

### 2.2 Parameters

- [ ] positional parameters
- [ ] default parameters
- [ ] rest parameters
- [ ] variadic calls

### 2.3 Function behavior

- [ ] closures
- [ ] captured variables
- [ ] lexical `this` for arrow functions
- [ ] normal `this` for regular functions
- [ ] function return values
- [ ] higher-order functions
- [ ] callback support

### 2.4 Invocation helpers

- [ ] method calls (`obj.fn()`)
- [ ] function values in variables
- [ ] function values in arrays
- [ ] function values in objects
- [ ] `.bind()` equivalent
- [ ] `.call()` equivalent
- [ ] `.apply()` equivalent

---

## 3. Objects and classes

### 3.1 Objects

- [ ] object property read
- [ ] object property write
- [ ] computed property names
- [ ] method definitions
- [ ] property enumeration
- [ ] object spread
- [ ] object destructuring

### 3.2 Classes

- [ ] `class`
- [ ] constructors
- [ ] instance methods
- [ ] static methods
- [ ] instance fields
- [ ] static fields
- [ ] getters
- [ ] setters
- [ ] `this`
- [ ] `new`

### 3.3 Inheritance

- [ ] `extends`
- [ ] `super`
- [ ] prototype chain or Jayess equivalent
- [ ] method overriding
- [ ] `instanceof` support

### 3.4 Encapsulation

- [ ] private fields if supported
- [ ] private methods if supported
- [ ] visibility rules if supported

---

## 4. Arrays, strings, and built-in data structures

### 4.1 Arrays

- [ ] array creation
- [ ] indexing
- [ ] mutation
- [ ] length
- [ ] iteration
- [ ] array destructuring
- [ ] rest elements
- [ ] spread elements

### 4.2 Strings

- [ ] string concatenation
- [ ] string indexing
- [ ] string length
- [ ] template strings
- [ ] unicode support

### 4.3 Built-in collections

- [ ] `Map`
- [ ] `Set`
- [ ] `WeakMap` if supported
- [ ] `WeakSet` if supported

### 4.4 Other built-ins

- [ ] `Date`
- [ ] `RegExp`
- [ ] `Symbol`
- [ ] `ArrayBuffer`
- [ ] typed arrays
- [ ] `DataView`

---

## 5. Iteration and generators

### 5.1 Iteration

- [ ] iterable protocol
- [ ] iterator protocol
- [ ] `for...of`
- [ ] custom iterables

### 5.2 Generators

- [ ] generator functions
- [ ] `yield`
- [ ] generator iteration
- [ ] async iterators if supported
- [ ] async generators if supported

---

## 6. Async model

### 6.1 Promises

- [ ] `Promise`
- [ ] resolve / reject
- [ ] chaining
- [ ] error propagation
- [ ] `Promise.all`
- [ ] `Promise.race`
- [ ] `Promise.allSettled` if supported
- [ ] `Promise.any` if supported

### 6.2 Async functions

- [ ] `async` functions
- [ ] `await`
- [ ] async error handling
- [ ] async return values

### 6.3 Scheduling

- [ ] event loop model
- [ ] microtask queue if supported
- [ ] timer queue
- [ ] cancellation model if supported

---

## 7. Modules and package system

### 7.1 Module syntax

- [ ] `import`
- [ ] `export`
- [ ] named exports
- [ ] default exports
- [ ] namespace imports
- [ ] re-exports
- [ ] `export *`
- [ ] `export * as ns`

### 7.2 Resolution

- [ ] relative imports
- [ ] parent-directory imports
- [ ] local project file imports
- [ ] package imports from `node_modules`
- [ ] scoped package imports
- [ ] package entry resolution
- [ ] `package.json` reading
- [ ] module initialization order
- [ ] circular imports
- [ ] clear diagnostics for unsupported JS packages

---

## 8. Error handling and diagnostics

### 8.1 Compiler diagnostics

- [ ] lexer errors
- [ ] parser errors
- [ ] semantic errors
- [ ] type errors if types exist
- [ ] module resolution errors
- [ ] lifetime / escape diagnostics
- [ ] source spans
- [ ] helpful messages

### 8.2 Runtime errors

- [ ] exceptions
- [ ] stack traces
- [ ] source locations in stack traces
- [ ] uncaught exception handling

---

## 9. Memory / lifetime behavior

### 9.1 Scope behavior

- [ ] lexical lifetime model
- [ ] block cleanup for non-escaping values
- [ ] escaping value retention
- [ ] globals not cleaned as locals
- [ ] closure-captured value retention

### 9.2 Escape cases

- [ ] returned values
- [ ] closure captures
- [ ] values stored in objects
- [ ] values stored in arrays
- [ ] values assigned to global/module state

### 9.3 Runtime / lowering validation

- [ ] scope-exit cleanup correctness
- [ ] escaping object correctness
- [ ] closure environment correctness
- [ ] no use-after-scope bugs

---

## 10. Standard library / runtime APIs

### 10.1 Process and environment

- [ ] command-line args
- [ ] environment variables
- [ ] current working directory
- [ ] exit codes
- [ ] stdin
- [ ] stdout
- [ ] stderr
- [ ] process info
- [ ] high-resolution time
- [ ] signals

### 10.2 Filesystem

- [ ] read file
- [ ] write file
- [ ] append file
- [ ] delete file
- [ ] rename / move file
- [ ] copy file
- [ ] stat / metadata
- [ ] file permissions
- [ ] file exists check
- [ ] create directory
- [ ] recursive directory creation
- [ ] remove directory
- [ ] list directory
- [ ] recursive directory listing
- [ ] symlink support if supported
- [ ] file watching if supported
- [ ] file streams

### 10.3 Path utilities

- [ ] path join
- [ ] path resolve
- [ ] path normalize
- [ ] basename
- [ ] dirname
- [ ] extension extraction
- [ ] relative path calculation

### 10.4 URL and query utilities

- [ ] URL parsing
- [ ] URL formatting
- [ ] query string parse
- [ ] query string stringify
- [ ] percent encoding / decoding
- [ ] file URL support if supported

### 10.5 Buffers and binary data

- [ ] binary buffer type
- [ ] string encoding / decoding
- [ ] byte slicing
- [ ] byte copying
- [ ] endian-aware reads/writes
- [ ] typed arrays
- [ ] binary stream support

### 10.6 Streams

- [ ] readable streams
- [ ] writable streams
- [ ] duplex streams
- [ ] transform streams
- [ ] piping
- [ ] backpressure handling

---

## 11. Networking

### 11.1 HTTP

- [ ] HTTP server
- [ ] HTTP client
- [ ] request object
- [ ] response object
- [ ] headers
- [ ] status codes
- [ ] request body reading
- [ ] response body writing
- [ ] streaming bodies
- [ ] keep-alive
- [ ] timeout handling

### 11.2 HTTPS

- [ ] HTTPS server
- [ ] HTTPS client
- [ ] TLS certificate loading
- [ ] private key loading
- [ ] CA / trust configuration
- [ ] certificate verification
- [ ] secure defaults

### 11.3 TCP

- [ ] TCP client sockets
- [ ] TCP server sockets
- [ ] connect
- [ ] listen
- [ ] accept
- [ ] read/write data
- [ ] close socket
- [ ] socket errors
- [ ] timeout support
- [ ] backpressure handling

### 11.4 TLS

- [ ] TLS client
- [ ] TLS server
- [ ] certificate handling
- [ ] ALPN if supported
- [ ] hostname verification

### 11.5 UDP

- [ ] UDP sockets
- [ ] send datagrams
- [ ] receive datagrams
- [ ] bind socket
- [ ] multicast if supported
- [ ] broadcast if supported

### 11.6 DNS

- [ ] hostname lookup
- [ ] reverse lookup
- [ ] custom resolver support if desired
- [ ] IP utilities

---

## 12. Crypto and compression

### 12.1 Crypto

- [ ] random bytes
- [ ] hashing
- [ ] HMAC
- [ ] symmetric encryption
- [ ] asymmetric encryption if supported
- [ ] digital signatures
- [ ] key generation
- [ ] secure compare

### 12.2 Compression

- [ ] gzip
- [ ] deflate
- [ ] brotli if supported
- [ ] compression streams

---

## 13. Concurrency and processes

### 13.1 Child processes

- [ ] spawn process
- [ ] exec command
- [ ] stdin/stdout/stderr piping
- [ ] process exit status
- [ ] signal handling
- [ ] process cleanup

### 13.2 Workers / threading

- [ ] worker threads if supported
- [ ] message passing
- [ ] shared memory if supported
- [ ] atomics if supported

---

## 14. OS and system APIs

- [ ] platform detection
- [ ] architecture detection
- [ ] temp directory
- [ ] hostname
- [ ] uptime
- [ ] CPU info
- [ ] memory info
- [ ] user info
- [ ] environment inspection

---

## 15. Optional type system

### 15.1 Basic typing

- [ ] variable type annotations
- [ ] parameter type annotations
- [ ] return type annotations
- [ ] property type annotations
- [ ] local type inference

### 15.2 Core types

- [ ] `number`
- [ ] `string`
- [ ] `boolean`
- [ ] `bigint`
- [ ] `void`
- [ ] `null`
- [ ] `undefined`
- [ ] `any`
- [ ] `unknown`
- [ ] `never`
- [ ] object types
- [ ] array types
- [ ] tuple types

### 15.3 Structured types

- [ ] interfaces
- [ ] type aliases
- [ ] optional properties
- [ ] readonly properties
- [ ] function types
- [ ] callable types
- [ ] index signatures

### 15.4 Advanced types

- [ ] union types
- [ ] intersection types
- [ ] literal types
- [ ] discriminated unions
- [ ] generics
- [ ] generic constraints
- [ ] enums if supported

### 15.5 Type system policy

- [ ] optional typing only
- [ ] erased at compile time
- [ ] typed/untyped interop
- [ ] cast / assertion syntax
- [ ] runtime type checks if supported

---

## 16. Tooling

- [ ] CLI compile command
- [ ] CLI run command
- [ ] target selection (`--target`)
- [ ] output file selection
- [ ] emit LLVM IR
- [ ] emit native executable
- [ ] diagnostics formatting
- [ ] source maps if supported
- [ ] REPL if supported
- [ ] formatter if supported
- [ ] package init command if supported
- [ ] test runner if supported

---

## 17. Cross-platform support

- [ ] Linux x64
- [ ] Linux arm64
- [ ] macOS x64
- [ ] macOS arm64
- [ ] Windows x64
- [ ] correct target triple handling
- [ ] platform-specific runtime linkage
- [ ] path handling across OSes
- [ ] file permission behavior across OSes
- [ ] networking behavior across OSes

---

## 18. Testing coverage

- [ ] lexer tests
- [ ] parser tests
- [ ] AST tests
- [ ] semantic tests
- [ ] type-checking tests
- [ ] lifetime / escape tests
- [ ] codegen tests
- [ ] LLVM IR tests
- [ ] runtime tests
- [ ] filesystem tests
- [ ] network tests
- [ ] module resolution tests
- [ ] cross-platform tests
- [ ] e2e native executable tests
- [ ] regression tests for fixed bugs
