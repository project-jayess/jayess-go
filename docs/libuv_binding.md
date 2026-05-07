# libuv Binding Boundary

libuv support is optional binding experiment support only. Standard Jayess
programs should use Jayess-owned runtime services for timers, microtasks,
filesystem operations, process helpers, child-process spawning, TCP, UDP, DNS,
HTTP, HTTPS, and streams.

## When To Use libuv

Use an explicit libuv binding only when a package is experimenting with libuv
APIs directly or needs behavior that is intentionally outside the standard
Jayess runtime service layer.

## Distribution

An app that imports only standard Jayess packages must not package libuv assets
or require libuv to be installed. If an app explicitly imports a libuv binding,
that binding must declare all redistributable source, shared libraries, headers,
licenses, and notices needed by the final distribution.

## Separation

Keep libuv binding docs and tests separate from core runtime docs. Core runtime
tests should prove Jayess services work without libuv; libuv tests should only
validate the optional binding model.
