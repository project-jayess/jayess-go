# TCP And UDP

Jayess TCP and UDP support is implemented with Go-owned internal runtime
helpers. It does not require libuv, c-ares, shell tools, or user-installed socket
libraries.

## TCP

The TCP runtime provides:

- client and server handles
- connect, listen, accept, read, write, and close helpers
- timeout metadata on clients, servers, and sockets
- last-error tracking
- duplex `IOStream` access for connected sockets

TCP sockets expose the shared Jayess stream model so higher-level packages can
reuse pipe and backpressure behavior.

```js
const server = tcp.server();
tcp.listen(server, "127.0.0.1:9000");

const client = tcp.client();
const socket = tcp.connect(client, "127.0.0.1:9000");
tcp.write(socket, "hello");
```

## UDP

The UDP runtime provides:

- socket creation
- bind
- send
- receive
- close
- timeout metadata
- broadcast intent metadata

Multicast joins are explicitly unsupported for now because Go's standard library
does not expose portable group-management helpers without extra packages. A
future Jayess-owned implementation can add this without requiring end-user
installation steps.

```js
const socket = udp.socket();
udp.bind(socket, "127.0.0.1:0");
udp.send(socket, "hello", "127.0.0.1:9001");
const packet = udp.receive(socket);
```
