import "http";

function main() {
  const server = http.createServer();
  server.on("request", (req, res) => {
    const request = http.requestObject(req);
    const response = http.responseObject(res);
    http.status(response, 200);
    http.writeBody(response, "hello " + http.readBody(request));
    return http.headers(request);
  });
  server.once("listening", () => process.stdout.write("listening\n"));
  server.on("error", (err) => process.stderr.write(String(err)));
  const client = http.withTimeout(http.request("http://127.0.0.1:3000"), 1000);
  const kept = http.keepAlive(client);
  http.status(kept, 200);
  http.writeBody(kept, "ready");
  return server || http.streamBody(kept);
}
