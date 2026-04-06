# Language Overview

This page describes the Jayess language surface currently implemented in the compiler.

## Entry Point

Programs typically declare:

```javascript
function main(args) {
  console.log(args);
  return 0;
}
```

- `main(args)` receives command-line arguments.
- `args` is array-like and printable.

## Variables

Jayess supports:

- `var` for mutable block-scoped bindings
- `const` for immutable bindings

`let` is not supported.

Examples:

```javascript
var count = 1;
count = "kimchi";

const settings = {};
settings.enabled = true;
```

`const` prevents rebinding, but object mutation is allowed.

## Values

Jayess supports:

- numbers
- strings
- booleans
- `null`
- `undefined`
- arrays
- objects
- functions
- class instances

## Expressions

Supported expression features include:

- arithmetic: `+`, `-`, `*`, `/`
- comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`
- strict comparison: `===`, `!==`
- logical operators: `&&`, `||`, `!`
- nullish coalescing: `??`
- optional chaining: `obj?.name`, `obj?.[key]`, `obj?.call?.()`
- compound assignment: `+=`, `-=`, `*=`, `/=`, `??=`, `||=`, `&&=`
- `typeof`
- `instanceof`
- `new.target` inside constructor code paths

String concatenation works with `+`:

```javascript
console.log("hello " + "world");
```

Template strings are supported:

```javascript
var name = "kimchi";
console.log(`hello ${name}`);
```

## Objects

Object literals support:

- ordinary properties
- computed keys
- object literal methods

Examples:

```javascript
var key = "name";
var obj = {
  [key]: "kimchi",
  greet() {
    return "hello";
  }
};

console.log(obj[key]);
console.log(obj.greet());
```

Object mutation is supported at runtime:

```javascript
obj.spicy = 10;
delete obj.spicy;
```

## Arrays

Array literals and index access are supported:

```javascript
var values = [1, 2, 3];
console.log(values[0]);
values[1] = 10;
```

Supported array operations include:

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

`for...of` works with arrays.

## Destructuring

Supported:

- object destructuring
- array destructuring
- destructuring assignment
- parameter destructuring
- destructuring defaults
- object and array rest elements

Examples:

```javascript
const { name = "unknown", ...rest } = profile;
const [head, ...tail] = values;

function show({ title = "untitled" }) {
  console.log(title);
}
```

Current boundary:

- top-level destructuring is still rejected
- destructuring rest must be an identifier, not another nested pattern

## Functions

Jayess supports:

- function declarations
- function expressions
- arrow functions
- closures
- default parameters
- rest parameters
- spread in arrays and calls

Examples:

```javascript
function add(a, b = 1) {
  return a + b;
}

var twice = (value) => value * 2;
var wrap = function(name) {
  return `hello ${name}`;
};
```

Functions are first-class values. They can be:

- assigned to variables
- stored in arrays and objects
- returned from functions
- passed as callbacks
- given properties

Examples:

```javascript
function greet(name) {
  return `hi ${name}`;
}

greet.label = "greeter";
console.log(greet.label);
```

JS-like helpers are available:

- `fn.call(thisArg, ...args)`
- `fn.apply(thisArg, argsArray)`
- `fn.bind(thisArg, ...args)`

## Control Flow

Supported control flow includes:

- `if / else if / else`
- `while`
- `for`
- `for...of`
- `for...in`
- `switch`
- `break`
- `continue`
- `try / catch / finally`
- `throw`

Compile-time errors are still compile-time diagnostics. `try / catch / finally` only applies to runtime-thrown Jayess values.

## Input and Timing

Supported builtins:

- `readLine(prompt)`
- `readKey(prompt)`
- `sleep(milliseconds)`

Examples:

```javascript
var name = readLine("Name: ");
readKey("Press any key");
sleep(500);
```
