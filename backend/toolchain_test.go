package backend

import (
	"bufio"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"jayess-go/compiler"
	"jayess-go/target"
)

func readHTTPTestRequest(conn net.Conn) (string, error) {
	var builder strings.Builder
	reader := bufio.NewReader(conn)
	contentLength := 0

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return "", err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		builder.WriteString(line)
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}
		if colon := strings.Index(trimmed, ":"); colon >= 0 {
			name := strings.TrimSpace(trimmed[:colon])
			value := strings.TrimSpace(trimmed[colon+1:])
			if strings.EqualFold(name, "Content-Length") {
				if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
					contentLength = parsed
				}
			}
		}
	}
	if contentLength > 0 {
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			return "", err
		}
		builder.Write(body)
	}
	return builder.String(), nil
}

func writeHTTPTestFailure(conn net.Conn) {
	if conn != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 500 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
	}
}

func TestBuildExecutableRunsCompiledProgram(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log("hello native");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "hello-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "hello native") {
		t.Fatalf("expected program output to contain hello native, got: %s", string(out))
	}
}

func TestBuildExecutablePassesRuntimeArgs(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log(process.argv());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "args-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath, "kimchi", "jjigae")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "kimchi") || !strings.Contains(text, "jjigae") {
		t.Fatalf("expected argv output to contain runtime args, got: %s", text)
	}
}

func TestBuildExecutableSupportsPromiseThenRejectAndAwaitCatch(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
async function loadValue(): number {
  return 7;
}

function main(args) {
  var resolved = Promise.resolve(2).then((value) => value + 3);
  console.log(await resolved);
  loadValue().then((value) => {
    console.log(value);
    return value;
  });
  try {
    await Promise.reject(new Error("kimchi"));
  } catch (err) {
    console.log(err.message);
  }
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "promise-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"5", "7", "kimchi"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected promise output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsPromiseAllAndRace(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var values = await Promise.all([Promise.resolve("kimchi"), 12, Promise.resolve("jjigae")]);
  console.log(values);
  console.log(await Promise.race([Promise.resolve("first"), Promise.resolve("second")]));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "promise-combinators-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"kimchi", "12", "jjigae", "first"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected promise combinator output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsTimerPromiseRace(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var winner = await Promise.race([
    sleepAsync(25, "slow"),
    sleepAsync(0, "fast")
  ]);
  console.log(winner);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "timer-promise-race-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "fast") {
		t.Fatalf("expected timer Promise.race output to contain fast, got: %s", text)
	}
	if strings.Contains(text, "slow") {
		t.Fatalf("expected slower timer promise not to win race, got: %s", text)
	}
}

func TestBuildExecutableSupportsTimersNamespace(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var id = timers.setTimeout(() => {
    console.log("cancelled");
    return 0;
  }, 0);
  timers.clearTimeout(id);
  console.log(await timers.sleep(0, "ready"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "timers-namespace-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "ready") {
		t.Fatalf("expected timers namespace output to contain ready, got: %s", text)
	}
	if strings.Contains(text, "cancelled") {
		t.Fatalf("expected cancelled namespaced timer not to run, got: %s", text)
	}
}

func TestBuildExecutableSupportsPromiseFinallyAndAllSettled(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var resolved = Promise.resolve("value").finally(() => {
    console.log("cleanup");
    return 0;
  });
  console.log(await resolved);
  try {
    await Promise.reject(new Error("bad")).finally(() => {
      console.log("rejected cleanup");
      return 0;
    });
  } catch (err) {
    console.log(err.message);
  }
  var settled = await Promise.allSettled([Promise.resolve("ok"), Promise.reject("no")]);
  console.log(settled[0].status, settled[0].value);
  console.log(settled[1].status, settled[1].reason);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "promise-finally-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"cleanup", "value", "rejected cleanup", "bad", "fulfilled", "ok", "rejected", "no"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected Promise.finally/allSettled output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsPromiseAnyAndAggregateError(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log(await Promise.any([Promise.reject("no"), Promise.resolve("yes")]));
  try {
    await Promise.any([Promise.reject("first"), Promise.reject("second")]);
  } catch (err) {
    console.log(err.name, err.message);
    console.log(err.errors[0], err.errors[1]);
  }
  var custom = new AggregateError(["a", "b"], "custom message");
  console.log(custom.name, custom.message, custom.errors[1]);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "promise-any-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"yes", "AggregateError", "All promises were rejected", "first", "second", "custom message", "b"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected Promise.any/AggregateError output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableThrowsTypeErrorForNonFunctionInvocation(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var value = 1;
  try {
    value();
  } catch (err) {
    console.log(err.name, err.message);
  }
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "type-error-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"TypeError", "value is not a function"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected TypeError output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableRunsPromiseCallbacksAsMicrotasks(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  Promise.resolve("micro").then((value) => {
    console.log(value);
    return value;
  });
  console.log("sync");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "microtask-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	syncIndex := strings.Index(text, "sync")
	microIndex := strings.Index(text, "micro")
	if syncIndex < 0 || microIndex < 0 {
		t.Fatalf("expected sync and micro output, got: %s", text)
	}
	if syncIndex > microIndex {
		t.Fatalf("expected microtask to run after sync code, got: %s", text)
	}
}

func TestBuildExecutableSupportsTimersAndAsyncArrowFunctions(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var later = async (value: number): number => value + 1;
  setTimeout(() => {
    console.log("timer");
    return 0;
  }, 0);
  Promise.resolve("micro").then((value) => {
    console.log(value);
    return value;
  });
  console.log(await later(4));
  console.log("sync");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "timer-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"5", "sync", "micro", "timer"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected timer output to contain %q, got: %s", want, text)
		}
	}
	microIndex := strings.Index(text, "micro")
	timerIndex := strings.Index(text, "timer")
	if microIndex < 0 || timerIndex < 0 || microIndex > timerIndex {
		t.Fatalf("expected microtask output before timer output, got: %s", text)
	}
}

func TestBuildExecutableSupportsClearTimeout(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  var id = setTimeout(() => {
    console.log("cancelled");
    return 0;
  }, 0);
  clearTimeout(id);
  setTimeout(() => {
    console.log("kept");
    return 0;
  }, 0);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "clear-timeout-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if strings.Contains(text, "cancelled") {
		t.Fatalf("expected cancelled timer not to run, got: %s", text)
	}
	if !strings.Contains(text, "kept") {
		t.Fatalf("expected kept timer to run, got: %s", text)
	}
}

func TestBuildExecutableSupportsFsAndPathRuntimeSurface(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "note.txt");
  fs.writeFile(file, "kimchi");
  console.log(path.basename(file));
  console.log(path.extname(file));
  console.log(fs.exists(file));
  console.log(fs.readFile(file));
  console.log(await fs.readFileAsync(file, "utf8"));
  await fs.writeFileAsync(path.join("tmp", "async-note.txt"), "async kimchi");
  console.log(await fs.readFileAsync(path.join("tmp", "async-note.txt"), "utf8"));
  console.log(fs.stat(file).isFile);
  console.log(fs.readDir("tmp")[0].name);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "fs-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"note.txt", ".txt", "true", "kimchi"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected fs/path output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsFsStreams(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "stream.txt");
  var writer = fs.createWriteStream(file);
  console.log(writer.write("kim"));
  console.log(writer.write("chi"));
  writer.on("finish", () => {
    console.log("removed-finish");
    return 0;
  });
  writer.removeListener("finish");
  writer.on("finish", () => {
    console.log("finished");
    return 0;
  });
  writer.on("finish", () => {
    console.log("finished-two");
    return 0;
  });
  writer.once("finish", () => {
    console.log("finished-once");
    return 0;
  });
  var skippedFinish = () => {
    console.log("skipped-specific");
    return 0;
  };
  writer.on("finish", skippedFinish);
  writer.off("finish", skippedFinish);
  console.log("finish-count:" + writer.listenerCount("finish"));
  console.log("finish-event:" + writer.eventNames().includes("finish"));
  writer.end();
  console.log(writer.closed);
  console.log(writer.writableEnded);
  writer.on("finish", () => {
    console.log("finished-late");
    return 0;
  });
  writer.once("finish", () => {
    console.log("finished-once-late");
    return 0;
  });

  var reader = fs.createReadStream(file);
  console.log(reader.read(3));
  console.log(reader.read(3));
  console.log(reader.read());
  console.log(reader.readableEnded);
  reader.close();
  console.log(reader.closed);

  var eventReader = fs.createReadStream(file);
  eventReader.on("end", () => {
    console.log("ended");
    return 0;
  });
  eventReader.on("data", (chunk) => {
    console.log("data:" + chunk);
    return 0;
  });

  var onceReader = fs.createReadStream(file);
  onceReader.once("data", (chunk) => {
    console.log("once-data:" + chunk);
    return 0;
  });
  onceReader.once("end", () => {
    console.log("once-ended");
    return 0;
  });
  console.log("once-end-count:" + onceReader.listenerCount("end"));
  onceReader.read();
  onceReader.read();
  console.log("once-end-after:" + onceReader.listenerCount("end"));

  var pipeTarget = path.join("tmp", "stream-copy.txt");
  fs.createReadStream(file).pipe(fs.createWriteStream(pipeTarget));
  console.log(fs.readFile(pipeTarget, "utf8"));

  var missing = fs.createReadStream(path.join("tmp", "missing-stream.txt"));
  missing.on("error", (err) => {
    console.log("read-error:" + err.message);
    return 0;
  });
  missing.on("error", (err) => {
    console.log("read-error-two:" + err.message);
    return 0;
  });
  missing.once("error", (err) => {
    console.log("read-error-once:" + err.message);
    return 0;
  });
  console.log("error-count:" + missing.listenerCount("error"));
  console.log("error-event:" + missing.eventNames().includes("error"));
  console.log(missing.errored);

  var binaryFile = path.join("tmp", "binary.bin");
  var outBytes = new Uint8Array(4);
  outBytes[0] = 65;
  outBytes[1] = 0;
  outBytes[2] = 255;
  outBytes[3] = 66;
  var binaryWriter = fs.createWriteStream(binaryFile);
  console.log("bytes-write:" + binaryWriter.write(outBytes));
  binaryWriter.end();
  var binaryReader = fs.createReadStream(binaryFile);
  var inBytes = binaryReader.readBytes(4);
  console.log("bytes-read:" + inBytes.length + ":" + inBytes[0] + ":" + inBytes[1] + ":" + inBytes[2] + ":" + inBytes[3]);
  var slicedBytes = inBytes.slice(1, 3);
  console.log("bytes-slice:" + slicedBytes.length + ":" + slicedBytes[0] + ":" + slicedBytes[1]);
  console.log("bytes-includes:" + inBytes.includes(255) + ":" + inBytes.includes(13));
  console.log("bytes-text:" + inBytes.slice(0, 1).toString() + inBytes.slice(3, 4).toString());
  console.log("bytes-end:" + binaryReader.readBytes(1));

  var textBytes = Uint8Array.fromString("jayess");
  var textBytesFile = path.join("tmp", "text-bytes.bin");
  var textBytesWriter = fs.createWriteStream(textBytesFile);
  textBytesWriter.write(textBytes);
  textBytesWriter.end();
  var textBytesRead = fs.createReadStream(textBytesFile).readBytes(6);
  console.log("from-string:" + textBytes.length + ":" + textBytes[0] + ":" + textBytes.toString() + ":" + textBytesRead.toString());
  var hexBytes = Uint8Array.fromString("4142ff", "hex");
  console.log("hex-bytes:" + hexBytes.length + ":" + hexBytes[0] + ":" + hexBytes[1] + ":" + hexBytes[2] + ":" + hexBytes.toString("hex"));
  console.log("utf8-bytes:" + Uint8Array.fromString("kimchi", "utf-8").toString("utf8"));
  var base64Bytes = Uint8Array.fromString("a2ltY2hp", "base64");
  console.log("base64-bytes:" + base64Bytes.length + ":" + base64Bytes.toString() + ":" + base64Bytes.toString("base64"));
  console.log("base64-pad:" + Uint8Array.fromString("QQ==", "base64").toString() + ":" + Uint8Array.fromString("QUI=", "base64").toString());
  var concatBytes = Uint8Array.concat(Uint8Array.fromString("kim"), Uint8Array.fromString("chi"));
  console.log("bytes-concat:" + concatBytes.length + ":" + concatBytes.toString());
  console.log("bytes-concat-method:" + Uint8Array.fromString("jay").concat(Uint8Array.fromString("ess")).toString());
  console.log("bytes-equals-same");
  console.log(concatBytes.equals(Uint8Array.fromString("kimchi")));
  console.log("bytes-equals-diff");
  console.log(concatBytes.equals(Uint8Array.fromString("kim")));
  console.log("bytes-equals-static");
  console.log(Uint8Array.equals(concatBytes, Uint8Array.fromString("kimchi")));
  console.log("bytes-compare-equal:" + Uint8Array.compare(concatBytes, Uint8Array.fromString("kimchi")));
  console.log("bytes-compare-less:" + Uint8Array.fromString("kim").compare(Uint8Array.fromString("kimchi")));
  console.log("bytes-compare-greater:" + Uint8Array.fromString("kimchi").compare(Uint8Array.fromString("kim")));
  console.log("bytes-index-of-byte:" + concatBytes.indexOf(99) + ":" + concatBytes.indexOf(120));
  console.log("bytes-index-of-seq:" + concatBytes.indexOf(Uint8Array.fromString("chi")) + ":" + concatBytes.indexOf(Uint8Array.fromString("hot")));
  console.log("bytes-prefix-suffix:" + concatBytes.startsWith(Uint8Array.fromString("kim")) + ":" + concatBytes.endsWith(Uint8Array.fromString("chi")));
  console.log("bytes-prefix-suffix-byte:" + concatBytes.startsWith(107) + ":" + concatBytes.endsWith(105));
  var setBytes = new Uint8Array(6);
  setBytes.set(Uint8Array.fromString("kim"), 1);
  setBytes.set([65, 66], 4);
  console.log("bytes-set:" + setBytes.length + ":" + setBytes.toString("hex"));
  var copiedBytes = Uint8Array.fromString("abcdef");
  copiedBytes.copyWithin(2, 0, 3);
  console.log("bytes-copy-within:" + copiedBytes.toString());
  var copiedOverlap = Uint8Array.fromString("abcdef");
  copiedOverlap.copyWithin(1, 0, 5);
  console.log("bytes-copy-overlap:" + copiedOverlap.toString());
  var viewBuffer = new ArrayBuffer(9);
  var view = new DataView(viewBuffer);
  view.setUint8(0, 255);
  view.setUint16(1, 4660, false);
  view.setUint16(3, 4660, true);
  view.setUint32(5, 66051, false);
  var viewBytes = new Uint8Array(viewBuffer);
  console.log("data-view-bytes:" + viewBytes.toString("hex"));
  console.log("data-view-read:" + view.getUint8(0) + ":" + view.getUint16(1, false) + ":" + view.getUint16(3, true) + ":" + view.getUint32(5, false));
  var signedBuffer = new ArrayBuffer(8);
  var signedView = new DataView(signedBuffer);
  signedView.setInt8(0, -1);
  signedView.setInt16(1, -2, false);
  signedView.setInt16(3, -3, true);
  signedView.setInt32(4, -123456, false);
  var signedBytes = new Uint8Array(signedBuffer);
  console.log("data-view-signed-bytes:" + signedBytes.toString("hex"));
  console.log("data-view-signed-read:" + signedView.getInt8(0) + ":" + signedView.getInt16(1, false) + ":" + signedView.getInt16(3, true) + ":" + signedView.getInt32(4, false));
  var floatBuffer = new ArrayBuffer(12);
  var floatView = new DataView(floatBuffer);
  floatView.setFloat32(0, 1.5, false);
  floatView.setFloat64(4, -2.5, true);
  var floatBytes = new Uint8Array(floatBuffer);
  console.log("data-view-float-bytes:" + floatBytes.toString("hex"));
  console.log("data-view-float-read:" + floatView.getFloat32(0, false) + ":" + floatView.getFloat64(4, true));

  var badWriter = fs.createWriteStream(path.join("tmp", "missing-dir", "stream.txt"));
  badWriter.on("error", (err) => {
    console.log("write-error:" + err.message);
    return 0;
  });
  console.log(badWriter.errored);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "fs-streams-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"true", "kim", "chi", "null", "finish-count:3", "finish-event:true", "finished", "finished-two", "finished-once", "finished-late", "finished-once-late", "data:kimchi", "ended", "once-data:kimchi", "once-end-count:1", "once-ended", "once-end-after:0", "read-error:", "read-error-two:", "read-error-once:", "error-count:2", "error-event:true", "bytes-write:true", "bytes-read:4:65:0:255:66", "bytes-slice:2:0:255", "bytes-includes:true:false", "bytes-text:AB", "bytes-end:null", "from-string:6:106:jayess:jayess", "hex-bytes:3:65:66:255:4142ff", "utf8-bytes:kimchi", "base64-bytes:6:kimchi:a2ltY2hp", "base64-pad:A:AB", "bytes-concat:6:kimchi", "bytes-concat-method:jayess", "bytes-equals-same", "bytes-equals-diff", "bytes-equals-static", "bytes-compare-equal:0", "bytes-compare-less:-1", "bytes-compare-greater:1", "bytes-index-of-byte:3:-1", "bytes-index-of-seq:3:-1", "bytes-prefix-suffix:true:true", "bytes-prefix-suffix-byte:true:true", "bytes-set:6:006b696d4142", "bytes-copy-within:ababcf", "bytes-copy-overlap:aabcde", "data-view-bytes:ff1234341200010203", "data-view-read:255:4660:4660:66051", "data-view-signed-bytes:fffffefdfffe1dc0", "data-view-signed-read:-1:-2:-3:-123456", "data-view-float-bytes:3fc0000000000000000004c0", "data-view-float-read:1.5:-2.5", "write-error:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected fs stream output to contain %q, got: %s", want, text)
		}
	}
	if strings.Contains(text, "removed-finish") {
		t.Fatalf("expected removed finish listener not to run, got: %s", text)
	}
	if strings.Contains(text, "skipped-specific") {
		t.Fatalf("expected specifically removed finish listener not to run, got: %s", text)
	}
}

func TestBuildExecutableSupportsNativeWrapperImports(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	nativeDir := filepath.Join(workdir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	nativeSource := `#include "jayess_runtime.h"

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { jayess_add } from "./native/math.c";

function main(args) {
  console.log(jayess_add(3, 4));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "7") {
		t.Fatalf("expected native wrapper output to contain 7, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPackageImports(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@demo", "math")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
export function add(a, b) {
  return a + b;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "@demo/math";

function main(args) {
  console.log(add(5, 6));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "pkg-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "11") {
		t.Fatalf("expected package-import output to contain 11, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPathHelperEdgeCases(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  console.log(path.resolve("a", "..", "b"));
  console.log(path.relative("tmp", path.join("tmp", "nested", "file.txt")));
  console.log(path.format(path.parse(path.join("tmp", "nested", "file.txt"))));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "path-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected three output lines, got: %q", string(out))
	}

	expectedResolve := filepath.Clean(filepath.Join(workdir, "b"))
	expectedRelative := filepath.Join("nested", "file.txt")
	expectedFormat := filepath.Join("tmp", "nested", "file.txt")

	if lines[0] != expectedResolve {
		t.Fatalf("expected resolved path %q, got %q", expectedResolve, lines[0])
	}
	if lines[1] != expectedRelative {
		t.Fatalf("expected relative path %q, got %q", expectedRelative, lines[1])
	}
	if lines[2] != expectedFormat {
		t.Fatalf("expected formatted path %q, got %q", expectedFormat, lines[2])
	}
}

func TestBuildObjectSupportsConfiguredCrossTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target build test: %v", err)
	}

	source := `
function main(args) {
  console.log("cross");
  return 0;
}
`
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}
			result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("Compile returned error: %v", err)
			}
			outputPath := filepath.Join(t.TempDir(), targetName+".o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildExecutableSupportsProcessPathAndRecursiveFsEdgeCases(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  fs.mkdir(path.join("tmp", "a", "b"), { recursive: true });
  fs.writeFile(path.join("tmp", "a", "b", "note.txt"), "kimchi");
  fs.copyDir("tmp", "copy");
  console.log(process.arch());
  console.log(process.threadPoolSize());
  console.log(path.sep);
  console.log(path.delimiter);
  console.log(fs.readFile(path.join("copy", "a", "b", "note.txt"), "utf8"));
  console.log(fs.readFile("missing.txt"));
  console.log(fs.stat("missing.txt"));
  console.log(fs.remove("copy", { recursive: true }));
  console.log(fs.exists("copy"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "stdlib-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n")
	if len(lines) < 9 {
		t.Fatalf("expected at least nine output lines, got %q", string(out))
	}
	if lines[0] == "" {
		t.Fatalf("expected process.arch output, got %q", lines[0])
	}
	if lines[1] != "4" {
		t.Fatalf("expected process.threadPoolSize 4, got %q", lines[1])
	}
	if lines[2] != string(filepath.Separator) {
		t.Fatalf("expected path.sep %q, got %q", string(filepath.Separator), lines[2])
	}
	expectedDelimiter := ":"
	if runtime.GOOS == "windows" {
		expectedDelimiter = ";"
	}
	if lines[3] != expectedDelimiter {
		t.Fatalf("expected path.delimiter %q, got %q", expectedDelimiter, lines[3])
	}
	if lines[4] != "kimchi" {
		t.Fatalf("expected copied file contents, got %q", lines[4])
	}
	if lines[5] != "undefined" {
		t.Fatalf("expected missing file read to return undefined, got %q", lines[5])
	}
	if lines[6] != "undefined" {
		t.Fatalf("expected missing file stat to return undefined, got %q", lines[6])
	}
	if lines[7] != "true" {
		t.Fatalf("expected recursive remove success, got %q", lines[7])
	}
	if lines[8] != "false" {
		t.Fatalf("expected removed directory to be absent, got %q", lines[8])
	}
}

func TestBuildExecutableSupportsMathIntrinsics(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main() {
  console.log("math:" + Math.floor(1.8) + ":" + Math.ceil(1.2) + ":" + Math.round(1.5) + ":" + Math.min(1, 2) + ":" + Math.max(1, 2) + ":" + Math.abs(-2) + ":" + Math.pow(2, 3) + ":" + Math.sqrt(9));
  console.log(Math.random() >= 0);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "math-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "math:1:2:2:1:2:2:8:3") {
		t.Fatalf("expected math intrinsic output, got: %s", text)
	}
	if !strings.Contains(text, "true") {
		t.Fatalf("expected Math.random comparison output, got: %s", text)
	}
}

func TestBuildExecutableSupportsUrlAndQuerystring(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main() {
  var parsed = url.parse("https://example.com/api?q=kimchi&lang=en#top");
  console.log("url-host:" + parsed.host);
  console.log("url-path:" + parsed.pathname);
  console.log("url-query:" + parsed.queryObject.q + ":" + parsed.queryObject.lang);
  console.log("url-format:" + url.format({ protocol: "https:", host: "example.com", pathname: "/api", query: "q=kimchi", hash: "top" }));
  var qs = querystring.parse("name=kimchi%20man&spicy=10");
  console.log("query-parse:" + qs.name + ":" + qs.spicy);
  console.log("query-stringify:" + querystring.stringify({ name: "kimchi man" }));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "url-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"url-host:example.com",
		"url-path:/api",
		"url-query:kimchi:en",
		"url-format:https://example.com/api?q=kimchi#top",
		"query-parse:kimchi man:10",
		"query-stringify:name=kimchi%20man",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected url/querystring output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpMessageHelpers(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
  function main() {
  var requestText = http.formatRequest({
    method: "POST",
    path: "/submit",
    version: "HTTP/1.1",
    headers: { Host: "example.com" },
    body: "kimchi=1"
  });
  var request = http.parseRequest(requestText);
  console.log("http-request:" + request.method + ":" + request.path + ":" + request.version + ":" + request.headers.Host + ":" + request.body);
  var responseText = http.formatResponse({
    version: "HTTP/1.1",
    status: 201,
    reason: "Created",
    headers: { "Content-Type": "text/plain" },
    body: "done"
  });
    var response = http.parseResponse(responseText);
    console.log("http-response:" + response.version + ":" + response.status + ":" + response.reason + ":" + response.statusText + ":" + response.ok + ":" + response.headers["Content-Type"] + ":" + response.body);
    console.log("http-response-bytes:" + response.bodyBytes.length + ":" + response.bodyBytes.toString());
    return 0;
  }
  `, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-message-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"http-request:POST:/submit:HTTP/1.1:example.com:kimchi=1",
		"http-response:HTTP/1.1:201:Created:Created:true:text/plain:done",
		"http-response-bytes:4:done",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected http helper output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpClient(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 3)
	go func() {
		for _, expected := range []struct {
			requestLine string
			body        string
			response    string
		}{
			{requestLine: "GET /hello?name=kimchi HTTP/1.1", body: "hello", response: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nhello"},
			{requestLine: "GET /hello?name=kimchi HTTP/1.1", body: "hello chunked", response: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n6\r\nhello \r\n7\r\nchunked\r\n0\r\n\r\n"},
			{requestLine: "POST /submit HTTP/1.1", body: "posted", response: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nposted"},
		} {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			requestText, err := readHTTPTestRequest(conn)
			if err != nil {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- err
				return
			}
			if !strings.Contains(requestText, expected.requestLine) {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("unexpected request line: %s", requestText)
				return
			}
			if !strings.Contains(requestText, "Host: 127.0.0.1") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing host header: %s", requestText)
				return
			}
			if !strings.Contains(requestText, "Connection: close") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing connection header: %s", requestText)
				return
			}
			if strings.Contains(requestText, "POST /submit HTTP/1.1") && !strings.Contains(requestText, "Content-Length: 8") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing content-length header: %s", requestText)
				return
			}
			if _, err := conn.Write([]byte(expected.response)); err != nil {
				conn.Close()
				serverErr <- err
				return
			}
			conn.Close()
			serverErr <- nil
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var response = http.get({ host: "127.0.0.1", port: %d, path: "/hello?name=kimchi" });
  console.log("http-get:" + response.status + ":" + response.reason + ":" + response.statusText + ":" + response.ok + ":" + response.headers["Content-Type"] + ":" + response.body);
  console.log("http-get-bytes:" + response.bodyBytes.length + ":" + response.bodyBytes.toString());
  var fromUrl = http.get("http://127.0.0.1:%d/hello?name=kimchi");
  console.log("http-get-url:" + fromUrl.status + ":" + fromUrl.body);
  var posted = http.request({ url: "http://127.0.0.1:%d/submit", method: "POST", body: "kimchi=1" });
  console.log("http-request-url:" + posted.status + ":" + posted.body);
  return 0;
}
`, port, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 3; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	if !strings.Contains(text, "http-get:200:OK:OK:true:text/plain:hello") {
		t.Fatalf("expected http client output, got: %s", text)
	}
	if !strings.Contains(text, "http-get-bytes:5:hello") {
		t.Fatalf("expected http client body bytes output, got: %s", text)
	}
	if !strings.Contains(text, "http-get-url:200:hello chunked") {
		t.Fatalf("expected http URL get output, got: %s", text)
	}
	if !strings.Contains(text, "http-request-url:200:posted") {
		t.Fatalf("expected http URL request output, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpClientStream(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 2)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				requestText, err := readHTTPTestRequest(conn)
				if err != nil {
					writeHTTPTestFailure(conn)
					serverErr <- err
					return
				}
				switch {
				case strings.Contains(requestText, "GET /headers-first HTTP/1.1"):
					if !strings.Contains(requestText, "Host: 127.0.0.1") {
						writeHTTPTestFailure(conn)
						serverErr <- fmt.Errorf("missing stream host header: %s", requestText)
						return
					}
					if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\nConnection: close\r\n\r\n")); err != nil {
						serverErr <- err
						return
					}
					time.Sleep(700 * time.Millisecond)
					_, _ = conn.Write([]byte("hello"))
					serverErr <- nil
				case strings.Contains(requestText, "POST /stream-body HTTP/1.1"):
					if !strings.Contains(requestText, "Content-Length: 8") {
						writeHTTPTestFailure(conn)
						serverErr <- fmt.Errorf("missing stream request content-length header: %s", requestText)
						return
					}
					if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 10\r\nConnection: close\r\n\r\nhello")); err != nil {
						serverErr <- err
						return
					}
					time.Sleep(60 * time.Millisecond)
					if _, err := conn.Write([]byte("world")); err != nil {
						serverErr <- err
						return
					}
					serverErr <- nil
				default:
					writeHTTPTestFailure(conn)
					serverErr <- fmt.Errorf("unexpected stream request: %s", requestText)
				}
			}(conn)
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var head = http.getStream("http://127.0.0.1:%d/headers-first");
  console.log("http-stream-head:" + head.status + ":" + head.ok + ":" + head.statusText + ":" + typeof head.bodyStream.read);
  head.bodyStream.close();

  var streamed = http.requestStream({ host: "127.0.0.1", port: %d, path: "/stream-body", method: "POST", body: "kimchi=1" });
  var first = streamed.bodyStream.read(5);
  var second = streamed.bodyStream.read(5);
  var done = streamed.bodyStream.read(1);
  console.log("http-stream-body:" + streamed.status + ":" + first + ":" + second + ":" + done + ":" + streamed.bodyStream.readableEnded);
  return 0;
}
`, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-stream-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 2; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	if !strings.Contains(text, "http-stream-head:200:true:OK:function") {
		t.Fatalf("expected streamed header output, got: %s", text)
	}
	if !strings.Contains(text, "http-stream-body:200:hello:world:null:true") {
		t.Fatalf("expected streamed body output, got: %s", text)
	}
	if elapsed >= 650*time.Millisecond {
		t.Fatalf("expected streamed header request to return before delayed body finished, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsHttpClientAsync(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 2)
	go func() {
		for _, expected := range []struct {
			requestLine string
			body        string
			response    string
		}{
			{requestLine: "GET /async?name=kimchi HTTP/1.1", body: "async chunked", response: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n6\r\nasync \r\n7\r\nchunked\r\n0\r\n\r\n"},
			{requestLine: "POST /async-submit HTTP/1.1", body: "async-posted", response: "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nasync-posted"},
		} {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			requestText, err := readHTTPTestRequest(conn)
			if err != nil {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- err
				return
			}
			if !strings.Contains(requestText, expected.requestLine) {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("unexpected async request line: %s", requestText)
				return
			}
			if !strings.Contains(requestText, "Host: 127.0.0.1") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing async host header: %s", requestText)
				return
			}
			if !strings.Contains(requestText, "Connection: close") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing async connection header: %s", requestText)
				return
			}
			if strings.Contains(requestText, "POST /async-submit HTTP/1.1") && !strings.Contains(requestText, "Content-Length: 8") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing async content-length header: %s", requestText)
				return
			}
			if _, err := conn.Write([]byte(expected.response)); err != nil {
				conn.Close()
				serverErr <- err
				return
			}
			conn.Close()
			serverErr <- nil
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main(args) {
  var getPromise = http.getAsync("http://127.0.0.1:%d/async?name=kimchi");
  console.log("http-get-async-promise:" + (typeof getPromise.then));
  var response = await getPromise;
  console.log("http-get-async:" + response.status + ":" + response.ok + ":" + response.statusText + ":" + response.body);
  console.log("http-get-async-bytes:" + response.bodyBytes.length + ":" + response.bodyBytes.toString());
  var posted = await http.requestAsync({ url: "http://127.0.0.1:%d/async-submit", method: "POST", body: "kimchi=1" });
  console.log("http-request-async:" + posted.status + ":" + posted.body);
  return 0;
}
`, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-async-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 2; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	for _, want := range []string{
		"http-get-async-promise:function",
		"http-get-async:200:true:OK:async chunked",
		"http-get-async-bytes:13:async chunked",
		"http-request-async:200:async-posted",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected async http output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpClientStreamAsync(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 2)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				requestText, err := readHTTPTestRequest(conn)
				if err != nil {
					writeHTTPTestFailure(conn)
					serverErr <- err
					return
				}
				switch {
				case strings.Contains(requestText, "GET /headers-first HTTP/1.1"):
					if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\nConnection: close\r\n\r\n")); err != nil {
						serverErr <- err
						return
					}
					time.Sleep(700 * time.Millisecond)
					_, _ = conn.Write([]byte("hello"))
					serverErr <- nil
				case strings.Contains(requestText, "POST /stream-body HTTP/1.1"):
					if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 10\r\nConnection: close\r\n\r\nhello")); err != nil {
						serverErr <- err
						return
					}
					time.Sleep(60 * time.Millisecond)
					if _, err := conn.Write([]byte("world")); err != nil {
						serverErr <- err
						return
					}
					serverErr <- nil
				default:
					writeHTTPTestFailure(conn)
					serverErr <- fmt.Errorf("unexpected async stream request: %s", requestText)
				}
			}(conn)
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var head = await http.getStreamAsync("http://127.0.0.1:%d/headers-first");
  console.log("http-stream-async-head:" + head.status + ":" + head.ok + ":" + head.statusText + ":" + typeof head.bodyStream.read);
  head.bodyStream.close();

  var streamed = await http.requestStreamAsync({ host: "127.0.0.1", port: %d, path: "/stream-body", method: "POST", body: "kimchi=1" });
  var first = streamed.bodyStream.read(5);
  var second = streamed.bodyStream.read(5);
  var done = streamed.bodyStream.read(1);
  console.log("http-stream-async-body:" + streamed.status + ":" + first + ":" + second + ":" + done + ":" + streamed.bodyStream.readableEnded);
  return 0;
}
`, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-stream-async-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 2; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	if !strings.Contains(text, "http-stream-async-head:200:true:OK:function") {
		t.Fatalf("expected async streamed header output, got: %s", text)
	}
	if !strings.Contains(text, "http-stream-async-body:200:hello:world:null:true") {
		t.Fatalf("expected async streamed body output, got: %s", text)
	}
	if elapsed >= 650*time.Millisecond {
		t.Fatalf("expected async streamed header request to return before delayed body finished, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsHttpsClient(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hello":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("secure-hello"))
		case "/echo":
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(r.Method + ":" + string(body)))
		default:
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var insecure = false;
  var syncResult = https.get({ url: "%s/hello", rejectUnauthorized: insecure });
  console.log("https-get:" + syncResult.status + ":" + syncResult.ok + ":" + syncResult.statusText + ":" + syncResult.body);
  console.log("https-get-bytes:" + syncResult.bodyBytes.length + ":" + syncResult.bodyBytes.toString());
  var syncRequest = https.request({ url: "%s/hello", rejectUnauthorized: insecure });
  console.log("https-request-sync:" + syncRequest.status + ":" + syncRequest.body);
  var postRequest = https.request({ url: "%s/echo", method: "POST", body: "kimchi=1", rejectUnauthorized: insecure });
  console.log("https-request-post:" + postRequest.status + ":" + postRequest.body);
  var asyncResult = await https.requestAsync({ url: "%s/hello", rejectUnauthorized: insecure });
  console.log("https-request-async:" + asyncResult.status + ":" + asyncResult.body);
  var asyncPost = await https.requestAsync({ url: "%s/echo", method: "POST", body: "kimchi=2", rejectUnauthorized: insecure });
  console.log("https-request-async-post:" + asyncPost.status + ":" + asyncPost.body);
  return 0;
}
`, server.URL, server.URL, server.URL, server.URL, server.URL), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-client-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"https-get:200:true:OK:secure-hello",
		"https-get-bytes:12:secure-hello",
		"https-request-sync:200:secure-hello",
		"https-request-async:200:secure-hello",
		"https-request-post:200:POST:kimchi=1",
		"https-request-async-post:200:POST:kimchi=2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected https output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpsClientStream(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/headers-first":
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(700 * time.Millisecond)
			_, _ = w.Write([]byte("hello"))
		case "/stream-body":
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || string(body) != "kimchi=1" {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Length", "10")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var insecure = false;
  var head = https.getStream({ url: "%s/headers-first", rejectUnauthorized: insecure });
  console.log("https-stream-head:" + head.status + ":" + head.ok + ":" + head.statusText + ":" + typeof head.bodyStream.read);
  head.bodyStream.close();

  var streamed = https.requestStream({ url: "%s/stream-body", method: "POST", body: "kimchi=1", rejectUnauthorized: insecure });
  var first = streamed.bodyStream.read(5);
  var second = streamed.bodyStream.read(5);
  var done = streamed.bodyStream.read(1);
  console.log("https-stream-body:" + streamed.status + ":" + first + ":" + second + ":" + done + ":" + streamed.bodyStream.readableEnded);
  return 0;
}
`, server.URL, server.URL), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-client-stream-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "https-stream-head:200:true:OK:function") {
		t.Fatalf("expected streamed https header output, got: %s", text)
	}
	if !strings.Contains(text, "https-stream-body:200:hello:world:null:true") {
		t.Fatalf("expected streamed https body output, got: %s", text)
	}
	if elapsed >= 650*time.Millisecond {
		t.Fatalf("expected streamed https header request to return before delayed body finished, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsHttpsRedirectControl(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("secure-redirected"))
		default:
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var insecure = false;
  var followed = https.get({ url: "%s/redirect", rejectUnauthorized: insecure });
  console.log("https-redirect-follow:" + followed.status + ":" + followed.body);
  console.log("https-redirect-follow-meta:" + followed.redirected + ":" + followed.redirectCount + ":" + followed.url);

  var stopped = https.get({ url: "%s/redirect", maxRedirects: 0, rejectUnauthorized: insecure });
  console.log("https-redirect-stop:" + stopped.status + ":" + stopped.statusText + ":" + stopped.headers.Location);
  console.log("https-redirect-stop-meta:" + stopped.redirected + ":" + stopped.redirectCount + ":" + stopped.url);
  return 0;
}
`, server.URL, server.URL), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-redirect-control-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "https-redirect-follow:200:secure-redirected") {
		t.Fatalf("expected followed https redirect output, got: %s", text)
	}
	if !strings.Contains(text, fmt.Sprintf("https-redirect-follow-meta:true:1:%s/final", server.URL)) {
		t.Fatalf("expected followed https redirect metadata, got: %s", text)
	}
	if !strings.Contains(text, "https-redirect-stop:302:Found:/final") {
		t.Fatalf("expected stopped https redirect output, got: %s", text)
	}
	if !strings.Contains(text, fmt.Sprintf("https-redirect-stop-meta:false:0:%s/redirect", server.URL)) {
		t.Fatalf("expected stopped https redirect metadata, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpsClientStreamAsync(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/headers-first":
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(700 * time.Millisecond)
			_, _ = w.Write([]byte("hello"))
		case "/stream-body":
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || string(body) != "kimchi=1" {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Length", "10")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(60 * time.Millisecond)
			_, _ = w.Write([]byte("world"))
		default:
			http.NotFound(w, r)
		}
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var insecure = false;
  var head = await https.getStreamAsync({ url: "%s/headers-first", rejectUnauthorized: insecure });
  console.log("https-stream-async-head:" + head.status + ":" + head.ok + ":" + head.statusText + ":" + typeof head.bodyStream.read);
  head.bodyStream.close();

  var streamed = await https.requestStreamAsync({ url: "%s/stream-body", method: "POST", body: "kimchi=1", rejectUnauthorized: insecure });
  var first = streamed.bodyStream.read(5);
  var second = streamed.bodyStream.read(5);
  var done = streamed.bodyStream.read(1);
  console.log("https-stream-async-body:" + streamed.status + ":" + first + ":" + second + ":" + done + ":" + streamed.bodyStream.readableEnded);
  return 0;
}
`, server.URL, server.URL), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-client-stream-async-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "https-stream-async-head:200:true:OK:function") {
		t.Fatalf("expected async streamed https header output, got: %s", text)
	}
	if !strings.Contains(text, "https-stream-async-body:200:hello:world:null:true") {
		t.Fatalf("expected async streamed https body output, got: %s", text)
	}
	if elapsed >= 650*time.Millisecond {
		t.Fatalf("expected async streamed https header request to return before delayed body finished, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsTlsCapabilitySurface(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main() {
  console.log("tls-cap:" + tls.isAvailable() + ":" + tls.backend());
  console.log("https-cap:" + https.isAvailable() + ":" + https.backend());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "tls-capability-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	wantAvailable := "true"
	wantBackend := "openssl"
	if runtime.GOOS == "windows" {
		wantBackend = "schannel"
	}
	if !strings.Contains(text, "tls-cap:"+wantAvailable+":"+wantBackend) {
		t.Fatalf("expected tls capability output, got: %s", text)
	}
	if !strings.Contains(text, "https-cap:"+wantAvailable+":"+wantBackend) {
		t.Fatalf("expected https capability output, got: %s", text)
	}
}

func TestBuildExecutableSupportsTlsConnect(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	server.TLS = &tls.Config{NextProtos: []string{"jayess-test", "http/1.1"}}
	server.StartTLS()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("Atoi returned error: %v", err)
	}

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var socket = tls.connect({ host: "%s", port: %d, rejectUnauthorized: false, alpnProtocols: ["jayess-test", "http/1.1"] });
  console.log("tls-socket:" + socket.secure + ":" + socket.backend + ":" + socket.connected + ":" + socket.protocol);
  var cert = socket.getPeerCertificate();
  console.log("tls-cert:" + (cert != undefined) + ":" + (cert.subject != "") + ":" + (cert.issuer != "") + ":" + cert.backend + ":" + cert.authorized);
  console.log("tls-cert-detail:" + (cert.subjectCN != "") + ":" + (cert.issuerCN != "") + ":" + (cert.serialNumber != ""));
  console.log("tls-cert-validity:" + (cert.validFrom != "") + ":" + (cert.validTo != ""));
  console.log("tls-cert-sans:" + (cert.subjectAltNames.length > 0));
  console.log("tls-alpn:" + socket.alpnProtocol + ":" + socket.alpnProtocols.length + ":" + socket.alpnProtocols[0] + ":" + socket.alpnProtocols[1]);
  console.log("tls-write:" + socket.write("ping"));
  socket.close();
  console.log("tls-closed:" + socket.closed + ":" + socket.connected);
  return 0;
}
`, serverURL.Hostname(), port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "tls-connect-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	wantTLSBackend := "openssl"
	if runtime.GOOS == "windows" {
		wantTLSBackend = "schannel"
	}
	if !strings.Contains(text, "tls-socket:true:"+wantTLSBackend+":true:TLS") {
		t.Fatalf("expected tls socket output, got: %s", text)
	}
	if !strings.Contains(text, "tls-cert:true:true:true:"+wantTLSBackend+":false") {
		t.Fatalf("expected tls certificate output, got: %s", text)
	}
	if !strings.Contains(text, "tls-cert-detail:true:true:true") {
		t.Fatalf("expected tls certificate detail output, got: %s", text)
	}
	if !strings.Contains(text, "tls-cert-validity:true:true") {
		t.Fatalf("expected tls certificate validity output, got: %s", text)
	}
	if !strings.Contains(text, "tls-cert-sans:true") {
		t.Fatalf("expected tls certificate SAN output, got: %s", text)
	}
	if !strings.Contains(text, "tls-alpn:jayess-test:2:jayess-test:http/1.1") {
		t.Fatalf("expected tls ALPN output, got: %s", text)
	}
	if !strings.Contains(text, "tls-write:true") {
		t.Fatalf("expected tls write output, got: %s", text)
	}
	if !strings.Contains(text, "tls-closed:true:false") {
		t.Fatalf("expected tls close output, got: %s", text)
	}
}

func TestBuildExecutableSupportsTlsTrustConfiguration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	server.TLS = &tls.Config{NextProtos: []string{"http/1.1"}}
	server.StartTLS()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("Atoi returned error: %v", err)
	}

	cert := server.Certificate()
	if cert == nil {
		t.Fatalf("expected server certificate")
	}
	serverName := serverURL.Hostname()
	if len(cert.DNSNames) > 0 {
		serverName = cert.DNSNames[0]
	}

	workdir := t.TempDir()
	caPath := filepath.Join(workdir, "server-cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		t.Fatalf("failed to encode certificate PEM")
	}
	if err := os.WriteFile(caPath, pemBytes, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	source := ""
	source = fmt.Sprintf(`
function main() {
  var socket = tls.connect({
    host: "%s",
    port: %d,
    serverName: "%s",
    caFile: "%s",
    trustSystem: false,
    alpnProtocols: "http/1.1"
  });
  console.log("tls-trust-custom:" + socket.authorized + ":" + socket.alpnProtocol + ":" + socket.protocol);
  socket.close();
  return 0;
}
`, serverURL.Hostname(), port, serverName, caPath)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "tls-trust-config-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "tls-trust-custom:true:http/1.1:TLS") {
		t.Fatalf("expected custom trust configuration output, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpsTrustConfiguration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "secure-trust")
	}))
	server.TLS = &tls.Config{NextProtos: []string{"http/1.1"}}
	server.StartTLS()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	cert := server.Certificate()
	if cert == nil {
		t.Fatalf("expected server certificate")
	}
	serverName := serverURL.Hostname()
	if len(cert.DNSNames) > 0 {
		serverName = cert.DNSNames[0]
	}

	workdir := t.TempDir()
	caPath := filepath.Join(workdir, "https-server-cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		t.Fatalf("failed to encode certificate PEM")
	}
	if err := os.WriteFile(caPath, pemBytes, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	source := ""
	source = fmt.Sprintf(`
function main() {
  var result = https.get({
    url: "%s/hello",
    serverName: "%s",
    caFile: "%s",
    trustSystem: false
  });
  console.log("https-trust-custom:" + result.status + ":" + result.body);
  return 0;
}
`, server.URL, serverName, caPath)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "https-trust-config-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "https-trust-custom:200:secure-trust") {
		t.Fatalf("expected https custom trust output, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpsGetBoundary(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("secure-hello"))
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var insecure = false;
  var ok = https.request({ url: "%s/hello", rejectUnauthorized: insecure });
  console.log("https-get-boundary-ok:" + ok.status + ":" + ok.body);

  try {
    https.get({ url: "%s/hello", method: "POST", body: "kimchi=1", rejectUnauthorized: insecure });
  } catch (err) {
    console.log("https-get-boundary-error:" + err.message);
  }

  try {
    await https.getAsync({ url: "%s/hello", method: "POST", body: "kimchi=1", rejectUnauthorized: insecure });
  } catch (err) {
    console.log("https-get-async-boundary-error:" + err.message);
  }
  return 0;
}
`, server.URL, server.URL, server.URL), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-request-boundary-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"https-get-boundary-ok:200:secure-hello",
		"https-get-boundary-error:HTTPS request bodies and custom methods are not supported yet",
		"https-get-async-boundary-error:HTTPS request bodies and custom methods are not supported yet",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected https get boundary output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpClientContentLengthBeforeClose(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		requestText, err := readHTTPTestRequest(conn)
		if err != nil {
			conn.Close()
			serverErr <- err
			return
		}
		if !strings.Contains(requestText, "GET /slow-close HTTP/1.1") {
			conn.Close()
			serverErr <- fmt.Errorf("unexpected request line: %s", requestText)
			return
		}
		if _, err := conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nhello")); err != nil {
			conn.Close()
			serverErr <- err
			return
		}
		time.Sleep(700 * time.Millisecond)
		conn.Close()
		serverErr <- nil
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var response = http.get("http://127.0.0.1:%d/slow-close");
  console.log("http-content-length:" + response.status + ":" + response.body);
  return 0;
}
`, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-content-length-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server returned error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "http-content-length:200:hello") {
		t.Fatalf("expected content-length http output, got: %s", text)
	}
	if elapsed >= 650*time.Millisecond {
		t.Fatalf("expected client to finish before delayed close, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsHttpClientTimeout(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 2)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			_, err = readHTTPTestRequest(conn)
			if err != nil {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- err
				return
			}
			time.Sleep(500 * time.Millisecond)
			_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\nConnection: close\r\n\r\nhello"))
			conn.Close()
			serverErr <- nil
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var syncResult = http.get({ host: "127.0.0.1", port: %d, path: "/timeout", timeout: 100 });
  console.log("http-timeout-sync:" + syncResult);
  var asyncResult = await http.getAsync({ host: "127.0.0.1", port: %d, path: "/timeout", timeout: 100 });
  console.log("http-timeout-async:" + asyncResult);
  return 0;
}
`, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-timeout-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 2; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	if !strings.Contains(text, "http-timeout-sync:undefined") {
		t.Fatalf("expected sync timeout output, got: %s", text)
	}
	if !strings.Contains(text, "http-timeout-async:undefined") {
		t.Fatalf("expected async timeout output, got: %s", text)
	}
	if elapsed >= 450*time.Millisecond {
		t.Fatalf("expected HTTP timeout requests to fail fast, elapsed=%s output=%s", elapsed, text)
	}
}

func TestBuildExecutableSupportsHttpClientRedirects(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	serverErr := make(chan error, 4)
	go func() {
		expected := []struct {
			requestLine string
			bodyNeedle  string
			response    string
			noBody      bool
		}{
			{
				requestLine: "GET /redirect-sync HTTP/1.1",
				response:    fmt.Sprintf("HTTP/1.1 302 Found\r\nLocation: http://127.0.0.1:%d/final-sync\r\nContent-Length: 0\r\nConnection: close\r\n\r\n", port),
			},
			{
				requestLine: "GET /final-sync HTTP/1.1",
				response:    "HTTP/1.1 200 OK\r\nContent-Length: 10\r\nConnection: close\r\n\r\nredirected",
			},
			{
				requestLine: "POST /redirect-async HTTP/1.1",
				bodyNeedle:  "kimchi=1",
				response:    "HTTP/1.1 303 See Other\r\nLocation: /final-async\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
			},
			{
				requestLine: "GET /final-async HTTP/1.1",
				response:    "HTTP/1.1 200 OK\r\nContent-Length: 16\r\nConnection: close\r\n\r\nasync-redirected",
				noBody:      true,
			},
		}
		for _, want := range expected {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			requestText, err := readHTTPTestRequest(conn)
			if err != nil {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- err
				return
			}
			if !strings.Contains(requestText, want.requestLine) {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("unexpected redirect request line: %s", requestText)
				return
			}
			if !strings.Contains(requestText, "Host: 127.0.0.1") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing redirect host header: %s", requestText)
				return
			}
			if want.bodyNeedle != "" && !strings.Contains(requestText, want.bodyNeedle) {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("missing redirect request body: %s", requestText)
				return
			}
			if want.noBody && strings.Contains(requestText, "Content-Length: 8") {
				writeHTTPTestFailure(conn)
				conn.Close()
				serverErr <- fmt.Errorf("redirect follow-up should not preserve POST body: %s", requestText)
				return
			}
			if _, err := conn.Write([]byte(want.response)); err != nil {
				conn.Close()
				serverErr <- err
				return
			}
			conn.Close()
			serverErr <- nil
		}
	}()

	result, err := compiler.Compile(fmt.Sprintf(`
function main() {
  var syncResult = http.get({ host: "127.0.0.1", port: %d, path: "/redirect-sync" });
  console.log("http-redirect-sync:" + syncResult.status + ":" + syncResult.body);
  console.log("http-redirect-sync-meta:" + syncResult.redirected + ":" + syncResult.redirectCount + ":" + syncResult.url);
  var asyncResult = await http.requestAsync({ url: "http://127.0.0.1:%d/redirect-async", method: "POST", body: "kimchi=1" });
  console.log("http-redirect-async:" + asyncResult.status + ":" + asyncResult.body);
  console.log("http-redirect-async-meta:" + asyncResult.redirected + ":" + asyncResult.redirectCount + ":" + asyncResult.url);
  console.log("http-redirect-ok:" + syncResult.ok + ":" + asyncResult.ok + ":" + syncResult.statusText + ":" + asyncResult.statusText);
  return 0;
}
`, port, port), compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-client-redirect-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	for i := 0; i < 4; i++ {
		if err := <-serverErr; err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	}
	text := string(out)
	if !strings.Contains(text, "http-redirect-sync:200:redirected") {
		t.Fatalf("expected sync redirect output, got: %s", text)
	}
	if !strings.Contains(text, fmt.Sprintf("http-redirect-sync-meta:true:1:http://127.0.0.1:%d/final-sync", port)) {
		t.Fatalf("expected sync redirect metadata output, got: %s", text)
	}
	if !strings.Contains(text, "http-redirect-async:200:async-redirected") {
		t.Fatalf("expected async redirect output, got: %s", text)
	}
	if !strings.Contains(text, fmt.Sprintf("http-redirect-async-meta:true:1:http://127.0.0.1:%d/final-async", port)) {
		t.Fatalf("expected async redirect metadata output, got: %s", text)
	}
	if !strings.Contains(text, "http-redirect-ok:true:true:OK:OK") {
		t.Fatalf("expected redirect ok/statusText output, got: %s", text)
	}
}

func TestBuildExecutableSupportsDnsLookup(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main() {
  var result = dns.lookup("localhost");
  var all = dns.lookupAll("localhost");
  console.log("dns-host:" + result.host);
  console.log("dns-family:" + (result.family === 4 || result.family === 6));
  console.log("dns-address:" + typeof result.address);
  console.log("dns-all:" + (all.length >= 1) + ":" + all[0].host + ":" + typeof all[0].address + ":" + (all[0].family === 4 || all[0].family === 6));
  console.log("dns-empty:" + dns.lookup(""));
  console.log("dns-all-empty:" + dns.lookupAll(""));
  console.log("dns-reverse:" + typeof dns.reverse("127.0.0.1"));
  console.log("dns-reverse-invalid:" + dns.reverse("not an ip"));
  console.log("net-is-ip:" + net.isIP("127.0.0.1") + ":" + net.isIP("::1") + ":" + net.isIP("kimchi"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "dns-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"dns-host:localhost",
		"dns-family:true",
		"dns-address:string",
		"dns-all:true:localhost:string:true",
		"dns-empty:undefined",
		"dns-all-empty:undefined",
		"dns-reverse:string",
		"dns-reverse-invalid:undefined",
		"net-is-ip:4:6:0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected dns lookup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsNetConnect(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		buffer := make([]byte, 4)
		if _, err := io.ReadFull(conn, buffer); err != nil {
			serverErr <- err
			return
		}
		if string(buffer) != "ping" {
			serverErr <- fmt.Errorf("expected ping, got %q", string(buffer))
			return
		}
		if _, err := conn.Write([]byte("pong")); err != nil {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	source := fmt.Sprintf(`
function main() {
  var socket = net.connect({ host: "127.0.0.1", port: %d });
  var removed = () => {
    console.log("socket-removed");
    return 0;
  };
  socket.on("close", () => {
    console.log("socket-close");
    return 0;
  });
  socket.once("close", () => {
    console.log("socket-close-once");
    return 0;
  });
  socket.on("close", removed);
  console.log("socket-listeners-before:" + socket.listenerCount("close"));
  console.log("socket-events-before:" + socket.eventNames().includes("close"));
  socket.off("close", removed);
  console.log("socket-listeners-after:" + socket.listenerCount("close"));
  console.log("socket-connected:" + socket.connected);
  console.log("socket-state-before:" + socket.readable + ":" + socket.writable);
  console.log("socket-peer:" + socket.remoteAddress + ":" + socket.remotePort + ":" + socket.remoteFamily);
  console.log("socket-local:" + typeof socket.localAddress + ":" + (socket.localPort > 0) + ":" + socket.localFamily);
  console.log("socket-address:" + typeof socket.address().address + ":" + (socket.address().port > 0) + ":" + socket.address().family);
  console.log("socket-remote:" + socket.remote().address + ":" + socket.remote().port + ":" + socket.remote().family);
  console.log("socket-options:" + (socket.setNoDelay(true) === socket) + ":" + (socket.setKeepAlive(true) === socket));
  console.log("socket-timeout:" + (socket.setTimeout(250) === socket) + ":" + socket.timeout);
  console.log("socket-bytes-before:" + socket.bytesRead + ":" + socket.bytesWritten);
  console.log("socket-write:" + socket.write("ping"));
  console.log("socket-read:" + socket.read(4));
  console.log("socket-bytes-after:" + socket.bytesRead + ":" + socket.bytesWritten);
  socket.end();
  console.log("socket-closed:" + socket.closed);
  console.log("socket-state-after:" + socket.readable + ":" + socket.writable);
  socket.on("close", () => {
    console.log("socket-close-late");
    return 0;
  });
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "net-connect-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server returned error: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"socket-listeners-before:3",
		"socket-events-before:true",
		"socket-listeners-after:2",
		"socket-connected:true",
		"socket-state-before:true:true",
		fmt.Sprintf("socket-peer:127.0.0.1:%d:4", port),
		"socket-local:string:true:4",
		"socket-address:string:true:4",
		fmt.Sprintf("socket-remote:127.0.0.1:%d:4", port),
		"socket-options:true:true",
		"socket-timeout:true:250",
		"socket-bytes-before:0:0",
		"socket-write:true",
		"socket-read:pong",
		"socket-bytes-after:4:4",
		"socket-close",
		"socket-close-once",
		"socket-closed:true",
		"socket-state-after:false:false",
		"socket-close-late",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected net.connect output to contain %q, got: %s", want, text)
		}
	}
	if strings.Contains(text, "socket-removed") {
		t.Fatalf("expected removed close listener not to run, got: %s", text)
	}
}

func TestBuildExecutableSupportsNetListen(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	port := reserved.Addr().(*net.TCPAddr).Port
	reserved.Close()

	source := fmt.Sprintf(`
function main() {
  var server = net.listen({ host: "127.0.0.1", port: %d });
  var removed = () => {
    console.log("server-removed");
    return 0;
  };
  server.on("close", () => {
    console.log("server-close");
    return 0;
  });
  server.once("close", () => {
    console.log("server-close-once");
    return 0;
  });
  server.on("close", removed);
  console.log("server-listening:" + server.listening);
  console.log("server-port:" + server.port);
  console.log("server-listeners-before:" + server.listenerCount("close"));
  server.off("close", removed);
  console.log("server-listeners-after:" + server.listenerCount("close"));
  console.log("server-events-before:" + server.eventNames().includes("close"));
  console.log("server-timeout:" + (server.setTimeout(400) === server) + ":" + server.timeout);
  console.log("server-address:" + server.address().address + ":" + server.address().port + ":" + server.address().family);
  var socket = server.accept();
  console.log("server-connections:" + server.connectionsAccepted);
  console.log("server-accepted:" + socket.remoteAddress + ":" + socket.remoteFamily);
  console.log("server-local:" + socket.localAddress + ":" + socket.localPort + ":" + socket.localFamily);
  console.log("server-socket-bytes-before:" + socket.bytesRead + ":" + socket.bytesWritten);
  console.log("server-read:" + socket.read(4));
  console.log("server-write:" + socket.write("pong"));
  console.log("server-socket-bytes-after:" + socket.bytesRead + ":" + socket.bytesWritten);
  socket.end();
  server.close();
  console.log("server-closed:" + server.closed);
  server.on("close", () => {
    console.log("server-close-late");
    return 0;
  });
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "net-listen-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	type runResult struct {
		out []byte
		err error
	}
	done := make(chan runResult, 1)
	go func() {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, err := cmd.CombinedOutput()
		done <- runResult{out: out, err: err}
	}()

	var client net.Conn
	for i := 0; i < 40; i++ {
		client, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	defer client.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatalf("client Write returned error: %v", err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(client, buffer); err != nil {
		t.Fatalf("client ReadFull returned error: %v", err)
	}
	if string(buffer) != "pong" {
		t.Fatalf("expected pong from server, got %q", string(buffer))
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	for _, want := range []string{
		"server-listening:true",
		fmt.Sprintf("server-port:%d", port),
		"server-listeners-before:3",
		"server-listeners-after:2",
		"server-events-before:true",
		"server-timeout:true:400",
		fmt.Sprintf("server-address:127.0.0.1:%d:4", port),
		"server-connections:1",
		"server-accepted:127.0.0.1:4",
		fmt.Sprintf("server-local:127.0.0.1:%d:4", port),
		"server-socket-bytes-before:0:0",
		"server-read:ping",
		"server-write:true",
		"server-socket-bytes-after:4:4",
		"server-close",
		"server-close-once",
		"server-closed:true",
		"server-close-late",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected net.listen output to contain %q, got: %s", want, text)
		}
	}
	if strings.Contains(text, "server-removed") {
		t.Fatalf("expected removed server close listener not to run, got: %s", text)
	}
}

func TestBuildExecutableSupportsNetAsyncMethods(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	port := reserved.Addr().(*net.TCPAddr).Port
	reserved.Close()

	source := fmt.Sprintf(`
function main() {
  var server = net.listen({ host: "127.0.0.1", port: %d });
  server.setTimeout(500);
  var acceptPromise = server.acceptAsync();
  console.log("accept-promise:" + (typeof acceptPromise.then));
  var socket = await acceptPromise;
  console.log("async-accepted:" + socket.remoteFamily);
  console.log("async-read:" + await socket.readAsync(4));
  console.log("async-write:" + await socket.writeAsync("pong"));
  console.log("async-socket-bytes:" + socket.bytesRead + ":" + socket.bytesWritten);
  socket.end();
  server.close();
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "net-async-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	type runResult struct {
		out []byte
		err error
	}
	done := make(chan runResult, 1)
	go func() {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, err := cmd.CombinedOutput()
		done <- runResult{out: out, err: err}
	}()

	var client net.Conn
	for i := 0; i < 40; i++ {
		client, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	defer client.Close()

	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatalf("client Write returned error: %v", err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(client, buffer); err != nil {
		t.Fatalf("client ReadFull returned error: %v", err)
	}
	if string(buffer) != "pong" {
		t.Fatalf("expected pong from async server, got %q", string(buffer))
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled async server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	for _, want := range []string{
		"accept-promise:function",
		"async-accepted:4",
		"async-read:ping",
		"async-write:true",
		"async-socket-bytes:4:4",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected net async output to contain %q, got: %s", want, text)
		}
	}
}

func nativeOutputPath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".exe")
	}
	return filepath.Join(dir, name)
}
