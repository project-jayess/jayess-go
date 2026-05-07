# Crypto and TLS

Jayess crypto is Go-owned internal runtime support for the common primitives
needed by CLI, server, and compiler-style programs. Supported crypto imports do
not require OpenSSL, zlib, libuv, or another user-installed native library.

## Internal Crypto

The internal `crypto` module supports:

- `crypto.randomBytes(size)`
- `crypto.hash(algorithm, data)`
- `crypto.hmac(algorithm, key, data)`
- `crypto.encrypt(algorithm, key, nonce, data)`
- `crypto.decrypt(algorithm, key, nonce, data)`
- `crypto.generateKey(algorithm)`
- `crypto.sign(key, data)`
- `crypto.verify(key, data, signature)`
- `crypto.publicEncrypt(key, data)`
- `crypto.privateDecrypt(key, data)`
- `crypto.secureCompare(left, right)`

The first internal digest set is `md5`, `sha1`, `sha224`, `sha256`, `sha384`,
and `sha512`. Algorithm names are case-insensitive and may include `-` or `_`.
Unsupported algorithms fail with a runtime error instead of silently falling
back to a system library.

The first internal key and encryption set is `aes-256-gcm`, `ed25519`, and
`rsa-oaep`. Key import/export helpers use standard PEM encodings. Certificate
parsing uses Go's `x509` package and returns stable certificate metadata for
runtime checks and diagnostics.

OpenSSL-backed bindings are now optional escape hatches for algorithms and
protocol features that Jayess does not own internally yet. They are not required
for the supported crypto, TLS, or HTTPS paths above.

## TLS And HTTPS

TLS and HTTPS use Jayess-owned Go runtime support for supported paths. The
runtime models certificate/key pairs, trust stores, hostname checks, ALPN, and
HTTPS client/server configuration without requiring OpenSSL to be installed by
the end user.

```js
const cert = tls.certificate("./certs/server.crt", "./certs/server.key");
const server = https.createServer({ cert: cert }, (req, res) => {
	http.status(res, 200);
	http.writeBody(res, "ok");
});

const client = https.request("https://localhost:8443", https.secureDefaults());
```

## Distribution

Internal crypto does not add end-user installation steps. Applications importing
the supported `crypto` helpers should compile and package with the Jayess runtime
only. Optional third-party crypto bindings remain allowed, but they must be
declared as explicit bindings and packaged by the compiler.

## Example

```js
function main() {
	const nonce = crypto.randomBytes(16);
	const digest = crypto.hash("sha256", "hello");
	const mac = crypto.hmac("sha256", "key", "hello");
	const key = crypto.generateKey("ed25519");
	const sig = crypto.sign(key, digest);
	const ok = crypto.verify(key, digest, sig);
	return crypto.secureCompare(digest, mac) || ok || nonce;
}
```
