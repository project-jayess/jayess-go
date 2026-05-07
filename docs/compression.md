# Compression

Jayess compression is implemented with Go-owned internal runtime helpers. The
supported paths do not require zlib, Brotli, or another user-installed native
library.

## Supported Formats

- `compression.gzip(data)`
- `compression.gunzip(data)`
- `compression.deflate(data)`
- `compression.inflate(data)`
- `compression.createCompressStream(format, source)`
- `compression.createDecompressStream(format, source)`

The stream helpers use the shared Jayess `IOStream` model and currently support
`gzip` and `deflate`.

## Brotli

Brotli remains explicitly unsupported by the internal runtime. Go's standard
library does not include Brotli, and Jayess should not require end users to
install or receive a separate Brotli library for normal compression support.

Calls to `compression.brotliCompress` and `compression.brotliDecompress` must
return a stable unsupported-format error until one of those options exists. They
must not silently package an external Brotli library.

## Distribution

`gzip` and `deflate` are internal runtime helpers and must not package external
`zlib` assets. Brotli is also not packaged while it is unsupported.

## Example

```js
function main(data) {
	const gz = compression.gzip(data);
	const plain = compression.gunzip(gz);

	const input = stream.readable(plain);
	const compressed = compression.createCompressStream("gzip", input);
	return compression.createDecompressStream("gzip", compressed);
}
```
