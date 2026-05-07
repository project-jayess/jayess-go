# DNS

Jayess DNS support is implemented with Go-owned internal runtime helpers. It
does not require c-ares, shell tools, libuv, or user-installed resolver
libraries.

## API Shape

- `dns.lookup(host)` resolves a host to IP records.
- `dns.reverse(address)` performs reverse lookup for an IP address.
- `dns.resolver(servers)` creates resolver configuration from DNS servers.
- `dns.isIP(address)` returns `4`, `6`, or `0`.
- `dns.parseIP(address)` returns parsed IP metadata.

## Resolver Behavior

The runtime uses Go's resolver. When a resolver configuration includes servers,
Jayess uses Go's resolver dial hook and sends DNS traffic to those servers. When
no servers are configured, Jayess uses the platform resolver through Go.

Timeouts are explicit runtime configuration. Invalid addresses fail before a
reverse lookup is attempted.

## Example

```js
function main(host) {
	const resolver = dns.resolver(["1.1.1.1", "8.8.8.8"]);
	const records = dns.lookup(host, resolver);
	const parsed = dns.parseIP("127.0.0.1");
	return records || parsed;
}
```
