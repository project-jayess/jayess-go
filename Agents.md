# AGENTS.md — Jayess native programming language Compiler

You are an AI coding agent working inside the **Jayess programming language compiler** repository.

Your job is to make **small, correct, incremental** changes while preserving compiler correctness, language semantics, and repository structure.

Jayess is a **JavaScript-like native programming language** implemented in **Go**, with an **LLVM-based backend** and **automatic memory handling**. The language aims to give JavaScript-like ergonomics while providing more predictable scope-based lifetime behavior for non-escaping values.

---

## 0) Non-negotiable rules

### Language / files
- The compiler implementation language is **Go**.
- Prefer **Go** for compiler, analysis, backend, CLI, and runtime integration work unless the task explicitly requires another language already used by the repository.
- Do not introduce TypeScript or unrelated frontend stacks unless the task explicitly requires them.
- Follow the repository’s existing Go package structure, naming, and file layout.

### Compiler safety
- **Do not silently change language semantics** unless the task explicitly requires it.
- **Do not introduce breaking syntax changes** unless the task explicitly requires it.
- **Do not guess semantic behavior** when parser, analyzer, lifetime, or runtime behavior is unclear.
- Prefer small, isolated changes over broad refactors.

### Generated / derived files
- **Do not edit generated output** unless the task explicitly requires it.
- If a file is produced by code generation, tests, snapshots, fixture generation, LLVM IR emission, or packaging, treat it as generated unless the task clearly says otherwise.

### Code size and file maintainability

- Keep source files small, focused, and reviewable.
- Do **not** keep adding unrelated logic to already-large files.
- If a file is becoming too large, prefer extracting one clear responsibility into a focused helper file or package.
- Avoid creating or expanding files toward massive sizes such as thousands of lines unless there is a strong architectural reason.
- Files over roughly **1,000 lines** should be treated as refactor candidates.
- Files over roughly **2,000 lines** should not receive more unrelated logic unless the task explicitly requires it.
- Files over roughly **10,000 lines** are considered unhealthy and should be gradually split through safe, behavior-preserving refactors.

When reducing file size:

- extract only one responsibility at a time
- preserve behavior exactly
- avoid broad renaming
- avoid formatting unrelated code
- keep public APIs stable unless the task requires a change
- add or preserve focused tests
- run relevant tests before committing

Do **not** perform huge cleanup refactors in one step. Prefer small extraction commits.

### File organization rule (STRICT)

Compiler code and documentation must be organized into small, focused files.

Do NOT implement a large compiler feature by placing all logic into one huge file.
Do NOT create “god files” that mix parsing, semantic analysis, lowering, LLVM emission, runtime binding, CLI handling, and tests.

Each new file should have one clear responsibility.

Good examples:

- `parser/functions.go`
- `parser/classes.go`
- `semantic/scope.go`
- `semantic/imports.go`
- `lowering/functions.go`
- `lowering/classes.go`
- `codegen/emit_function.go`
- `codegen/emit_class.go`
- `backend/linker.go`
- `backend/target.go`

Bad examples:

- `compiler.go` containing lexer, parser, semantic analysis, lowering, and LLVM emission
- `utils.go` containing unrelated compiler logic
- `runtime_helpers.go` growing into thousands of lines of mixed behavior
- `big_feature.go` containing every part of a feature across all compiler stages

When adding a feature that touches multiple compiler stages, split the work by layer:

1. syntax/parser changes
2. AST changes
3. semantic analysis changes
4. lifetime/escape analysis changes
5. lowering changes
6. LLVM/codegen changes
7. runtime changes
8. tests

Each layer should be implemented in the package/file that owns that responsibility.

Do not bypass architecture by putting cross-stage logic into a shared helper file.
Shared helpers are allowed only when the helper is genuinely reusable and has a narrow purpose.

### Refactoring discipline (STRICT)

Refactoring in this repository must be **safe, incremental, and behavior-preserving**.

This is a compiler and runtime project — careless refactoring can silently break semantics, lifetime behavior, or code generation.

#### Core rule

Refactoring must **NOT change behavior** unless the task explicitly requires it.

#### Allowed refactoring scope

When refactoring, agents must:

- choose **one target file or one subsystem**
- identify **one clear responsibility to improve**
- perform **one focused change**
- verify behavior before moving to the next step

Do NOT refactor multiple subsystems at once.

#### Safe refactoring steps

When modifying a large or problematic file:

1. identify a **cohesive responsibility** (e.g. function call dispatch, environment handling, IR emission)
2. extract it into:
   - a helper function, OR
   - a new file/package (if large enough)
3. keep original function as a thin wrapper (temporarily if needed)
4. verify:
   - compiler builds
   - tests pass
   - behavior is unchanged
5. only then proceed to the next extraction

#### Behavior preservation checklist

After refactoring, verify:

- parsing behavior is unchanged
- semantic resolution is unchanged
- diagnostics are unchanged
- lifetime / escape behavior is unchanged
- generated code is unchanged (or intentionally improved)
- runtime behavior is unchanged

If any of these change unintentionally → rollback or fix.

#### Runtime-specific refactoring rules

Runtime code must NOT be refactored into:

- repeated per-arity functions (`call_one`, `call_two`, …)
- large copy-paste dispatch blocks
- hardcoded limits (e.g. max 13 arguments)

Instead:

- prefer **generic representations** (e.g. `argc` / `argv`, call frame structs)
- replace duplication with loops or shared helpers
- ensure support for dynamic behavior (closures, `this`, `.bind`, spread/rest)


#### Large file refactoring strategy

For files larger than ~2000 lines:

- do NOT rewrite the file in one pass
- split across multiple commits:
  - extract helper logic
  - isolate data structures
  - separate concerns into files
- keep each commit:
  - small
  - reviewable
  - buildable

For files larger than ~10,000 lines:

- prioritize breaking them into multiple files
- do NOT continue adding new logic to them

#### Forbidden refactoring behavior

- Do NOT rewrite entire subsystems “for cleanliness”
- Do NOT mix refactoring with feature changes
- Do NOT rename large sets of identifiers without necessity
- Do NOT change public/runtime ABI unless required
- Do NOT silently change semantics while refactoring

#### When refactoring is required

Refactoring SHOULD be performed when:

- file size becomes unmanageable
- logic is duplicated
- scalability issues exist (e.g. fixed-arity dispatch)
- responsibilities are unclear or mixed
- runtime/compiler boundaries are violated

#### Commit discipline for refactoring

Refactoring commits should:

- be clearly labeled (e.g. `refactor: extract call dispatcher`)
- describe:
  - what was moved
  - why it was moved
  - confirmation that behavior is preserved
- avoid mixing with unrelated changes

### No giant implementation dumps

Agents must not solve tasks by pasting a huge, monolithic implementation into a single file.

If a feature requires substantial code, split it into small, reviewable pieces by compiler stage and responsibility.

A valid implementation should make it easy for a reviewer to answer:

- What layer owns this code?
- What responsibility does this file have?
- What behavior changed?
- Which tests prove it?

---

## 1) Project intent

Jayess is a **JavaScript-like native compiled language** with:
- native programming language
- lexical scopes
- class, functions and block-based scope behavior
- automatic memory handling
- scope-based cleanup for non-escaping values
- retention/promotion of escaping values when required
- LLVM-based code generation
- npm-based package management and import resolution
- not copy of JavaScript's runtime model

Jayess aims to be **syntactically compatible with JavaScript**, meaning most Jayess code resembles JavaScript.

However:
- Jayess is a **separate compiled language**, not a JavaScript runtime.
- Not all valid JavaScript code is valid Jayess code.
- Jayess may enforce stricter and more predictable semantics than JavaScript.

It should preserve familiar syntax and developer ergonomics where useful, while allowing stricter and more predictable behavior than JavaScript.

Agents must protect that design goal.

---

## 1.1 Build output and platform targets

Jayess is a **native compiled language**.

### Primary output
- The compiler must produce **native executables**.
- Output should be placed under:
  - `./build/` (or repo-defined output directory)

Examples:
- Linux: `build/linux/<name>`
- macOS: `build/mac/<name>`
- Windows: `build/windows/<name>.exe`

### Supported platforms
The compiler must support building for:

- Linux (x64, arm64 if supported)
- macOS (x64, arm64)
- Windows (x64 at minimum)

### Cross-platform requirement
- Do **not assume host-only compilation**.
- Codegen, runtime, and linking must work across supported platforms.
- Avoid hardcoding OS-specific behavior unless properly isolated.

### Target configuration
The compiler should support specifying a target:

Examples:
- `--target=linux-x64`
- `--target=darwin-arm64`
- `--target=windows-x64`

If target is not specified:
- default to host platform

### Runtime linkage
- Generated executables must include or link against the Jayess runtime.
- Runtime behavior must be consistent across all supported platforms.
- Do not introduce platform-specific behavior differences unless explicitly required.

### LLVM usage constraints
- LLVM backend must generate correct output for all supported targets.
- Do not use LLVM features that only work on a single platform unless guarded.
- Ensure correct target triple handling.

### Forbidden behavior
- Do NOT generate binaries that only run on the developer's machine.
- Do NOT assume Linux-only paths, syscalls, or ABI.
- Do NOT embed platform-specific hacks in frontend or semantic layers.
- Do NOT bypass cross-platform issues by disabling features.

---

## 1.2 Package management and import resolution

Jayess produces **native executables**, but uses the **npm ecosystem** for package management and dependency resolution.

### Package manager
- Jayess uses **npm** as the default package manager.
- Project dependencies are declared in `package.json`.
- Installed packages are resolved from `node_modules/`.
- Do **not** introduce a separate Jayess-specific package manager unless explicitly required.

### Import resolution
- The compiler must support resolving imports from:
  - relative paths
  - local project files
  - npm packages installed in `node_modules`
- Imports should follow the repository’s defined Jayess resolution rules, while remaining compatible with npm-based dependency layout where applicable.

Examples:
- `import "./utils.js"`
- `import "../lib/math.js"`
- `import "lodash"`
- `import "@scope/pkg"`

### Native binding resolution
- Native C/C++ interop should use **manual `*.bind.js` files** as the binding model.
- Do **not** treat direct `.c` / `.cc` / `.cpp` imports as the intended long-term language model.
- Do **not** introduce new manifest-based native binding formats when `*.bind.js` can express the same information.
- A `*.bind.js` file is expected to declare:
  - `sources`
  - `includeDirs`
  - `cflags`
  - `ldflags`
  - `exports`
- Jayess source should import native bindings through the binding file, for example:
  - `import { add } from "./native/math.bind.js"`
- `*.bind.js` may also include editor-friendly placeholder exports such as:
  - `const f = () => {}; export const add = f;`
  while `export default { ... }` remains the compiler-facing source of truth.
- Native implementation code should use the low-level runtime boundary header:
  - `#include "jayess_runtime.h"`
- Preserve the distinction between:
  - Jayess modules
  - manual native binding modules
  - native implementation source files

#### *.bind.js example:
```
const f = () => {};
export const add = f;
export default {
  sources: ["./src/mylib.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};

```

### package.json awareness
- Compiler and tooling may use `package.json` for:
  - dependency discovery
  - package entry resolution
  - project metadata
  - build configuration if the repo defines such behavior
- Do **not** ignore `package.json` when implementing package resolution.

### Node ecosystem compatibility
- Jayess is **not** the Node.js runtime.
- npm is used for dependency management, not to imply that compiled Jayess programs run on Node.js.
- Do **not** assume Node-only runtime APIs are automatically available in Jayess.

### Resolution safety rules
- Do **not** hardcode Linux-only or machine-specific `node_modules` paths.
- Do **not** assume all npm packages are valid Jayess packages unless the repo explicitly supports JS interop or foreign package handling.
- Do **not** silently reinterpret unsupported JavaScript packages as valid Jayess source packages.

### If JS interop exists
If the repository defines JavaScript interoperability:
- keep that behavior explicit and isolated
- do not blur the boundary between:
  - Jayess source packages
  - JS libraries
  - runtime bindings
- preserve predictable diagnostics when a package is installed through npm but is not directly compilable by Jayess

### Forbidden behavior
- Do NOT invent a second package lock or parallel package registry unless the task explicitly requires it.
- Do NOT break npm-style scoped packages.
- Do NOT bypass package resolution rules by flattening everything into ad hoc local paths.

---

## 2) Core design principles to preserve

### Language philosophy
- Jayess should feel familiar to JavaScript users.
- Jayess may intentionally be stricter and more predictable than JavaScript.
- Prefer predictable semantics over clever or surprising behavior.
- Avoid adding legacy-JavaScript-style quirks unless explicitly required.

### Memory philosophy
- Programmers should **not manually allocate or free memory**.
- Non-escaping values should be eligible for cleanup at scope end.
- Escaping values must remain valid beyond the defining scope.
- Compiler/runtime changes must preserve correctness of value lifetime.

### Compiler philosophy
- Keep frontend, semantic analysis, code generation, and runtime responsibilities clearly separated.
- Avoid mixing parser logic, semantic analysis, and LLVM lowering in the same place unless the repository already does so.
- Prefer explicit compiler stages and understandable data flow.

---

## 3) Folder ownership boundaries (STRICT)

> Exact folder names may differ by repo state. Follow the actual repository layout first.  
> The ownership model below describes the intended separation of responsibilities.

### Frontend / lexing / parsing
Typical areas:
- `./lexer/**`
- `./parser/**`
- `./ast/**`
- `./syntax/**`
- `./internal/lexer/**`
- `./internal/parser/**`

Typical work here:
- token definitions
- lexing rules
- parser behavior
- AST node definitions
- syntax-only validation
- syntax error reporting

Rules:
- Do **not** put semantic/lifetime/codegen logic into lexer code.
- Do **not** put LLVM generation directly into parser code unless the architecture already does so.

### Semantic analysis / symbol resolution / type or lifetime analysis
Typical areas:
- `./semantic/**`
- `./analyzer/**`
- `./scope/**`
- `./types/**`
- `./lifetime/**`
- `./escape/**`
- `./internal/semantic/**`

Typical work here:
- symbol binding
- scope resolution
- declaration validation
- name lookup
- type checking / type inference
- escape analysis
- lifetime analysis
- semantic diagnostics

Rules:
- Do **not** mix raw parsing concerns into semantic passes.
- Do **not** patch semantic issues in codegen if the real fix belongs in analysis.

### IR / lowering / LLVM backend
Typical areas:
- `./ir/**`
- `./lowering/**`
- `./codegen/**`
- `./llvm/**`
- `./backend/**`
- `./internal/codegen/**`

Typical work here:
- lowering AST/semantic structures into compiler IR
- LLVM IR generation
- value representation decisions
- function emission
- control flow lowering
- allocation/lifetime lowering
- calling convention handling

Rules:
- Do **not** change syntax or semantic rules here unless the task explicitly requires it.
- Do **not** bury frontend fixes in LLVM special cases unless unavoidable and documented.

### Runtime / support library
Typical areas:
- `./runtime/**`
- `./stdlib/**`
- `./support/**`

Typical work here:
- runtime helpers
- managed values
- strings/arrays/objects support
- closure environment support
- escaping value support
- standard support code required by compiled programs

Rules:
- Runtime must support compiler semantics, not redefine them ad hoc.
- Do **not** hide compiler bugs with runtime workarounds unless explicitly documented.

### CLI / tooling / developer UX
Typical areas:
- `./cmd/**`
- `./cli/**`
- `./tools/**`
- `./scripts/**`

Typical work here:
- compiler CLI commands
- input/output file handling
- diagnostics formatting
- build/test helpers
- debug tooling

Rules:
- Keep CLI concerns separate from core compiler logic.
- Do not bury important compiler behavior in CLI-only code.

### Tests / fixtures
Typical areas:
- `./test/**`
- `./tests/**`
- `./fixtures/**`
- `./testdata/**`

Typical work here:
- lexer/parser tests
- semantic tests
- escape/lifetime tests
- codegen tests
- runtime behavior tests
- regression fixtures

Rules:
- Add or update tests whenever behavior changes.
- Do not mass-rewrite fixtures without clear reason.
- Preserve focused regression coverage.

### Reference projects and external examples

- `./refs/**` may contain cloned external projects, examples, notes, or reference implementations.
- Treat `./refs/**` as **read-only reference material** unless the task explicitly says to update it.
- Use files under `./refs/**` only to understand design ideas, APIs, package support, native bindings, runtime behavior, or implementation examples.
- Do **not** copy large sections of external code into Jayess.
- Do **not** modify cloned reference projects as part of normal compiler work.
- Do **not** treat `./refs/**` as production Jayess source code.
- Do **not** include `./refs/**` in compiler builds, generated output, tests, or packaging unless explicitly required.

If using `./refs/**` for guidance:

- summarize the idea in Jayess terms
- implement the smallest Jayess-specific version needed
- respect licenses of referenced projects
- keep copied code out unless license and task scope clearly allow it

### Target triple handling
- Always use explicit LLVM target triples when generating code.
- Do not rely on implicit host defaults for cross-compilation.

---

## 4) Forbidden cross-boundary behavior

- Do **not** implement parser behavior inside runtime files.
- Do **not** implement semantic rules inside CLI-only code.
- Do **not** fix analyzer bugs by inserting arbitrary backend special cases unless absolutely necessary.
- Do **not** hide lifetime or escape issues by “just keeping everything alive forever.”
- Do **not** bypass repository structure by placing new compiler phases in random utility files.

If a task crosses multiple boundaries, keep each responsibility in its proper layer.

---

## 5) Language semantics rules for agents

When working on Jayess, assume these principles unless the repo explicitly defines otherwise.

### Scope and lifetime
- Jayess uses lexical scope.
- A local binding normally belongs to its defining scope.
- Values that do **not escape** a scope may be cleaned up when that scope ends.
- Values that **escape** must remain valid after the scope exits.

### Escape behavior
A value is likely escaping if it is:
- returned
- captured by a closure
- stored into a longer-lived structure
- assigned into global or module state
- otherwise retained beyond the local scope

Do not implement scope-end cleanup rules that break escaping values.

### Globals
- Globals or module-level bindings may outlive local scopes.
- Do not accidentally apply local cleanup rules to global state.

### Closures
- Closures may require captured values to outlive the declaring scope.
- Do not assume captured locals can remain purely scope-local values.

### Semantics stability
If a task is about implementation details, preserve:
- existing syntax
- existing AST shape where practical
- existing diagnostic meaning
- existing tests unless intentionally changing behavior

## 5.1 Variable declaration rules

Jayess simplifies JavaScript variable declarations.

### Rule
- Variable lifetime must align with lexical scope boundaries.
- Scope exit may trigger cleanup for non-escaping values.
- Jayess intentionally redefines `var` to behave as a modern block-scoped variable.

### Supported keywords
- `var` — mutable variable
- `const` — immutable binding

### Removed JavaScript behavior
- `let` is **not supported**
- JavaScript-style `var` semantics are **not supported**

### Jayess `var` semantics
- `var` is **block-scoped**, not function-scoped
- `var` behaves similarly to JavaScript `let`
- `var` does **not** use JavaScript-style hoisting
- variables must not be used before declaration (unless explicitly allowed by the language spec)

Example:
```js
if (true) {
  var x = 10;
}
console.log(x); // error (block scoped)
```
### Jayess `const` semantics
- `const` is block-scoped
- `const` must be initialized at declaration
- reassignment is not allowed

Example:
```js
const x = 10;
x = 20; // error
```
### Shadowing
- Inner scope variables may shadow outer scope variables
- Shadowing rules must be handled in semantic analysis

### Forbidden behavior
- Do NOT implement JavaScript-style function-scoped var
- Do NOT implement JavaScript hoisting behavior for var
- Do NOT reintroduce let unless explicitly required by the task


---

## 6) Go-specific engineering expectations

### Package discipline
- Keep packages focused and cohesive.
- Do not create circular dependencies.
- Prefer clear package ownership over convenience imports.
- Do not dump unrelated compiler logic into shared `util` packages.

### Errors and diagnostics
- Preserve structured diagnostics where the repo already has them.
- Do not replace helpful compiler diagnostics with generic `panic` or vague errors.
- Use `panic` only for truly impossible internal states, not normal user-facing compile errors.

### Public vs internal APIs
- Keep internal compiler details internal when possible.
- Avoid exposing unstable compiler internals without reason.
- Respect existing `internal/` boundaries.

### Formatting and style
- Keep code gofmt-friendly.
- Preserve existing naming conventions and package style.
- Prefer simple, readable Go over clever abstractions.

---

## 7) Workflow expectations

### Before coding
- Identify the correct boundary for the change.
- Read the smallest relevant set of files first.
- Check whether the task is:
  - lexing/parsing
  - AST
  - semantic analysis
  - lifetime/escape analysis
  - lowering/codegen
  - runtime
  - CLI/tooling
  - tests

### While coding
- Make the minimum necessary change.
- Preserve existing patterns and naming where reasonable.
- Add comments only where they prevent future confusion or mistakes.
- Avoid broad refactors unless explicitly required.
- Keep compiler stages understandable and separate.

### File growth discipline

Before adding significant code to an existing file, check whether the file is already large or mixed-responsibility.

If the change would add more than roughly 150-250 lines to one file, consider whether the logic should be split into:

- a focused helper function
- a new focused file in the same package
- a small internal package, only if package boundaries are clear

Do not create a new package just to hide messy code.
Do not create generic dumping-ground files such as:

- `helpers.go`
- `common.go`
- `misc.go`
- `utils.go`

unless the existing repository already uses that pattern and the new code clearly belongs there.

Prefer names that explain compiler responsibility:

- `resolve_imports.go`
- `check_classes.go`
- `lower_calls.go`
- `emit_closures.go`
- `runtime_bindings.go`

### After coding
- Ensure the compiler still builds.
- Ensure changed behavior is covered by tests where practical.
- Check whether the change affects:
  - parsing
  - name resolution
  - diagnostics
  - lifetime/escape handling
  - backend/codegen correctness
  - runtime behavior

If any of those changed, add or update focused tests.

---

## 8) Testing expectations

Prefer targeted validation.

Examples:
- Syntax change → lexer/parser/AST test
- Name resolution change → semantic/scope test
- Lifetime or escape change → dedicated regression test
- Codegen change → IR/backend/output test
- Runtime memory behavior change → runtime regression test

When fixing a bug:
- add a regression test if the repo has a test suite for that layer

Do not rely only on “it builds” as proof of correctness.

---

## 9) Performance and optimization rules

Jayess is a compiler project, so performance changes must be careful.

- Do not introduce correctness regressions for micro-optimizations.
- Do not assume an optimization is valid without checking semantics.
- Prefer correctness first, then optimize.
- Keep optimization logic isolated where possible.
- If changing allocation or lifetime behavior, verify escaping and closure cases.

### 9.1 Scalable design rules

code must prefer scalable designs over fixed, repetitive special cases.

Do **not** implement behavior by writing many near-identical functions that only differ by argument count, type count, or small numeric limits.

Avoid patterns such as:

- `call_one`, `call_two`, `call_three`, ... up to an arbitrary limit
- huge `if/else` or `switch` chains for every supported argument count
- duplicated function pointer typedefs for every arity
- logic that only works up to a hardcoded number such as 13 arguments
- copy-pasted code blocks where a loop, table, call frame, or shared helper would be clearer

If runtime behavior needs to support variable argument counts, dynamic calls, callbacks, `.call`, `.apply`, `.bind`, spread, or rest parameters, prefer a general call representation such as:

```c
typedef struct jayess_call_frame {
    jayess_value *this_value;
    jayess_value *env;
    size_t argc;
    jayess_value **argv;
} jayess_call_frame;

typedef jayess_value *(*jayess_callable)(jayess_call_frame *frame);
```

### 9.2 Agent logs and temporary files
- All AI agent logs, notes, and temporary files must be placed under `./dev-agent/`
- Do NOT create tracking, log, or analysis files outside of `./dev-agent/`.
- Track refactoring progress in `./dev-agent/refactoring.md`. Clear contents of `refactoring.md` before starting refactoring for a new file

---

## 10) “If you are unsure” rule

If there is ambiguity about:
- syntax meaning
- AST ownership
- type/lifetime rules
- escape behavior
- lowering expectations
- runtime responsibilities
- whether a file is generated

then:

- **Do not guess silently.**
- Make the safest incremental change possible.
- Document assumptions and risks in the repo’s review/task log if such a log exists.
- Prefer preserving current behavior over speculative redesign.

---

## 11) Output discipline for agents

- Do not create new subsystems unless the task requires them.
- Do not rename or move large groups of files without strong reason.
- Do not rewrite large parts of the compiler just to “clean things up” unless explicitly asked.
- Do not add dependencies casually.
- Keep diffs narrow, reviewable, and single-purpose.
- Preserve the separation between:
  - syntax
  - semantics
  - lifetime analysis
  - code generation
  - runtime

---

## 12) Reference architecture mindset

When reasoning about changes, think in this rough pipeline:

1. source text
2. tokens
3. AST
4. semantic/scope analysis
5. lifetime/escape analysis
6. lowering / intermediate representation
7. LLVM generation
8. runtime interaction / final output

Avoid collapsing these stages unless the repository explicitly does so.

---
