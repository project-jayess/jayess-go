# Modules

Jayess supports local module imports and npm-style package resolution through `node_modules`.

## Imports

Supported forms:

```javascript
import "./utils.js";
import { add, twice } from "./lib/math.js";
import { add as sum } from "./lib/math.js";
import thing from "./lib/module.js";
import thing, { add as sum } from "./lib/module.js";
import * as ns from "./lib/module.js";

import { add } from "@demo/math";
import thing from "@demo/math";
import * as ns from "@demo/math";
```

Relative imports and package imports are both supported.

## Exports

Supported forms:

```javascript
export function add(a, b) {
  return a + b;
}

export const VERSION = "0.1.0";
export var counter = 0;

export default function current() {
  return counter;
}

export default 123;
export { counter, add as sum };
export { add as sum } from "@demo/math";
export * from "./more.js";
export * as math from "./more.js";
```

## Visibility

Modules are private by default. Only exported bindings are visible to importers.

Top-level `public` and `private` are not used for module visibility.

## Package Resolution

Jayess uses npm-style package layout:

- dependencies from `package.json`
- installed packages from `node_modules/`
- scoped packages like `@scope/pkg`

Jayess is not the Node.js runtime. Package management is npm-based, but compiled programs are native executables.

If a package is installed but does not expose a supported Jayess `.js` entrypoint, the compiler reports that explicitly instead of silently treating it as a valid module.

## Native Source Imports

Jayess can also link native wrapper sources via `import`:

```javascript
import { jayess_add } from "./native/math.c";
import "./native/math.c";
```

See [Native Interop](/C:/Users/ncksd/Documents/it/jayess/jayess-go/docs/native-interop.md).
