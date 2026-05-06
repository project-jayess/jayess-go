# Destructuring

Jayess supports selected destructuring forms for variables, assignments, and
parameters where the parser, semantic checks, lowering, and backend agree on a
stable representation.

## Object Patterns

Object destructuring reads named properties from a value and binds them to local
names or assignment targets.

```js
const point = { x: 1, y: 2 };
const { x, y } = point;
```

## Array Patterns

Array destructuring reads indexed values.

```js
const pair = [10, 20];
const [left, right] = pair;
```

## Limits

Unsupported nested, rest, default, or computed patterns should be rejected with
clear diagnostics. Do not rely on unsupported JavaScript destructuring behavior
being accepted.
