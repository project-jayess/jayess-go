# Jayess feature checklist

## 1. Core language

### 1.1 Lexical / syntax

- [x] single-line comments (`//`)
- [x] block comments (`/* */`)
- [x] identifiers
- [x] keywords / reserved words
- [x] hashbang comments at the start of `.js` files
- [x] reject unsupported `enum` and `const enum` declarations with a clear diagnostic
- [x] reject unsupported type aliases, interfaces, ambient/module/namespace declarations, abstract class/member modifiers, readonly/override/accessor/class access modifiers, class `implements` clauses with organized computed class member parsing, annotations including catch and destructuring binding annotations, variable/class/catch/destructuring binding definite assignment and const assertions, function/class/arrow return annotations with organized arrow unsupported probes with organized unsupported type tests, type predicate/assertion return annotations, optional parameter/property/variable/destructuring binding markers, function/class method overload declarations, parameter property modifiers, function/class/method/arrow generic type parameters, type-only module declarations/specifiers with organized import specifier parsing with organized export specifier parsing, import-equals/export-equals/export-as-namespace module syntax, type expression suffixes in statements and nested expressions, top-level/class/member/parameter decorators, JSX, and angle-bracket assertions with clear diagnostics and organized diagnostic, declaration probe, and modifier probe helpers
- [x] semicolon handling
- [x] empty statements
- [x] trailing commas in parameter and argument lists
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
- [x] object literals with organized method/accessor parsing helpers
- [x] array literals
- [x] template strings
- [x] reject unsupported regular expression literals with a clear diagnostic

### 1.3 Variables and bindings

- [x] `var` declarations
- [x] `const` declarations
- [x] reject unsupported `let` declarations with a clear diagnostic
- [x] reject unsupported `using` declarations with a clear diagnostic
- [x] reject unsupported `public` / top-level `private` with clear diagnostics
- [x] block scope
- [x] lexical scope
- [x] shadowing
- [x] declaration without hoisting
- [x] no use before declaration
- [x] reject reassignment to `const` bindings
- [x] destructuring declarations with organized binding-pattern parser files
- [x] array destructuring elisions with organized array binding parsing
- [x] default values in declarations

### 1.4 Operators

- [x] arithmetic operators
- [x] modulo operator
- [x] exponentiation operator
- [x] comparison operators
- [x] logical operators
- [x] assignment operators
- [x] bitwise compound assignment operators
- [x] bitwise operators
- [x] unary operators
- [x] unary plus operator
- [x] update operators (`++`, `--`)
- [x] line terminator handling before postfix update operators
- [x] ternary operator
- [x] optional chaining
- [x] nullish coalescing
- [x] `typeof`
- [x] `typeof` permits undeclared identifier operands
- [x] `void`
- [x] `delete`
- [x] reject deleting identifiers
- [x] reject deleting private class members
- [x] `in`
- [x] `instanceof`

### 1.5 Control flow

- [x] `if`
- [x] `else`
- [x] `switch`
- [x] reject duplicate direct declarations across switch clauses
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
- [x] destructuring catch bindings
- [x] `finally`
- [x] `debugger` statement
- [x] reject unsupported `with` statements with a clear diagnostic
- [x] preserve async/generator context in loop and switch expressions

---

## 2. Functions

### 2.1 Basic functions

- [x] function declarations
- [x] function expressions
- [x] reject line terminator after `async` in async function declarations/expressions with a clear diagnostic
- [x] arrow functions
- [x] reject line terminator before arrow `=>` with a clear diagnostic
- [x] reject line terminator after `async` in async arrow functions with a clear diagnostic
- [x] anonymous functions
- [x] nested functions
- [x] recursion
- [x] first-class functions

### 2.2 Parameters

- [x] positional parameters
- [x] default parameters
- [x] method context in parameter defaults
- [x] method context in destructuring parameter defaults
- [x] rest parameters
- [x] destructuring parameters
- [x] default values in destructuring parameters
- [x] reject duplicate parameter bindings
- [x] function-local `arguments` binding and organized closure capture construction
- [x] nested regular functions receive their own `arguments` binding
- [x] variadic calls

### 2.3 Function behavior

- [x] parse and analyze closures
- [x] parse and analyze captured variables
- [x] parse and analyze lexical `this` for arrow functions
- [x] preserve class field/static block context for arrow functions
- [x] normal `this` for regular functions
- [x] parse and analyze `this` expressions
- [x] parse and analyze function return values
- [x] higher-order functions
- [x] callback support

### 2.4 Invocation helpers

- [x] method calls (`obj.fn()`)
- [x] function values in variables
- [x] function values in arrays
- [x] function values in objects
- [x] parse and analyze `.bind()` invocation helper shapes
- [x] parse and analyze `.call()` invocation helper shapes
- [x] parse and analyze `.apply()` invocation helper shapes

---

## 3. Objects and classes

### 3.1 Objects

- [x] object property read
- [x] object property write
- [x] keyword member property names
- [x] computed property names
- [x] method definitions
- [x] async object methods with organized async modifier lookahead
- [x] reject line terminator after `async` in async object methods
- [x] generator object methods
- [x] getter and setter definitions
- [x] computed getter and setter definitions
- [x] reject rest parameters in setters
- [x] shorthand properties
- [x] keyword object property names
- [x] parse and analyze `for...in` property enumeration
- [x] object spread
- [x] object destructuring with organized object binding parsing
- [x] keyword object destructuring property names
- [x] computed object destructuring property names

### 3.2 Classes

- [x] `class`
- [x] constructors
- [x] empty class elements
- [x] instance methods
- [x] static methods
- [x] static initialization blocks with organized class modifier lookahead helpers
- [x] reject static class members named `prototype`
- [x] computed class member names
- [x] instance fields with organized class field parsing
- [x] static fields with organized class field parsing
- [x] `this` and private access in class field initializers
- [x] `super` in derived class static initialization blocks
- [x] getters with organized class accessor parsing
- [x] setters with organized class accessor parsing
- [x] keyword class member names
- [x] class members named `static`
- [x] async class methods
- [x] line terminator after `async` starts a class field, not an async method
- [x] generator class methods
- [x] private async/generator class methods
- [x] `this`
- [x] parse and analyze `new` operator with constructor arguments
- [x] parse and analyze `new` expressions
- [x] parse and analyze `new.target` expressions
- [x] reject optional chaining in `new` targets

### 3.3 Inheritance

- [x] `extends`
- [x] reject non-constructable local `extends` targets
- [x] `super`
- [x] parse and analyze prototype-chain method access shapes
- [x] parse and analyze method override shapes
- [x] parse and analyze `instanceof` support

### 3.4 Encapsulation

- [x] private fields if supported
- [x] private methods if supported
- [x] visibility rules if supported

---

## 4. Arrays, strings, and built-in data structures

### 4.1 Arrays

- [x] array creation
- [x] array literal elisions
- [x] indexing
- [x] parse and analyze array index mutation
- [x] parse and analyze array length member access
- [x] parse and analyze array iteration with `for...of`
- [x] array destructuring
- [x] array destructuring elisions
- [x] rest elements
- [x] spread elements

### 4.2 Strings

- [x] parse and analyze string concatenation
- [x] parse and analyze string indexing
- [x] parse and analyze string length member access
- [x] parse and analyze template string interpolations
- [x] tagged template calls
- [x] parse and analyze unicode strings and identifiers

### 4.3 Built-in collections

- [x] parse and analyze `Map` construction and method access
- [x] parse and analyze `Set` construction and method access
- [x] parse and analyze `WeakMap` construction and method access if supported
- [x] parse and analyze `WeakSet` construction and method access if supported

### 4.4 Other built-ins

- [x] parse and analyze `Date` construction and method access
- [x] parse and analyze `RegExp` construction and method access
- [x] parse and analyze `Object` construction and static helper access
- [x] parse and analyze `Symbol` calls
- [x] parse and analyze `ArrayBuffer` construction
- [x] parse and analyze typed array construction and indexing
- [x] parse and analyze `DataView` construction and method access

---

## 5. Iteration and generators

### 5.1 Iteration

- [x] parse and analyze iterable protocol object shapes
- [x] parse and analyze iterator protocol object shapes
- [x] parse and analyze `for...of` with organized for-each parser helpers
- [x] parse and analyze `for await...of`
- [x] parse and analyze `for...in` with organized for-each parser helpers
- [x] reject initializers in `for...of` / `for...in` binding heads
- [x] destructuring bindings in `for...of` / `for...in`
- [x] assignment targets in `for...of` / `for...in`
- [x] parse and analyze custom iterable object shapes

### 5.2 Generators

- [x] parse and analyze generator functions
- [x] parse and analyze `yield` expressions
- [x] bare `yield`
- [x] delegated `yield*`
- [x] parse and analyze generator result iteration shapes
- [x] parse and analyze async iterator consumption shapes if supported
- [x] parse and analyze async generator functions
- [x] parse and analyze async generator function expressions

---

## 6. Async model

### 6.1 Promises

- [x] parse and analyze `Promise` construction
- [x] parse and analyze resolve / reject
- [x] parse and analyze chaining
- [x] parse and analyze Promise error-propagation chains
- [x] parse and analyze `Promise.all`
- [x] parse and analyze `Promise.race`
- [x] parse and analyze `Promise.allSettled` if supported
- [x] parse and analyze `Promise.any` if supported

### 6.2 Async functions

- [x] parse and analyze `async` functions
- [x] parse and analyze `await` expressions
- [x] parse and analyze async try/catch/finally error-handling shapes
- [x] parse and analyze async return values

### 6.3 Scheduling

- [x] event loop model
- [x] parse and analyze microtask scheduling callbacks if supported
- [x] parse and analyze timer scheduling callbacks
- [x] parse and analyze timer cancellation calls if supported

---

## 7. Modules and package system

### 7.1 Module syntax

- [x] `import`
- [x] `export`
- [x] named exports
- [x] default exports
- [x] anonymous default function/class exports
- [x] async default function exports
- [x] async generator default function exports
- [x] namespace imports
- [x] re-exports
- [x] `default` in named import/export specifiers
- [x] string-literal import/export specifier names
- [x] keyword import/export specifier names
- [x] `import.meta` expressions
- [x] `export *`
- [x] `export * as ns`
- [x] string-literal and `default` namespace re-export aliases

### 7.2 Resolution

- [x] parse and analyze relative import specifiers
- [x] parse and analyze parent-directory import specifiers
- [x] parse and analyze local project file import specifiers
- [x] relative/local source file resolution helper
- [x] unified import resolution dispatch helper
- [x] unified import resolver rejects missing importer paths before dispatch
- [x] unified import resolver routes dot-prefixed malformed source specifiers to source diagnostics
- [x] AST module dependency source extraction helper
- [x] AST module dependency compaction helper
- [x] resolved AST module dependency helper
- [x] compact resolved AST module dependency helper
- [x] resolved module dependency compaction helper
- [x] compact resolved AST module dependency helper deduplicates by resolved path
- [x] resolved module dependencies feed module graph initialization order
- [x] compact resolved module dependencies feed module graph initialization order
- [x] parsed program dependencies feed module graph helper
- [x] compact parsed program dependencies feed module graph helper
- [x] parse and analyze package import specifiers from `node_modules`
- [x] resolve Jayess stdlib import specifiers before `node_modules`
- [x] detect native binding modules from `export default bind(...)`
- [x] extract binding manifests from `export default bind(...)`
- [x] resolved binding imports feed native build planning with organized build path helpers and organized module export binding lowering
- [x] parse and analyze scoped package import specifiers
- [x] reject malformed package and scoped package import specifiers, including empty and dot path segments, with clear diagnostics
- [x] package resolver rejects malformed package specifiers before filesystem lookup
- [x] package resolver rejects whitespace-padded package specifiers before filesystem lookup
- [x] package resolver rejects scheme-like package specifiers before filesystem lookup
- [x] source resolver rejects empty source specifiers before filesystem lookup
- [x] source resolver rejects whitespace-padded source specifiers before filesystem lookup
- [x] source resolver rejects dot-only relative source specifier forms before filesystem lookup
- [x] source resolver rejects malformed relative source path segments before filesystem lookup
- [x] source resolver rejects absolute and scheme-like source specifiers before filesystem lookup
- [x] resolver rejects query and fragment module specifiers before filesystem lookup
- [x] module and resolver paths reject backslash separators with clear diagnostics
- [x] package entry resolution helper
- [x] `package.json` reading for package entry metadata
- [x] package entry metadata rejects unsafe package-relative paths
- [x] package entry metadata rejects whitespace-only package-relative paths
- [x] package entry metadata rejects scheme-like package-relative paths
- [x] package entry metadata rejects query and fragment package-relative paths
- [x] node_modules package import resolution helper
- [x] module graph single import edge add helper
- [x] module graph single import edge remove helper
- [x] module graph all matching import edges remove helper
- [x] module graph clear module imports helper
- [x] module graph replace module imports helper
- [x] module graph remove module helper
- [x] module graph clone helper
- [x] module graph compact direct import helper
- [x] module graph module/dependency/dependent count helpers
- [x] module graph dependency/dependent count map helpers
- [x] module graph import edge count helper
- [x] module graph deterministic import edge listing helper
- [x] module graph construction from import edges helper
- [x] module graph construction from import map helper
- [x] module graph export to import map helper
- [x] module graph export to dependent map helper
- [x] module graph construction from dependent map helper
- [x] module graph export to transitive dependency map helper
- [x] module graph export to transitive dependent map helper
- [x] module graph transitive dependency/dependent count helpers
- [x] module graph direct dependency predicate helper
- [x] module graph dependency inspection helper
- [x] module graph dependent inspection helper
- [x] module graph dependency depth and depth map helpers
- [x] module graph dependency depth grouping helper
- [x] module graph dependency depth level listing helper
- [x] module graph dependency depth layer listing helper
- [x] module graph dependency depth layer width listing helper
- [x] module graph dependency depth width map helper
- [x] module graph exact dependency depth filter helper
- [x] module graph bounded dependency depth filter helper
- [x] module graph beyond dependency depth filter helper
- [x] module graph dependency depth range filter helper
- [x] module graph widest dependency depth helper
- [x] module graph deepest dependency modules helper
- [x] module graph longest dependency path helper
- [x] module graph longest dependency path map helper
- [x] module graph dependent depth and depth map helpers
- [x] module graph dependent depth grouping helper
- [x] module graph dependent depth level listing helper
- [x] module graph dependent depth layer listing helper
- [x] module graph dependent depth layer width listing helper
- [x] module graph dependent depth width map helper
- [x] module graph exact dependent depth filter helper
- [x] module graph bounded dependent depth filter helper
- [x] module graph beyond dependent depth filter helper
- [x] module graph dependent depth range filter helper
- [x] module graph widest dependent depth helper
- [x] module graph deepest dependent modules helper
- [x] module graph longest dependent path helper
- [x] module graph longest dependent path map helper
- [x] module graph transitive dependency predicate helper
- [x] module graph multi-entry transitive dependency predicate helper
- [x] module graph transitive dependency inspection helper
- [x] module graph multi-entry transitive dependency inspection helper
- [x] module graph multi-entry transitive dependency count helper
- [x] module graph transitive dependency set helper
- [x] module graph transitive dependency count map helper
- [x] module graph transitive dependency set map helper
- [x] module graph multi-entry transitive dependency set helper
- [x] module graph transitive dependent predicate helper
- [x] module graph transitive dependent inspection helper
- [x] module graph transitive dependent set helper
- [x] module graph transitive dependent count map helper
- [x] module graph transitive dependent set map helper
- [x] module graph multi-module transitive dependent inspection helper
- [x] module graph multi-module transitive dependent predicate helper
- [x] module graph multi-module transitive dependent count helper
- [x] module graph multi-module transitive dependent set helper
- [x] module graph deterministic module listing helper
- [x] module graph deterministic root module listing helper
- [x] module graph reachable initialization subgraph helper
- [x] module graph entry reachable subgraph helper
- [x] module graph entry reachable module listing helper
- [x] module graph entry reachable module count helper
- [x] module graph entry reachable module set helper
- [x] module graph entry reachable module predicate helper
- [x] module graph entry reachable module order helper
- [x] module graph reachable module map helper
- [x] module graph reachable module count map helper
- [x] module graph reachable module set map helper
- [x] module graph reachable module order map helper
- [x] module graph multi-entry reachable subgraph helper
- [x] module graph multi-entry reachable module listing helper
- [x] module graph multi-entry reachable module count helper
- [x] module graph multi-entry reachable module set helper
- [x] module graph multi-entry reachable module predicate helper
- [x] module graph multi-entry reachable module order helper
- [x] module graph shared initialization batch comparison helper
- [x] module graph shared initialization entry map helper
- [x] module graph shared initialization batch index map helper
- [x] module graph shared initialization batch width helper
- [x] module graph shared initialization batch extrema helper
- [x] module graph shared widest initialization batch helper
- [x] module graph shared narrowest initialization batch helper
- [x] module graph shared initialization batch summary map helper
- [x] module graph entry initialization batch listing helper
- [x] module graph entry same initialization batch helper
- [x] module graph entry initialization batch index map helper
- [x] module graph entry initialization batch count helper
- [x] module graph entry initialization batch width listing helper
- [x] module graph entry initialization batch width range helper
- [x] module graph entry widest initialization batch helper
- [x] module graph entry narrowest initialization batch helper
- [x] module graph multi-entry initialization batch listing helper
- [x] module graph multi-entry same initialization batch helper
- [x] module graph multi-entry initialization batch index map helper
- [x] module graph multi-entry initialization batch count helper
- [x] module graph multi-entry initialization batch width listing helper
- [x] module graph multi-entry initialization batch width range helper
- [x] module graph multi-entry widest initialization batch helper
- [x] module graph multi-entry narrowest initialization batch helper
- [x] module graph initialization batch listing helper
- [x] module graph full same initialization batch helper
- [x] module graph full initialization batch index map helper
- [x] module graph initialization batch map helper
- [x] module graph initialization batch count helper
- [x] module graph initialization batch count map helper
- [x] module graph initialization batch width listing helper
- [x] module graph initialization batch width map helper
- [x] module graph initialization batch width range helper
- [x] module graph widest initialization batch helper
- [x] module graph narrowest initialization batch helper
- [x] module graph widest initialization batch map helper
- [x] module graph narrowest initialization batch map helper
- [x] module graph deterministic leaf module listing helper
- [x] module graph root and leaf predicate helpers
- [x] module graph isolated module listing and predicate helpers
- [x] module initialization order helper
- [x] module graph shared initialization order comparison helper
- [x] module graph entry initialization order comparison helper
- [x] module graph entry reverse initialization order comparison helper
- [x] module graph entry initialization order position lookup helper
- [x] module graph entry initialization order position helper
- [x] module graph multi-entry initialization order helper
- [x] module graph multi-entry initialization order comparison helper
- [x] module graph multi-entry reverse initialization order comparison helper
- [x] module graph multi-entry initialization order position lookup helper
- [x] module graph multi-entry initialization order position helper
- [x] module graph full initialization order helper
- [x] module graph full initialization order comparison helper
- [x] module graph full reverse initialization order comparison helper
- [x] module graph full initialization order position lookup helper
- [x] module graph full initialization order position helper
- [x] module graph initialization order map helper
- [x] module graph initialization order position map helper
- [x] circular import detection helper
- [x] full module graph acyclic validation helper
- [x] module graph acyclic predicate helper
- [x] clear diagnostics for unsupported absolute/URL/scheme-like module specifiers
- [x] clear diagnostics for unsupported import attributes/assertions
- [x] clear diagnostics for unsupported dynamic import expressions

---

## 8. Error handling and diagnostics

### 8.1 Compiler diagnostics

- [x] lexer reports unexpected characters and illegal tokens
- [x] lexer reports unterminated block comments, strings, and templates
- [x] parser reports expected-token and statement-terminator errors
- [x] parser surfaces lexer diagnostics for illegal tokens
- [x] semantic errors
- [x] type errors if types exist
- [x] module diagnostics for unsupported package specifiers
- [x] lifetime / escape diagnostics
- [x] source spans for parser statement dispatch organization and parser and semantic diagnostics
- [x] parser diagnostics include source spans
- [x] semantic diagnostics include source spans
- [x] helpful semantic diagnostics for common mutability mistakes
- [x] duplicate binding diagnostics include destructured binding names

### 8.2 Runtime errors

- [x] exceptions
- [x] stack traces
- [x] source locations in stack traces
- [x] uncaught exception handling

### 8.3 Control-flow semantic validation

- [x] reject `return` outside functions
- [x] reject `break` outside loops and switches
- [x] reject `continue` outside loops
- [x] validate labeled `break` / `continue` targets with organized control-jump validation
- [x] reject duplicate `default` clauses in `switch`
- [x] reject optional chains as assignment/update targets with organized assignment-target validation
- [x] reject `await` outside async functions
- [x] reject `yield` outside generator functions
- [x] reject `this` outside methods
- [x] reject `super` outside derived class methods
- [x] reject non-constructable local `new` targets
- [x] reject non-constructable local `instanceof` targets with organized constructability analysis helpers
- [x] reject invalid private member access with organized private member analysis helpers
- [x] reject invalid private member assignment
- [x] reject duplicate constructors and private class members
- [x] reject direct `super()` calls outside constructors

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

- [x] cleanup/destructor calls are inserted at scope exit for non-escaping values with organized cleanup binding extraction and cleanup op emission helpers
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
- [x] object/array replacement releases previous stored values safely
- [x] closure environment cleanup releases captured values safely

#### 9.5.5 Double-free and invalid use prevention

- [x] no double-free is possible for Jayess-managed values
- [x] no use-after-free is possible for Jayess-managed values
- [x] freed/closed runtime values cannot be reused silently
- [x] invalid value usage reports a runtime error or compiler diagnostic
- [x] pointer/reference validity is preserved across compiler, runtime, and native binding boundaries

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
- [x] object/array replacement does not leak previous value
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

### 10.0 Built-in globals

- [x] semantic recognition for documented MVP globals (`console`, `print`, `sleep`, `readLine`, `readKey`)
- [x] semantic recognition for numeric globals (`NaN`, `Infinity`)
- [x] semantic recognition for numeric helper globals (`parseInt`, `parseFloat`, `isNaN`, `isFinite`)
- [x] semantic recognition for standard namespace globals (`Math`, `JSON`)
- [x] semantic recognition for standard error constructors (`Error`, `EvalError`, `RangeError`, `ReferenceError`, `SyntaxError`, `TypeError`, `URIError`, `AggregateError`)
- [x] semantic recognition for standard collection/date/regexp constructors (`Array`, `Date`, `RegExp`, `Map`, `Set`, `WeakMap`, `WeakSet`)
- [x] semantic recognition for standard binary-data constructors (`ArrayBuffer`, typed arrays, `DataView`) and `Symbol`
- [x] semantic recognition for documented async/timer globals (`Promise`, `setTimeout`, `clearTimeout`, `setInterval`, `clearInterval`, `queueMicrotask`)
- [x] semantic recognition for standard global utilities (`globalThis`, URI encode/decode helpers)
- [x] runtime implementation validation for documented MVP globals

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

### 10.7 OS and CLI runtime implementation

- [x] Go-side stdin/stdout/stderr stream services are implemented
- [x] Go-side process exit code state is implemented
- [x] Go-side child process spawn/exec helpers are implemented
- [x] Go-side filesystem operation helpers are implemented
- [x] Go-side file read/write stream helpers are implemented
- [x] Go-side terminal detection helpers are implemented
- [x] Terminal standard-library surface is declared
- [x] OS/CLI runtime services are documented in `docs/os_cli_runtime.md`
- [x] OS/CLI runtime service tests are covered under `test/`

### 10.8 OS and CLI runtime integration

- [x] OS/CLI stdlib calls lower to direct runtime symbols in LLVM IR
- [x] process stdin/stdout/stderr stream operations lower to runtime symbols
- [x] OS/CLI stdlib imports trigger automatic app distribution runtime assets
- [x] CLI OS/runtime example exists under `examples/`
- [x] CLI OS/runtime example compiles to LLVM in smoke tests
- [x] OS/CLI runtime service smoke test runs filesystem, stream, process, child process, and terminal helpers

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
- [x] AST tests with organized control statement nodes
- [x] semantic tests with organized object literal analysis and organized function expression analysis
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
>     import { bind } from "ffi"
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
>     import { bind } from "ffi"
>
>     const f = () => {};
>     export const add = f;
>
>     export default bind({
>       sources: ["./src/mylib.c"],
>       includeDirs: ["./include"],
>       cflags: [],
>       ldflags: [],
>       exports: {
>         add: { symbol: "mylib_add", type: "function" }
>       }
>     });
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
- [x] `platforms` field is supported for target-specific source/include/flag overrides with organized platform manifest parsing
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
- [x] binding-listed native sources and shared libraries can be linked into emitted native executables
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
- [x] Jayess-facing LLVM package API surface and minimal IR builder are modeled for future compiler self-hosting
- [x] LLVM IR can be emitted directly
- [x] native executables can be built from LLVM IR output
- [x] object files can be emitted directly as a supported artifact and from the CLI
- [x] emitted bitcode if supported
- [x] static libraries can be emitted if supported
- [x] shared libraries can be emitted if supported

### 24.1a Shared library artifacts

- [x] CLI can emit target-aware shared libraries directly
- [x] Linux shared library output uses `.so`
- [x] macOS shared library output uses `.dylib`
- [x] Windows shared library output uses `.dll`
- [x] default shared library naming follows platform conventions
- [x] shared library builds use target-specific clang linking, executable-local bundled/ref tool discovery, missing-tool diagnostics, optional LLVM C API object emission, and an internal lld C++ shim
- [x] tooling compile plans route `--emit=shared` to shared-library build plans
- [x] shared library emission is covered by tests
- [x] platform distribution packages can be assembled under `dist/<platform>` with the compiler at package root, bundled LLVM/Clang/lld tools under `tools/bin`, LLVM runtime libraries, license notices, compressed archives, checksums, and organized dist test helpers
- [x] app distribution plans copy native shared libraries beside executables so users do not need separate library installation

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

- [x] statement lifetime is safe with organized returned-expression escape marking
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

---

## 33. Remaining compiler completion work

This section tracks the practical work still needed before Jayess is a complete
native JavaScript-like compiler. These are intentionally scoped to the current
project direction: no package manager requirement, limited type-system goals, and
native libraries supplied through explicit bindings.

### 33.1 Real LLVM lowering

These tasks track generalized lowering mechanisms only. Do not add one-off
literal/operator cases here; add support by wiring AST families through shared
emitters, runtime calls, and control-flow builders.

- [x] create a central LLVM expression emitter entry point
- [x] create a central LLVM statement emitter entry point
- [x] create shared runtime value-constructor emission helpers
- [x] preserve source positions in backend diagnostics
- [x] keep emitted local scope restoration isolated behind scope helpers
- [x] add focused tests for completed lowering mechanisms
- [x] replace constant `main` return-code folding with the general expression/statement emitter
- [x] represent mutable locals as LLVM slots with `alloca`, `load`, and `store`
- [x] lower lexical environments through one scope stack used by variables, blocks, functions, and catch bindings
- [x] lower assignment targets through one target-address/value abstraction for identifiers, members, and indexes
- [x] lower expression evaluation through a reusable left-to-right sequencing helper
- [x] add one runtime-call emission helper for lowered operators and value APIs
- [x] lower unary operators through a shared operator dispatcher
- [x] lower binary arithmetic and bitwise operators through a shared operator dispatcher
- [x] lower equality and relational operators through a shared comparison dispatcher
- [x] lower logical, nullish, ternary, and comma expressions through the sequencing and basic-block helpers
- [x] lower update expressions through the assignment-target abstraction
- [x] lower `typeof`, `void`, `delete`, `in`, and `instanceof` through operator dispatchers and runtime calls
- [x] lower object construction through runtime allocation APIs, including computed keys and spread
- [x] lower array construction through runtime allocation APIs, including elisions and spread
- [x] lower member access through runtime property APIs
- [x] lower index access through runtime index APIs
- [x] lower destructuring through the same target abstraction used by assignment
- [x] add an LLVM basic-block builder for branches, loops, exits, and fallthrough
- [x] lower `if` through the basic-block builder
- [x] lower `switch` through the basic-block builder
- [x] lower `while` as the primitive loop form in the backend
- [x] lower `do...while`, `for`, `for...in`, and `for...of` by translating them to the primitive `while` lowering path
- [x] lower `break` and `continue` through structured loop/switch exit stacks
- [x] lower labels through the same structured exit stacks
- [x] lower `return` through the shared abrupt-completion and cleanup path
- [x] lower `throw` through the shared abrupt-completion and cleanup path
- [x] lower `try`, `catch`, and `finally` through the shared abrupt-completion and cleanup path
- [x] connect lifetime/escape analysis to retain/release/destructor cleanup emission
- [x] lower function declarations and function expressions through the callable runtime ABI
- [x] lower closures and captured variables through the lexical environment lowering
- [x] lower calls, default/rest parameters, `arguments`, `.bind`, `.call`, and `.apply` through the callable runtime ABI
- [x] lower class constructors, `this`, `super`, `new`, and `new.target` through object/function lowering primitives
- [x] lower class fields, methods, accessors, private members, and static blocks through object/function lowering primitives
- [x] lower module initialization and imports through runtime module bindings
- [x] lower exports, re-exports, defaults, namespace objects, and live bindings through runtime module bindings
- [x] lower native binding wrapper calls through the runtime ABI
- [x] add executable lowering tests for each generalized mechanism before marking it done

### 33.2 Dynamic values, objects, and arrays

- [x] implement a runtime value representation for Jayess dynamic values
- [x] implement object allocation, property lookup, property write, and computed keys
- [x] implement array allocation, indexed read/write, growth, and length behavior
- [x] implement object spread, array spread, and destructuring runtime behavior
- [x] implement `for...in` and `for...of` runtime support for supported object/array shapes
- [x] document dynamic object/array runtime ownership and mutation rules

### 33.3 Functions and closures

- [x] implement callable runtime values for lowered function declarations and function expressions
- [x] implement runtime call frames, argument passing, default parameters, and rest parameters
- [x] implement runtime closure environment allocation and captured variable access
- [x] implement runtime arrow-function lexical `this`
- [x] implement `.bind`, `.call`, and `.apply` runtime behavior
- [x] cover recursive and higher-order function execution in tests

### 33.4 Classes and dispatch

- [x] define runtime class instance layout for fields, private fields, methods, and accessors
- [x] implement runtime constructor, `new`, `this`, and `super` support for lowering
- [x] implement runtime dispatch-table behavior for methods
- [x] implement runtime static fields and static initialization blocks
- [x] implement runtime getter/setter invocation and private member access checks
- [x] add executable tests for inheritance and method dispatch

### 33.5 Modules

- [x] wire module graph ordering into backend module initialization
- [x] lower imports and exports into runtime module bindings
- [x] implement live binding behavior where required by Jayess semantics
- [x] initialize modules exactly once and preserve deterministic order
- [x] add executable tests for local imports, re-exports, defaults, and namespace imports

### 33.6 Native bindings

- [x] compile binding `sources` as part of the main compiler workflow
- [x] link binding object files and declared libraries into executable/shared outputs
- [x] resolve binding include dirs and library dirs relative to the binding module
- [x] validate exported binding symbols against generated wrapper expectations
- [x] surface clear diagnostics for missing headers, missing libraries, and missing symbols
- [x] support binding-owned shared library runtime asset collection for app distributions

### 33.7 App distribution

- [x] expose app distribution from the compiler CLI, such as `jayess package` or `--emit=dist`
- [x] copy executable plus required `.so`, `.dylib`, or `.dll` files into an app dist folder
- [x] emit diagnostics for unresolved runtime shared library dependencies
- [x] preserve project-provided binding library licenses in app distributions when supplied
- [x] document static-vs-shared native binding shipping behavior
- [x] add end-to-end tests for app dist output layout

### 33.7.1 Imported dependency packaging policy

- [x] use the resolved import graph as the source of truth for app distribution dependency collection
- [x] distinguish source modules, Jayess packages, native binding modules, built-in packages, and external package imports in the dependency graph
- [x] attach package/binding distribution metadata to each imported dependency during resolver or build planning
- [x] fail app distribution when an imported dependency requires runtime assets that are not declared or cannot be resolved
- [x] keep non-redistributable platform SDKs as documented build-machine requirements, not end-user installation steps

### 33.7.2 Native dependency build inputs

- [x] build declared binding-owned native `sources` before executable/package creation when they are imported by the program
- [x] include imported binding object files and static libraries in the executable link plan
- [x] collect imported binding `sharedLibraries` as runtime assets for the app distribution
- [x] collect imported binding helper data/assets when the binding/package declares them
- [x] support platform-specific dependency inputs from binding `platforms.<target>` overrides during app distribution
- [x] emit actionable diagnostics for missing headers, source files, libraries, assets, or unsupported platform dependency declarations

### 33.7.3 Package dependency metadata

- [x] define a small package distribution metadata format for imported Jayess packages that need runtime assets
- [x] support package-declared runtime assets such as shared libraries, data files, helper executables, config templates, and license files
- [x] resolve package runtime assets relative to the imported package root, not the caller's working directory
- [x] deduplicate assets that are required by multiple imported packages or bindings
- [x] preserve deterministic output paths for copied package assets

### 33.7.4 License and redistribution enforcement

- [x] require license/notice metadata for imported third-party runtime libraries copied into app distributions
- [x] copy imported dependency license/notice files into a stable `licenses/` layout
- [x] report diagnostics when a redistributable imported dependency lacks required license metadata
- [x] allow package metadata to mark platform SDK/system-framework dependencies as build-only or system-provided when redistribution is not allowed
- [x] document the difference between redistributable runtime assets and non-redistributable SDK inputs in `docs/app_distribution.md`

### 33.7.5 Self-contained app distribution verification

- [x] add tests where importing a native binding causes its shared library and license file to appear in the app distribution
- [x] add tests where importing a package causes package-declared data/helper assets to appear in the app distribution
- [x] add tests where unused packages do not add runtime assets to the app distribution
- [x] add tests for duplicate imported dependencies producing one copied runtime asset
- [x] add tests for missing imported runtime assets failing package creation with diagnostics
- [x] add smoke tests that run a packaged app from the Jayess-produced distribution without separate end-user dependency installation

### 33.8 Toolchain and release readiness

- [x] rebuild the bundled LLVM toolchain with Clang and lld enabled for release packages
- [x] verify `jayess-dist` with `--strict-tools=true` from a clean checkout
- [x] verify compiler SDK archives on Linux, macOS, and Windows targets
- [x] document required platform SDK boundaries for macOS and Windows linking
- [x] add release smoke tests that compile examples from the unpacked SDK
- [x] keep `refs` and `old_version` read-only during release packaging

### 33.9 Recommended implementation order

- [x] implement core runtime value representation before object/function/class lowering
- [x] implement dynamic objects and arrays before class field/private-member behavior
- [x] integrate native binding compile/link before exposing app distribution in the CLI
- [x] finish generalized LLVM lowering mechanisms before marking compiler completion
- [x] add executable tests incrementally for each lowering feature as it lands
- [x] keep examples aligned with the actually executable subset during compiler completion

---

## 34. Self-hosting readiness

### 34.1 Large-program parsing and modules

- [x] parse larger multi-file Jayess programs without excessive memory use or parser recursion failures
- [x] preserve stable module graph ordering for compiler-sized projects
- [x] report module import/export errors with file, line, and symbol context
- [x] add compiler-scale parser fixtures under `test/`

### 34.2 Compiler data structures

- [x] provide Jayess-native list/vector support suitable for tokens, AST children, and diagnostics
- [x] provide Jayess-native map/table support suitable for scopes, symbols, and module registries
- [x] provide Jayess-native structured records suitable for tokens, AST nodes, and type metadata
- [x] document ownership and mutation rules for compiler data structures

### 34.3 File, path, and source input APIs

- [x] expose file read/write APIs needed by a compiler frontend
- [x] expose directory traversal and path normalization APIs for module resolution
- [x] support deterministic source loading across Linux, macOS, and Windows paths
- [x] add tests for missing files, invalid paths, and relative import resolution

### 34.4 String and text processing

- [x] support efficient string indexing, slicing, concatenation, and comparison for lexer work
- [x] support Unicode-aware source positions where the language requires them
- [x] support stable byte-offset, line, and column tracking for diagnostics
- [x] add lexer-sized string processing benchmarks or stress tests

### 34.5 Diagnostics and recovery

- [x] define a Jayess-native diagnostic structure with severity, span, message, and notes
- [x] support recoverable parser and semantic diagnostics without stopping at the first error
- [x] support deterministic diagnostic ordering for multi-file projects
- [x] add tests for multiple syntax and semantic errors in one compile

### 34.6 Backend and toolchain APIs

- [x] expose a Jayess-callable LLVM API package for modules, types, functions, blocks, and instructions
- [x] expose a Jayess-callable object emission API backed by the bundled LLVM toolchain
- [x] expose a Jayess-callable lld/link API or stable compiler backend API for executable output
- [x] add examples that build a small object file or executable through the exposed backend API

### 34.7 Runtime independence

- [x] ensure Jayess-compiled tools can run without depending on Go-only compiler internals
- [x] expose runtime services needed by compiler tools through Jayess packages
- [x] support packaging Jayess-built compiler utilities with their runtime assets
- [x] add smoke tests for Jayess-built command-line tools

### 34.8 Compiler-scale performance

- [x] measure parse, semantic, lowering, and backend time on large Jayess source inputs
- [x] avoid quadratic behavior in module resolution, symbol lookup, and diagnostic collection
- [x] keep memory usage bounded for compiler-sized ASTs and module graphs
- [x] add performance baselines under `test/` or `temp/` generated fixtures

### 34.9 Self-hosting milestones

- [x] compile a Jayess-written lexer utility with the Go-hosted Jayess compiler
- [x] compile a Jayess-written parser utility with the Go-hosted Jayess compiler
- [x] compile a Jayess-written semantic checker utility with the Go-hosted Jayess compiler
- [x] compile a Jayess-written backend/toolchain utility with the Go-hosted Jayess compiler
- [x] compile a small Jayess-written compiler prototype using previously compiled Jayess utilities

---

## 35. Documentation completion checklist

All project documentation for this section should live under `docs/`. Keep each
document small and focused; use `README.md` only as the top-level entry point.

### 35.1 Language basics

- [x] document Jayess goals, supported JavaScript-like syntax, and deliberate non-goals in `docs/language_overview.md`
- [x] document project layout, compiler pipeline, and where each Go package fits in `docs/compiler_overview.md`
- [x] document source files, comments, literals, variables, constants, and scope rules in `docs/language_basics.md`
- [x] document primitive types, runtime value representation, truthiness, and equality behavior in `docs/types_and_values.md`
- [x] document expressions, operators, precedence, assignment, update, and type conversion behavior in `docs/expressions.md`
- [x] document statements, blocks, conditionals, loops, switch, labels, break, and continue in `docs/statements.md`

### 35.2 Functions, modules, and objects

- [x] document functions, closures, returns, parameters, call behavior, and lifetime rules in `docs/functions.md`
- [x] document arrays, objects, properties, indexing, member access, and mutation behavior in `docs/objects_and_arrays.md`
- [x] document destructuring, spread-like supported forms, and unsupported patterns in `docs/destructuring.md`
- [x] document classes, constructors, fields, methods, accessors, inheritance, and current limits in `docs/classes.md`
- [x] document imports, exports, module resolution, project loading, and diagnostic behavior in `docs/modules.md`
- [x] document unsupported JavaScript features with clear alternatives or expected errors in `docs/unsupported_features.md`

### 35.3 Compiler stages

- [x] document lexer output, token model, source positions, and text handling in `docs/lexer.md`
- [x] document parser AST nodes, parser recovery, and syntax diagnostic conventions in `docs/parser.md`
- [x] document semantic checks, symbol tables, type/value assumptions, and module checks in `docs/semantic_analysis.md`
- [x] document generalized lowering strategy and which high-level constructs lower to core forms in `docs/lowering.md`
- [x] document LLVM backend responsibilities, runtime calls, emitted IR shape, and link flow in `docs/llvm_backend.md`
- [x] document diagnostics format, severity rules, note handling, and deterministic ordering in `docs/diagnostics.md`

### 35.4 Runtime and standard packages

- [x] document runtime value APIs, object/array helpers, string helpers, and memory/lifetime expectations in `docs/runtime_values.md`
- [x] document file, path, source text, and compiler data-structure runtime APIs in `docs/runtime_services.md`
- [x] document built-in HTML, XML, and CSS parsing packages with examples in `docs/parsing_packages.md`
- [x] document SQLite package setup, linking expectations, and examples in `docs/sqlite.md`
- [x] document crypto/TLS package setup, OpenSSL expectations, and examples in `docs/crypto_tls.md`
- [x] document async/runtime integration expectations where libuv support is available in `docs/async_runtime.md`

### 35.5 Native binding and external libraries

- [x] document native binding manifest format, bind file structure, platform fields, and validation rules in `docs/native_bindings.md`
- [x] document how developers install external native libraries locally before binding them in `docs/native_library_setup.md`
- [x] document include paths, library paths, link flags, runtime library placement, and distribution layout in `docs/native_linking.md`
- [x] document how app distribution collects native runtime assets needed by built executables in `docs/app_distribution.md`
- [x] document licensing responsibilities for third-party native libraries included by developers in `docs/third_party_licenses.md`
- [x] document examples for binding a small C library and shipping its runtime files in `docs/native_binding_examples.md`

### 35.6 Networking and embedded services

- [x] document libcurl networking package setup, supported transfer APIs, and error handling in `docs/network_curl.md`
- [x] document Mongoose embedded web server setup, lifecycle, routing model, and shutdown behavior in `docs/network_mongoose.md`
- [x] document picohttpparser package purpose, low-level HTTP parsing API, and ownership rules in `docs/network_http_parser.md`
- [x] document TLS/networking integration boundaries between curl, OpenSSL, and embedded server packages in `docs/network_tls.md`
- [x] document network package distribution requirements for native libraries and licenses in `docs/network_distribution.md`
- [x] document minimal HTTP client and embedded HTTP server examples in `docs/network_examples.md`

### 35.7 Graphics, UI, and app distribution

- [x] document raylib setup, native library requirements, asset placement, and graphics examples in `docs/graphics_raylib.md`
- [x] document GLFW/SDL-style binding expectations if developers provide those bindings themselves in `docs/graphics_native_bindings.md`
- [x] document webview native app support, platform dependencies, and packaging behavior in `docs/webview_apps.md`
- [x] document app distribution command usage, output layout, runtime assets, and troubleshooting in `docs/app_distribution.md`
- [x] document compiler/toolchain distribution command usage, LLVM/Clang/lld expectations, and licenses in `docs/toolchain_distribution.md`
- [x] document cross-platform release notes for Linux, macOS, and Windows differences in `docs/platform_notes.md`

### 35.8 Self-hosting and contributor documentation

- [x] document self-hosting readiness, milestones, remaining risks, and how to run milestone tests in `docs/self_hosting.md`
- [x] document compiler data structures intended for future Jayess-written compiler components in `docs/compiler_data_structures.md`
- [x] document performance baselines, benchmark commands, and expected large-program behavior in `docs/performance.md`
- [x] document testing strategy, test package layout, smoke tests, and release verification commands in `docs/testing.md`
- [x] document contributor rules for keeping files small, avoiding protected directories, and using `temp/` in `docs/contributing.md`
- [x] document release checklist from clean checkout through distributable smoke test in `docs/release.md`

---
