# Objects and Arrays

Objects and arrays are dynamic runtime values backed by Jayess runtime helpers.

## Objects

Object literals create mutable property containers.

```js
const user = { name: "Ada", count: 1 };
user.count = user.count + 1;
```

Properties can be accessed with dot syntax or computed member syntax when the
expression is supported by lowering and backend emission.

## Arrays

Arrays are ordered containers with indexed access.

```js
const items = [1, 2, 3];
const first = items[0];
```

Array operations are emitted through runtime helpers where dynamic behavior is
needed.

## Identity

Objects and arrays compare by identity unless a runtime operation explicitly
implements another behavior.
