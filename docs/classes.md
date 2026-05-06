# Classes

Jayess supports a focused JavaScript-like class subset.

## Supported Forms

- `class Name { ... }`
- `export class Name { ... }`
- `export default class Name { ... }`
- constructors
- instance fields and methods
- `#private` instance fields and methods
- static fields and methods
- static private fields and methods
- `new Name(...)`
- single inheritance with `extends`
- `super(...)`, `super.method()`, and `super.property`

## Example

```js
class Counter {
  constructor(start) {
    this.value = start;
  }

  inc() {
    this.value = this.value + 1;
    return this.value;
  }
}
```

## Limits

The compiler rejects class features that are not implemented by parser,
semantic, lowering, and backend stages. Prefer small explicit classes over
dynamic metaprogramming patterns.
