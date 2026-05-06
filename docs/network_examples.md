# Network Examples

Network examples should stay small and show the distribution contract clearly.

## HTTP Client Shape

```js
import { get } from "./native/curl.js";

function main() {
  const response = get("https://example.com");
  console.log(response.status);
  return 0;
}
```

## Embedded Server Shape

```js
import { serve } from "./native/server.js";

function main() {
  serve("127.0.0.1:8080", (request) => {
    return { status: 200, body: "ok" };
  });
  return 0;
}
```

The actual import paths depend on the project's binding layout.
