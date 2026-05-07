# Network TLS Boundaries

Jayess TLS and HTTPS use compiler-owned Go runtime helpers for supported paths.
Applications should not require end users to install OpenSSL, libcurl, libuv, or
another TLS library after packaging.

## Boundaries

- `tls` owns TLS configuration, certificate loading, trust stores, ALPN, and
  hostname verification data structures.
- `https` owns HTTPS client/server facades and reuses the internal HTTP request,
  response, handler, stream, and event model.
- optional native bindings may still expose OpenSSL, libcurl, or platform TLS,
  but those are explicit package imports and not required for internal
  `tls`/`https`.

## Distribution

Internal TLS and HTTPS do not add shared-library runtime assets. If a future
target needs redistributable TLS backend assets, the compiler must package them
automatically when `tls` or `https` is imported.

## Configuration

Certificates, keys, and trust store paths should be explicit application
configuration, not hidden global state.

## Example

```js
const cert = tls.certificate("./certs/server.crt", "./certs/server.key");
const server = https.createServer({ cert: cert }, (req, res) => {
	http.status(res, 200);
	http.writeBody(res, "secure hello");
});

server.listen("127.0.0.1:8443");
```
