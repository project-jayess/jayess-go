# Classes

Jayess supports JavaScript-like classes with a native compiled implementation.

## Supported Syntax

Examples:

```javascript
class Counter {
  value = 1;
  #secret = 9;

  constructor(step) {
    this.step = step;
  }

  total() {
    return this.value + this.step + this.#secret;
  }

  static label() {
    return "Counter";
  }
}
```

Supported class features:

- `class Name { ... }`
- `export class Name { ... }`
- `export default class Name { ... }`
- `new Name(...)`
- constructors
- instance field initializers
- instance methods
- static fields
- static methods
- `this`
- `extends`
- `super(...)`
- `super.method()`
- `super.property`
- `#private` instance fields and methods
- `static #private` fields and methods

## Inheritance

Single inheritance is supported:

```javascript
class Animal {
  sound() {
    return "noise";
  }
}

class Dog extends Animal {
  sound() {
    return "woof";
  }

  parentSound() {
    return super.sound();
  }
}
```

`instanceof` works for direct classes and constructor aliases.

## Methods and Function Values

Methods can be extracted and used as function values:

```javascript
var dog = new Dog();
var speak = dog.sound;
console.log(speak());
```

Direct method calls and extracted method references both use the current callable runtime model.

## Private Members

Jayess uses JavaScript-style `#private` syntax for class members:

```javascript
class Example {
  #count = 0;

  #read() {
    return this.#count;
  }
}
```

Top-level `private` and `public` are not supported. Module visibility is controlled by `export`.

## Notes

- Jayess classes are implemented explicitly in the compiler pipeline.
- The syntax is JavaScript-like, but the runtime is Jayess-specific.
- The current model is intentionally limited to single inheritance and the existing class surface above.
