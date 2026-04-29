# Jayess feature checklist

## 1. Core language

### 1.1 Lexical / syntax

- [x] single-line comments (`//`)
- [x] block comments (`/* */`)
- [x] identifiers
- [x] keywords / reserved words
- [x] semicolon handling
- [x] operator precedence
- [x] grouping with parentheses
- [x] comma expressions if supported

### 1.2 Literals

- [x] number literals
- [x] string literals
- [x] boolean literals
- [x] `null`
- [x] `undefined` or Jayess equivalent
- [x] bigint literals
- [x] object literals
- [x] array literals
- [x] template strings

### 1.3 Variables and bindings

- [x] `var` declarations
- [x] `const` declarations
- [x] block scope
- [x] lexical scope
- [x] shadowing
- [x] declaration without hoisting
- [x] no use before declaration
- [x] destructuring declarations
- [x] default values in declarations

### 1.4 Operators

- [x] arithmetic operators
- [x] comparison operators
- [x] logical operators
- [x] assignment operators
- [x] bitwise operators
- [x] unary operators
- [x] ternary operator
- [x] optional chaining
- [x] nullish coalescing
- [x] `typeof`
- [x] `instanceof`

### 1.5 Control flow

- [x] `if`
- [x] `else`
- [x] `switch`
- [x] `for`
- [x] `while`
- [x] `do while`
- [x] `for...of`
- [x] `break`
- [x] `continue`
- [x] labeled statements if supported
- [x] `return`
- [x] `throw`
- [x] `try`
- [x] `catch`
- [x] `finally`

---

## 2. Functions

### 2.1 Basic functions

- [x] function declarations
- [x] function expressions
- [x] arrow functions
- [x] anonymous functions
- [x] nested functions
- [x] recursion
- [x] first-class functions

### 2.2 Parameters

- [x] positional parameters
- [x] default parameters
- [x] rest parameters
- [x] variadic calls

### 2.3 Function behavior

- [x] closures
- [x] captured variables
- [x] lexical `this` for arrow functions
- [x] normal `this` for regular functions
- [x] function return values
- [x] higher-order functions
- [x] callback support

### 2.4 Invocation helpers

- [x] method calls (`obj.fn()`)
- [x] function values in variables
- [x] function values in arrays
- [x] function values in objects
- [x] `.bind()` equivalent
- [x] `.call()` equivalent
- [x] `.apply()` equivalent

---

## 3. Objects and classes

### 3.1 Objects

- [x] object property read
- [x] object property write
- [x] computed property names
- [x] method definitions
- [x] property enumeration
- [x] object spread
- [x] object destructuring

### 3.2 Classes

- [x] `class`
- [x] constructors
- [x] instance methods
- [x] static methods
- [x] instance fields
- [x] static fields
- [x] getters
- [x] setters
- [x] `this`
- [x] `new`

### 3.3 Inheritance

- [x] `extends`
- [x] `super`
- [x] prototype chain or Jayess equivalent
- [x] method overriding
- [x] `instanceof` support

### 3.4 Encapsulation

- [x] private fields if supported
- [x] private methods if supported
- [x] visibility rules if supported

---

## 4. Arrays, strings, and built-in data structures

### 4.1 Arrays

- [x] array creation
- [x] indexing
- [x] mutation
- [x] length
- [x] iteration
- [x] array destructuring
- [x] rest elements
- [x] spread elements

### 4.2 Strings

- [x] string concatenation
- [x] string indexing
- [x] string length
- [x] template strings
- [x] unicode support

### 4.3 Built-in collections

- [x] `Map`
- [x] `Set`
- [x] `WeakMap` if supported
- [x] `WeakSet` if supported

### 4.4 Other built-ins

- [x] `Date`
- [x] `RegExp`
- [x] `Symbol`
- [x] `ArrayBuffer`
- [x] typed arrays
- [x] `DataView`

---

## 5. Iteration and generators

### 5.1 Iteration

- [x] iterable protocol
- [x] iterator protocol
- [x] `for...of`
- [x] custom iterables

### 5.2 Generators

- [x] generator functions
- [x] `yield`
- [x] generator iteration
- [x] async iterators if supported
- [x] async generators if supported

---

## 6. Async model

### 6.1 Promises

- [x] `Promise`
- [x] resolve / reject
- [x] chaining
- [x] error propagation
- [x] `Promise.all`
- [x] `Promise.race`
- [x] `Promise.allSettled` if supported
- [x] `Promise.any` if supported

### 6.2 Async functions

- [x] `async` functions
- [x] `await`
- [x] async error handling
- [x] async return values

### 6.3 Scheduling

- [x] event loop model
- [x] microtask queue if supported
- [x] timer queue
- [x] cancellation model if supported

---

## 7. Modules and package system

### 7.1 Module syntax

- [x] `import`
- [x] `export`
- [x] named exports
- [x] default exports
- [x] namespace imports
- [x] re-exports
- [x] `export *`
- [x] `export * as ns`

### 7.2 Resolution

- [x] relative imports
- [x] parent-directory imports
- [x] local project file imports
- [x] package imports from `node_modules`
- [x] scoped package imports
- [x] package entry resolution
- [x] `package.json` reading
- [x] module initialization order
- [x] circular imports
- [x] clear diagnostics for unsupported JS packages

---

## 8. Error handling and diagnostics

### 8.1 Compiler diagnostics

- [x] lexer errors
- [x] parser errors
- [x] semantic errors
- [x] type errors if types exist
- [x] module resolution errors
- [x] lifetime / escape diagnostics
- [x] source spans
- [x] helpful messages

### 8.2 Runtime errors

- [x] exceptions
- [x] stack traces
- [x] source locations in stack traces
- [x] uncaught exception handling

---

## 9. Memory / lifetime behavior (NO GC — scope-based)

This section defines **compiler and runtime correctness requirements**
for Jayess’s scope-based lifetime system.

Jayess does NOT use garbage collection.
Memory safety is enforced through **lexical lifetime + escape analysis**.

### 9.1 Lifetime model (semantic rules)

- [x] values are destroyed at lexical scope exit by default
- [x] lifetime is determined by lexical scope, not runtime reachability
- [x] variables are invalid outside their defining scope
- [x] globals/module-level values outlive all local scopes
- [x] closures extend lifetime of captured variables
- [x] function return values extend lifetime beyond function scope
- [x] lifetime rules are consistent across all language constructs

### 9.2 Escape analysis (compile-time)

The compiler must detect when a value escapes its defining scope.

- [x] returned values are treated as escaping
- [x] closure-captured variables are treated as escaping
- [x] values assigned to outer scopes are treated as escaping
- [x] values stored in objects are treated as escaping
- [x] values stored in arrays are treated as escaping
- [x] values passed to unknown/external functions are conservatively treated as escaping
- [x] escape detection is conservative (prefer false positive over unsoundness)

Classification:
- [x] non-escaping values → eligible for scope-based cleanup
- [x] escaping values → must NOT be destroyed at scope exit

### 9.3 Closure and environment handling

- [x] closures correctly capture variables, not copies (unless specified)
- [x] captured variables remain valid after outer scope exits
- [x] closure environments are heap-allocated or otherwise extended safely
- [x] multiple closures sharing variables behave correctly
- [x] mutation of captured variables is reflected across closures
- [x] no dangling references inside closures

### 9.4 Lowering and code generation

- [x] cleanup/destructor calls are inserted at scope exit for non-escaping values
- [x] escaping values are NOT cleaned up at scope exit
- [x] cleanup is emitted for all control-flow paths:
  - [x] normal block exit
  - [x] early return
  - [x] break / continue
  - [x] exception paths (if supported)
- [x] no cleanup is skipped due to control-flow complexity
- [x] no duplicate cleanup is generated

### 9.5 Runtime memory safety

### 9.5 Scope-based runtime memory safety

Jayess uses automatic scope-based memory management:
- programmers must not manually allocate/free Jayess values
- non-escaping values are cleaned up at scope exit
- escaping values are promoted/retained for the required lifetime
- containers, closures, globals, and native handles must preserve value validity

A memory-safety item may only be marked done when covered by focused tests.

#### 9.5.1 Ownership model

- [x] document ownership rules for every public `jayess_value *` runtime helper
- [x] distinguish owned, borrowed, copied, retained, and closed values
- [x] `jayess_value_from_*` returns runtime-owned values
- [x] borrowed string/bytes views are valid only during the current native call
- [x] copied string/bytes buffers have explicit free helpers

#### 9.5.2 Scope cleanup

- [x] non-escaping local values are cleaned up at lexical scope exit
- [x] cleanup runs on all scope exits: normal exit, return, break, continue, and error paths
- [x] returned values are not cleaned up before the caller receives them
- [x] globals and module-level values are not cleaned up as local scope values

#### 9.5.3 Escaping values

- [x] returned values remain valid for the required lifetime
- [x] values stored in objects/arrays remain valid
- [x] closure-captured values remain valid after the declaring scope exits
- [x] values assigned into globals/module state remain valid
- [x] values passed into longer-lived native handles remain valid only through copied/owned data

#### 9.5.4 Containers and closures

- [x] container references objects/arrays remain valid
- [x] closure environments remain valid
- [x] object/array insert operations retain or otherwise preserve stored values
- [ ] object/array replacement releases previous stored values safely
- [x] closure environment cleanup releases captured values safely

#### 9.5.5 Double-free and invalid use prevention

- [ ] no double-free is possible for Jayess-managed values
- [ ] no use-after-free is possible for Jayess-managed values
- [ ] freed/closed runtime values cannot be reused silently
- [x] invalid value usage reports a runtime error or compiler diagnostic
- [ ] pointer/reference validity is preserved across compiler, runtime, and native binding boundaries

#### 9.5.6 Native binding safety

- [x] native wrappers must not store borrowed Jayess pointers beyond the current call
- [x] native wrappers must copy strings/bytes for long-lived native state
- [x] managed native handles become invalid after close
- [x] repeated close on managed native handles is safe
- [x] using a closed managed native handle reports a runtime error
- [x] native finalizers run at most once

#### 9.5.7 Required regression tests

- [x] returning local object from function remains valid
- [x] returning local array from function remains valid
- [x] storing local value into object property remains valid
- [x] storing local value into array remains valid
- [x] closure capture remains valid after outer function returns
- [x] non-escaping temporary is cleaned up at scope exit
- [x] cleanup still runs on early return
- [x] cleanup still runs on break/continue
- [ ] object/array replacement does not leak previous value
- [x] double close of managed native handle is safe
- [x] use after managed native handle close reports an error

### 9.6 Containers and references

- [x] storing a value in object/array extends its lifetime
- [x] nested containers correctly propagate escape behavior
- [x] removing values from containers does not prematurely free if still referenced
- [x] shared references behave consistently
- [x] aliasing does not break lifetime guarantees

### 9.7 Validation and testing

- [x] scope-exit cleanup correctness is verified via tests
- [x] escape cases are covered by regression tests
- [x] closure lifetime behavior is tested
- [x] container escape behavior is tested
- [x] returned value lifetime is tested
- [x] cross-function lifetime behavior is tested
- [x] complex control-flow lifetime cases are tested
- [x] stress tests exist for nested scopes and closures

---

## 10. Standard library / runtime APIs

### 10.1 Process and environment

- [x] command-line args
- [x] environment variables
- [x] current working directory
- [x] exit codes
- [x] stdin
- [x] stdout
- [x] stderr
- [x] process info
- [x] high-resolution time
- [x] signals

### 10.2 Filesystem

- [x] read file
- [x] write file
- [x] append file
- [x] delete file
- [x] rename / move file
- [x] copy file
- [x] stat / metadata
- [x] file permissions
- [x] file exists check
- [x] create directory
- [x] recursive directory creation
- [x] remove directory
- [x] list directory
- [x] recursive directory listing
- [x] symlink support if supported
- [x] file watching if supported
- [x] file streams

### 10.3 Path utilities

- [x] path join
- [x] path resolve
- [x] path normalize
- [x] basename
- [x] dirname
- [x] extension extraction
- [x] relative path calculation

### 10.4 URL and query utilities

- [x] URL parsing
- [x] URL formatting
- [x] query string parse
- [x] query string stringify
- [x] percent encoding / decoding
- [x] file URL support if supported

### 10.5 Buffers and binary data

- [x] binary buffer type
- [x] string encoding / decoding
- [x] byte slicing
- [x] byte copying
- [x] endian-aware reads/writes
- [x] typed arrays
- [x] binary stream support

### 10.6 Streams

- [x] readable streams
- [x] writable streams
- [x] duplex streams
- [x] transform streams
- [x] piping
- [x] backpressure handling

---

## 11. Networking

### 11.1 HTTP

- [x] HTTP server
- [x] HTTP client
- [x] request object
- [x] response object
- [x] headers
- [x] status codes
- [x] request body reading
- [x] response body writing
- [x] streaming bodies
- [x] keep-alive
- [x] timeout handling

### 11.2 HTTPS

- [x] HTTPS server
- [x] HTTPS client
- [x] TLS certificate loading
- [x] private key loading
- [x] CA / trust configuration
- [x] certificate verification
- [x] secure defaults

### 11.3 TCP

- [x] TCP client sockets
- [x] TCP server sockets
- [x] connect
- [x] listen
- [x] accept
- [x] read/write data
- [x] close socket
- [x] socket errors
- [x] timeout support
- [x] backpressure handling

### 11.4 TLS

- [x] TLS client
- [x] TLS server
- [x] certificate handling
- [x] ALPN if supported
- [x] hostname verification

### 11.5 UDP

- [x] UDP sockets
- [x] send datagrams
- [x] receive datagrams
- [x] bind socket
- [x] multicast if supported
- [x] broadcast if supported

### 11.6 DNS

- [x] hostname lookup
- [x] reverse lookup
- [x] custom resolver support if desired
- [x] IP utilities

---

## 12. Crypto and compression

### 12.1 Crypto

- [x] random bytes
- [x] hashing
- [x] HMAC
- [x] symmetric encryption
- [x] asymmetric encryption if supported
- [x] digital signatures
- [x] key generation
- [x] secure compare

### 12.2 Compression

- [x] gzip
- [x] deflate
- [x] brotli if supported
- [x] compression streams

---

## 13. Concurrency and processes

### 13.1 Child processes

- [x] spawn process
- [x] exec command
- [x] stdin/stdout/stderr piping
- [x] process exit status
- [x] signal handling
- [x] process cleanup

### 13.2 Workers / threading

- [x] worker threads if supported
- [x] message passing
- [x] shared memory if supported
- [x] atomics if supported

---

## 14. OS and system APIs

- [x] platform detection
- [x] architecture detection
- [x] temp directory
- [x] hostname
- [x] uptime
- [x] CPU info
- [x] memory info
- [x] user info
- [x] environment inspection

---

## 15. Optional type system

### 15.1 Basic typing

- [x] variable type annotations
- [x] parameter type annotations
- [x] return type annotations
- [x] property type annotations
- [x] local type inference

### 15.2 Core types

- [x] `number`
- [x] `string`
- [x] `boolean`
- [x] `bigint`
- [x] `void`
- [x] `null`
- [x] `undefined`
- [x] `any`
- [x] `unknown`
- [x] `never`
- [x] object types
- [x] array types
- [x] tuple types

### 15.3 Structured types

- [x] interfaces
- [x] type aliases
- [x] optional properties
- [x] readonly properties
- [x] function types
- [x] callable types
- [x] index signatures

### 15.4 Advanced types

- [x] union types
- [x] intersection types
- [x] literal types
- [x] discriminated unions
- [x] generics
- [x] generic constraints
- [x] enums if supported

### 15.5 Type system policy

- [x] optional typing only
- [x] erased at compile time
- [x] typed/untyped interop
- [x] cast / assertion syntax
- [x] runtime type checks if supported

---

## 16. Tooling

- [x] CLI compile command
- [x] CLI run command
- [x] target selection (`--target`)
- [x] output file selection
- [x] emit LLVM IR
- [x] emit native executable
- [x] diagnostics formatting
- [x] package init command if supported
- [x] Showing bug or error detail with LLVM debug info (DWARF)
- [x] test runner if supported

---

## 17. Cross-platform support

- [x] Linux x64
- [x] Linux arm64
- [x] macOS x64
- [x] macOS arm64
- [x] Windows x64
- [x] correct target triple handling
- [x] platform-specific runtime linkage
- [x] path handling across OSes
- [x] file permission behavior across OSes
- [x] networking behavior across OSes

---

## 18. Testing coverage

- [x] lexer tests
- [x] parser tests
- [x] AST tests
- [x] semantic tests
- [x] type-checking tests
- [x] lifetime / escape tests
- [x] codegen tests
- [x] LLVM IR tests
- [x] runtime tests
- [x] filesystem tests
- [x] network tests
- [x] module resolution tests
- [x] cross-platform tests
- [x] e2e native executable tests
- [x] regression tests for fixed bugs

---

## 19. Manual C/C++ binding support

This section defines how Jayess interoperates with native code through
manually-authored `*.bind.js` binding files. Jayess should not depend on
auto-generated wrapping, manifest JSON, or direct native source imports as the long-term model.

> **Note: Example wrapper**
>
>     export default {
>       sources: ["./src/mylib.c"],
>       includeDirs: ["./include"],
>       cflags: [],
>       ldflags: [],
>       exports: {
>         add: { symbol: "mylib_add", type: "function" }
>       }
>     };
>
> **Note: Editor-friendly placeholder exports**
>
>     const f = () => {};
>     export const add = f;
>
>     export default {
>       sources: ["./src/mylib.c"],
>       includeDirs: ["./include"],
>       cflags: [],
>       ldflags: [],
>       exports: {
>         add: { symbol: "mylib_add", type: "function" }
>       }
>     };
>
> **Note: Example usage**
>
>     import { add } from "./mylib.bind.js";
>
>     function main(args) {
>       console.log(add(3, 4));
>       return 0;
>     }

---

### 19.1 Binding file model

- [x] import native bindings from `*.bind.js`
- [x] clear distinction between Jayess source modules and native binding modules
- [x] `*.bind.js` is the single supported manual binding format
- [x] diagnostics for unsupported or malformed binding targets
- [x] relative `*.bind.js` imports work predictably
- [x] package-local `*.bind.js` imports work predictably if supported

---

### 19.2 Binding schema and symbol mapping

- [x] `sources` field is supported
- [x] `includeDirs` field is supported
- [x] `cflags` field is supported
- [x] `ldflags` field is supported
- [x] `platforms` field is supported for target-specific source/include/flag overrides
- [x] `exports` field is supported
- [x] named placeholder JS exports can coexist with binding metadata
- [x] placeholder exports can be reused through shared stubs like `const f = () => {}; export const add = f;`
- [x] exported native symbol names resolve correctly
- [x] binding files can expose functions
- [x] binding files can expose values/variables if supported
- [x] malformed export declarations produce clear diagnostics

> **Note: Example C binding target**
>
>     #include "jayess_runtime.h"
>
>     jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
>       return jayess_value_from_number(
>         jayess_value_to_number(a) + jayess_value_to_number(b)
>       );
>     }

---

### 19.3 Type and value conversion

- [x] number ↔ native numeric types
- [x] string ↔ C string conversion
- [x] boolean ↔ native boolean conversion
- [x] null / undefined ↔ native null handling
- [x] object/array passing rules if supported
- [x] buffer / binary memory interop if supported
- [x] pointer/handle representation rules if supported

---

### 19.4 Memory and ownership rules

- [x] ownership of returned values is clearly defined
- [x] native bindings do not return invalid/dangling Jayess values
- [x] Jayess runtime values passed to native code remain valid for required lifetime
- [x] string/buffer allocation and freeing rules are documented
- [x] no double-free across Jayess/native boundary
- [x] no use-after-free across Jayess/native boundary

---

### 19.5 Error handling and safety

- [x] invalid binding imports produce clear diagnostics
- [x] native symbol resolution failures are reported clearly
- [x] native type mismatch errors are handled safely
- [x] binding-side exceptions/errors can propagate into Jayess diagnostics if supported
- [x] unsafe native operations are clearly restricted or documented

---

### 19.6 Build and compilation model

- [x] sources listed by `*.bind.js` are compiled during build
- [x] binding-listed native sources can be linked into emitted native executables
- [x] multiple `*.bind.js` files can be included in one build
- [x] platform-specific native compilation rules are supported
- [x] include path handling for runtime headers is supported
- [x] native build failures produce useful diagnostics
- [x] duplicate native source compilation across wrappers is handled safely
- [x] shared native helper/source ownership rules are documented

---

### 19.7 Low-level runtime header model

- [x] low-level native binding header is `jayess_runtime.h`
- [x] `jayess_runtime.h` exposes low-level runtime control directly
- [x] no separate high-level wrapper helper header is required
- [x] low-level native header usage is documented clearly

---

### 19.8 Native library and platform use cases

- [x] manual bindings can expose engine APIs to Jayess
- [x] manual bindings can expose platform APIs to Jayess
- [x] manual bindings can expose rendering/audio/input libraries
- [x] manual bindings can expose third-party C libraries safely
- [x] manual bindings can be used for performance-critical code paths

---

### 19.9 Testing coverage

- [x] regression tests for `*.bind.js` import resolution
- [x] regression tests for binding symbol resolution
- [x] tests for primitive value conversion
- [x] tests for string conversion
- [x] tests for error handling across boundary
- [x] e2e tests compiling Jayess + native bindings into executable
- [x] cross-platform tests for manual binding builds

---

## 20. Audio and media native bindings

This section tracks compiler/runtime support for exposing native audio libraries
through manual Jayess `*.bind.js` bindings. For an LLVM-native language, this is
primarily a native interop and build-model problem rather than a JavaScript-style
runtime API problem.

### 20.1 Audio binding model

- [x] compiler can import audio `*.bind.js` modules
- [x] audio bindings can be linked into native executables
- [x] audio APIs can be called from Jayess safely
- [x] audio callbacks can be bridged safely if supported

### 20.2 Playback and device access

- [x] enumerate output devices if supported
- [x] enumerate input devices if supported
- [x] open playback device
- [x] open capture device if supported
- [x] configure sample rate / channels / format
- [x] start / stop / pause playback
- [x] audio buffer submission
- [x] streaming playback

### 20.3 Asset and decoding support

- [x] load WAV if supported
- [x] load OGG if supported
- [x] load MP3 if supported
- [x] load FLAC if supported
- [x] raw PCM buffer support
- [x] decoded audio data can be exposed as Jayess buffers

### 20.4 Real-time audio behavior

- [x] low-latency playback path if supported
- [x] underrun / device-loss handling
- [x] thread-safe audio callback interaction if callbacks are supported
- [x] audio state can be synchronized with worker/thread model if needed

### 20.5 Audio library targets

- [x] SDL audio manual bindings if supported
- [x] OpenAL manual bindings if supported
- [x] miniaudio manual bindings if supported
- [x] PortAudio manual bindings if supported
- [x] platform-native audio backends can be bound manually if needed

### 20.6 Testing coverage

- [x] compile tests for audio binding imports
- [x] runtime tests for playback/capture surface if supported
- [x] e2e native executable tests for audio bindings
- [x] cross-platform tests for audio binding builds

---

## 21. GTK native GUI support

This section tracks Jayess support for native GTK application bindings through
manual `*.bind.js` modules. GTK support is naturally platform-limited and
will likely be strongest on Linux first.

### 21.1 GTK binding model

- [x] compiler can import GTK binding modules
- [x] GTK binding-listed native files can be compiled and linked
- [x] Jayess can call GTK APIs safely through bindings
- [x] GTK types/handles can be represented safely in Jayess

### 21.2 Application lifecycle

- [x] initialize GTK runtime
- [x] create application/window
- [x] enter main loop
- [x] quit main loop
- [x] clean shutdown without invalid handles

### 21.3 Widgets and layout

- [x] create labels
- [x] create buttons
- [x] create text inputs
- [x] create containers/layout widgets
- [x] set widget properties
- [x] add child widgets
- [x] show/hide widgets

### 21.4 Events and callbacks

- [x] connect signal handlers
- [x] button click handling
- [x] input/change events
- [x] window close events
- [x] callback lifetime/ownership is safe

### 21.5 Drawing and assets

- [x] image/widget asset loading if supported
- [x] custom drawing if supported
- [x] text rendering support through GTK stack

### 21.6 Platform/build model

- [x] pkg-config based GTK build discovery if supported
- [x] include/library path handling for GTK headers/libs
- [x] Linux-native GTK build support
- [x] macOS GTK build support if supported
- [x] Windows GTK build support if supported
- [x] useful diagnostics when GTK toolchain/deps are missing

### 21.7 Testing coverage

- [x] compile tests for GTK binding imports
- [x] smoke tests for window creation if supported
- [x] event/callback tests if supported
- [x] cross-platform GTK binding build tests

---

## 22. GLFW graphics/windowing support

This section tracks support for GLFW-backed native windowing/graphics bindings.
For Jayess as an LLVM-native language, GLFW is a good fit because it exposes a
portable C API and can pair with OpenGL/Vulkan/Metal abstractions through manual bindings.

### 22.1 GLFW binding model

- [x] compiler can import GLFW binding modules
- [x] GLFW bindings can be linked into native executables
- [x] GLFW handles can be represented safely in Jayess
- [x] binding lifecycle/ownership rules are defined

### 22.2 Window and context management

- [x] initialize GLFW
- [x] create window
- [x] destroy window
- [x] poll events
- [x] swap buffers
- [x] create OpenGL context if supported
- [x] Vulkan surface integration if supported

### 22.3 Input handling

- [x] keyboard input callbacks
- [x] mouse button callbacks
- [x] cursor position callbacks
- [x] scroll callbacks
- [x] gamepad/joystick input if supported

### 22.4 Rendering integration

- [x] OpenGL function access if supported
- [x] Vulkan integration if supported
- [x] timing/frame loop helpers
- [x] resize handling
- [x] fullscreen/windowed mode switching if supported

### 22.5 Asset and media integration

- [x] image loading can be paired with GLFW rendering path if supported
- [x] audio integration can coexist with GLFW app loop
- [x] worker/thread model can interoperate with render loop safely

### 22.6 Platform/build model

- [x] Linux GLFW build support
- [x] macOS GLFW build support
- [x] Windows GLFW build support
- [x] native link flags are handled correctly per platform
- [x] useful diagnostics when GLFW toolchain/deps are missing

### 22.7 Testing coverage

- [x] compile tests for GLFW binding imports
- [x] smoke tests for window/context creation if supported
- [x] input callback tests if supported
- [x] cross-platform GLFW binding build tests

---

## 23. Webview native app support

This section tracks support for embedding native webviews in Jayess-compiled
applications. For an LLVM-native language, this should be built through manual binding
modules and platform-native webview backends rather than browser-style runtime
emulation.

### 23.1 Webview binding model

- [x] compiler can import webview binding modules
- [x] webview binding-listed native files can be compiled and linked
- [x] webview handles can be represented safely in Jayess
- [x] binding lifecycle/ownership rules are defined

### 23.2 Window and host lifecycle

- [x] create webview window
- [x] destroy webview window
- [x] set window title
- [x] set window size
- [x] show/hide window if supported
- [x] enter webview event loop
- [x] clean shutdown without invalid handles

### 23.3 Content loading

- [x] load inline HTML
- [x] load local file content if supported
- [x] navigate to URL
- [x] serve local app content through embedded HTTP server if supported
- [x] inject JavaScript into the webview

### 23.4 Jayess to JavaScript bridge

- [x] expose Jayess functions to JavaScript
- [x] receive messages/events from JavaScript
- [x] pass strings/JSON safely across boundary
- [x] callback lifetime/ownership is safe
- [x] error propagation across bridge is defined

### 23.5 App integration

- [x] webview can coexist with native HTTP server support
- [x] webview can integrate with worker/thread model safely
- [x] webview can integrate with filesystem/path APIs
- [x] webview can integrate with GLFW/GTK host apps if desired

### 23.6 Platform/build model

- [x] Linux webview build support
- [x] macOS webview build support
- [x] Windows webview build support
- [x] native link flags are handled correctly per platform
- [x] useful diagnostics when webview toolchain/deps are missing

### 23.7 Testing coverage

- [x] compile tests for webview binding imports
- [x] smoke tests for window creation/navigation if supported
- [x] bridge callback tests if supported
- [x] cross-platform webview binding build tests

---

## 24. LLVM compiler support

This section tracks Jayess as an LLVM-native compiler, not just a language with a
native runtime. It covers the compiler/backend work needed to emit robust LLVM IR,
target multiple platforms, interoperate with LLVM tooling, and keep generated code
diagnosable and optimizable.

### 24.1 Core LLVM backend

- [x] Jayess lowers to LLVM IR
- [x] LLVM IR can be emitted directly
- [x] native executables can be built from LLVM IR output
- [x] object files can be emitted directly as a supported artifact
- [x] emitted bitcode if supported
- [x] static libraries can be emitted if supported
- [x] shared libraries can be emitted if supported

### 24.1a Shared library artifacts

- [x] CLI can emit shared libraries directly
- [x] Linux shared library output uses `.so`
- [x] macOS shared library output uses `.dylib`
- [x] Windows shared library output uses `.dll`
- [x] default shared library naming follows platform conventions
- [x] shared library emission is covered by tests

### 24.2 Target and code generation support

- [x] host target triple detection
- [x] explicit target triple selection
- [x] cross-target build configuration exists
- [x] target CPU selection if supported
- [x] target feature flags if supported
- [x] relocation model configuration if supported
- [x] code model configuration if supported

### 24.3 LLVM optimization pipeline

- [x] opt level selection (`O0` / `O1` / `O2` / `O3` / `Oz`) if supported
- [x] debug-friendly no-opt builds are supported
- [x] optimization pipeline is configurable from CLI/build surface
- [x] generated IR remains valid across optimization levels
- [x] optimization regressions are covered by tests

### 24.4 Debug information and diagnostics

- [x] DWARF or platform-native debug info emission if supported
- [x] source locations are preserved into emitted LLVM IR where supported
- [x] function names remain useful in generated debug output
- [x] native crash/debug workflows can map back to Jayess source reasonably
- [x] debug builds are covered by regression tests

### 24.5 LLVM toolchain interoperability

- [x] emitted IR can be verified with LLVM verifier tools if available
- [x] emitted IR works with `opt` if available
- [x] emitted objects work with `llc`/system linker flows if supported
- [x] ABI expectations are stable across LLVM/clang toolchain use

### 24.6 Runtime and backend correctness

- [x] generated code links correctly with the Jayess runtime
- [x] generated code links correctly with manual native bindings
- [x] calling convention choices are documented
- [x] LLVM backend behavior for exceptions/errors is documented
- [x] data layout assumptions are documented
- [x] backend invariants are protected by regression tests

### 24.7 Platform coverage

- [x] Linux LLVM-native executable builds
- [x] macOS LLVM-native executable builds
- [x] Windows LLVM-native executable builds
- [x] macOS target triples can emit cross-target LLVM-native object files
- [x] Windows target triples can emit cross-target LLVM-native object files
- [x] missing macOS executable SDK/sysroot boundary is diagnosed clearly
- [x] missing Windows executable SDK/runtime boundary is diagnosed clearly
- Current host-side blocker for macOS executable proof:
  Apple SDK/sysroot availability is still required for `darwin-*` targets.
- Current host-side blocker for Windows executable proof:
  Windows SDK plus C runtime headers/libs are still required for `windows-x64`.
- [x] cross-platform object/library emission is tested if supported
- [x] per-platform LLVM/linker quirks are documented

### 24.8 Testing coverage

- [x] LLVM IR tests
- [x] codegen tests
- [x] e2e native executable tests
- [x] explicit emitted-IR snapshot tests if supported
- [x] LLVM verifier/validation tests if supported
- [x] cross-target LLVM backend tests

---

## 25. SQLite native database support

This section tracks first-class SQLite support through manual `*.bind.js` bindings
and native source/library integration. The goal is a practical embedded database
surface for Jayess-compiled applications rather than a JS-only abstraction layer.

### 25.1 Binding and build model

- [x] compiler can import SQLite binding modules
- [x] SQLite binding-listed native files can be compiled and linked
- [x] SQLite library can be built from vendored source if present
- [x] SQLite handles can be represented safely in Jayess
- [x] binding lifecycle/ownership rules are defined

### 25.2 Core database API

- [x] open database
- [x] close database
- [x] execute SQL directly
- [x] prepare statement
- [x] finalize statement
- [x] reset statement
- [x] clear statement bindings

### 25.3 Value binding and row access

- [x] bind null
- [x] bind integer
- [x] bind float
- [x] bind string
- [x] bind blob
- [x] read columns by index
- [x] read columns by name if supported
- [x] row iteration helpers are provided

### 25.4 Transactions and pragmas

- [x] begin transaction
- [x] commit transaction
- [x] rollback transaction
- [x] useful pragma/configuration helpers if supported
- [x] busy timeout configuration if supported

### 25.5 Safety and correctness

- [x] statement lifetime is safe
- [x] database handle lifetime is safe
- [x] blob/string ownership across boundary is safe
- [x] SQLite errors propagate into Jayess diagnostics usefully

### 25.6 Testing coverage

- [x] compile tests for SQLite binding imports
- [x] smoke tests for create/query/update/delete
- [x] statement binding tests
- [x] transaction tests

---

## 26. libuv async runtime integration

This section tracks optional integration with `libuv` as a native event-loop and
async I/O backend. This is a systems/runtime feature, not just a thin library
binding.

### 26.1 Binding and build model

- [x] compiler can import libuv binding modules
- [x] libuv binding-listed native files can be compiled and linked
- [x] libuv can be built from vendored source if present
- [x] libuv loop/handle types can be represented safely in Jayess
- [x] binding lifecycle/ownership rules are defined

### 26.2 Event loop integration

- [x] create uv loop
- [x] run uv loop
- [x] stop uv loop
- [x] close uv loop safely
- [x] Jayess timers can coexist with libuv loop if supported
- [x] microtask/promise behavior with libuv loop is defined

### 26.3 Async I/O primitives

- [x] TCP integration if supported
- [x] UDP integration if supported
- [x] filesystem async operations if supported
- [x] polling/watcher integration if supported
- [x] process/spawn integration if supported
- [x] signal watcher integration if supported

### 26.4 Safety and threading

- [x] handle lifetime is safe
- [x] callback lifetime across boundary is safe
- [x] thread/loop ownership rules are documented
- [x] libuv errors propagate into Jayess diagnostics usefully

### 26.5 Testing coverage

- [x] compile tests for libuv binding imports
- [x] event loop smoke tests
- [x] timer/fs/network integration tests
- [x] process/signal integration tests if supported

---

## 27. OpenSSL native crypto/TLS integration

This section tracks explicit OpenSSL-backed capabilities beyond the current runtime
surface, including direct binding/library coverage and richer TLS/crypto features.

### 27.1 Binding and build model

- [x] compiler can import OpenSSL binding modules
- [x] OpenSSL binding-listed native files can be compiled and linked
- [x] vendored OpenSSL integration is supported if present
- [x] useful diagnostics when OpenSSL headers/libs are missing

### 27.2 Crypto primitives

- [x] hashing through OpenSSL bindings
- [x] HMAC through OpenSSL bindings
- [x] symmetric encryption through OpenSSL bindings
- [x] asymmetric encryption through OpenSSL bindings
- [x] digital signatures through OpenSSL bindings
- [x] key generation through OpenSSL bindings
- [x] random bytes through OpenSSL bindings

### 27.3 TLS integration

- [x] TLS client through OpenSSL bindings
- [x] TLS server through OpenSSL bindings
- [x] certificate loading
- [x] trust store configuration
- [x] hostname verification
- [x] ALPN if supported

### 27.4 Safety and correctness

- [x] key/certificate handle lifetime is safe
- [x] OpenSSL errors propagate into Jayess diagnostics usefully
- [x] version/feature differences are handled safely

### 27.5 Testing coverage

- [x] compile tests for OpenSSL binding imports
- [x] crypto smoke tests
- [x] TLS client/server smoke tests if supported
- [x] certificate/trust configuration tests

---

## 28. libcurl networking and transfer support

This section tracks `libcurl` as a native HTTP/HTTPS/file transfer backend available
through manual bindings and packaged runtime support.

### 28.1 Binding and build model

- [x] compiler can import libcurl binding modules
- [x] libcurl binding-listed native files can be compiled and linked
- [x] vendored libcurl integration is supported if present
- [x] curl easy handle types can be represented safely in Jayess
- [x] useful diagnostics when curl headers/libs are missing

### 28.2 Core transfer API

- [x] create easy handle
- [x] configure URL/method/headers/body
- [x] perform transfer
- [x] read status/headers/body
- [x] cleanup easy handle

### 28.3 Advanced transfer features

- [x] HTTPS support
- [x] redirect control
- [x] timeout control
- [x] upload support
- [x] download-to-file support if supported
- [x] cookie support if supported
- [x] proxy support if supported

### 28.4 Async/multi integration

- [x] multi-handle support if supported
- [x] integration with Jayess async model if supported
- [x] streaming body delivery if supported

### 28.5 Testing coverage

- [x] compile tests for libcurl binding imports
- [x] HTTP/HTTPS smoke tests
- [x] timeout/redirect tests
- [x] streaming or multi tests if supported

---

## 29. Mongoose embedded web server support

This section tracks Mongoose as a native embedded HTTP/WebSocket/server library
available to Jayess through manual bindings.

### 29.1 Binding and build model

- [x] compiler can import Mongoose binding modules
- [x] Mongoose binding-listed native files can be compiled and linked
- [x] Mongoose can be built from vendored source if present
- [x] Mongoose manager/connection handles can be represented safely in Jayess
- [x] useful diagnostics when Mongoose build deps are missing

### 29.2 HTTP server core

- [x] create server manager
- [x] bind/listen on host/port
- [x] accept HTTP requests
- [x] read method/path/query/headers/body
- [x] send status/headers/body responses
- [x] clean shutdown

### 29.3 Extended server features

- [x] static file serving if supported
- [x] route dispatch helpers if supported
- [x] chunked/streaming responses if supported
- [x] HTTPS serving if supported
- [x] WebSocket upgrade/handling if supported
- [x] embedded app/content serving for webview integration if supported

### 29.4 Event loop and callback integration

- [x] Mongoose event callbacks can invoke Jayess safely
- [x] callback lifetime/ownership is safe
- [x] server loop integration with Jayess runtime is defined
- [x] error propagation into Jayess diagnostics is useful

### 29.5 Testing coverage

- [x] compile tests for Mongoose binding imports
- [x] HTTP server smoke tests
- [x] routing/response tests
- [x] HTTPS/WebSocket tests if supported

---

## 30. picohttpparser low-level HTTP parsing

This section tracks `picohttpparser` as a low-level, performance-oriented HTTP parsing
dependency exposed through Jayess bindings and higher-level server packages.

### 30.1 Binding and build model

- [x] compiler can import picohttpparser binding modules
- [x] picohttpparser binding-listed native files can be compiled and linked
- [x] picohttpparser can be built from vendored source if present
- [x] useful diagnostics when picohttpparser build inputs are missing

### 30.2 Parsing surface

- [x] parse HTTP request line and headers
- [x] parse HTTP response line and headers
- [x] incremental parsing support if supported
- [x] chunked parsing helpers if supported
- [x] malformed input errors are surfaced cleanly

### 30.3 Integration

- [x] parser output can be converted into Jayess objects safely
- [x] parser can be used by higher-level HTTP/Mongoose packages
- [x] parser can be used in performance-critical paths

### 30.4 Testing coverage

- [x] compile tests for picohttpparser binding imports
- [x] request/response parse tests
- [x] malformed input tests
- [x] incremental parsing tests if supported

---

## 31. raylib native graphics/game support

This section tracks `raylib` as a native graphics/window/input/audio/game-dev
library exposed to Jayess through manual bindings.

### 31.1 Binding and build model

- [x] compiler can import raylib binding modules
- [x] raylib binding-listed native files can be compiled and linked
- [x] raylib can be built from vendored source if present
- [x] raylib handles/types can be represented safely in Jayess
- [x] useful diagnostics when raylib build deps are missing

### 31.2 Window and application lifecycle

- [x] initialize raylib
- [x] create window
- [x] set window title
- [x] set window size
- [x] detect window close requests
- [x] close window cleanly
- [x] frame/update loop helpers

### 31.3 Rendering API

- [x] begin drawing
- [x] end drawing
- [x] clear background
- [x] draw text
- [x] draw basic shapes
- [x] draw textures/images if supported
- [x] custom color/value types can be passed safely

### 31.4 Input and timing

- [x] keyboard input
- [x] mouse input
- [x] gamepad input if supported
- [x] timing/frame delta helpers
- [x] fullscreen/window mode switching if supported

### 31.5 Assets and media

- [x] load images if supported
- [x] load textures if supported
- [x] unload image/texture resources safely
- [x] audio playback integration if supported
- [x] file/path integration for game assets

### 31.6 Safety and integration

- [x] callback lifetime/ownership is safe if callbacks are supported
- [x] resource handle lifetime is safe
- [x] render loop can coexist with Jayess async/runtime model safely
- [x] errors propagate into Jayess diagnostics usefully

### 31.7 Testing coverage

- [x] compile tests for raylib binding imports
- [x] smoke tests for window creation if supported
- [x] render/input loop tests if supported
- [x] cross-platform raylib binding build tests

---

## 32. HTML, XML, and CSS parsing (built-in)

This section tracks Jayess-native parsing support for HTML, XML, and CSS.

Parsers are implemented directly in Jayess (or compiler/runtime code),
without relying on external libraries. The goal is predictable behavior,
full control, and easy integration with Jayess tooling and AST systems.

---

### 32.1 General parser design

- [x] parsers are implemented without external dependencies
- [x] tokenization (lexer) is clearly separated from parsing
- [x] AST structures are defined for HTML, XML, and CSS
- [x] parsers preserve source spans (file/line/column)
- [x] error handling is consistent with Jayess diagnostics
- [x] malformed input produces recoverable errors where possible
- [x] parsing performance is acceptable for large files
- [x] memory usage is predictable and safe (no leaks)

---

### 32.2 HTML parser

- [x] parse full HTML documents
- [x] parse HTML fragments
- [x] support standard HTML tag syntax
- [x] support self-closing tags
- [x] support nested elements
- [x] parse attributes (name/value)
- [x] support boolean attributes
- [x] parse text nodes
- [x] parse comments (`<!-- -->`)
- [x] handle common malformed HTML cases gracefully
- [x] maintain DOM-like tree structure
- [x] preserve node order
- [x] serialize AST back to HTML string

---

### 32.3 XML parser

- [x] parse XML documents
- [x] enforce strict well-formed rules
- [x] parse element names and nesting
- [x] parse attributes
- [x] parse text nodes
- [x] parse comments
- [x] parse processing instructions if supported
- [x] parse CDATA sections if supported
- [x] support XML namespaces if supported
- [x] detect and report syntax errors clearly
- [x] maintain tree structure
- [x] serialize AST back to XML string

---

### 32.4 CSS parser

- [x] parse CSS stylesheets
- [x] tokenize selectors
- [x] parse rule blocks
- [x] parse declarations (property/value)
- [x] parse values (numbers, strings, units)
- [x] parse comments (`/* */`)
- [x] parse at-rules (`@media`, `@import`) if supported
- [x] maintain rule order
- [x] build CSS AST structure
- [x] serialize AST back to CSS string

---

### 32.5 Selector and query support

- [x] simple selector matching for HTML nodes
- [x] tag selectors
- [x] id selectors (`#id`)
- [x] class selectors (`.class`)
- [x] attribute selectors if supported
- [x] descendant selectors
- [x] child selectors (`>`)
- [x] basic pseudo-class support if supported

---

### 32.6 AST and transformation support

- [x] create nodes programmatically
- [x] modify node attributes
- [x] add/remove nodes
- [x] replace nodes
- [x] traverse tree (DFS/BFS)
- [x] query nodes
- [x] clone nodes
- [x] transform trees safely without breaking structure

---

### 32.7 Serialization and formatting

- [x] serialize HTML/XML/CSS back to string
- [x] preserve structure correctness
- [x] optional pretty-print formatting
- [x] optional minification support
- [x] preserve or strip comments based on options

---

### 32.8 Integration with Jayess

- [x] parsers integrate with file system APIs
- [x] parsers integrate with module system if needed
- [x] AST nodes can be used in user programs
- [x] parser errors integrate with Jayess diagnostics
- [x] source spans align with compiler error reporting

---

### 32.9 Testing coverage

- [x] HTML parsing tests
- [x] XML parsing tests
- [x] large file parsing tests
- [x] CSS parsing tests
- [x] malformed input tests
- [x] serialization round-trip tests
- [x] source span correctness tests

## 33. Refactoring and maintainability

This section tracks safe refactoring of large or hard-to-maintain files.

Refactoring must improve structure without changing Jayess language behavior,
compiler output, runtime behavior, diagnostics, or public APIs unless explicitly required.

### Refactoring discipline

When refactoring a large file, agents must:

- choose one target file or package
- identify one responsibility to extract
- move only that responsibility
- preserve behavior exactly
- avoid broad renaming
- avoid formatting unrelated code
- avoid changing public APIs unless required
- avoid mixing refactoring with feature work
- run focused tests for the affected package
- document what was moved and why

---

### 33.1 Runtime refactoring

- [x] refactor `jayess_runtime.c`
- [x] refactor `jayess_runtime.h`
- [x] split public runtime type declarations into a dedicated header
- [x] split runtime value helpers if file is too large
- [x] split string/buffer helpers if file is too large
- [x] split object/array helpers if file is too large
- [x] split error/exception helpers if file is too large
- [x] split path/filesystem helpers if file is too large
- [x] split bigint/numeric helpers if file is too large
- [x] split typed-array/data-view helpers if file is too large
- [x] split crypto/encoding helpers if file is too large
- [x] split async scheduler/worker/process helpers if file is too large
- [x] split network/TLS/HTTP helpers if file is too large
- [x] split stream/event/compression helpers if file is too large

---

### 33.2 Compiler refactoring

- [x] refactor AST code
- [x] refactor lexer code
- [x] refactor parser code
- [x] refactor semantic code
- [x] refactor type system code
- [x] refactor lifetime/escape code
- [x] refactor lowering code
- [x] refactor IR code
- [x] refactor codegen code
- [x] refactor LLVM backend code
- [x] refactor target/platform code
- [x] refactor compiler orchestration code
- [x] refactor CLI/cmd code
- [x] refactor native binding build code

---

### 33.3 Verification after refactoring

After every refactor, agents must verify behavior is unchanged.

- [x] existing tests still pass
- [x] affected package tests pass
- [x] compiler still builds
- [x] generated LLVM IR is unchanged where behavior should be unchanged
- [x] native executable output is unchanged for existing fixtures
- [x] diagnostics remain unchanged unless intentionally improved
- [x] source spans remain correct
- [x] no public API was changed accidentally
- [x] no generated files were manually edited
- [x] no unrelated files were reformatted

---

### 33.4 Refactoring acceptance criteria

A refactor is only complete when:

- [x] target file line count is reduced or responsibility is clearer
- [x] extracted code has a clear single purpose
- [x] package boundaries remain clean
- [x] imports remain non-circular
- [x] tests pass
- [x] behavior is preserved
- [x] future changes are easier to make incrementally

---
