# Network TLS Boundaries

Networking packages should keep TLS responsibilities explicit.

## Boundaries

- libcurl can own TLS for client transfers when built against TLS support
- OpenSSL bindings can expose lower-level crypto or TLS primitives
- embedded server packages should document whether they terminate TLS directly
  or expect a proxy

## Distribution

When TLS relies on shared native libraries, package those libraries and license
files with the application distribution.

## Configuration

Certificates, keys, and trust store paths should be explicit application
configuration, not hidden global state.

## Example Configuration

```js
const tls = {
  certFile: "./certs/server.crt",
  keyFile: "./certs/server.key",
  trustStore: "./certs/ca.pem"
};
```
