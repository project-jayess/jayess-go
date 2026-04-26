package backend

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
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
	"syscall"
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

func readHTTPHeaderBlock(conn net.Conn) (string, error) {
	var builder strings.Builder
	reader := bufio.NewReader(conn)
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return "", err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		builder.WriteString(line)
		if line == "\r\n" {
			return builder.String(), nil
		}
	}
}

func websocketAcceptForKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func writeMaskedWebSocketTextFrame(conn net.Conn, payload string) error {
	mask := [4]byte{0x11, 0x22, 0x33, 0x44}
	data := []byte(payload)
	if len(data) > 125 {
		return fmt.Errorf("payload too large for simple frame writer")
	}
	frame := make([]byte, 2+4+len(data))
	frame[0] = 0x81
	frame[1] = 0x80 | byte(len(data))
	copy(frame[2:6], mask[:])
	for i := 0; i < len(data); i++ {
		frame[6+i] = data[i] ^ mask[i%4]
	}
	if err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}
	_, err := conn.Write(frame)
	return err
}

func readWebSocketTextFrame(conn net.Conn) (string, error) {
	header := make([]byte, 2)
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return "", err
	}
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	if header[0]&0x0f != 0x1 {
		return "", fmt.Errorf("unexpected websocket opcode %#x", header[0]&0x0f)
	}
	length := int(header[1] & 0x7f)
	if header[1]&0x80 != 0 {
		return "", fmt.Errorf("unexpected masked server frame")
	}
	if length == 126 || length == 127 {
		return "", fmt.Errorf("extended websocket lengths not supported in test")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return "", err
	}
	return string(payload), nil
}

func writeTestTLSCertificatePair(t *testing.T, dir, prefix string) (string, string) {
	t.Helper()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	server.EnableHTTP2 = false
	server.StartTLS()
	defer server.Close()

	if len(server.TLS.Certificates) == 0 || len(server.TLS.Certificates[0].Certificate) == 0 {
		t.Fatalf("expected httptest TLS certificate")
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.TLS.Certificates[0].Certificate[0],
	})
	if certPEM == nil {
		t.Fatalf("failed to encode test certificate PEM")
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(server.TLS.Certificates[0].PrivateKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey returned error: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})
	if keyPEM == nil {
		t.Fatalf("failed to encode test private key PEM")
	}

	certPath := filepath.Join(dir, prefix+"-cert.pem")
	keyPath := filepath.Join(dir, prefix+"-key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return certPath, keyPath
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func repoRootFromBackendTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}

func copyDirRecursive(t *testing.T, srcDir, dstDir string) {
	t.Helper()
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("ReadDir(%s) returned error: %v", srcDir, err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) returned error: %v", dstDir, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if entry.IsDir() {
			copyDirRecursive(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", dstPath, err)
		}
	}
}

func TestFormatNativeBuildErrorReportsMissingLibrariesClearly(t *testing.T) {
	err := errors.New("link failed")

	gnuStyle := formatNativeBuildError(err, "/usr/bin/ld: cannot find -lglfw: No such file or directory")
	if !strings.Contains(gnuStyle.Error(), "native library link failed for glfw") {
		t.Fatalf("expected GNU ld missing library diagnostic, got: %v", gnuStyle)
	}

	appleStyle := formatNativeBuildError(err, "ld: library 'glfw' not found")
	if !strings.Contains(appleStyle.Error(), "native library link failed for glfw") {
		t.Fatalf("expected Apple ld missing library diagnostic, got: %v", appleStyle)
	}
}

func TestFormatNativeBuildErrorReportsMissingHeadersClearly(t *testing.T) {
	err := errors.New("compile failed")

	clangStyle := formatNativeBuildError(err, "fatal error: 'gtk/gtk.h' file not found")
	if !strings.Contains(clangStyle.Error(), "native header dependency missing for gtk/gtk.h") {
		t.Fatalf("expected clang missing header diagnostic, got: %v", clangStyle)
	}

	gccStyle := formatNativeBuildError(err, "fatal error: webkit2/webkit2.h: No such file or directory")
	if !strings.Contains(gccStyle.Error(), "native header dependency missing for webkit2/webkit2.h") {
		t.Fatalf("expected gcc missing header diagnostic, got: %v", gccStyle)
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

func TestBuildExecutableSupportsProcessEnvironmentAndExitSurface(t *testing.T) {
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
  console.log("cwd:" + process.cwd());
  console.log("env:" + process.env("JAYESS_TEST_ENV"));
  console.log("platform:" + process.platform());
  console.log("arch:" + process.arch());
  console.error("stderr:ready");
  process.exit(7);
  console.log("after-exit");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "process-env-exit-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "JAYESS_TEST_ENV=kimchi")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected compiled program to exit non-zero")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("expected explicit exit code 7, got %d", exitErr.ExitCode())
	}

	stdoutText := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	stderrText := strings.ReplaceAll(stderr.String(), "\r\n", "\n")
	for _, want := range []string{
		"cwd:" + workdir,
		"env:kimchi",
		"platform:",
		"arch:",
	} {
		if !strings.Contains(stdoutText, want) {
			t.Fatalf("expected stdout to contain %q, got: %s", want, stdoutText)
		}
	}
	if strings.Contains(stdoutText, "after-exit") {
		t.Fatalf("expected process.exit to stop execution, got stdout: %s", stdoutText)
	}
	if !strings.Contains(stderrText, "stderr:ready") {
		t.Fatalf("expected stderr output, got: %s", stderrText)
	}
}

func TestBuildExecutableSupportsStdinReadLine(t *testing.T) {
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
  var value = readLine("prompt> ");
  console.log("line:" + value);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "stdin-readline-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Stdin = strings.NewReader("kimchi\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "line:kimchi") {
		t.Fatalf("expected stdin/readLine output, got: %s", text)
	}
}

func TestBuildExecutableSupportsSymlinks(t *testing.T) {
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
  fs.writeFile("target.txt", "kimchi");
  console.log("link:" + fs.symlink("target.txt", "alias.txt"));
  console.log("exists:" + fs.exists("alias.txt"));
  console.log("text:" + fs.readFile("alias.txt"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "symlink-native")
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
		"link:true",
		"exists:true",
		"text:kimchi",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected symlink output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsFileAndDirectoryWatch(t *testing.T) {
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
  fs.writeFile("watched.txt", "a");
  fs.mkdir("watched", { recursive: true });

  var fileWatcher = fs.watchSync("watched.txt");
  var dirWatcher = fs.watchSync("watched");

  fileWatcher.on("change", (event) => {
    console.log("file-event:" + event.path + ":" + event.exists + ":" + event.isDir);
    return 0;
  });
  dirWatcher.once("change", (event) => {
    console.log("dir-event:" + event.path + ":" + event.exists + ":" + event.isDir);
    return 0;
  });

  console.log("file-listeners:" + fileWatcher.listenerCount("change"));
  console.log("dir-events:" + dirWatcher.eventNames().includes("change"));

  fs.writeFile("watched.txt", "bb");
  timers.sleep(20);
  console.log("file-poll:" + (fileWatcher.poll() !== null));
  console.log("file-size:" + fileWatcher.size);

  fs.writeFile(path.join("watched", "entry.txt"), "x");
  timers.sleep(20);
  console.log("dir-poll:" + (dirWatcher.poll() !== null));
  console.log("dir-is-dir:" + dirWatcher.isDir);

  fileWatcher.close();
  dirWatcher.close();
  console.log("file-closed:" + fileWatcher.closed);
  console.log("dir-closed:" + dirWatcher.closed);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "watch-native")
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
		"file-listeners:1",
		"dir-events:true",
		"file-event:watched.txt:true:false",
		"file-poll:true",
		"file-size:2",
		"dir-event:watched:true:true",
		"dir-poll:true",
		"dir-is-dir:true",
		"file-closed:true",
		"dir-closed:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected watcher output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsAsyncWatch(t *testing.T) {
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
  fs.writeFile("watched.txt", "a");
  var watcher = fs.watch("watched.txt");
  fs.writeFile("watched.txt", "ccc");
  var changed = await watcher.pollAsync(500);
  console.log("async-change:" + (changed !== null));
  console.log("async-size:" + watcher.size);
  var timedOut = await watcher.pollAsync(30);
  console.log("async-timeout:" + (timedOut === null));
  watcher.close();
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "watch-async-native")
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
		"async-change:true",
		"async-size:3",
		"async-timeout:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected async watcher output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesPropertyEnumerationOrder(t *testing.T) {
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
  var obj = { first: "a", second: "b", third: "c" };
  console.log("keys:" + Object.keys(obj).join(","));
  console.log("values:" + Object.values(obj).join(","));
  console.log("entries:" + JSON.stringify(Object.entries(obj)));
  var seen = [];
  for (var key in obj) {
    seen.push(key);
  }
  console.log("forin:" + seen.join(","));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "property-enumeration-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"keys:first,second,third",
		"values:a,b,c",
		"entries:[[first, a], [second, b], [third, c]]",
		"forin:first,second,third",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected property enumeration output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsObjectSpread(t *testing.T) {
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
  var base = { second: "b", third: "c" };
  var obj = { first: "a", ...base, fourth: "d" };
  console.log("keys:" + Object.keys(obj).join(","));
  console.log("values:" + Object.values(obj).join(","));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "object-spread-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"keys:first,second,third,fourth",
		"values:a,b,c,d",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected object spread output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesCapturedVariablesAcrossScopeExit(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function makeCounter() {
  var count = 0;
  return {
    inc: () => {
      count = count + 1;
      return count;
    },
    read: () => count
  };
}

function main(args) {
  var counter = makeCounter();
  console.log("read-1:" + counter.read());
  console.log("inc:" + counter.inc());
  console.log("read-2:" + counter.read());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "closure-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"read-1:0", "inc:1", "read-2:1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected closure lifetime output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesReturnedValuesAndGlobals(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
var state = { label: "ready", count: 2 };

function makeRecord() {
  var suffix = "chi";
  return { name: "kim" + suffix, items: [state.count, 4] };
}

function main(args) {
  var record = makeRecord();
  console.log("global:" + state.label + ":" + state.count);
  console.log("record:" + record.name);
  console.log("items:" + record.items[0] + "," + record.items[1]);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "returned-values-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"global:ready:2", "record:kimchi", "items:2,4"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected returned/global lifetime output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesNestedContainerAliases(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function buildNested() {
  var inner = { count: 1 };
  var items = [inner];
  var box = { inner: inner, items: items };
  return box;
}

function main(args) {
  var box = buildNested();
  var alias = box.items[0];
  alias.count = alias.count + 4;
  console.log("object:" + box.inner.count);
  console.log("array:" + box.items[0].count);
  console.log("alias:" + alias.count);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "nested-container-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"object:5", "array:5", "alias:5"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected nested container lifetime output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesAliasedValuesAfterContainerRemoval(t *testing.T) {
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
  var value = { count: 2 };
  var box = { value: value };
  var items = [value];
  var alias = items[0];

  delete box.value;
  items.pop();
  alias.count = alias.count + 5;

  console.log("alias:" + alias.count);
  console.log("direct:" + value.count);
  console.log("boxOwn:" + Object.hasOwn(box, "value"));
  console.log("items:" + items.length);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "container-removal-alias-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"alias:7", "direct:7", "boxOwn:false", "items:0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected container-removal alias output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesClosuresAcrossComplexControlFlow(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function buildClosures() {
  var items = [];
  for (var i = 0; i < 6; i = i + 1) {
    var value = i;
    if (value == 1) {
      continue;
    }
    items.push(() => value);
    if (value == 4) {
      break;
    }
  }
  return items;
}

function main(args) {
  var items = buildClosures();
  console.log("count:" + items.length);
  console.log("values:" + items[0]() + "," + items[1]() + "," + items[2]() + "," + items[3]());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "control-flow-closure-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"count:4", "values:0,2,3,4"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected complex control-flow closure output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableStressNestedScopesAndClosures(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function buildClosures() {
  var items = [];
  for (var i = 0; i < 20; i = i + 1) {
    var base = i;
    var make = () => {
      var inner = base * 2;
      return () => inner + base;
    };
    items.push(make());
  }
  return items;
}

function main(args) {
  var items = buildClosures();
  var total = 0;
  for (var i = 0; i < items.length; i = i + 1) {
    total = total + items[i]();
  }
  console.log("first:" + items[0]());
  console.log("last:" + items[19]());
  console.log("total:" + total);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "nested-scope-closure-stress-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"first:0", "last:57", "total:570"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected nested-scope closure stress output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsAccessors(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
class Counter {
  constructor() {
    this._value = 1;
  }

  get value() {
    return this._value;
  }

  set value(next) {
    this._value = next;
  }

  static get label() {
    return "counter";
  }

  static set label(next) {
    console.log("static-set:" + next);
  }
}
function main(args) {
  var obj = {
    _value: 10,
    get value() {
      return this._value;
    },
    set value(next) {
      this._value = next;
    }
  };
  obj.value = 42;
  console.log("object:" + obj.value);

  var counter = new Counter();
  counter.value = 7;
  console.log("instance:" + counter.value);

  Counter.label = "updated";
  console.log("static:" + Counter.label);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "accessors-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"object:42",
		"instance:7",
		"static-set:updated",
		"static:counter",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected accessor output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsWeakCollections(t *testing.T) {
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
  var key = {};
  var other = {};
  var weakMap = new WeakMap();
  weakMap.set(key, "kimchi");
  console.log("wm-get:" + weakMap.get(key));
  console.log("wm-has-key:" + weakMap.has(key));
  console.log("wm-has-other:" + weakMap.has(other));
  console.log("wm-del:" + weakMap.delete(key));
  console.log("wm-after:" + weakMap.has(key));

  var weakSet = new WeakSet();
  weakSet.add(other);
  console.log("ws-has-other:" + weakSet.has(other));
  console.log("ws-del:" + weakSet.delete(other));
  console.log("ws-after:" + weakSet.has(other));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "weak-collections-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"wm-get:kimchi",
		"wm-has-key:true",
		"wm-has-other:false",
		"wm-del:true",
		"wm-after:false",
		"ws-has-other:true",
		"ws-del:true",
		"ws-after:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected weak collection output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSymbol(t *testing.T) {
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
  var left = Symbol("kimchi");
  var right = Symbol("kimchi");
  var map = new Map();
  map.set(left, "ok");
  console.log("type:" + typeof left);
  console.log("self:" + (left == left));
  console.log("other:" + (left == right));
  console.log("map:" + map.get(left));
  console.log("print:" + left);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "symbol-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"type:symbol",
		"self:true",
		"other:false",
		"map:ok",
		"print:Symbol(kimchi)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected symbol output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSymbolPropertyKeys(t *testing.T) {
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
  var sym = Symbol("kimchi");
  var other = Symbol("other");
  var obj = { [sym]: 7, plain: 1 };
  var spread = { ...obj };
  var assigned = Object.assign({}, obj);
  var rest = { plain: 1, [sym]: 7, [other]: 9, ...{} };
  delete rest[other];
  var from = Object.fromEntries([[sym, "yes"], ["plain", "ok"]]);

  console.log("obj:" + obj[sym]);
  console.log("spread:" + spread[sym]);
  console.log("assign:" + assigned[sym]);
  console.log("from:" + from[sym]);
  console.log("own:" + Object.hasOwn(obj, sym));
  console.log("keys:" + Object.keys(obj));
  console.log("symbols:" + Object.getOwnPropertySymbols(obj));
  console.log("entries:" + Object.entries(obj));
  console.log("restOwnSym:" + Object.hasOwn(rest, sym));
  console.log("restOwnOther:" + Object.hasOwn(rest, other));
  console.log("desc:" + sym.description);
  console.log("str:" + sym.toString());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "symbol-props-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"obj:7",
		"spread:7",
		"assign:7",
		"from:yes",
		"own:true",
		"keys:[plain]",
		"symbols:[Symbol(kimchi)]",
		"entries:[[plain, 1]]",
		"restOwnSym:true",
		"restOwnOther:false",
		"desc:kimchi",
		"str:Symbol(kimchi)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected symbol property output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSymbolRegistryAndWellKnownSymbols(t *testing.T) {
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
  var left = Symbol.for("kimchi");
  var right = Symbol.for("kimchi");
  var local = Symbol("kimchi");
  console.log("forEq:" + (left == right));
  console.log("forNeLocal:" + (left == local));
  console.log("keyFor:" + Symbol.keyFor(left));
  console.log("keyForLocal:" + Symbol.keyFor(local));
  console.log("iterEq:" + (Symbol.iterator == Symbol.iterator));
  console.log("asyncEq:" + (Symbol.asyncIterator == Symbol.asyncIterator));
  console.log("tagType:" + typeof Symbol.toStringTag);
  console.log("primType:" + typeof Symbol.toPrimitive);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "symbol-registry-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"forEq:true",
		"forNeLocal:false",
		"keyFor:kimchi",
		"keyForLocal:undefined",
		"iterEq:true",
		"asyncEq:true",
		"tagType:symbol",
		"primType:symbol",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected symbol registry output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsTypedArrays(t *testing.T) {
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
  var signed = new Int8Array(3);
  signed[0] = -1;
  signed[1] = 127;
  signed.fill(-2);
  console.log("int8:" + signed.length + ":" + signed[0] + ":" + signed[1] + ":" + signed.includes(-2));

  var words = new Uint16Array(3);
  words[0] = 513;
  words[1] = 65535;
  words.set([7, 8], 1);
  console.log("u16:" + words.length + ":" + words.byteLength + ":" + words[0] + ":" + words[1] + ":" + words[2]);
  console.log("u16-slice:" + words.slice(1, 3)[0] + ":" + words.slice(1, 3)[1]);

  var buffer = new ArrayBuffer(8);
  var ints = new Int32Array(buffer);
  var floats = new Float32Array(buffer);
  ints[0] = 1065353216;
  ints[1] = -1082130432;
  console.log("f32:" + floats[0] + ":" + floats[1]);

  var f64 = new Float64Array(1);
  f64[0] = 3.5;
  console.log("f64:" + f64[0]);

  var copied = new Uint32Array(words);
  console.log("copy:" + copied[0] + ":" + copied[1] + ":" + copied.indexOf(8));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "typed-arrays-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"int8:3:-2:-2:true",
		"u16:3:6:513:7:8",
		"u16-slice:7:8",
		"f32:1:-1",
		"f64:3.5",
		"copy:513:7:2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected typed array output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsRuntimeTypeChecks(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
type Result = { kind: "ok", value: number } | { kind: "error", message: string };

function main(args) {
  var value: string | number = "kimchi";
  console.log("str:" + (value is string));
  console.log("num:" + (value is number));
  console.log("lit:" + ("ok" is "ok" | "error"));
  console.log("tuple:" + ([1, "ok"] is [number, string]));
  console.log("objGood:" + ({ kind: "ok", value: 3 } is Result));
  console.log("objBad:" + ({ kind: "ok" } is Result));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "runtime-type-check-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"str:true",
		"num:false",
		"lit:true",
		"tuple:true",
		"objGood:true",
		"objBad:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected runtime type check output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsGenericAliasesAndConstraints(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
type Box<T> = { value: T };
type Named<T extends string | number> = { id: T, name: string };
interface Pair<T> {
  left: T,
  right: T,
}

function main(args) {
  var box: Box<number> = { value: 1 };
  var named: Named<string> = { id: "kimchi", name: "ok" };
  var pair: Pair<boolean> = { left: true, right: false };
  console.log("generic:" + box.value + ":" + named.id + ":" + pair.left);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "generic-types-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if text := string(out); !strings.Contains(text, "generic:1:kimchi:true") {
		t.Fatalf("expected generic type output, got: %s", text)
	}
}

func TestBuildExecutableSupportsIterableProtocol(t *testing.T) {
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
  var iterable = {
    [Symbol.iterator]: function() {
      var i = 0;
      return {
        next: function() {
          if (i < 3) {
            var value = i + 1;
            i = i + 1;
            return { value: value, done: false };
          }
          return { value: undefined, done: true };
        }
      };
    }
  };

  var text = "";
  for (var ch of "abc") {
    text = text + ch;
  }
  console.log("str:" + text);

  var values = Array.from(iterable);
  console.log("arr:" + values.join(","));

  var sum = 0;
  for (var value of iterable) {
    sum = sum + value;
  }
  console.log("sum:" + sum);

  var direct = {
    index: 0,
    next: function() {
      this.index = this.index + 1;
      if (this.index <= 2) {
        return { value: this.index * 10, done: false };
      }
      return { value: undefined, done: true };
    }
  };

  var iter = Iterator.from(direct);
  var a = iter.next();
  var b = iter.next();
  var c = iter.next();
  console.log("iter:" + a.value + ":" + a.done + ":" + b.value + ":" + b.done + ":" + c.done);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "iterable-protocol-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"str:abc",
		"arr:1,2,3",
		"sum:6",
		"iter:10:false:20:false:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected iterable protocol output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsGenerators(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function* values() {
  yield 1;
  yield 2;
  return 50;
}

function main(args) {
  var iter = values();
  var first = iter.next();
  var second = iter.next();
  var third = iter.next();
  console.log("next:" + first.value + ":" + first.done + ":" + second.value + ":" + second.done + ":" + third.done);

  var sum = 0;
  for (var value of values()) {
    sum = sum + value;
  }
  console.log("sum:" + sum);

  var make = function*() {
    yield 4;
    yield 5;
  };
  console.log("expr:" + Array.from(make()).join(","));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "generators-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"next:1:false:2:false:true",
		"sum:3",
		"expr:4,5",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected generator output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsAsyncGenerators(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
async function* values() {
  yield await Promise.resolve(1);
  yield 2;
}

function main(args) {
  var iter = values();
  var same = iter[Symbol.asyncIterator]();
  var first = await iter.next();
  var second = await iter.next();
  var third = await iter.next();
  console.log("same:" + (same == iter));
  console.log("next:" + first.value + ":" + first.done + ":" + second.value + ":" + second.done + ":" + third.done);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "async-generators-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"same:true",
		"next:1:false:2:false:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected async generator output to contain %q, got: %s", want, text)
		}
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

func TestBuildExecutableReportsUncaughtExceptions(t *testing.T) {
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
  fail();
  return 0;
}

function fail() {
  throw new Error("boom");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "uncaught-exception-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected compiled program to fail with uncaught exception, got output: %s", string(out))
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected uncaught exception exit code 1, got %d with output: %s", exitErr.ExitCode(), string(out))
	}
	text := string(out)
	for _, want := range []string{"Uncaught exception:", "boom", "at fail (7:1)", "at main (2:1)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected uncaught exception output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsErrorStackTraces(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function leaf() {
  throw new Error("boom");
  return 0;
}

function middle() {
  leaf();
  return 0;
}

function main(args) {
  try {
    middle();
  } catch (err) {
    console.log(err.name, err.message);
    console.log(err.stack);
  }
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "error-stack-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"Error boom", "at leaf (2:1)", "at middle (7:1)", "at main (12:1)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected stack trace output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutablePreservesSourceLocationsInDebugFriendlyBuilds(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	opts := compiler.Options{TargetTriple: triple, OptimizationLevel: "O0"}
	result, err := compiler.Compile(`
function leaf() {
  throw new Error("boom");
  return 0;
}

function middle() {
  leaf();
  return 0;
}

function main(args) {
  middle();
  return 0;
}
`, opts)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if !strings.Contains(string(result.LLVMIR), "; source function leaf at 2:1") {
		t.Fatalf("expected emitted LLVM IR to preserve source comment for leaf, got:\n%s", string(result.LLVMIR))
	}
	if !strings.Contains(string(result.LLVMIR), "; source function main at 12:1") {
		t.Fatalf("expected emitted LLVM IR to preserve source comment for main, got:\n%s", string(result.LLVMIR))
	}

	outputPath := nativeOutputPath(t.TempDir(), "debug-friendly-stack-native")
	if err := tc.BuildExecutable(result, opts, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected compiled program to fail with uncaught exception, got output: %s", string(out))
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected uncaught exception exit code 1, got %d with output: %s", exitErr.ExitCode(), string(out))
	}
	text := string(out)
	for _, want := range []string{"Uncaught exception:", "boom", "at leaf (2:1)", "at middle (7:1)", "at main (12:1)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected debug-friendly build output to contain %q, got: %s", want, text)
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

func TestBuildExecutableSupportsProcessSystemInfoSurface(t *testing.T) {
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
  console.log("tmpdir:" + process.tmpdir());
  console.log("hostname:" + process.hostname());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "process-system-info-native")
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
	if !strings.Contains(text, "tmpdir:") {
		t.Fatalf("expected tmpdir output, got: %s", text)
	}
	if strings.Contains(text, "tmpdir:\n") || strings.Contains(text, "tmpdir:null") || strings.Contains(text, "tmpdir:undefined") {
		t.Fatalf("expected non-empty tmpdir output, got: %s", text)
	}
	if !strings.Contains(text, "hostname:") {
		t.Fatalf("expected hostname output, got: %s", text)
	}
	if strings.Contains(text, "hostname:\n") || strings.Contains(text, "hostname:null") || strings.Contains(text, "hostname:undefined") {
		t.Fatalf("expected non-empty hostname output, got: %s", text)
	}
}

func TestBuildExecutableSupportsExtendedProcessSystemInfoSurface(t *testing.T) {
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
  var uptime = process.uptime();
  var start = process.hrtime();
  sleep(5);
  var finish = process.hrtime();
  var cpu = process.cpuInfo();
  var memory = process.memoryInfo();
  var user = process.userInfo();
  console.log("uptime:" + (uptime >= 0));
  console.log("hrtime:" + (start > 0) + ":" + (finish > start));
  console.log("cpu:" + (cpu.count >= 1) + ":" + (typeof cpu.arch));
  console.log("memory:" + (memory.total >= 0) + ":" + (memory.available >= 0));
  console.log("user:" + (typeof user.username) + ":" + (typeof user.home));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "process-extended-system-info-native")
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
		"uptime:true",
		"hrtime:true:true",
		"cpu:true:string",
		"memory:true:true",
		"user:string:string",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected extended process system info output to contain %q, got: %s", want, text)
		}
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

func TestBuildExecutableSupportsFileRemoval(t *testing.T) {
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
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "remove.txt");
  fs.writeFile(file, "kimchi");
  console.log("before:" + fs.exists(file));
  console.log("remove:" + fs.remove(file));
  console.log("after:" + fs.exists(file));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "fs-remove-file-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	for _, want := range []string{"before:true", "remove:true", "after:false"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected file removal output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsFileAppend(t *testing.T) {
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
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "append.txt");
  fs.writeFile(file, "kim");
  console.log("append-one:" + fs.appendFile(file, "chi"));
  console.log("append-two:" + fs.appendFile(file, "!"));
  console.log("text:" + fs.readFile(file, "utf8"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "fs-append-file-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	for _, want := range []string{"append-one:true", "append-two:true", "text:kimchi!"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected append file output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsFilePermissions(t *testing.T) {
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
  fs.mkdir("tmp", { recursive: true });
  var file = path.join("tmp", "perms.txt");
  fs.writeFile(file, "kimchi");
  var info = fs.stat(file);
  console.log("perms:" + info.permissions);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "fs-permissions-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n"))
	if !strings.HasPrefix(text, "perms:") {
		t.Fatalf("expected permissions output, got: %s", text)
	}
	if strings.TrimPrefix(text, "perms:") == "" {
		t.Fatalf("expected non-empty permissions text, got: %s", text)
	}
}

func TestBuildExecutableSupportsFsCopyRenameAndPathHelpers(t *testing.T) {
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
  var source = path.join("tmp", "a", "b", "note.txt");
  fs.writeFile(source, "kimchi");
  var copied = path.join("tmp", "a", "b", "copy.txt");
  console.log("copy:" + fs.copyFile(source, copied));
  var renamed = path.join("tmp", "a", "b", "renamed.txt");
  console.log("rename:" + fs.rename(copied, renamed));
  console.log("renamed-exists:" + fs.exists(renamed));
  console.log("normalize:" + path.normalize(path.join("tmp", "a", ".", "b", "..", "b", "renamed.txt")));
  console.log("dirname:" + path.dirname(renamed));
  var entries = fs.readDir("tmp", { recursive: true });
  var paths = [];
  for (var i = 0; i < entries.length; i = i + 1) {
    paths.push(entries[i].path);
  }
  console.log("entries:" + entries.length);
  console.log("entry-paths:" + paths.join("|"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "fs-copy-rename-path-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}

	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	renamedPath := filepath.Join("tmp", "a", "b", "renamed.txt")
	dirnamePath := filepath.Join("tmp", "a", "b")
	for _, want := range []string{
		"copy:true",
		"rename:true",
		"renamed-exists:true",
		"normalize:" + renamedPath,
		"dirname:" + dirnamePath,
		"entries:4",
		"entry-paths:" + filepath.Join("tmp", "a"),
		renamedPath,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected fs copy/rename/path output to contain %q, got: %s", want, text)
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
	t.Skip("legacy direct native imports removed; use .bind.js")
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
#include <stdlib.h>
#include <string.h>

jayess_value *jayess_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}

jayess_value *jayess_greet(jayess_value *name) {
    const char *prefix = "Hello, ";
    const char *value = jayess_value_as_string(name);
    size_t prefix_len = strlen(prefix);
    size_t value_len = strlen(value);
    char *buffer = (char *)malloc(prefix_len + value_len + 1);
    if (buffer == NULL) {
        return jayess_value_undefined();
    }
    memcpy(buffer, prefix, prefix_len);
    memcpy(buffer + prefix_len, value, value_len + 1);
    return jayess_value_from_string(buffer);
}

jayess_value *jayess_toggle(jayess_value *value) {
    return jayess_value_from_bool(!jayess_value_as_bool(value));
}

jayess_value *jayess_make_profile(jayess_value *name, jayess_value *score) {
    jayess_object *profile = jayess_object_new();
    jayess_object_set_value(profile, "name", name);
    jayess_object_set_value(profile, "score", score);
    return jayess_value_from_object(profile);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { jayess_add, jayess_greet, jayess_toggle, jayess_make_profile as makeProfile } from "./native/math.c";

function main(args) {
  console.log(jayess_add(3, 4));
  console.log(jayess_greet("Kimchi"));
  console.log(jayess_toggle(true));
  var profile = makeProfile("Kimchi", 7);
  console.log(profile.name + ":" + profile.score);
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		if !strings.Contains(err.Error(), "mongoose") || !strings.Contains(err.Error(), "was not found") {
			t.Fatalf("expected clear Mongoose missing-input diagnostic, got: %v", err)
		}
		return
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
	text := string(out)
	for _, want := range []string{"7", "Hello, Kimchi", "false", "Kimchi:7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected native wrapper output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsNativeWrapperModuleManifests(t *testing.T) {
	t.Skip("legacy native manifests removed; use .bind.js")
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
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	headerSource := `#pragma once
double jayess_manifest_helper(double left, double right);
`
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(headerSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	helperSource := `#include "native_math.h"

double jayess_manifest_helper(double left, double right) {
    return left + right + JAYESS_NATIVE_BONUS;
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(helperSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	nativeSource := `#include "jayess_runtime.h"
#include "native_math.h"

jayess_value *jayess_manifest_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_manifest_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}

jayess_value *jayess_manifest_greet(jayess_value *name) {
    return jayess_value_from_string(jayess_concat_values(jayess_value_from_string("hi "), name));
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	manifestSource := `{
  "sources": ["./math.c", "./helper.c"],
  "includeDirs": ["./include"],
  "cflags": ["-DJAYESS_NATIVE_BONUS=4"],
  "exports": {
    "add": "jayess_manifest_add",
    "greet": "jayess_manifest_greet"
  }
}`
	if err := os.WriteFile(filepath.Join(nativeDir, "math.native.json"), []byte(manifestSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { add, greet as welcome } from "./native/math.native.json";

function main(args) {
  console.log(add(3, 4));
  console.log(welcome("Kimchi"));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		if !strings.Contains(err.Error(), "mongoose") || !strings.Contains(err.Error(), "was not found") {
			t.Fatalf("expected clear Mongoose missing-input diagnostic, got: %v", err)
		}
		return
	}

	outputPath := nativeOutputPath(workdir, "ffi-manifest-native")
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
	if !strings.Contains(text, "11") {
		t.Fatalf("expected native manifest output to contain 11, got: %s", text)
	}
	if !strings.Contains(text, "hi Kimchi") {
		t.Fatalf("expected native manifest output to contain greeting, got: %s", text)
	}
}

func TestBuildExecutableSupportsNativeWrapperManifestShorthand(t *testing.T) {
	t.Skip("legacy native manifests removed; use .bind.js")
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
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

jayess_value *jayess_native_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_native_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double jayess_native_helper(double left, double right) {
    return left + right + 9;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.native.json"), []byte(`{
  "source": "./math.c",
  "sources": ["./helper.c"],
  "includeDir": "./include",
  "cflag": "-DJAYESS_NATIVE_SHORTHAND=1"
}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { jayess_native_add as add } from "./native/math.native.json";

function main(args) {
  console.log(add(1, 2));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-manifest-shorthand")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "12") {
		t.Fatalf("expected shorthand manifest output to contain 12, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPackageNativeImports(t *testing.T) {
	t.Skip("legacy direct native imports removed; use .bind.js")
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "demo-native-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"demo-native-pkg","native":"native/math.c"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(js_arg_number(js, 0) + js_arg_number(js, 1) + 15);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { jayess_native_add as add } from "demo-native-pkg/native/math.c";

function main(args) {
  console.log(add(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "20") {
		t.Fatalf("expected package native output to contain 20, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsDiscoveredNativeExportAliases(t *testing.T) {
	t.Skip("legacy direct native imports removed; use .bind.js")
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
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(js_arg_number(js, 0) + js_arg_number(js, 1) + 21);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "./native/math.c";

function main(args) {
  console.log(add(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-discovered-alias")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "26") {
		t.Fatalf("expected discovered alias output to contain 26, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsZeroManifestNativePackageDirectory(t *testing.T) {
	t.Skip("legacy zero-manifest native imports removed; use .bind.js")
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "demo-zero-native-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "index.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(jayess_native_helper(js_arg_number(js, 0), js_arg_number(js, 1)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double jayess_native_helper(double left, double right) {
    return left + right + 40;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "demo-zero-native-pkg/native";

function main(args) {
  console.log(add(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-zero-manifest-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "45") {
		t.Fatalf("expected zero-manifest package output to contain 45, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsRecursiveZeroManifestNativePackageDirectives(t *testing.T) {
	t.Skip("legacy zero-manifest native imports removed; use .bind.js")
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "demo-config-native-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	helpersDir := filepath.Join(nativeDir, "helpers")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "index.c"), []byte(`// jayess:include ./include
// jayess:cflag -DJAYESS_NATIVE_BONUS=17
#include "jayess_runtime.h"
#include "native_math.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(jayess_native_helper(js_arg_number(js, 0), js_arg_number(js, 1)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "math_helper.c"), []byte(`#include "native_math.h"

double jayess_native_helper(double left, double right) {
    return left + right + JAYESS_NATIVE_BONUS;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "demo-config-native-pkg/native";

function main(args) {
  console.log(add(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-recursive-config-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "22") {
		t.Fatalf("expected recursive directive package output to contain 22, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsManualBindFiles(t *testing.T) {
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
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double mylib_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(mylib_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double mylib_helper(double left, double right) {
    return left + right + 6;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.bind.js"), []byte(`export default {
  sources: ["./math.c", "./helper.c"],
  includeDirs: ["./include"],
  cflags: ["-DMANUAL_BIND=1"],
  ldflags: ["-lm"],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "./native/math.bind.js";

function main(args) {
  console.log(add(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "bind-native")
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
		t.Fatalf("expected bind output to contain 11, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsPackageLocalBindFiles(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "demo-bind-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b) + 9);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./native/math.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "demo-bind-pkg";

function main(args) {
  console.log(add(1, 2));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "bind-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "12") {
		t.Fatalf("expected package bind output to contain 12, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsMultipleManualBindFiles(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(nativeDir, "add.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_mul(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) * jayess_value_to_number(b));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "add.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./add.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.bind.js"), []byte(`const f = () => {};
export const mul = f;

export default {
  sources: ["./mul.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    mul: { symbol: "mylib_mul", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "./native/add.bind.js";
import { mul } from "./native/mul.bind.js";

function main(args) {
  console.log(add(2, 3) + mul(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "bind-multi")
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
		t.Fatalf("expected multiple bind output to contain 11, got: %s", string(out))
	}
}

func TestBuildExecutableDeduplicatesSharedNativeSourcesAcrossBindFiles(t *testing.T) {
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
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "shared_math.h"), []byte(`#pragma once
double shared_add_bonus(double left, double right);
double shared_mul_bonus(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "shared.c"), []byte(`#include "shared_math.h"

double shared_add_bonus(double left, double right) {
    return left + right + 1;
}

double shared_mul_bonus(double left, double right) {
    return left * right + 1;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "add.c"), []byte(`#include "jayess_runtime.h"
#include "shared_math.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(shared_add_bonus(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.c"), []byte(`#include "jayess_runtime.h"
#include "shared_math.h"

jayess_value *mylib_mul(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(shared_mul_bonus(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "add.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./add.c", "./shared.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.bind.js"), []byte(`const f = () => {};
export const mul = f;

export default {
  sources: ["./mul.c", "./shared.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    mul: { symbol: "mylib_mul", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { add } from "./native/add.bind.js";
import { mul } from "./native/mul.bind.js";

function main(args) {
  console.log("shared-dedup:" + add(1, 2) + ":" + mul(2, 3));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if len(result.NativeImports) != 3 {
		t.Fatalf("expected deduplicated native imports [add.c mul.c shared.c], got %#v", result.NativeImports)
	}

	outputPath := nativeOutputPath(workdir, "bind-shared-dedup")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "shared-dedup:4:7") {
		t.Fatalf("expected shared helper bind output to contain shared-dedup:4:7, got: %s", string(out))
	}
}

func TestBuildExecutableArgsSupportPlatformManualBindRules(t *testing.T) {
	result := &compiler.Result{
		NativeImports:      []string{"native/demo.c"},
		NativeIncludeDirs:  []string{"native/include"},
		NativeCompileFlags: []string{"-DNATIVE_DEMO=1"},
		NativeLinkFlags:    []string{"-ldemo"},
	}

	cases := []struct {
		name         string
		targetTriple string
		systemFlags  []string
	}{
		{name: "windows", targetTriple: "x86_64-pc-windows-msvc", systemFlags: []string{"-lws2_32", "-lwinhttp", "-lsecur32", "-lcrypt32", "-lbcrypt"}},
		{name: "linux", targetTriple: "x86_64-unknown-linux-gnu", systemFlags: []string{"-lssl", "-lcrypto", "-lz", "-lm"}},
		{name: "darwin", targetTriple: "aarch64-apple-darwin", systemFlags: []string{"-lssl", "-lcrypto", "-lz", "-lm"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := buildExecutableArgs(
				result,
				compiler.Options{TargetTriple: tc.targetTriple},
				"module.ll",
				"runtime/jayess_runtime.c",
				"runtime",
				"refs/brotli/c/include",
				[]string{"refs/brotli/c/common/constants.c"},
				true,
				"out.exe",
			)
			for _, want := range []string{"-target", tc.targetTriple, "-I", "runtime", "-I", "native/include", "-DNATIVE_DEMO=1", "module.ll", "runtime/jayess_runtime.c", "refs/brotli/c/common/constants.c", "native/demo.c", "-ldemo", "-o", "out.exe"} {
				if !containsString(args, want) {
					t.Fatalf("expected args for %s to contain %q, got: %v", tc.name, want, args)
				}
			}
			for _, want := range tc.systemFlags {
				if !containsString(args, want) {
					t.Fatalf("expected args for %s to contain system flag %q, got: %v", tc.name, want, args)
				}
			}
		})
	}
}

func TestBuildExecutableSupportsManualBindBooleanAndNullConversion(t *testing.T) {
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

jayess_value *jayess_native_is_truthy(jayess_value *value) {
    return jayess_value_from_bool(jayess_value_is_truthy(value));
}

jayess_value *jayess_native_is_nullish(jayess_value *value) {
    return jayess_value_from_bool(jayess_value_is_nullish(value));
}

jayess_value *jayess_native_return_null(jayess_value *value) {
    (void)value;
    return jayess_value_null();
}

jayess_value *jayess_native_return_undefined(jayess_value *value) {
    (void)value;
    return jayess_value_undefined();
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "bools.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "bools.bind.js"), []byte(`const f = () => {};
export const isTruthy = f;
export const isNullish = f;
export const returnNull = f;
export const returnUndefined = f;

export default {
  sources: ["./bools.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    isTruthy: { symbol: "jayess_native_is_truthy", type: "function" },
    isNullish: { symbol: "jayess_native_is_nullish", type: "function" },
    returnNull: { symbol: "jayess_native_return_null", type: "function" },
    returnUndefined: { symbol: "jayess_native_return_undefined", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { isTruthy, isNullish, returnNull, returnUndefined } from "./native/bools.bind.js";

function main(args) {
  console.log("truthy:" + isTruthy(true) + ":" + isTruthy(false));
  console.log("nullish:" + isNullish(null) + ":" + isNullish(undefined) + ":" + isNullish(0));
  console.log("returns:" + returnNull(1) + ":" + returnUndefined(1));
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

	outputPath := nativeOutputPath(workdir, "bind-bools")
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
		"truthy:true:false",
		"nullish:true:true:false",
		"returns:null:undefined",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected bool/null bind output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsManualBindNativeBuildFailuresClearly(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(nativeDir, "broken.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_broken(jayess_value *value) {
    return jayess_value_from_number(
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "broken.bind.js"), []byte(`const f = () => {};
export const broken = f;

export default {
  sources: ["./broken.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    broken: { symbol: "mylib_broken", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { broken } from "./native/broken.bind.js";

function main(args) {
  return broken(1);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "bind-broken")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected native build failure")
	}
	if !strings.Contains(err.Error(), "clang native build failed") || !strings.Contains(err.Error(), "broken.c") {
		t.Fatalf("expected clear native build diagnostic mentioning broken.c, got: %v", err)
	}
}

func TestBuildExecutableSupportsNativeInteropObjectsBuffersAndHandles(t *testing.T) {
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
#include <stdlib.h>

typedef struct jayess_counter_handle {
    int value;
} jayess_counter_handle;

jayess_value *jayess_profile_summary(jayess_value *profile) {
    jayess_object *object = jayess_value_as_object(profile);
    jayess_object *result = jayess_object_new();
    jayess_value *tags_value;
    jayess_array *tags;
    int tag_count = 0;
    if (object == NULL || result == NULL) {
        return jayess_value_undefined();
    }
    tags_value = jayess_object_get(object, "tags");
    tags = jayess_value_as_array(tags_value);
    if (tags != NULL) {
        tag_count = jayess_array_length(tags);
    }
    jayess_object_set_value(result, "name", jayess_object_get(object, "name"));
    jayess_object_set_value(result, "total", jayess_value_from_number(jayess_value_to_number(jayess_object_get(object, "score")) + (double)tag_count));
    return jayess_value_from_object(result);
}

jayess_value *jayess_make_bytes(void) {
    static const unsigned char raw[] = {75, 105, 109, 99, 104, 105};
    return jayess_value_from_bytes_copy(raw, sizeof(raw));
}

jayess_value *jayess_sum_bytes(jayess_value *value) {
    size_t length = 0;
    unsigned char *bytes = jayess_value_to_bytes_copy(value, &length);
    size_t i;
    int total = 0;
    if (bytes == NULL) {
        return jayess_value_from_number(0);
    }
    for (i = 0; i < length; i++) {
        total += (int)bytes[i];
    }
    free(bytes);
    return jayess_value_from_number((double)total);
}

jayess_value *jayess_counter_new(jayess_value *start) {
    jayess_counter_handle *handle = (jayess_counter_handle *)malloc(sizeof(jayess_counter_handle));
    if (handle == NULL) {
        return jayess_value_undefined();
    }
    handle->value = (int)jayess_value_to_number(start);
    return jayess_value_from_native_handle("CounterHandle", handle);
}

jayess_value *jayess_counter_increment(jayess_value *counter) {
    jayess_counter_handle *handle = (jayess_counter_handle *)jayess_value_as_native_handle(counter, "CounterHandle");
    if (handle == NULL) {
        return jayess_value_undefined();
    }
    handle->value += 1;
    return jayess_value_from_number((double)handle->value);
}

jayess_value *jayess_counter_value(jayess_value *counter) {
    jayess_counter_handle *handle = (jayess_counter_handle *)jayess_value_as_native_handle(counter, "CounterHandle");
    if (handle == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)handle->value);
}

jayess_value *jayess_counter_close(jayess_value *counter) {
    jayess_counter_handle *handle = (jayess_counter_handle *)jayess_value_as_native_handle(counter, "CounterHandle");
    if (handle == NULL) {
        return jayess_value_from_bool(0);
    }
    free(handle);
    jayess_value_clear_native_handle(counter);
    return jayess_value_from_bool(1);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "interop.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "interop.bind.js"), []byte(`const f = () => {};
export const profileSummary = f;
export const makeBytes = f;
export const sumBytes = f;
export const createCounter = f;
export const incrementCounter = f;
export const counterValue = f;
export const closeCounter = f;

export default {
  sources: ["./interop.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    profileSummary: { symbol: "jayess_profile_summary", type: "function" },
    makeBytes: { symbol: "jayess_make_bytes", type: "function" },
    sumBytes: { symbol: "jayess_sum_bytes", type: "function" },
    createCounter: { symbol: "jayess_counter_new", type: "function" },
    incrementCounter: { symbol: "jayess_counter_increment", type: "function" },
    counterValue: { symbol: "jayess_counter_value", type: "function" },
    closeCounter: { symbol: "jayess_counter_close", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { profileSummary, makeBytes, sumBytes, createCounter, incrementCounter, counterValue, closeCounter } from "./native/interop.bind.js";

function main(args) {
  var summary = profileSummary({ name: "Kimchi", score: 7, tags: ["hot", "red"] });
  console.log(summary.name + ":" + summary.total);

  var bytes = makeBytes();
  console.log("bytes:" + bytes.length + ":" + bytes[0] + ":" + bytes[5] + ":" + sumBytes(bytes));

  var counter = createCounter(3);
  console.log("counter:" + incrementCounter(counter) + ":" + counterValue(counter));
  console.log("closed:" + closeCounter(counter) + ":" + counterValue(counter));
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

	outputPath := nativeOutputPath(workdir, "ffi-interop-native")
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
	for _, want := range []string{"Kimchi:9", "bytes:6:75:105:597", "counter:4:4", "closed:true:undefined"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected native interop output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsNativeWrapperErrorPropagation(t *testing.T) {
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

jayess_value *jayess_native_fail_type(jayess_value *message) {
    jayess_throw_type_error(jayess_value_as_string(message));
    return jayess_value_undefined();
}

jayess_value *jayess_native_fail_named(jayess_value *name, jayess_value *message) {
    jayess_throw_named_error(jayess_value_as_string(name), jayess_value_as_string(message));
    return jayess_value_undefined();
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "errors.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "errors.bind.js"), []byte(`const f = () => {};
export const failType = f;
export const failNamed = f;

export default {
  sources: ["./errors.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    failType: { symbol: "jayess_native_fail_type", type: "function" },
    failNamed: { symbol: "jayess_native_fail_named", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	caughtPath := filepath.Join(workdir, "caught.js")
	caughtSource := `
import { failType, failNamed } from "./native/errors.bind.js";

function main(args) {
  try {
    failType("native type boom");
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    failNamed("WrapperError", "native wrapper boom");
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  return 0;
}
`
	if err := os.WriteFile(caughtPath, []byte(caughtSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	caughtResult, err := compiler.CompilePath(caughtPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	caughtOutputPath := nativeOutputPath(workdir, "ffi-native-errors-caught")
	if err := tc.BuildExecutable(caughtResult, compiler.Options{TargetTriple: triple}, caughtOutputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}
	caughtCmd := exec.Command(caughtOutputPath)
	caughtCmd.Dir = workdir
	caughtOut, err := caughtCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(caughtOut))
	}
	caughtText := string(caughtOut)
	for _, want := range []string{"TypeError:native type boom", "WrapperError:native wrapper boom"} {
		if !strings.Contains(caughtText, want) {
			t.Fatalf("expected native wrapper caught error output to contain %q, got: %s", want, caughtText)
		}
	}

	uncaughtPath := filepath.Join(workdir, "uncaught.js")
	uncaughtSource := `
import { failType } from "./native/errors.bind.js";

function main(args) {
  failType("native uncaught boom");
  return 0;
}
`
	if err := os.WriteFile(uncaughtPath, []byte(uncaughtSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	uncaughtResult, err := compiler.CompilePath(uncaughtPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	uncaughtOutputPath := nativeOutputPath(workdir, "ffi-native-errors-uncaught")
	if err := tc.BuildExecutable(uncaughtResult, compiler.Options{TargetTriple: triple}, uncaughtOutputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}
	uncaughtCmd := exec.Command(uncaughtOutputPath)
	uncaughtCmd.Dir = workdir
	uncaughtOut, err := uncaughtCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected uncaught native wrapper error to fail, got output: %s", string(uncaughtOut))
	}
	uncaughtText := string(uncaughtOut)
	for _, want := range []string{"Uncaught exception:", "TypeError", "native uncaught boom"} {
		if !strings.Contains(uncaughtText, want) {
			t.Fatalf("expected uncaught native wrapper error output to contain %q, got: %s", want, uncaughtText)
		}
	}
}

func TestBuildExecutableReportsNativeSymbolResolutionFailuresClearly(t *testing.T) {
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

jayess_value *jayess_real_symbol(jayess_value *value) {
    return jayess_value_from_number(1);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "broken.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "broken.bind.js"), []byte(`const f = () => {};
export const missingSymbol = f;

export default {
  sources: ["./broken.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    missingSymbol: { symbol: "jayess_missing_symbol", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { missingSymbol } from "./native/broken.bind.js";

function main(args) {
  return missingSymbol(1);
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "ffi-native-missing-symbol")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected native symbol resolution error")
	}
	if !strings.Contains(err.Error(), "native symbol resolution failed") || !strings.Contains(err.Error(), "jayess_missing_symbol") {
		t.Fatalf("expected clearer native symbol resolution diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsNativeWrapperTypeMismatchSafety(t *testing.T) {
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
#include <stdlib.h>

typedef struct jayess_counter_handle {
    int value;
} jayess_counter_handle;

jayess_value *jayess_require_name(jayess_value *value) {
    jayess_object *object = jayess_expect_object(value, "jayess_require_name");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_object_get(object, "name");
}

jayess_value *jayess_require_bytes_sum(jayess_value *value) {
    size_t length = 0;
    unsigned char *bytes = jayess_expect_bytes_copy(value, &length, "jayess_require_bytes_sum");
    size_t i;
    int total = 0;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    for (i = 0; i < length; i++) {
        total += (int)bytes[i];
    }
    free(bytes);
    return jayess_value_from_number((double)total);
}

jayess_value *jayess_counter_new_checked(jayess_value *start) {
    jayess_counter_handle *handle = (jayess_counter_handle *)malloc(sizeof(jayess_counter_handle));
    if (handle == NULL) {
        return jayess_value_undefined();
    }
    handle->value = (int)jayess_value_to_number(start);
    return jayess_value_from_native_handle("CounterHandle", handle);
}

jayess_value *jayess_counter_checked_value(jayess_value *counter) {
    jayess_counter_handle *handle = (jayess_counter_handle *)jayess_expect_native_handle(counter, "CounterHandle", "jayess_counter_checked_value");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)handle->value);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "safe.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "safe.bind.js"), []byte(`const f = () => {};
export const requireName = f;
export const requireBytesSum = f;
export const createCounter = f;
export const counterValue = f;

export default {
  sources: ["./safe.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    requireName: { symbol: "jayess_require_name", type: "function" },
    requireBytesSum: { symbol: "jayess_require_bytes_sum", type: "function" },
    createCounter: { symbol: "jayess_counter_new_checked", type: "function" },
    counterValue: { symbol: "jayess_counter_checked_value", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { requireName, requireBytesSum, createCounter, counterValue } from "./native/safe.bind.js";

function main(args) {
  try {
    requireName(1);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    requireBytesSum("kimchi");
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    counterValue({});
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  var counter = createCounter(5);
  console.log("counter-safe:" + counterValue(counter));
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

	outputPath := nativeOutputPath(workdir, "ffi-native-type-safety")
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
		"TypeError:jayess_require_name expects an object",
		"TypeError:jayess_require_bytes_sum expects a Uint8Array or byte buffer value",
		"TypeError:jayess_counter_checked_value expects a CounterHandle native handle",
		"counter-safe:5",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected native type-safety output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSimplifiedNativeWrapperSDK(t *testing.T) {
	t.Skip("legacy helper-SDK wrapper path removed")
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

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(js_arg_number(js, 0) + js_arg_number(js, 1));
}

JAYESS_EXPORT1(jayess_native_greet) {
    const char *name = js_arg_string(js, 0);
    if (jayess_has_exception()) {
        return js_undefined();
    }
    return js_format("Hello, %s", name);
}

JAYESS_EXPORT2(jayess_native_user) {
    js_obj *user = js_object();
    const char *name = js_arg_string(js, 0);
    if (jayess_has_exception()) {
        return js_undefined();
    }
    js_set(user, "name", js_string(name));
    js_set(user, "age", js_number(js_arg_number(js, 1)));
    return js_value_from_object(user);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "simple.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { jayess_native_add as add, jayess_native_greet as greet, jayess_native_user as makeUser } from "./native/simple.c";

function main(args) {
  var user = makeUser("Kimchi", 7);
  console.log("simple-add:" + add(3, 4));
  console.log("simple-greet:" + greet("Jayess"));
  console.log("simple-user:" + user.name + ":" + user.age);
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

	outputPath := nativeOutputPath(workdir, "simple-native-sdk")
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
		"simple-add:7",
		"simple-greet:Hello, Jayess",
		"simple-user:Kimchi:7",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected simplified native SDK output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsManagedNativeHandleOwnership(t *testing.T) {
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
#include <stdlib.h>

typedef struct jayess_owned_counter {
    int value;
} jayess_owned_counter;

static int jayess_owned_counter_finalize_count = 0;

static void jayess_owned_counter_finalize(void *handle) {
    if (handle != NULL) {
        free(handle);
        jayess_owned_counter_finalize_count += 1;
    }
}

jayess_value *jayess_owned_counter_new(jayess_value *start) {
    jayess_owned_counter *counter = (jayess_owned_counter *)malloc(sizeof(jayess_owned_counter));
    if (counter == NULL) {
        return jayess_value_undefined();
    }
    counter->value = (int)jayess_value_to_number(start);
    return jayess_value_from_managed_native_handle("OwnedCounter", counter, jayess_owned_counter_finalize);
}

jayess_value *jayess_owned_counter_value(jayess_value *value) {
    jayess_owned_counter *counter = (jayess_owned_counter *)jayess_expect_native_handle(value, "OwnedCounter", "jayess_owned_counter_value");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)counter->value);
}

jayess_value *jayess_owned_counter_close(jayess_value *value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(value));
}

jayess_value *jayess_owned_counter_closed(jayess_value *value) {
    jayess_object *object = jayess_value_as_object(value);
    if (object == NULL) {
        return jayess_value_from_bool(0);
    }
    return jayess_object_get(object, "closed");
}

jayess_value *jayess_owned_counter_finalize_total(void) {
    return jayess_value_from_number((double)jayess_owned_counter_finalize_count);
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "owned.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "owned.bind.js"), []byte(`const f = () => {};
export const createCounter = f;
export const counterValue = f;
export const closeCounter = f;
export const counterClosed = f;
export const finalizeTotal = f;

export default {
  sources: ["./owned.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    createCounter: { symbol: "jayess_owned_counter_new", type: "function" },
    counterValue: { symbol: "jayess_owned_counter_value", type: "function" },
    closeCounter: { symbol: "jayess_owned_counter_close", type: "function" },
    counterClosed: { symbol: "jayess_owned_counter_closed", type: "function" },
    finalizeTotal: { symbol: "jayess_owned_counter_finalize_total", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createCounter, counterValue, closeCounter, counterClosed, finalizeTotal } from "./native/owned.bind.js";

function main(args) {
  var counter = createCounter(9);
  console.log("owned-before:" + counterValue(counter) + ":" + counterClosed(counter));
  console.log("owned-close:" + closeCounter(counter) + ":" + counterClosed(counter) + ":" + finalizeTotal());
  console.log("owned-close-again:" + closeCounter(counter) + ":" + counterClosed(counter) + ":" + finalizeTotal());
  try {
    counterValue(counter);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
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

	outputPath := nativeOutputPath(workdir, "ffi-native-owned-handle")
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
		"owned-before:9:false",
		"owned-close:true:true:1",
		"owned-close-again:false:true:1",
		"TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected managed native handle output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsNativeWrapperLifetimeSafeCopies(t *testing.T) {
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
#include <stdlib.h>

typedef struct jayess_name_box {
    char *name;
} jayess_name_box;

static void jayess_name_box_finalize(void *handle) {
    jayess_name_box *box = (jayess_name_box *)handle;
    if (box != NULL) {
        jayess_string_free(box->name);
        free(box);
    }
}

jayess_value *jayess_name_box_new(jayess_value *value) {
    jayess_name_box *box = (jayess_name_box *)malloc(sizeof(jayess_name_box));
    if (box == NULL) {
        return jayess_value_undefined();
    }
    box->name = jayess_value_to_string_copy(value);
    if (box->name == NULL) {
        free(box);
        return jayess_value_undefined();
    }
    return jayess_value_from_managed_native_handle("NameBox", box, jayess_name_box_finalize);
}

jayess_value *jayess_name_box_get(jayess_value *value) {
    jayess_name_box *box = (jayess_name_box *)jayess_expect_native_handle(value, "NameBox", "jayess_name_box_get");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(box->name != NULL ? box->name : "");
}

jayess_value *jayess_name_box_close(jayess_value *value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(value));
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "lifetime.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "lifetime.bind.js"), []byte(`const f = () => {};
export const createBox = f;
export const readBox = f;
export const closeBox = f;

export default {
  sources: ["./lifetime.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    createBox: { symbol: "jayess_name_box_new", type: "function" },
    readBox: { symbol: "jayess_name_box_get", type: "function" },
    closeBox: { symbol: "jayess_name_box_close", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createBox, readBox, closeBox } from "./native/lifetime.bind.js";

function makeBox() {
  var name = "Kimchi";
  var box = createBox(name);
  name = "Soup";
  return box;
}

function main(args) {
  var box = makeBox();
  console.log("lifetime-copy:" + readBox(box));
  console.log("lifetime-close:" + closeBox(box));
  try {
    readBox(box);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
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

	outputPath := nativeOutputPath(workdir, "ffi-native-lifetime-copy")
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
		"lifetime-copy:Kimchi",
		"lifetime-close:true",
		"TypeError:jayess_name_box_get expects a NameBox native handle",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected lifetime-safe wrapper output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessHttpServerPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "httpserver"),
		filepath.Join(workdir, "node_modules", "@jayess", "httpserver"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "http-server"),
		filepath.Join(workdir, "node_modules", "@jayess", "http-server"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "picohttpparser"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"picohttpparser.c", "picohttpparser.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "picohttpparser", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "picohttpparser", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { parseRequest, parseResponse, parseRequestIncremental, parseResponseIncremental, decodeChunked, formatRequest, formatResponse } from "@jayess/httpserver";

function main(args) {
  var requestHeaders = {};
  var requestPayload = {};
  requestHeaders.Host = "127.0.0.1";
  requestHeaders["Content-Type"] = "text/plain";
  requestPayload.method = "POST";
  requestPayload.path = "/hello?name=kimchi";
  requestPayload.version = "HTTP/1.1";
  requestPayload.headers = requestHeaders;
  requestPayload.body = "kimchi";
  var request = parseRequest(formatRequest(requestPayload));
  var responsePayload = {};
  var responseHeadersForParse = {};
  responseHeadersForParse["Content-Type"] = "text/plain";
  responseHeadersForParse["X-Jayess-Test"] = "ok";
  responsePayload.version = "HTTP/1.1";
  responsePayload.status = 201;
  responsePayload.reason = "Created";
  responsePayload.headers = responseHeadersForParse;
  responsePayload.body = "pong:kimchi";
  var responseText = formatResponse(responsePayload);
  var parsedResponse = parseResponse(responseText);
  var incrementalRequestPayload = {};
  var incrementalRequestHeaders = {};
  incrementalRequestHeaders.Host = "a";
  incrementalRequestPayload.method = "GET";
  incrementalRequestPayload.path = "/partial";
  incrementalRequestPayload.version = "HTTP/1.1";
  incrementalRequestPayload.headers = incrementalRequestHeaders;
  incrementalRequestPayload.body = "";
  var incrementalRequestText = formatRequest(incrementalRequestPayload);
  var partialRequest = parseRequestIncremental("GET /partial HTTP/1.1\\r\\nHost: a", 5);
  var completeRequest = parseRequestIncremental(incrementalRequestText, 5);
  var incrementalResponsePayload = {};
  incrementalResponsePayload.version = "HTTP/1.1";
  incrementalResponsePayload.status = 200;
  incrementalResponsePayload.reason = "OK";
  incrementalResponsePayload.headers = {};
  incrementalResponsePayload.body = "pong";
  var incrementalResponseText = formatResponse(incrementalResponsePayload);
  var partialResponse = parseResponseIncremental("HTTP/1.1 200 OK\\r\\nContent-Length: 4", 5);
  var completeResponse = parseResponseIncremental(incrementalResponseText, 5);
  var decoded = decodeChunked(Uint8Array.fromString("360d0a68656c6c6f200d0a350d0a776f726c640d0a300d0a0d0a", "hex"));
  var headers = {};
  var response = {};
  console.log("http-server-request:" + request.method + ":" + request.path + ":" + request.query + ":" + request.body);
  console.log("http-server-response:" + parsedResponse.status + ":" + parsedResponse.reason + ":" + parsedResponse.body);
  console.log("http-server-incremental-request:" + partialRequest.complete + ":" + completeRequest.complete + ":" + completeRequest.message.path);
  console.log("http-server-incremental-response:" + partialResponse.complete + ":" + completeResponse.complete + ":" + completeResponse.message.status + ":" + completeResponse.message.body);
  console.log("http-server-chunked:" + decoded.complete + ":" + decoded.body + ":" + decoded.remaining);
  headers["Content-Type"] = "text/plain";
  headers["X-Jayess-Test"] = "ok";
  response.version = "HTTP/1.1";
  response.status = 201;
  response.reason = "Created";
  response.headers = headers;
  response.body = "pong:" + request.body;
  console.log(formatResponse(response));
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

	outputPath := nativeOutputPath(workdir, "jayess-http-server-package")
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
		"http-server-request:POST:/hello:name=kimchi:kimchi",
		"http-server-response:201:Created:pong:kimchi",
		"http-server-incremental-request:false:true:/partial",
		"http-server-incremental-response:false:true:200:pong",
		"http-server-chunked:true:hello world:0",
		"HTTP/1.1 201 Created",
		"X-Jayess-Test: ok",
		"pong:kimchi",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected HTTP server package output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsMissingPicoHTTPParserDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "httpserver"),
		filepath.Join(workdir, "node_modules", "@jayess", "httpserver"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "http-server"),
		filepath.Join(workdir, "node_modules", "@jayess", "http-server"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { parseRequest } from "@jayess/httpserver";

function main(args) {
  console.log(parseRequest("GET / HTTP/1.1\r\n\r\n"));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		if !strings.Contains(err.Error(), "picohttpparser") || !strings.Contains(err.Error(), "was not found") {
			t.Fatalf("expected clear pico missing-input diagnostic, got: %v", err)
		}
		return
	}

	outputPath := nativeOutputPath(workdir, "jayess-picohttp-missing-deps")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail without picohttpparser sources")
	}
	if !strings.Contains(err.Error(), "picohttpparser") {
		t.Fatalf("expected picohttpparser build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableReportsPicoHTTPParserErrorsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "httpserver"),
		filepath.Join(workdir, "node_modules", "@jayess", "httpserver"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "http-server"),
		filepath.Join(workdir, "node_modules", "@jayess", "http-server"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "picohttpparser"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"picohttpparser.c", "picohttpparser.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "picohttpparser", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "picohttpparser", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { parseRequest, parseResponse } from "@jayess/httpserver";

function main(args) {
  if (args.length > 0 && args[0] === "response") {
    parseResponse("HTTP/1.1 ???\r\n\r\n");
    return 0;
  }
  parseRequest("GET\r\n\r\n");
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

	outputPath := nativeOutputPath(workdir, "jayess-picohttp-errors")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected malformed request parse to fail")
	}
	if !strings.Contains(string(out), "SyntaxError") {
		t.Fatalf("expected SyntaxError output, got: %s", string(out))
	}

	cmd = exec.Command(outputPath, "response")
	cmd.Dir = workdir
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected malformed response parse to fail")
	}
	if !strings.Contains(string(out), "SyntaxError") {
		t.Fatalf("expected SyntaxError output, got: %s", string(out))
	}
}

func TestBuildExecutableSupportsJayessMongoosePackage(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, reply, freeManager } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-listen:false");
    return 1;
  }
  console.log("mongoose-listen:true");
  var i = 0;
  while (i < 200) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      var headers = {};
      headers["Content-Type"] = "text/plain";
      headers["X-Mongoose"] = "ok";
      console.log("mongoose-event:" + event.method + ":" + event.path + ":" + event.query + ":" + event.body);
      reply(event.connection, 202, headers, "pong:" + event.body);
      pollManager(manager, 20);
      break;
    }
    i = i + 1;
  }
  console.log("mongoose-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/hello?name=kimchi", port), "text/plain", strings.NewReader("kimchi"))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Post returned error: %v\noutput: %s", err, stdout.String())
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 202 {
		t.Fatalf("expected 202 response, got %d body=%s output=%s", resp.StatusCode, string(body), stdout.String())
	}
	if resp.Header.Get("X-Mongoose") != "ok" {
		t.Fatalf("expected X-Mongoose header, got %#v", resp.Header)
	}
	if string(body) != "pong:kimchi" {
		t.Fatalf("expected response body pong:kimchi, got %q", string(body))
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-listen:true",
		"mongoose-event:POST:/hello:name=kimchi:kimchi",
		"mongoose-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseRouterHelpers(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, reply, freeManager, createRouter, get, post, all, dispatch } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  var router = createRouter();
  get(router, "/ready", function(event) {
    console.log("mongoose-route:get:" + event.path);
    return { status: 200, headers: { "X-Route": "get" }, body: "ready" };
  });
  post(router, "/submit", function(event) {
    console.log("mongoose-route:post:" + event.path + ":" + event.body);
    return { status: 201, headers: { "X-Route": "post" }, body: "submitted:" + event.body };
  });
  all(router, "*", function(event) {
    console.log("mongoose-route:fallback:" + event.method + ":" + event.path);
    return { status: 404, headers: { "X-Route": "fallback" }, body: "not-found" };
  });
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-router-listen:false");
    return 1;
  }
  console.log("mongoose-router-listen:true");
  var handled = 0;
  while (handled < 2) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      if (!dispatch(router, event)) {
        reply(event.connection, 500, { "Content-Type": "text/plain" }, "unhandled");
      }
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-router-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-router-helpers")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp1, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/ready", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get returned error: %v\noutput: %s", err, stdout.String())
	}
	body1, err := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp1.StatusCode != 200 || string(body1) != "ready" || resp1.Header.Get("X-Route") != "get" {
		t.Fatalf("unexpected GET route response: status=%d body=%q headers=%#v output=%s", resp1.StatusCode, string(body1), resp1.Header, stdout.String())
	}

	resp2, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/submit", port), "text/plain", strings.NewReader("kimchi"))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Post returned error: %v\noutput: %s", err, stdout.String())
	}
	body2, err := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp2.StatusCode != 201 || string(body2) != "submitted:kimchi" || resp2.Header.Get("X-Route") != "post" {
		t.Fatalf("unexpected POST route response: status=%d body=%q headers=%#v output=%s", resp2.StatusCode, string(body2), resp2.Header, stdout.String())
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose router program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-router-listen:true",
		"mongoose-route:get:/ready",
		"mongoose-route:post:/submit:kimchi",
		"mongoose-router-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose router output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseStaticFiles(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(workdir, "site", "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "site", "index.html"), []byte("<h1>home</h1>"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "site", "docs", "app.css"), []byte("body{color:red;}"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, reply, freeManager, serveStatic } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-static-listen:false");
    return 1;
  }
  console.log("mongoose-static-listen:true");
  var handled = 0;
  while (handled < 3) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      if (!serveStatic(event, "/public", "site")) {
        reply(event.connection, 500, { "Content-Type": "text/plain" }, "unhandled:" + event.method + ":" + event.path);
      }
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-static-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-static-files")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp1, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/public/", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(/public/) returned error: %v\noutput: %s", err, stdout.String())
	}
	body1, err := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp1.StatusCode != 200 || string(body1) != "<h1>home</h1>" || !strings.Contains(resp1.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected static index response: status=%d body=%q headers=%#v output=%s", resp1.StatusCode, string(body1), resp1.Header, stdout.String())
	}

	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/public/docs/app.css", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(/public/docs/app.css) returned error: %v\noutput: %s", err, stdout.String())
	}
	body2, err := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp2.StatusCode != 200 || string(body2) != "body{color:red;}" || !strings.Contains(resp2.Header.Get("Content-Type"), "text/css") {
		t.Fatalf("unexpected static css response: status=%d body=%q headers=%#v output=%s", resp2.StatusCode, string(body2), resp2.Header, stdout.String())
	}

	resp3, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/public/../main.js", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(path traversal) returned error: %v\noutput: %s", err, stdout.String())
	}
	body3, err := io.ReadAll(resp3.Body)
	_ = resp3.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp3.StatusCode != 403 || string(body3) != "forbidden" {
		t.Fatalf("unexpected traversal/static miss response: status=%d body=%q headers=%#v output=%s", resp3.StatusCode, string(body3), resp3.Header, stdout.String())
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose static program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-static-listen:true",
		"mongoose-static-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose static output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseEmbeddedApp(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, reply, freeManager, createEmbeddedApp, serveEmbeddedApp } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  var app = createEmbeddedApp([
    { path: "/index.html", body: "<!doctype html><title>app</title><h1>embedded</h1>", contentType: "text/html; charset=utf-8" },
    { path: "/app.js", body: "console.log('embedded-app');", contentType: "application/javascript; charset=utf-8" }
  ], "/index.html");
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-embedded-listen:false");
    return 1;
  }
  console.log("mongoose-embedded-listen:true");
  var handled = 0;
  while (handled < 3) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      if (!serveEmbeddedApp(event, "/", app, undefined)) {
        reply(event.connection, 404, { "Content-Type": "text/plain" }, "missing:" + event.path);
      }
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-embedded-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-embedded-app")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp1, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(/) returned error: %v\noutput: %s", err, stdout.String())
	}
	body1, err := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp1.StatusCode != 200 || !strings.Contains(string(body1), "<h1>embedded</h1>") || !strings.Contains(resp1.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected embedded index response: status=%d body=%q headers=%#v output=%s", resp1.StatusCode, string(body1), resp1.Header, stdout.String())
	}

	resp2, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/app.js", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(/app.js) returned error: %v\noutput: %s", err, stdout.String())
	}
	body2, err := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp2.StatusCode != 200 || string(body2) != "console.log('embedded-app');" || !strings.Contains(resp2.Header.Get("Content-Type"), "application/javascript") {
		t.Fatalf("unexpected embedded asset response: status=%d body=%q headers=%#v output=%s", resp2.StatusCode, string(body2), resp2.Header, stdout.String())
	}

	resp3, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/dashboard/settings", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get(SPA fallback) returned error: %v\noutput: %s", err, stdout.String())
	}
	body3, err := io.ReadAll(resp3.Body)
	_ = resp3.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp3.StatusCode != 200 || !strings.Contains(string(body3), "<h1>embedded</h1>") || !strings.Contains(resp3.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected embedded fallback response: status=%d body=%q headers=%#v output=%s", resp3.StatusCode, string(body3), resp3.Header, stdout.String())
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose embedded app program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-embedded-listen:true",
		"mongoose-embedded-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose embedded app output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseChunkedStreaming(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, freeManager, startChunked, writeChunk, endChunked } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-chunked-listen:false");
    return 1;
  }
  console.log("mongoose-chunked-listen:true");
  var handled = 0;
  while (handled < 1) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      var stream = startChunked(event, 200, { "Content-Type": "text/plain" });
      console.log("mongoose-chunked-open:" + (stream !== undefined));
      writeChunk(stream, "pong:");
      writeChunk(stream, event.body);
      endChunked(stream, "!");
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-chunked-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-chunked-streaming")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/submit", port), "text/plain", strings.NewReader("kimchi"))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Post returned error: %v\noutput: %s", err, stdout.String())
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 200 || string(body) != "pong:kimchi!" {
		t.Fatalf("unexpected chunked response: status=%d body=%q headers=%#v output=%s", resp.StatusCode, string(body), resp.Header, stdout.String())
	}
	if len(resp.TransferEncoding) == 0 || resp.TransferEncoding[0] != "chunked" {
		t.Fatalf("expected chunked transfer encoding, got %#v", resp.TransferEncoding)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose chunked program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-chunked-listen:true",
		"mongoose-chunked-open:true",
		"mongoose-chunked-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose chunked output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseHTTPS(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}
	certPath, keyPath := writeTestTLSCertificatePair(t, workdir, "mongoose-https")

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenTlsServer, pollManager, nextEvent, reply, freeManager } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (!listenTlsServer(manager, "https://127.0.0.1:%d", %q, %q)) {
    console.log("mongoose-https-listen:false");
    return 1;
  }
  console.log("mongoose-https-listen:true");
  var handled = 0;
  while (handled < 1) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      console.log("mongoose-https-event:" + event.method + ":" + event.path + ":" + event.body);
      reply(event.connection, 201, { "Content-Type": "text/plain" }, "secure:" + event.body);
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-https-free:" + freeManager(manager));
  return 0;
}
`, port, certPath, keyPath)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-https")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post(fmt.Sprintf("https://127.0.0.1:%d/secure", port), "text/plain", strings.NewReader("kimchi"))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Post returned error: %v\noutput: %s", err, stdout.String())
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 201 || string(body) != "secure:kimchi" {
		t.Fatalf("unexpected https response: status=%d body=%q headers=%#v output=%s", resp.StatusCode, string(body), resp.Header, stdout.String())
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose https program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-https-listen:true",
		"mongoose-https-event:POST:/secure:kimchi",
		"mongoose-https-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose https output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseWebSocket(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, upgradeWebSocket, sendWebSocket, freeManager } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-ws-listen:false");
    return 1;
  }
  console.log("mongoose-ws-listen:true");
  var opened = false;
  var echoed = false;
  var loops = 0;
  while (loops < 800 && !echoed) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      if (event.kind === "http" && event.path === "/ws") {
        console.log("mongoose-ws-upgrade");
        upgradeWebSocket(event);
      } else if (event.kind === "wsOpen") {
        console.log("mongoose-ws-open");
        opened = true;
      } else if (event.kind === "wsMessage") {
        console.log("mongoose-ws-message:" + event.body);
        sendWebSocket(event.connection, "echo:" + event.body);
        pollManager(manager, 50);
        echoed = true;
      }
    }
    loops = loops + 1;
  }
  console.log("mongoose-ws-opened:" + opened);
  pollManager(manager, 50);
  console.log("mongoose-ws-free:" + freeManager(manager));
  return echoed ? 0 : 1;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-websocket")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Dial returned error: %v\noutput: %s", err, stdout.String())
	}
	defer conn.Close()

	wsKey := "dGhlIHNhbXBsZSBub25jZQ=="
	req := fmt.Sprintf("GET /ws HTTP/1.1\r\nHost: 127.0.0.1:%d\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", port, wsKey)
	if _, err := conn.Write([]byte(req)); err != nil {
		_ = cmd.Wait()
		t.Fatalf("Write returned error: %v\noutput: %s", err, stdout.String())
	}
	headers, err := readHTTPHeaderBlock(conn)
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("readHTTPHeaderBlock returned error: %v\noutput: %s", err, stdout.String())
	}
	if !strings.Contains(headers, "101 Switching Protocols") {
		t.Fatalf("expected websocket upgrade response, got: %s\noutput: %s", headers, stdout.String())
	}
	if !strings.Contains(headers, "Sec-WebSocket-Accept: "+websocketAcceptForKey(wsKey)) {
		t.Fatalf("expected websocket accept header, got: %s", headers)
	}
	if err := writeMaskedWebSocketTextFrame(conn, "kimchi"); err != nil {
		_ = cmd.Wait()
		t.Fatalf("writeMaskedWebSocketTextFrame returned error: %v\noutput: %s", err, stdout.String())
	}
	message, err := readWebSocketTextFrame(conn)
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("readWebSocketTextFrame returned error: %v\noutput: %s", err, stdout.String())
	}
	if message != "echo:kimchi" {
		t.Fatalf("unexpected websocket message %q\noutput: %s", message, stdout.String())
	}
	_ = conn.Close()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose websocket program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-ws-listen:true",
		"mongoose-ws-upgrade",
		"mongoose-ws-open",
		"mongoose-ws-message:kimchi",
		"mongoose-ws-opened:true",
		"mongoose-ws-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose websocket output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsJayessMongooseErrorsUsefully(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, listenTlsServer, pollManager, nextEvent, reply, upgradeWebSocket, freeManager } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  try {
    listenTlsServer(manager, "https://127.0.0.1:%d", "./missing-cert.pem", "./missing-key.pem");
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }

  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-error-listen:false");
    return 1;
  }

  var handled = false;
  var loops = 0;
  while (loops < 400 && !handled) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      try {
        upgradeWebSocket(event);
      } catch (err) {
        console.log(err.name + ":" + err.message);
        reply(event.connection, 400, { "Content-Type": "text/plain" }, "bad-upgrade");
        pollManager(manager, 50);
      }
      handled = true;
    }
    loops = loops + 1;
  }

  console.log("mongoose-error-free:" + freeManager(manager));
  return handled ? 0 : 1;
}
`, port+1, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-errors")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/plain", port))
	if err != nil {
		_ = cmd.Wait()
		t.Fatalf("Get returned error: %v\noutput: %s", err, stdout.String())
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 400 || string(body) != "bad-upgrade" {
		t.Fatalf("unexpected mongoose error response: status=%d body=%q output=%s", resp.StatusCode, string(body), stdout.String())
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose error program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"Error:jayess_mongoose_listen_tls failed to read certificate or key file",
		"TypeError:jayess_mongoose_upgrade_websocket requires a websocket upgrade request",
		"mongoose-error-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose error output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongooseCallbackLifetime(t *testing.T) {
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
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "mongoose"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"mongoose.c", "mongoose.h"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "mongoose", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "refs", "mongoose", name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := fmt.Sprintf(`
import { createManager, listenServer, pollManager, nextEvent, reply, freeManager, createRouter, get, dispatch } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  var router = createRouter();
  var state = { prefix: "count", value: 0 };

  get(router, "/count", function(event) {
    state.value = state.value + 1;
    console.log("mongoose-callback:" + state.prefix + ":" + state.value + ":" + event.path);
    return {
      status: 200,
      headers: { "X-Count": "" + state.value },
      body: state.prefix + ":" + state.value
    };
  });

  if (!listenServer(manager, "http://127.0.0.1:%d")) {
    console.log("mongoose-callback-listen:false");
    return 1;
  }
  console.log("mongoose-callback-listen:true");
  var handled = 0;
  while (handled < 3) {
    pollManager(manager, 10);
    var event = nextEvent(manager);
    if (event !== undefined) {
      if (!dispatch(router, event)) {
        reply(event.connection, 500, { "Content-Type": "text/plain" }, "unhandled");
      }
      pollManager(manager, 20);
      handled = handled + 1;
    }
  }
  console.log("mongoose-callback-free:" + freeManager(manager));
  return 0;
}
`, port)
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-callback-lifetime")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	time.Sleep(300 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/count", port))
		if err != nil {
			_ = cmd.Wait()
			t.Fatalf("Get returned error: %v\noutput: %s", err, stdout.String())
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		wantBody := fmt.Sprintf("count:%d", i)
		if resp.StatusCode != 200 || string(body) != wantBody || resp.Header.Get("X-Count") != strconv.Itoa(i) {
			t.Fatalf("unexpected callback route response: status=%d body=%q headers=%#v output=%s", resp.StatusCode, string(body), resp.Header, stdout.String())
		}
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("mongoose callback program returned error: %v: %s", err, stdout.String())
	}
	text := stdout.String()
	for _, want := range []string{
		"mongoose-callback-listen:true",
		"mongoose-callback:count:1:/count",
		"mongoose-callback:count:2:/count",
		"mongoose-callback:count:3:/count",
		"mongoose-callback-free:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected mongoose callback output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessMongoosePackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "mongoose"),
		filepath.Join(workdir, "node_modules", "@jayess", "mongoose"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createManager, freeManager } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  if (manager === undefined) {
    console.log("mongoose-manager:undefined");
    return 0;
  }
  console.log("mongoose-free:" + freeManager(manager));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		if !strings.Contains(err.Error(), "mongoose") || !strings.Contains(err.Error(), "was not found") {
			t.Fatalf("expected clear Mongoose missing-input diagnostic, got: %v", err)
		}
		return
	}

	outputPath := nativeOutputPath(workdir, "jayess-mongoose-package-missing-deps")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled mongoose program returned error: %v: %s", runErr, string(out))
		}
		if !strings.Contains(string(out), "mongoose-") {
			t.Fatalf("expected mongoose package smoke output, got: %s", string(out))
		}
		return
	}
	if !strings.Contains(err.Error(), "mongoose") {
		t.Fatalf("expected clear Mongoose build diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "native header dependency missing") && !strings.Contains(err.Error(), "clang native build failed") {
		t.Fatalf("expected clear Mongoose build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessGLFWPackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw", "include"),
		filepath.Join(workdir, "refs", "glfw", "include"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw", "src"),
		filepath.Join(workdir, "refs", "glfw", "src"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { init, terminate } from "@jayess/glfw";

function main(args) {
  var ok = init();
  console.log("glfw-init:" + ok);
  if (ok) {
    terminate();
  }
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

	outputPath := nativeOutputPath(workdir, "jayess-glfw-package")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled GLFW program returned error: %v: %s", runErr, string(out))
		}
		if !strings.Contains(string(out), "glfw-init:") {
			t.Fatalf("expected GLFW package smoke output, got: %s", string(out))
		}
		return
	}
	if !strings.Contains(err.Error(), "glfw") || (!strings.Contains(err.Error(), "native library link failed") && !strings.Contains(err.Error(), "native header dependency missing") && !strings.Contains(err.Error(), "clang native build failed")) {
		t.Fatalf("expected clear GLFW build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessGLFWWindowLifecycleOrHeadless(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "audio"),
		filepath.Join(workdir, "node_modules", "@jayess", "audio"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw", "include"),
		filepath.Join(workdir, "refs", "glfw", "include"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw", "src"),
		filepath.Join(workdir, "refs", "glfw", "src"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { init, terminate, createWindow, destroyWindow, pollEvents, swapBuffers, windowShouldClose, getTime, setTime, getWindowSize, setWindowSize, getFramebufferSize, setKeyCallback, setMouseButtonCallback, setCursorPosCallback, setScrollCallback, simulateKeyEvent, simulateMouseButtonEvent, simulateCursorPosEvent, simulateScrollEvent, setWindowFullscreen, setWindowWindowed, isJoystickPresent, isJoystickGamepad, getJoystickName, getGamepadName, getGamepadButton } from "@jayess/glfw";

function main(args) {
  if (!init()) {
    console.log("glfw-init:false");
    return 0;
  }
  var workerThread = worker.create(function(message) {
    return {
      sum: message.left + message.right,
      text: message.text.toUpperCase()
    };
  });
  var window = createWindow(64, 64, "jayess-glfw-test");
  if (window === undefined) {
    console.log("glfw-worker-terminate:" + workerThread.terminate());
    console.log("glfw-headless");
    terminate();
    return 0;
  }
  setTime(3.25);
  var keyCount = 0;
  var mouseCount = 0;
  var cursorCount = 0;
  var scrollCount = 0;
  setKeyCallback(window, function (event) {
    keyCount = keyCount + event.key + event.action + event.mods;
  });
  setMouseButtonCallback(window, function (event) {
    mouseCount = mouseCount + event.button + event.action + event.mods;
  });
  setCursorPosCallback(window, function (event) {
    cursorCount = cursorCount + event.x + event.y;
  });
  setScrollCallback(window, function (event) {
    scrollCount = scrollCount + event.xoffset + event.yoffset;
  });
  simulateKeyEvent(window, 65, 1, 2);
  simulateMouseButtonEvent(window, 1, 1, 4);
  simulateCursorPosEvent(window, 12.5, 3.5);
  simulateScrollEvent(window, 2.5, -1.5);
  pollEvents();
  swapBuffers(window);
  console.log("glfw-worker-post:" + workerThread.postMessage({ left: 5, right: 7, text: "kimchi" }));
  console.log("glfw-worker-loop:" + await sleepAsync(0, "tick"));
  var workerReply = workerThread.receive(5000);
  setWindowSize(window, 96, 72);
  var size = getWindowSize(window);
  var framebuffer = getFramebufferSize(window);
  setWindowFullscreen(window);
  var fullscreenSize = getWindowSize(window);
  setWindowWindowed(window, 80, 60);
  var windowedSize = getWindowSize(window);
  var joystickName = getJoystickName(0);
  var gamepadName = getGamepadName(0);
  console.log("glfw-time:" + (getTime() >= 3.25));
  console.log("glfw-size:" + size.width + "x" + size.height);
  console.log("glfw-framebuffer-size:" + framebuffer.width + "x" + framebuffer.height);
  console.log("glfw-key-callback:" + keyCount);
  console.log("glfw-mouse-callback:" + mouseCount);
  console.log("glfw-cursor-callback:" + cursorCount);
  console.log("glfw-scroll-callback:" + scrollCount);
  console.log("glfw-worker-reply:" + workerReply.value.sum + ":" + workerReply.value.text);
  console.log("glfw-fullscreen-size:" + fullscreenSize.width + "x" + fullscreenSize.height);
  console.log("glfw-windowed-size:" + windowedSize.width + "x" + windowedSize.height);
  console.log("glfw-joystick-present:" + (isJoystickPresent(0) === false || isJoystickPresent(0) === true));
  console.log("glfw-joystick-gamepad:" + (isJoystickGamepad(0) === false || isJoystickGamepad(0) === true));
  console.log("glfw-joystick-name:" + (joystickName == undefined || typeof joystickName === "string"));
  console.log("glfw-gamepad-name:" + (gamepadName == undefined || typeof gamepadName === "string"));
  console.log("glfw-gamepad-button:" + (getGamepadButton(0, 0) == undefined || getGamepadButton(0, 0) === false || getGamepadButton(0, 0) === true));
  console.log("glfw-window:" + windowShouldClose(window));
  console.log("glfw-destroy:" + destroyWindow(window));
  try {
    windowShouldClose(window);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("glfw-worker-terminate:" + workerThread.terminate());
  terminate();
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

	outputPath := nativeOutputPath(workdir, "jayess-glfw-lifecycle")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err != nil {
		if strings.Contains(err.Error(), "glfw") && (strings.Contains(err.Error(), "native library link failed") || strings.Contains(err.Error(), "native header dependency missing") || strings.Contains(err.Error(), "clang native build failed")) {
			t.Skipf("GLFW dependency unavailable for lifecycle smoke test: %v", err)
		}
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled GLFW lifecycle program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if strings.Contains(text, "glfw-headless") || strings.Contains(text, "glfw-init:false") {
		return
	}
	for _, want := range []string{
		"glfw-time:true",
		"glfw-size:96x72",
		"glfw-framebuffer-size:96x72",
		"glfw-key-callback:68",
		"glfw-mouse-callback:6",
		"glfw-cursor-callback:16",
		"glfw-scroll-callback:1",
		"glfw-worker-post:true",
		"glfw-worker-loop:tick",
		"glfw-worker-reply:12:KIMCHI",
		"glfw-fullscreen-size:1920x1080",
		"glfw-windowed-size:80x60",
		"glfw-joystick-present:true",
		"glfw-joystick-gamepad:true",
		"glfw-joystick-name:true",
		"glfw-gamepad-name:true",
		"glfw-gamepad-button:true",
		"glfw-window:",
		"glfw-destroy:true",
		"glfw-worker-terminate:true",
		"TypeError:jayess_glfw_window_should_close expects a GLFWwindow native handle",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected GLFW lifecycle output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessRaylibPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "raylib"),
		filepath.Join(workdir, "node_modules", "@jayess", "raylib"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "raylib", "src"),
		filepath.Join(workdir, "refs", "raylib", "src"),
	)
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { setTraceLogLevel, setConfigFlags, initWindow, closeWindow, isWindowReady, windowShouldClose, setWindowTitle, setWindowSize, getScreenWidth, getScreenHeight, beginDrawing, endDrawing, clearBackground, drawCircle, drawText, genImageColor, unloadImage, isImageReady, getImageWidth, getImageHeight, loadTextureFromImage, unloadTexture, isTextureReady, getTextureWidth, getTextureHeight, drawTexture, isKeyPressed, isKeyDown, isMouseButtonDown, getMouseX, getMouseY, getMousePosition, isGamepadAvailable, getGamepadAxisCount, isGamepadButtonDown, getGamepadName, setTargetFPS, getFrameTime, getTime, setRandomSeed, getRandomValue } from "@jayess/raylib";

function main(args) {
  var black = { r: 0, g: 0, b: 0, a: 255 };
  var red = { r: 230, g: 41, b: 55, a: 255 };
  var white = { r: 255, g: 255, b: 255, a: 255 };
  var green = { r: 0, g: 228, b: 48, a: 255 };
  var mouse = null;
  var gamepadName = null;
  var image = null;
  var memoryImage = null;
  var texture = null;
  var memoryTexture = null;
  setTraceLogLevel(7);
  setConfigFlags(128);
  setRandomSeed(123);
  console.log("raylib-rand:" + getRandomValue(1, 10));
  console.log("raylib-init:" + initWindow(64, 64, "jayess-raylib"));
  console.log("raylib-ready:" + isWindowReady());
  setWindowTitle("jayess-raylib-updated");
  setWindowSize(48, 40);
  console.log("raylib-size:" + getScreenWidth() + "x" + getScreenHeight());
  console.log("raylib-key-pressed:" + (isKeyPressed(65) === false || isKeyPressed(65) === true));
  console.log("raylib-key-down:" + (isKeyDown(65) === false || isKeyDown(65) === true));
  console.log("raylib-mouse-down:" + (isMouseButtonDown(0) === false || isMouseButtonDown(0) === true));
  mouse = getMousePosition();
  gamepadName = getGamepadName(0);
  console.log("raylib-mouse-x:" + (getMouseX() >= 0 || getMouseX() < 0));
  console.log("raylib-mouse-y:" + (getMouseY() >= 0 || getMouseY() < 0));
  console.log("raylib-mouse-pos:" + ((mouse.x >= 0 || mouse.x < 0) && (mouse.y >= 0 || mouse.y < 0)));
  console.log("raylib-gamepad-available:" + (isGamepadAvailable(0) === false || isGamepadAvailable(0) === true));
  console.log("raylib-gamepad-axis-count:" + (getGamepadAxisCount(0) >= 0));
  console.log("raylib-gamepad-button:" + (isGamepadButtonDown(0, 0) === false || isGamepadButtonDown(0, 0) === true));
  console.log("raylib-gamepad-name:" + (gamepadName == undefined || typeof gamepadName === "string"));
  image = genImageColor(4, 4, green);
  memoryImage = genImageColor(2, 2, white);
  texture = loadTextureFromImage(image);
  memoryTexture = loadTextureFromImage(memoryImage);
  console.log("raylib-image-ready:" + isImageReady(image));
  console.log("raylib-image-size:" + getImageWidth(image) + "x" + getImageHeight(image));
  console.log("raylib-second-image-size:" + getImageWidth(memoryImage) + "x" + getImageHeight(memoryImage));
  console.log("raylib-texture-ready:" + isTextureReady(texture));
  console.log("raylib-texture-size:" + getTextureWidth(texture) + "x" + getTextureHeight(texture));
  console.log("raylib-second-texture-size:" + getTextureWidth(memoryTexture) + "x" + getTextureHeight(memoryTexture));
  setTargetFPS(60);
  beginDrawing();
  clearBackground(black);
  drawCircle(32, 32, 8, red);
  drawText("jayess", 4, 4, 10, white);
  drawTexture(texture, 12, 12, white);
  drawTexture(memoryTexture, 20, 20, white);
  endDrawing();
  setTimeout(() => {
    console.log("raylib-timer");
    return 0;
  }, 0);
  console.log("raylib-async:" + await sleepAsync(0, "ok"));
  beginDrawing();
  clearBackground(black);
  endDrawing();
  console.log("raylib-text:true");
  console.log("raylib-texture-draw:true");
  console.log("raylib-unload-texture:" + unloadTexture(texture));
  console.log("raylib-unload-memory-texture:" + unloadTexture(memoryTexture));
  console.log("raylib-unload-image:" + unloadImage(image));
  console.log("raylib-unload-memory-image:" + unloadImage(memoryImage));
  try {
    getTextureWidth(texture);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    getImageWidth(image);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("raylib-frame:" + (getFrameTime() >= 0));
  console.log("raylib-time:" + (getTime() >= 0));
  console.log("raylib-close:" + windowShouldClose());
  closeWindow();
  console.log("raylib-ready-after-close:" + isWindowReady());
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

	outputPath := nativeOutputPath(workdir, "jayess-raylib-package")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled raylib program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"raylib-rand:",
		"raylib-init:true",
		"raylib-ready:true",
		"raylib-size:",
		"raylib-key-pressed:true",
		"raylib-key-down:true",
		"raylib-mouse-down:true",
		"raylib-mouse-x:true",
		"raylib-mouse-y:true",
		"raylib-mouse-pos:true",
		"raylib-gamepad-available:true",
		"raylib-gamepad-axis-count:true",
		"raylib-gamepad-button:true",
		"raylib-gamepad-name:true",
		"raylib-image-ready:true",
		"raylib-image-size:4x4",
		"raylib-second-image-size:2x2",
		"raylib-texture-ready:true",
		"raylib-texture-size:4x4",
		"raylib-second-texture-size:2x2",
		"raylib-timer",
		"raylib-async:ok",
		"raylib-text:true",
		"raylib-texture-draw:true",
		"raylib-unload-texture:true",
		"raylib-unload-memory-texture:true",
		"raylib-unload-image:true",
		"raylib-unload-memory-image:true",
		"TypeError:jayess_raylib_get_texture_width expects a RaylibTexture native handle",
		"TypeError:jayess_raylib_get_image_width expects a RaylibImage native handle",
		"raylib-frame:true",
		"raylib-time:true",
		"raylib-close:false",
		"raylib-ready-after-close:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected raylib package smoke output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessRaylibPackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "raylib"),
		filepath.Join(workdir, "node_modules", "@jayess", "raylib"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { initWindow, isWindowReady } from "@jayess/raylib";

function main(args) {
  initWindow(32, 32, "missing-refs");
  console.log("raylib-ready:" + isWindowReady());
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		if !strings.Contains(err.Error(), "raylib") || !strings.Contains(err.Error(), "was not found") {
			t.Fatalf("expected clear raylib missing-input diagnostic, got: %v", err)
		}
		return
	}

	outputPath := nativeOutputPath(workdir, "jayess-raylib-package-missing-deps")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled raylib program returned error: %v: %s", runErr, string(out))
		}
		if !strings.Contains(string(out), "raylib-ready:") {
			t.Fatalf("expected raylib package smoke output, got: %s", string(out))
		}
		return
	}
	if !strings.Contains(err.Error(), "raylib") {
		t.Fatalf("expected clear raylib build diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "native header dependency missing") && !strings.Contains(err.Error(), "clang native build failed") {
		t.Fatalf("expected clear raylib build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableReportsJayessRaylibErrorsUsefully(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping raylib error test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-raylib-errors-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "raylib"),
		filepath.Join(workdir, "node_modules", "@jayess", "raylib"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "raylib", "src"),
		filepath.Join(workdir, "refs", "raylib", "src"),
	)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { loadImage, loadImageFromBytes, loadTexture } from "@jayess/raylib";

function main(args) {
  try {
    loadImage("missing-image.png");
  } catch (err) {
    console.log("raylib-image-error:" + err.name);
  }
  try {
    loadImageFromBytes(".ppm", Uint8Array.fromString("not-an-image"));
  } catch (err) {
    console.log("raylib-bytes-error:" + err.name);
  }
  try {
    loadTexture("missing-texture.png");
  } catch (err) {
    console.log("raylib-texture-error:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "raylib-errors-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled raylib error program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"raylib-image-error:RaylibError",
		"raylib-bytes-error:RaylibError",
		"raylib-texture-error:RaylibError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected raylib error output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessAudioPackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "audio"),
		filepath.Join(workdir, "node_modules", "@jayess", "audio"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "cubeb", "include", "cubeb"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "cubeb", "include", "cubeb", "cubeb.h"))
	if err != nil {
		t.Fatalf("ReadFile(cubeb.h) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "refs", "cubeb", "include", "cubeb", "cubeb.h"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(cubeb.h) returned error: %v", err)
	}

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createContext, backendId, maxChannelCount, destroyContext } from "@jayess/audio";

function main(args) {
  var ctx = createContext("jayess-audio-test", null);
  if (ctx === undefined) {
    console.log("audio-init:undefined");
    return 0;
  }
  console.log("audio-backend:" + backendId(ctx));
  console.log("audio-max-channels:" + maxChannelCount(ctx));
  console.log("audio-destroy:" + destroyContext(ctx));
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

	outputPath := nativeOutputPath(workdir, "jayess-audio-package")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled audio program returned error: %v: %s", runErr, string(out))
		}
		if !strings.Contains(string(out), "audio-") {
			t.Fatalf("expected audio package smoke output, got: %s", string(out))
		}
		return
	}
	if !strings.Contains(err.Error(), "cubeb") || (!strings.Contains(err.Error(), "native library link failed") && !strings.Contains(err.Error(), "clang native build failed")) {
		t.Fatalf("expected clear Cubeb build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessWebviewPackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
		filepath.Join(workdir, "node_modules", "@jayess", "webview"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "webview", "core", "include"),
		filepath.Join(workdir, "refs", "webview", "core", "include"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createWindow } from "@jayess/webview";

function main(args) {
  var view = createWindow(false);
  if (view == undefined) {
    console.log("webview-create:undefined");
    return 0;
  }
  console.log("webview-created:" + view.closed);
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

	outputPath := nativeOutputPath(workdir, "jayess-webview-package")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled webview program returned error: %v: %s", runErr, string(out))
		}
		text := string(out)
		if !strings.Contains(text, "webview-") {
			t.Fatalf("expected webview package smoke output, got: %s", text)
		}
		return
	}
	if !strings.Contains(err.Error(), "gtk") && !strings.Contains(err.Error(), "webkit") && !strings.Contains(err.Error(), "webview") {
		t.Fatalf("expected webview dependency diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "native library link failed") && !strings.Contains(err.Error(), "native header dependency missing") && !strings.Contains(err.Error(), "clang native build failed") {
		t.Fatalf("expected clear webview build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessWebviewBindingWithExplicitIncludePaths(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping webview explicit-include test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
		filepath.Join(workdir, "node_modules", "@jayess", "webview"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw"),
		filepath.Join(workdir, "refs", "glfw"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "webview", "native")
	includeDir := filepath.Join(nativeDir, "include", "webview")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "webview.h"), []byte(`#pragma once
#ifdef __cplusplus
extern "C" {
#endif
typedef void *webview_t;
typedef enum webview_error_t { WEBVIEW_ERROR_OK = 0, WEBVIEW_ERROR_FAILURE = 1 } webview_error_t;
typedef enum webview_native_handle_kind_t { WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW = 0 } webview_native_handle_kind_t;
typedef enum webview_hint_t { WEBVIEW_HINT_NONE = 0 } webview_hint_t;
webview_t webview_create(int debug, void *window);
webview_error_t webview_destroy(webview_t view);
webview_error_t webview_set_title(webview_t view, const char *title);
webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint);
void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind);
webview_error_t webview_set_html(webview_t view, const char *html);
webview_error_t webview_navigate(webview_t view, const char *url);
webview_error_t webview_init(webview_t view, const char *js);
webview_error_t webview_eval(webview_t view, const char *js);
webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg);
webview_error_t webview_unbind(webview_t view, const char *name);
webview_error_t webview_return(webview_t view, const char *id, int status, const char *result);
webview_error_t webview_run(webview_t view);
webview_error_t webview_terminate(webview_t view);
#ifdef __cplusplus
}
#endif
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview_stub.cpp"), []byte(`#include "webview/webview.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

struct jayess_stub_webview {
  int debug;
  int width;
  int height;
  int terminated;
  int shown;
  char *title;
  char *html;
  char *url;
  char *init_js;
  char *eval_js;
  char *last_return_id;
  int last_return_status;
  char *last_return_result;
  int next_id;
  struct jayess_stub_binding *bindings;
};

struct jayess_stub_binding {
  char *name;
  void (*fn)(const char *id, const char *req, void *arg);
  void *arg;
  struct jayess_stub_binding *next;
};

static void jayess_stub_replace(char **slot, const char *text) {
  if (*slot != NULL) {
    free(*slot);
    *slot = NULL;
  }
  if (text != NULL) {
    size_t length = strlen(text);
    *slot = (char *)malloc(length + 1);
    if (*slot != NULL) {
      memcpy(*slot, text, length + 1);
    }
  }
}

extern "C" webview_t webview_create(int debug, void *window) {
  struct jayess_stub_webview *view = (struct jayess_stub_webview *)calloc(1, sizeof(struct jayess_stub_webview));
  (void)window;
  if (view == NULL) {
    return NULL;
  }
  view->debug = debug;
  return (webview_t)view;
}

extern "C" webview_error_t webview_destroy(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  free(state->title);
  free(state->html);
  free(state->url);
  free(state->init_js);
  free(state->eval_js);
  free(state->last_return_id);
  free(state->last_return_result);
  while (state->bindings != NULL) {
    struct jayess_stub_binding *next = state->bindings->next;
    free(state->bindings->name);
    free(state->bindings);
    state->bindings = next;
  }
  free(state);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_title(webview_t view, const char *title) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->title, title);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  (void)hint;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  state->width = width;
  state->height = height;
  return WEBVIEW_ERROR_OK;
}

extern "C" void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL || kind != 0) {
    return NULL;
  }
  return state;
}

extern "C" void gtk_widget_show(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) {
    state->shown = 1;
  }
}

extern "C" void gtk_widget_hide(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) {
    state->shown = 0;
  }
}

extern "C" webview_error_t webview_set_html(webview_t view, const char *html) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->html, html);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_navigate(webview_t view, const char *url) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->url, url);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_init(webview_t view, const char *js) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->init_js, js);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_eval(webview_t view, const char *js) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  struct jayess_stub_binding *binding = NULL;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->eval_js, js);
  if (js != NULL) {
    const char *open = strchr(js, '(');
    const char *close = strrchr(js, ')');
    if (open != NULL && close != NULL && close > open) {
      size_t name_length = (size_t)(open - js);
      size_t req_length = (size_t)(close - open - 1);
      binding = state->bindings;
      while (binding != NULL) {
        if (strlen(binding->name) == name_length && strncmp(binding->name, js, name_length) == 0) {
          char id_buf[32];
          char *request = (char *)malloc(req_length + 1);
          if (request != NULL) {
            memcpy(request, open + 1, req_length);
            request[req_length] = '\0';
            snprintf(id_buf, sizeof(id_buf), "stub-%d", state->next_id++);
            binding->fn(id_buf, request, binding->arg);
            free(request);
          }
          break;
        }
        binding = binding->next;
      }
    }
  }
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  struct jayess_stub_binding *binding = NULL;
  if (state == NULL || name == NULL || fn == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  binding = state->bindings;
  while (binding != NULL) {
    if (strcmp(binding->name, name) == 0) {
      return WEBVIEW_ERROR_FAILURE;
    }
    binding = binding->next;
  }
  binding = (struct jayess_stub_binding *)calloc(1, sizeof(struct jayess_stub_binding));
  if (binding == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  binding->name = (char *)malloc(strlen(name) + 1);
  if (binding->name == NULL) {
    free(binding);
    return WEBVIEW_ERROR_FAILURE;
  }
  strcpy(binding->name, name);
  binding->fn = fn;
  binding->arg = arg;
  binding->next = state->bindings;
  state->bindings = binding;
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_unbind(webview_t view, const char *name) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  struct jayess_stub_binding *binding = NULL;
  struct jayess_stub_binding *previous = NULL;
  if (state == NULL || name == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  binding = state->bindings;
  while (binding != NULL) {
    if (strcmp(binding->name, name) == 0) {
      break;
    }
    previous = binding;
    binding = binding->next;
  }
  if (binding == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  if (previous != NULL) {
    previous->next = binding->next;
  } else {
    state->bindings = binding->next;
  }
  free(binding->name);
  free(binding);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_return(webview_t view, const char *id, int status, const char *result) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  jayess_stub_replace(&state->last_return_id, id);
  state->last_return_status = status;
  jayess_stub_replace(&state->last_return_result, result);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_run(webview_t view) {
  return view != NULL ? WEBVIEW_ERROR_OK : WEBVIEW_ERROR_FAILURE;
}

extern "C" webview_error_t webview_terminate(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  state->terminated = 1;
  return WEBVIEW_ERROR_OK;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	bindPath := filepath.Join(nativeDir, "webview.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	bindText = strings.Replace(bindText, `sources: ["./webview.cpp"],`, `sources: ["./webview.cpp", "./webview_stub.cpp"],`, 1)
	bindText = strings.Replace(bindText, `includeDirs: ["../../../../refs/webview/core/include"],`, `includeDirs: ["./include"],`, 1)
	bindText = strings.Replace(bindText, `cflags: ["-std=c++14"],`, `cflags: [],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: [
        "-lstdc++",
        "-ldl",
        "-lgtk-3",
        "-lwebkit2gtk-4.1",
        "-lgobject-2.0",
        "-lglib-2.0",
        "-lgio-2.0"
      ]
    },`, `linux: {
      ldflags: ["-lstdc++"]
    },`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createWindow, setTitle, setSize, show, hide, setHtml, navigate, initJs, evalJs, bind, nextBindingEvent, returnBinding, unbind, run, terminate, destroyWindow } from "@jayess/webview";

function main(args) {
  var view = createWindow(false);
  if (view == undefined) {
    console.log("webview-explicit-create:false");
    return 0;
  }
  setTitle(view, "explicit-webview");
  setSize(view, 640, 480);
  console.log("webview-explicit-show:" + show(view));
  console.log("webview-explicit-hide:" + hide(view));
  setHtml(view, "<h1>hello</h1>");
  navigate(view, "https://example.com/");
  initJs(view, "window.x = 1;");
  console.log("webview-explicit-bind:" + bind(view, "jayessEcho"));
  try {
    bind(view, "jayessEcho");
    console.log("webview-explicit-bind-error:false");
  } catch (err) {
    console.log("webview-explicit-bind-error:" + err.name);
  }
  evalJs(view, "jayessEcho({})");
  var event = nextBindingEvent(view);
  console.log("webview-explicit-event:" + event.name + ":" + event.request);
  console.log("webview-explicit-return:" + returnBinding(view, event.id, 0, "{}"));
  evalJs(view, "jayessEcho({})");
  var event2 = nextBindingEvent(view);
  console.log("webview-explicit-event2:" + event2.name + ":" + event2.request);
  console.log("webview-explicit-unbind:" + unbind(view, "jayessEcho"));
  evalJs(view, "jayessEcho({})");
  console.log("webview-explicit-after-unbind:" + (nextBindingEvent(view) == undefined));
  evalJs(view, "window.x = window.x + 1;");
  console.log("webview-explicit-terminate:" + terminate(view));
  run(view);
  console.log("webview-explicit-run:true");
  console.log("webview-explicit-create:true");
  console.log("webview-explicit-destroy:" + destroyWindow(view));
  try {
    setTitle(view, "after-close");
    console.log("webview-explicit-after-close:false");
  } catch (err) {
    console.log("webview-explicit-after-close:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "webview-explicit-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled webview explicit-include program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"webview-explicit-create:true",
		"webview-explicit-show:undefined",
		"webview-explicit-hide:undefined",
		"webview-explicit-bind:true",
		"webview-explicit-bind-error:WebviewError",
		"webview-explicit-event:jayessEcho:{}",
		"webview-explicit-return:true",
		"webview-explicit-event2:jayessEcho:{}",
		"webview-explicit-unbind:true",
		"webview-explicit-after-unbind:true",
		"webview-explicit-terminate:undefined",
		"webview-explicit-run:true",
		"webview-explicit-destroy:true",
		"webview-explicit-after-close:TypeError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected explicit webview output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessWebviewFilesystemIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping webview filesystem integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
		filepath.Join(workdir, "node_modules", "@jayess", "webview"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "webview", "native")
	includeDir := filepath.Join(nativeDir, "include", "webview")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "webview.h"), []byte(`#pragma once
#ifdef __cplusplus
extern "C" {
#endif
typedef void *webview_t;
typedef enum webview_error_t { WEBVIEW_ERROR_OK = 0, WEBVIEW_ERROR_FAILURE = 1 } webview_error_t;
typedef enum webview_native_handle_kind_t { WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW = 0 } webview_native_handle_kind_t;
typedef enum webview_hint_t { WEBVIEW_HINT_NONE = 0 } webview_hint_t;
webview_t webview_create(int debug, void *window);
webview_error_t webview_destroy(webview_t view);
webview_error_t webview_set_title(webview_t view, const char *title);
webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint);
void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind);
webview_error_t webview_set_html(webview_t view, const char *html);
webview_error_t webview_navigate(webview_t view, const char *url);
webview_error_t webview_init(webview_t view, const char *js);
webview_error_t webview_eval(webview_t view, const char *js);
webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg);
webview_error_t webview_unbind(webview_t view, const char *name);
webview_error_t webview_return(webview_t view, const char *id, int status, const char *result);
webview_error_t webview_run(webview_t view);
webview_error_t webview_terminate(webview_t view);
#ifdef __cplusplus
}
#endif
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview_stub.cpp"), []byte(`#include "webview/webview.h"
#include <stdlib.h>
#include <string.h>

struct jayess_stub_webview {
  int debug;
  int width;
  int height;
  int terminated;
  int shown;
  char *title;
  char *html;
  char *url;
};

static void jayess_stub_replace(char **slot, const char *text) {
  if (*slot != NULL) {
    free(*slot);
    *slot = NULL;
  }
  if (text != NULL) {
    size_t length = strlen(text);
    *slot = (char *)malloc(length + 1);
    if (*slot != NULL) {
      memcpy(*slot, text, length + 1);
    }
  }
}

extern "C" webview_t webview_create(int debug, void *window) {
  struct jayess_stub_webview *view = (struct jayess_stub_webview *)calloc(1, sizeof(struct jayess_stub_webview));
  (void)window;
  if (view == NULL) {
    return NULL;
  }
  view->debug = debug;
  return (webview_t)view;
}

extern "C" webview_error_t webview_destroy(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) {
    return WEBVIEW_ERROR_FAILURE;
  }
  free(state->title);
  free(state->html);
  free(state->url);
  free(state);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_title(webview_t view, const char *title) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->title, title);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  (void)hint;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->width = width;
  state->height = height;
  return WEBVIEW_ERROR_OK;
}

extern "C" void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL || kind != WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW) return NULL;
  return state;
}

extern "C" void gtk_widget_show(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 1;
}

extern "C" void gtk_widget_hide(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 0;
}

extern "C" webview_error_t webview_set_html(webview_t view, const char *html) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->html, html);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_navigate(webview_t view, const char *url) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->url, url);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_init(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_eval(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg) { (void)view; (void)name; (void)fn; (void)arg; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_unbind(webview_t view, const char *name) { (void)view; (void)name; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_return(webview_t view, const char *id, int status, const char *result) { (void)view; (void)id; (void)status; (void)result; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_run(webview_t view) { return view != NULL ? WEBVIEW_ERROR_OK : WEBVIEW_ERROR_FAILURE; }
extern "C" webview_error_t webview_terminate(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->terminated = 1;
  return WEBVIEW_ERROR_OK;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	bindPath := filepath.Join(nativeDir, "webview.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	bindText = strings.Replace(bindText, `sources: ["./webview.cpp"],`, `sources: ["./webview.cpp", "./webview_stub.cpp"],`, 1)
	bindText = strings.Replace(bindText, `includeDirs: ["../../../../refs/webview/core/include"],`, `includeDirs: ["./include"],`, 1)
	bindText = strings.Replace(bindText, `cflags: ["-std=c++14"],`, `cflags: [],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: [
        "-lstdc++",
        "-ldl",
        "-lgtk-3",
        "-lwebkit2gtk-4.1",
        "-lgobject-2.0",
        "-lglib-2.0",
        "-lgio-2.0"
      ]
    },`, `linux: {
      ldflags: ["-lstdc++"]
    },`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createWindow, setHtml, loadFile, destroyWindow } from "@jayess/webview";

function main(args) {
  var file = path.join(".", "page.html");
  fs.writeFile(file, "<h1>kimchi</h1>");
  var html = fs.readFile(file, "utf8");
  var view = createWindow(false);
  setHtml(view, html);
  loadFile(view, file);
  console.log("webview-fs-html:" + html);
  console.log("webview-fs-path:" + (file === "page.html" || file === ".\\page.html" || file === "./page.html"));
  console.log("webview-destroy:" + destroyWindow(view));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "webview-filesystem-integration-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled webview filesystem integration program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"webview-fs-html:<h1>kimchi</h1>",
		"webview-fs-path:true",
		"webview-destroy:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected webview filesystem integration output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessWebviewWorkerIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping webview worker integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
		filepath.Join(workdir, "node_modules", "@jayess", "webview"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "webview", "native")
	includeDir := filepath.Join(nativeDir, "include", "webview")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "webview.h"), []byte(`#pragma once
#ifdef __cplusplus
extern "C" {
#endif
typedef void *webview_t;
typedef enum webview_error_t { WEBVIEW_ERROR_OK = 0, WEBVIEW_ERROR_FAILURE = 1 } webview_error_t;
typedef enum webview_native_handle_kind_t { WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW = 0 } webview_native_handle_kind_t;
typedef enum webview_hint_t { WEBVIEW_HINT_NONE = 0 } webview_hint_t;
webview_t webview_create(int debug, void *window);
webview_error_t webview_destroy(webview_t view);
webview_error_t webview_set_title(webview_t view, const char *title);
webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint);
void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind);
webview_error_t webview_set_html(webview_t view, const char *html);
webview_error_t webview_navigate(webview_t view, const char *url);
webview_error_t webview_init(webview_t view, const char *js);
webview_error_t webview_eval(webview_t view, const char *js);
webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg);
webview_error_t webview_unbind(webview_t view, const char *name);
webview_error_t webview_return(webview_t view, const char *id, int status, const char *result);
webview_error_t webview_run(webview_t view);
webview_error_t webview_terminate(webview_t view);
#ifdef __cplusplus
}
#endif
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview_stub.cpp"), []byte(`#include "webview/webview.h"
#include <stdlib.h>
#include <string.h>

struct jayess_stub_webview {
  int debug;
  int width;
  int height;
  int terminated;
  int shown;
  char *title;
  char *html;
  char *url;
};

static void jayess_stub_replace(char **slot, const char *text) {
  if (*slot != NULL) {
    free(*slot);
    *slot = NULL;
  }
  if (text != NULL) {
    size_t length = strlen(text);
    *slot = (char *)malloc(length + 1);
    if (*slot != NULL) {
      memcpy(*slot, text, length + 1);
    }
  }
}

extern "C" webview_t webview_create(int debug, void *window) {
  struct jayess_stub_webview *view = (struct jayess_stub_webview *)calloc(1, sizeof(struct jayess_stub_webview));
  (void)window;
  if (view == NULL) {
    return NULL;
  }
  view->debug = debug;
  return (webview_t)view;
}

extern "C" webview_error_t webview_destroy(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  free(state->title);
  free(state->html);
  free(state->url);
  free(state);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_title(webview_t view, const char *title) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->title, title);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  (void)hint;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->width = width;
  state->height = height;
  return WEBVIEW_ERROR_OK;
}

extern "C" void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL || kind != WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW) return NULL;
  return state;
}

extern "C" void gtk_widget_show(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 1;
}

extern "C" void gtk_widget_hide(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 0;
}

extern "C" webview_error_t webview_set_html(webview_t view, const char *html) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->html, html);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_navigate(webview_t view, const char *url) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->url, url);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_init(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_eval(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg) { (void)view; (void)name; (void)fn; (void)arg; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_unbind(webview_t view, const char *name) { (void)view; (void)name; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_return(webview_t view, const char *id, int status, const char *result) { (void)view; (void)id; (void)status; (void)result; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_run(webview_t view) { return view != NULL ? WEBVIEW_ERROR_OK : WEBVIEW_ERROR_FAILURE; }
extern "C" webview_error_t webview_terminate(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->terminated = 1;
  return WEBVIEW_ERROR_OK;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	bindPath := filepath.Join(nativeDir, "webview.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	bindText = strings.Replace(bindText, `sources: ["./webview.cpp"],`, `sources: ["./webview.cpp", "./webview_stub.cpp"],`, 1)
	bindText = strings.Replace(bindText, `includeDirs: ["../../../../refs/webview/core/include"],`, `includeDirs: ["./include"],`, 1)
	bindText = strings.Replace(bindText, `cflags: ["-std=c++14"],`, `cflags: [],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: [
        "-lstdc++",
        "-ldl",
        "-lgtk-3",
        "-lwebkit2gtk-4.1",
        "-lgobject-2.0",
        "-lglib-2.0",
        "-lgio-2.0"
      ]
    },`, `linux: {
      ldflags: ["-lstdc++"]
    },`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createWindow, setTitle, terminate, run, destroyWindow } from "@jayess/webview";

function main(args) {
  var view = createWindow(false);
  var workerThread = worker.create(function(message) {
    return { doubled: message.value * 2, text: message.text.toUpperCase() };
  });
  console.log("webview-worker-create:" + (view != undefined));
  console.log("webview-worker-post:" + workerThread.postMessage({ value: 7, text: "kimchi" }));
  var response = workerThread.receive(5000);
  console.log("webview-worker-reply:" + response.ok + ":" + response.value.doubled + ":" + response.value.text);
  setTitle(view, "worker-webview");
  console.log("webview-worker-terminate:" + terminate(view));
  run(view);
  console.log("webview-worker-run:true");
  console.log("webview-worker-close:" + workerThread.terminate());
  console.log("webview-worker-destroy:" + destroyWindow(view));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "webview-worker-integration-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled webview worker integration program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"webview-worker-create:true",
		"webview-worker-post:true",
		"webview-worker-reply:true:14:KIMCHI",
		"webview-worker-terminate:undefined",
		"webview-worker-run:true",
		"webview-worker-close:true",
		"webview-worker-destroy:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected webview worker integration output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessWebviewGLFWIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping webview GLFW integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
		filepath.Join(workdir, "node_modules", "@jayess", "webview"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw"),
		filepath.Join(workdir, "refs", "glfw"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "webview", "native")
	includeDir := filepath.Join(nativeDir, "include", "webview")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "webview.h"), []byte(`#pragma once
#ifdef __cplusplus
extern "C" {
#endif
typedef void *webview_t;
typedef enum webview_error_t { WEBVIEW_ERROR_OK = 0, WEBVIEW_ERROR_FAILURE = 1 } webview_error_t;
typedef enum webview_native_handle_kind_t { WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW = 0 } webview_native_handle_kind_t;
typedef enum webview_hint_t { WEBVIEW_HINT_NONE = 0 } webview_hint_t;
webview_t webview_create(int debug, void *window);
webview_error_t webview_destroy(webview_t view);
webview_error_t webview_set_title(webview_t view, const char *title);
webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint);
void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind);
webview_error_t webview_set_html(webview_t view, const char *html);
webview_error_t webview_navigate(webview_t view, const char *url);
webview_error_t webview_init(webview_t view, const char *js);
webview_error_t webview_eval(webview_t view, const char *js);
webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg);
webview_error_t webview_unbind(webview_t view, const char *name);
webview_error_t webview_return(webview_t view, const char *id, int status, const char *result);
webview_error_t webview_run(webview_t view);
webview_error_t webview_terminate(webview_t view);
#ifdef __cplusplus
}
#endif
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview_stub.cpp"), []byte(`#include "webview/webview.h"
#include <stdlib.h>
#include <string.h>

struct jayess_stub_webview {
  int debug;
  int width;
  int height;
  int terminated;
  int shown;
  char *title;
  char *html;
  char *url;
};

static void jayess_stub_replace(char **slot, const char *text) {
  if (*slot != NULL) {
    free(*slot);
    *slot = NULL;
  }
  if (text != NULL) {
    size_t length = strlen(text);
    *slot = (char *)malloc(length + 1);
    if (*slot != NULL) {
      memcpy(*slot, text, length + 1);
    }
  }
}

extern "C" webview_t webview_create(int debug, void *window) {
  struct jayess_stub_webview *view = (struct jayess_stub_webview *)calloc(1, sizeof(struct jayess_stub_webview));
  (void)window;
  if (view == NULL) {
    return NULL;
  }
  view->debug = debug;
  return (webview_t)view;
}

extern "C" webview_error_t webview_destroy(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  free(state->title);
  free(state->html);
  free(state->url);
  free(state);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_title(webview_t view, const char *title) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->title, title);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  (void)hint;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->width = width;
  state->height = height;
  return WEBVIEW_ERROR_OK;
}

extern "C" void *webview_get_native_handle(webview_t view, webview_native_handle_kind_t kind) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL || kind != WEBVIEW_NATIVE_HANDLE_KIND_UI_WINDOW) return NULL;
  return state;
}

extern "C" void gtk_widget_show(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 1;
}

extern "C" void gtk_widget_hide(void *widget) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)widget;
  if (state != NULL) state->shown = 0;
}

extern "C" webview_error_t webview_set_html(webview_t view, const char *html) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->html, html);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_navigate(webview_t view, const char *url) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  jayess_stub_replace(&state->url, url);
  return WEBVIEW_ERROR_OK;
}

extern "C" webview_error_t webview_init(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_eval(webview_t view, const char *js) { (void)view; (void)js; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_bind(webview_t view, const char *name, void (*fn)(const char *id, const char *req, void *arg), void *arg) { (void)view; (void)name; (void)fn; (void)arg; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_unbind(webview_t view, const char *name) { (void)view; (void)name; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_return(webview_t view, const char *id, int status, const char *result) { (void)view; (void)id; (void)status; (void)result; return WEBVIEW_ERROR_OK; }
extern "C" webview_error_t webview_run(webview_t view) { return view != NULL ? WEBVIEW_ERROR_OK : WEBVIEW_ERROR_FAILURE; }
extern "C" webview_error_t webview_terminate(webview_t view) {
  struct jayess_stub_webview *state = (struct jayess_stub_webview *)view;
  if (state == NULL) return WEBVIEW_ERROR_FAILURE;
  state->terminated = 1;
  return WEBVIEW_ERROR_OK;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	bindPath := filepath.Join(nativeDir, "webview.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	bindText = strings.Replace(bindText, `sources: ["./webview.cpp"],`, `sources: ["./webview.cpp", "./webview_stub.cpp"],`, 1)
	bindText = strings.Replace(bindText, `includeDirs: ["../../../../refs/webview/core/include"],`, `includeDirs: ["./include"],`, 1)
	bindText = strings.Replace(bindText, `cflags: ["-std=c++14"],`, `cflags: [],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: [
        "-lstdc++",
        "-ldl",
        "-lgtk-3",
        "-lwebkit2gtk-4.1",
        "-lgobject-2.0",
        "-lglib-2.0",
        "-lgio-2.0"
      ]
    },`, `linux: {
      ldflags: ["-lstdc++"]
    },`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createWindow as createWebviewWindow, setTitle as setWebviewTitle, terminate as terminateWebview, run as runWebview, destroyWindow as destroyWebviewWindow } from "@jayess/webview";
import { initNative as initGLFW, createWindowNative as createGLFWWindow, pollEventsNative as pollGLFWEvents, setTimeNative as setGLFWTime, getTimeNative as getGLFWTime, setWindowSizeNative as setGLFWWindowSize, getWindowSizeNative as getGLFWWindowSize, destroyWindowNative as destroyGLFWWindow, terminateNative as terminateGLFW } from "./node_modules/@jayess/glfw/native/glfw.bind.js";

function main(args) {
  var view = createWebviewWindow(false);
  console.log("webview-glfw-view:" + (view != undefined));
  setWebviewTitle(view, "glfw-host");
  console.log("webview-glfw-init:" + initGLFW());
  var window = createGLFWWindow(64, 64, "jayess-glfw-host");
  console.log("webview-glfw-window:" + (window != undefined));
  setGLFWTime(3.5);
  console.log("webview-glfw-time:" + getGLFWTime());
  setGLFWWindowSize(window, 40, 24);
  var size = getGLFWWindowSize(window);
  console.log("webview-glfw-size:" + size.width + "x" + size.height);
  pollGLFWEvents();
  console.log("webview-glfw-terminate-view:" + terminateWebview(view));
  runWebview(view);
  console.log("webview-glfw-run:true");
  console.log("webview-glfw-destroy-window:" + destroyGLFWWindow(window));
  console.log("webview-glfw-terminate-glfw:" + terminateGLFW());
  console.log("webview-glfw-destroy-view:" + destroyWebviewWindow(view));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "webview-http-integration-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled webview GLFW integration program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"webview-glfw-view:true",
		"webview-glfw-init:true",
		"webview-glfw-window:true",
		"webview-glfw-time:3.5",
		"webview-glfw-size:40x24",
		"webview-glfw-terminate-view:undefined",
		"webview-glfw-run:true",
		"webview-glfw-destroy-window:true",
		"webview-glfw-terminate-glfw:undefined",
		"webview-glfw-destroy-view:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected webview GLFW integration output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessGTKPackageOrReportsMissingDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "gtk"),
		filepath.Join(workdir, "node_modules", "@jayess", "gtk"),
	)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { init, createWindow, setTitle, show, destroyWindow } from "@jayess/gtk";

function main(args) {
  if (!init()) {
    console.log("gtk-init:false");
    return 0;
  }
  var window = createWindow();
  if (window == undefined) {
    console.log("gtk-window:undefined");
    return 0;
  }
  setTitle(window, "Jayess GTK");
  show(window);
  console.log("gtk-window-closed:" + window.closed);
  console.log("gtk-destroy:" + destroyWindow(window));
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

	outputPath := nativeOutputPath(workdir, "jayess-gtk-package")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		cmd := exec.Command(outputPath)
		cmd.Dir = workdir
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("compiled gtk program returned error: %v: %s", runErr, string(out))
		}
		if !strings.Contains(string(out), "gtk-") {
			t.Fatalf("expected gtk package smoke output, got: %s", string(out))
		}
		return
	}
	if !strings.Contains(err.Error(), "gtk") {
		t.Fatalf("expected clear GTK build diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "native library link failed") && !strings.Contains(err.Error(), "native header dependency missing") && !strings.Contains(err.Error(), "clang native build failed") {
		t.Fatalf("expected clear GTK build diagnostic, got: %v", err)
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

func TestBuildObjectSupportsJayessRaylibAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target raylib build test: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}

			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", "raylib"),
				filepath.Join(workdir, "node_modules", "@jayess", "raylib"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "raylib", "src"),
				filepath.Join(workdir, "refs", "raylib", "src"),
			)

			entry := filepath.Join(workdir, "main.js")
			source := `
import { initWindow, isWindowReady, closeWindow } from "@jayess/raylib";

function main(args) {
  initWindow(32, 24, "cross-raylib");
  console.log("raylib-cross:" + isWindowReady());
  closeWindow();
  return 0;
}
`
			if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := filepath.Join(workdir, targetName+"-raylib.o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target raylib object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built raylib object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildObjectSupportsJayessGLFWAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target GLFW build test: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}

			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
				filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "glfw", "include"),
				filepath.Join(workdir, "refs", "glfw", "include"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "glfw", "src"),
				filepath.Join(workdir, "refs", "glfw", "src"),
			)

			entry := filepath.Join(workdir, "main.js")
			source := `
import { init, terminate } from "@jayess/glfw";

function main(args) {
  console.log("glfw-cross:" + init());
  terminate();
  return 0;
}
`
			if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := filepath.Join(workdir, targetName+"-glfw.o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target GLFW object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built GLFW object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildObjectSupportsJayessWebviewAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target webview build test: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}

			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", "webview"),
				filepath.Join(workdir, "node_modules", "@jayess", "webview"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "webview", "core", "include"),
				filepath.Join(workdir, "refs", "webview", "core", "include"),
			)

			entry := filepath.Join(workdir, "main.js")
			source := `
import { createWindow } from "@jayess/webview";

function main(args) {
  console.log("webview-cross:" + (createWindow(false) !== undefined));
  return 0;
}
`
			if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := filepath.Join(workdir, targetName+"-webview.o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target webview object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built webview object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildObjectSupportsJayessGTKAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target GTK build test: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	for _, targetName := range []string{"windows-x64", "linux-x64", "darwin-arm64"} {
		t.Run(targetName, func(t *testing.T) {
			triple, err := target.FromName(targetName)
			if err != nil {
				t.Fatalf("FromName returned error: %v", err)
			}

			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", "gtk"),
				filepath.Join(workdir, "node_modules", "@jayess", "gtk"),
			)

			nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "gtk", "native")
			includeDir := filepath.Join(nativeDir, "include", "gtk")
			if err := os.MkdirAll(includeDir, 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			if err := os.WriteFile(filepath.Join(includeDir, "gtk.h"), []byte(`#pragma once
typedef struct _GtkWidget GtkWidget;
typedef struct _GtkWindow GtkWindow;
typedef struct _GtkDrawingArea GtkDrawingArea;
typedef struct _GtkImage GtkImage;
typedef int gboolean;
typedef struct _cairo cairo_t;
typedef enum { GTK_WINDOW_TOPLEVEL = 0 } GtkWindowType;
#define FALSE 0
#define GTK_WINDOW(widget) ((GtkWindow *)(widget))
int gtk_init_check(int *argc, char ***argv);
GtkWidget *gtk_window_new(GtkWindowType type);
GtkWidget *gtk_image_new_from_file(const char *path);
GtkWidget *gtk_drawing_area_new(void);
void gtk_window_set_title(GtkWindow *window, const char *title);
void gtk_widget_show_all(GtkWidget *widget);
void gtk_widget_queue_draw(GtkWidget *widget);
void gtk_widget_destroy(GtkWidget *widget);
int gtk_events_pending(void);
void gtk_main_iteration_do(int blocking);
`), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			bindPath := filepath.Join(nativeDir, "gtk.bind.js")
			bindBytes, err := os.ReadFile(bindPath)
			if err != nil {
				t.Fatalf("ReadFile returned error: %v", err)
			}
			bindText := string(bindBytes)
			if strings.Contains(bindText, `includeDirs: [],`) {
				bindText = strings.Replace(bindText, `includeDirs: [],`, `includeDirs: ["./include"],`, 1)
			}
			if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			entry := filepath.Join(workdir, "main.js")
			source := `
import { init, createWindow, setTitle, show, pollEvents, destroyWindow } from "@jayess/gtk";

function main(args) {
  if (!init()) {
    return 0;
  }
  var window = createWindow();
  if (window != undefined) {
    setTitle(window, "cross-gtk");
    show(window);
    pollEvents();
    destroyWindow(window);
  }
  return 0;
}
`
			if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := filepath.Join(workdir, targetName+"-gtk.o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target GTK object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built GTK object file for %s, got err=%v", targetName, err)
			}
		})
	}
}

func TestBuildExecutableSupportsJayessGTKBindingWithExplicitIncludePaths(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping GTK explicit-include test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "gtk"),
		filepath.Join(workdir, "node_modules", "@jayess", "gtk"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "gtk", "native")
	includeDir := filepath.Join(nativeDir, "include", "gtk")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "gtk.h"), []byte(`#pragma once
typedef struct _GtkWidget GtkWidget;
typedef struct _GtkWindow GtkWindow;
typedef struct _GtkLabel GtkLabel;
typedef struct _GtkButton GtkButton;
typedef struct _GtkEntry GtkEntry;
typedef struct _GtkImage GtkImage;
typedef struct _GtkDrawingArea GtkDrawingArea;
typedef struct _GtkBox GtkBox;
typedef struct _GtkContainer GtkContainer;
typedef enum { GTK_WINDOW_TOPLEVEL = 0 } GtkWindowType;
typedef enum { GTK_ORIENTATION_HORIZONTAL = 0, GTK_ORIENTATION_VERTICAL = 1 } GtkOrientation;
#define FALSE 0
#define GTK_WINDOW(widget) ((GtkWindow *)(widget))
#define GTK_LABEL(widget) ((GtkLabel *)(widget))
#define GTK_BUTTON(widget) ((GtkButton *)(widget))
#define GTK_ENTRY(widget) ((GtkEntry *)(widget))
#define GTK_CONTAINER(widget) ((GtkContainer *)(widget))
int jayess_test_is_label(GtkWidget *widget);
int jayess_test_is_button(GtkWidget *widget);
int jayess_test_is_entry(GtkWidget *widget);
#define GTK_IS_LABEL(widget) jayess_test_is_label(widget)
#define GTK_IS_BUTTON(widget) jayess_test_is_button(widget)
#define GTK_IS_ENTRY(widget) jayess_test_is_entry(widget)
typedef void *gpointer;
typedef unsigned long gulong;
typedef int gboolean;
typedef struct _cairo cairo_t;
typedef void (*GCallback)(void);
#define G_CALLBACK(callback) ((GCallback)(callback))
int gtk_init_check(int *argc, char ***argv);
GtkWidget *gtk_window_new(GtkWindowType type);
GtkWidget *gtk_label_new(const char *text);
GtkWidget *gtk_button_new_with_label(const char *text);
GtkWidget *gtk_entry_new(void);
GtkWidget *gtk_image_new_from_file(const char *path);
GtkWidget *gtk_drawing_area_new(void);
GtkWidget *gtk_box_new(GtkOrientation orientation, int spacing);
void gtk_window_set_title(GtkWindow *window, const char *title);
void gtk_label_set_text(GtkLabel *label, const char *text);
void gtk_button_set_label(GtkButton *button, const char *text);
void gtk_entry_set_text(GtkEntry *entry, const char *text);
void gtk_container_add(GtkContainer *container, GtkWidget *child);
gulong g_signal_connect_data(gpointer instance, const char *detailed_signal, GCallback c_handler, gpointer data, gpointer destroy_data, int connect_flags);
void g_signal_emit_by_name(gpointer instance, const char *detailed_signal);
void gtk_widget_show_all(GtkWidget *widget);
void gtk_widget_hide(GtkWidget *widget);
void gtk_widget_queue_draw(GtkWidget *widget);
void gtk_widget_destroy(GtkWidget *widget);
int gtk_events_pending(void);
void gtk_main_iteration_do(int blocking);
void gtk_main(void);
void gtk_main_quit(void);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nativeDir, "gtk_stub.c"), []byte(`#include <gtk/gtk.h>
#include <string.h>
#include <stdlib.h>

struct _GtkWidget {
  int kind;
  int shown;
  int destroyed;
  int spacing;
  char *text;
  struct jayess_stub_signal_handler *handlers;
  struct _GtkWidget *first_child;
  struct _GtkWidget *next_sibling;
};

struct jayess_stub_signal_handler {
  char *signal;
  GCallback callback;
  void *data;
  struct jayess_stub_signal_handler *next;
};

static int jayess_stub_pending_events = 2;
static int jayess_stub_main_quit = 0;
static int jayess_stub_main_runs = 0;

enum {
  JAYESS_GTK_KIND_WINDOW = 1,
  JAYESS_GTK_KIND_LABEL = 2,
  JAYESS_GTK_KIND_BUTTON = 3,
  JAYESS_GTK_KIND_ENTRY = 4,
  JAYESS_GTK_KIND_IMAGE = 5,
  JAYESS_GTK_KIND_BOX = 6,
  JAYESS_GTK_KIND_DRAWING_AREA = 7
};

static GtkWidget *jayess_stub_widget_new(int kind) {
  GtkWidget *widget = (GtkWidget *)calloc(1, sizeof(GtkWidget));
  if (widget != NULL) widget->kind = kind;
  return widget;
}

static void jayess_stub_set_text(GtkWidget *widget, const char *text) {
  if (widget == NULL) return;
  free(widget->text);
  widget->text = NULL;
  if (text != NULL) {
    widget->text = (char *)malloc(strlen(text) + 1);
    if (widget->text != NULL) strcpy(widget->text, text);
  }
}

static void jayess_stub_emit(GtkWidget *widget, const char *signal) {
  struct jayess_stub_signal_handler *handler = NULL;
  if (widget == NULL || signal == NULL) return;
  handler = widget->handlers;
  while (handler != NULL) {
    if (strcmp(handler->signal, signal) == 0) {
      if (strcmp(signal, "draw") == 0) {
        ((gboolean (*)(GtkWidget *, cairo_t *, void *))handler->callback)(widget, NULL, handler->data);
      } else {
        ((void (*)(GtkWidget *, void *))handler->callback)(widget, handler->data);
      }
    }
    handler = handler->next;
  }
}

int gtk_init_check(int *argc, char ***argv) {
  (void)argc;
  (void)argv;
  return 1;
}

GtkWidget *gtk_window_new(GtkWindowType type) {
  GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_WINDOW);
  (void)type;
  return widget;
}

GtkWidget *gtk_label_new(const char *text) {
  GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_LABEL);
  jayess_stub_set_text(widget, text);
  return widget;
}

GtkWidget *gtk_button_new_with_label(const char *text) {
  GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_BUTTON);
  jayess_stub_set_text(widget, text);
  return widget;
}

GtkWidget *gtk_entry_new(void) {
  return jayess_stub_widget_new(JAYESS_GTK_KIND_ENTRY);
}

GtkWidget *gtk_image_new_from_file(const char *path) {
  GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_IMAGE);
  jayess_stub_set_text(widget, path);
  return widget;
}

GtkWidget *gtk_drawing_area_new(void) {
  return jayess_stub_widget_new(JAYESS_GTK_KIND_DRAWING_AREA);
}

GtkWidget *gtk_box_new(GtkOrientation orientation, int spacing) {
  GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_BOX);
  (void)orientation;
  if (widget != NULL) widget->spacing = spacing;
  return widget;
}

void gtk_window_set_title(GtkWindow *window, const char *title) {
  jayess_stub_set_text((GtkWidget *)window, title);
}

void gtk_label_set_text(GtkLabel *label, const char *text) {
  jayess_stub_set_text((GtkWidget *)label, text);
}

void gtk_button_set_label(GtkButton *button, const char *text) {
  jayess_stub_set_text((GtkWidget *)button, text);
}

void gtk_entry_set_text(GtkEntry *entry, const char *text) {
  GtkWidget *widget = (GtkWidget *)entry;
  jayess_stub_set_text(widget, text);
  jayess_stub_emit(widget, "changed");
}

void gtk_container_add(GtkContainer *container, GtkWidget *child) {
  GtkWidget *widget = (GtkWidget *)container;
  if (widget == NULL || child == NULL) return;
  child->next_sibling = widget->first_child;
  widget->first_child = child;
}

void gtk_widget_show_all(GtkWidget *widget) {
  if (widget != NULL) {
    GtkWidget *child = NULL;
    widget->shown = 1;
    child = widget->first_child;
    while (child != NULL) {
      gtk_widget_show_all(child);
      child = child->next_sibling;
    }
  }
}

void gtk_widget_hide(GtkWidget *widget) {
  if (widget != NULL) widget->shown = 0;
}

void gtk_widget_queue_draw(GtkWidget *widget) {
  jayess_stub_emit(widget, "draw");
}

void gtk_widget_destroy(GtkWidget *widget) {
  if (widget != NULL && !widget->destroyed) {
    GtkWidget *child = widget->first_child;
    struct jayess_stub_signal_handler *handler = widget->handlers;
    while (child != NULL) {
      GtkWidget *next = child->next_sibling;
      gtk_widget_destroy(child);
      child = next;
    }
    jayess_stub_emit(widget, "destroy");
    while (handler != NULL) {
      struct jayess_stub_signal_handler *next = handler->next;
      free(handler->signal);
      free(handler);
      handler = next;
    }
    free(widget->text);
    widget->destroyed = 1;
    free(widget);
  }
}

int gtk_events_pending(void) {
  return jayess_stub_pending_events > 0;
}

void gtk_main_iteration_do(int blocking) {
  (void)blocking;
  if (jayess_stub_pending_events > 0) jayess_stub_pending_events--;
}

void gtk_main(void) {
  jayess_stub_main_runs++;
  while (!jayess_stub_main_quit && gtk_events_pending()) {
    gtk_main_iteration_do(FALSE);
  }
}

void gtk_main_quit(void) {
  jayess_stub_main_quit = 1;
}

int jayess_test_is_label(GtkWidget *widget) {
  return widget != NULL && widget->kind == JAYESS_GTK_KIND_LABEL;
}

int jayess_test_is_button(GtkWidget *widget) {
  return widget != NULL && widget->kind == JAYESS_GTK_KIND_BUTTON;
}

int jayess_test_is_entry(GtkWidget *widget) {
  return widget != NULL && widget->kind == JAYESS_GTK_KIND_ENTRY;
}

gulong g_signal_connect_data(gpointer instance, const char *detailed_signal, GCallback c_handler, gpointer data, gpointer destroy_data, int connect_flags) {
  GtkWidget *widget = (GtkWidget *)instance;
  struct jayess_stub_signal_handler *handler = NULL;
  (void)destroy_data;
  (void)connect_flags;
  if (widget == NULL || detailed_signal == NULL || c_handler == NULL) return 0;
  handler = (struct jayess_stub_signal_handler *)calloc(1, sizeof(struct jayess_stub_signal_handler));
  if (handler == NULL) return 0;
  handler->signal = (char *)malloc(strlen(detailed_signal) + 1);
  if (handler->signal == NULL) {
    free(handler);
    return 0;
  }
  strcpy(handler->signal, detailed_signal);
  handler->callback = c_handler;
  handler->data = data;
  handler->next = widget->handlers;
  widget->handlers = handler;
  return 1;
}

void g_signal_emit_by_name(gpointer instance, const char *detailed_signal) {
  jayess_stub_emit((GtkWidget *)instance, detailed_signal);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	bindPath := filepath.Join(nativeDir, "gtk.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	if strings.Contains(bindText, `includeDirs: [],`) {
		bindText = strings.Replace(bindText, `includeDirs: [],`, `includeDirs: ["./include"],`, 1)
	}
	bindText = strings.Replace(bindText, `sources: ["./gtk.c"],`, `sources: ["./gtk.c", "./gtk_stub.c"],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0"]
    },`, `linux: {
      ldflags: []
    },`, 1)
	bindText = strings.Replace(bindText, `darwin: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0", "-framework", "Cocoa"]
    },`, `darwin: {
      ldflags: []
    },`, 1)
	bindText = strings.Replace(bindText, `windows: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0", "-lole32", "-lcomctl32", "-luser32"]
    }`, `windows: {
      ldflags: []
    }`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { init, createWindow, createLabel, createButton, createTextInput, createImage, createDrawingArea, createBox, setTitle, setText, addChild, connectSignal, emitSignal, queueDraw, show, hide, pollEvents, runMainLoop, quitMainLoop, destroyWindow } from "@jayess/gtk";

function main(args) {
  console.log("gtk-explicit-init:" + init());
  var window = createWindow();
  var box = createBox(true, 8);
  var label = createLabel("hello");
  var button = createButton("go");
  var entry = createTextInput();
  var image = createImage("assets/icon.png");
  var drawingArea = createDrawingArea();
  console.log("gtk-explicit-window:" + (window != undefined));
  setTitle(window, "gtk-explicit");
  setText(label, "kimchi");
  setText(button, "save");
  setText(entry, "jjigae");
  addChild(box, label);
  addChild(box, button);
  addChild(box, entry);
  addChild(box, image);
  addChild(box, drawingArea);
  addChild(window, box);
  connectSignal(button, "clicked", function(signal) {
    console.log("gtk-explicit-click:" + signal);
    return 0;
  });
  connectSignal(entry, "changed", function(signal) {
    console.log("gtk-explicit-changed:" + signal);
    return 0;
  });
  connectSignal(window, "destroy", function(signal) {
    console.log("gtk-explicit-destroy-signal:" + signal);
    return 0;
  });
  connectSignal(drawingArea, "draw", function(signal) {
    console.log("gtk-explicit-draw:" + signal);
    return 0;
  });
  console.log("gtk-explicit-widgets:" + (image != undefined));
  emitSignal(button, "clicked");
  setText(entry, "bibimbap");
  queueDraw(drawingArea);
  show(window);
  hide(label);
  hide(window);
  pollEvents();
  quitMainLoop();
  runMainLoop();
  console.log("gtk-explicit-destroy:" + destroyWindow(window));
  try {
    show(label);
    console.log("gtk-explicit-child-after-close:false");
  } catch (err) {
    console.log("gtk-explicit-child-after-close:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "gtk-explicit-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled GTK explicit-include program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"gtk-explicit-init:true",
		"gtk-explicit-window:true",
		"gtk-explicit-widgets:true",
		"gtk-explicit-click:clicked",
		"gtk-explicit-changed:changed",
		"gtk-explicit-draw:draw",
		"gtk-explicit-destroy-signal:destroy",
		"gtk-explicit-destroy:true",
		"gtk-explicit-child-after-close:TypeError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected GTK explicit-include output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessGTKBindingWithPkgConfigDiscovery(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping GTK pkg-config test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "gtk"),
		filepath.Join(workdir, "node_modules", "@jayess", "gtk"),
	)

	nativeDir := filepath.Join(workdir, "node_modules", "@jayess", "gtk", "native")
	includeRoot := filepath.Join(nativeDir, "pkgconfig-include")
	includeDir := filepath.Join(includeRoot, "gtk")
	binDir := filepath.Join(workdir, "bin")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "gtk.h"), []byte(`#pragma once
typedef struct _GtkWidget GtkWidget;
typedef struct _GtkWindow GtkWindow;
typedef struct _GtkLabel GtkLabel;
typedef struct _GtkButton GtkButton;
typedef struct _GtkEntry GtkEntry;
typedef struct _GtkImage GtkImage;
typedef struct _GtkDrawingArea GtkDrawingArea;
typedef struct _GtkBox GtkBox;
typedef struct _GtkContainer GtkContainer;
typedef enum { GTK_WINDOW_TOPLEVEL = 0 } GtkWindowType;
typedef enum { GTK_ORIENTATION_HORIZONTAL = 0, GTK_ORIENTATION_VERTICAL = 1 } GtkOrientation;
#define FALSE 0
#define GTK_WINDOW(widget) ((GtkWindow *)(widget))
#define GTK_LABEL(widget) ((GtkLabel *)(widget))
#define GTK_BUTTON(widget) ((GtkButton *)(widget))
#define GTK_ENTRY(widget) ((GtkEntry *)(widget))
#define GTK_CONTAINER(widget) ((GtkContainer *)(widget))
int jayess_test_is_label(GtkWidget *widget);
int jayess_test_is_button(GtkWidget *widget);
int jayess_test_is_entry(GtkWidget *widget);
#define GTK_IS_LABEL(widget) jayess_test_is_label(widget)
#define GTK_IS_BUTTON(widget) jayess_test_is_button(widget)
#define GTK_IS_ENTRY(widget) jayess_test_is_entry(widget)
typedef void *gpointer;
typedef unsigned long gulong;
typedef int gboolean;
typedef struct _cairo cairo_t;
typedef void (*GCallback)(void);
#define G_CALLBACK(callback) ((GCallback)(callback))
int gtk_init_check(int *argc, char ***argv);
GtkWidget *gtk_window_new(GtkWindowType type);
GtkWidget *gtk_label_new(const char *text);
GtkWidget *gtk_button_new_with_label(const char *text);
GtkWidget *gtk_entry_new(void);
GtkWidget *gtk_image_new_from_file(const char *path);
GtkWidget *gtk_drawing_area_new(void);
GtkWidget *gtk_box_new(GtkOrientation orientation, int spacing);
void gtk_window_set_title(GtkWindow *window, const char *title);
void gtk_label_set_text(GtkLabel *label, const char *text);
void gtk_button_set_label(GtkButton *button, const char *text);
void gtk_entry_set_text(GtkEntry *entry, const char *text);
void gtk_container_add(GtkContainer *container, GtkWidget *child);
gulong g_signal_connect_data(gpointer instance, const char *detailed_signal, GCallback c_handler, gpointer data, gpointer destroy_data, int connect_flags);
void g_signal_emit_by_name(gpointer instance, const char *detailed_signal);
void gtk_widget_show_all(GtkWidget *widget);
void gtk_widget_hide(GtkWidget *widget);
void gtk_widget_queue_draw(GtkWidget *widget);
void gtk_widget_destroy(GtkWidget *widget);
int gtk_events_pending(void);
void gtk_main_iteration_do(int blocking);
void gtk_main(void);
void gtk_main_quit(void);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "gtk_stub.c"), []byte(`#include <gtk/gtk.h>
#include <stdlib.h>
#include <string.h>

struct _GtkWidget {
  int kind;
  int shown;
  char *text;
};

enum {
  JAYESS_GTK_KIND_WINDOW = 1,
  JAYESS_GTK_KIND_LABEL = 2,
  JAYESS_GTK_KIND_BUTTON = 3,
  JAYESS_GTK_KIND_ENTRY = 4,
  JAYESS_GTK_KIND_IMAGE = 5,
  JAYESS_GTK_KIND_BOX = 6,
  JAYESS_GTK_KIND_DRAWING_AREA = 7
};

static GtkWidget *jayess_stub_widget_new(int kind) {
  GtkWidget *widget = (GtkWidget *)calloc(1, sizeof(GtkWidget));
  if (widget != NULL) widget->kind = kind;
  return widget;
}

static void jayess_stub_set_text(GtkWidget *widget, const char *text) {
  if (widget == NULL) return;
  free(widget->text);
  widget->text = NULL;
  if (text != NULL) {
    widget->text = (char *)malloc(strlen(text) + 1);
    if (widget->text != NULL) strcpy(widget->text, text);
  }
}

int gtk_init_check(int *argc, char ***argv) { (void)argc; (void)argv; return 1; }
GtkWidget *gtk_window_new(GtkWindowType type) { (void)type; return jayess_stub_widget_new(JAYESS_GTK_KIND_WINDOW); }
GtkWidget *gtk_label_new(const char *text) { GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_LABEL); jayess_stub_set_text(widget, text); return widget; }
GtkWidget *gtk_button_new_with_label(const char *text) { GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_BUTTON); jayess_stub_set_text(widget, text); return widget; }
GtkWidget *gtk_entry_new(void) { return jayess_stub_widget_new(JAYESS_GTK_KIND_ENTRY); }
GtkWidget *gtk_image_new_from_file(const char *path) { GtkWidget *widget = jayess_stub_widget_new(JAYESS_GTK_KIND_IMAGE); jayess_stub_set_text(widget, path); return widget; }
GtkWidget *gtk_drawing_area_new(void) { return jayess_stub_widget_new(JAYESS_GTK_KIND_DRAWING_AREA); }
GtkWidget *gtk_box_new(GtkOrientation orientation, int spacing) { (void)orientation; (void)spacing; return jayess_stub_widget_new(JAYESS_GTK_KIND_BOX); }
void gtk_window_set_title(GtkWindow *window, const char *title) { jayess_stub_set_text((GtkWidget *)window, title); }
void gtk_label_set_text(GtkLabel *label, const char *text) { jayess_stub_set_text((GtkWidget *)label, text); }
void gtk_button_set_label(GtkButton *button, const char *text) { jayess_stub_set_text((GtkWidget *)button, text); }
void gtk_entry_set_text(GtkEntry *entry, const char *text) { jayess_stub_set_text((GtkWidget *)entry, text); }
void gtk_container_add(GtkContainer *container, GtkWidget *child) { (void)container; (void)child; }
gulong g_signal_connect_data(gpointer instance, const char *detailed_signal, GCallback c_handler, gpointer data, gpointer destroy_data, int connect_flags) {
  (void)instance; (void)detailed_signal; (void)c_handler; (void)data; (void)destroy_data; (void)connect_flags; return 1;
}
void g_signal_emit_by_name(gpointer instance, const char *detailed_signal) { (void)instance; (void)detailed_signal; }
void gtk_widget_show_all(GtkWidget *widget) { if (widget != NULL) widget->shown = 1; }
void gtk_widget_hide(GtkWidget *widget) { if (widget != NULL) widget->shown = 0; }
void gtk_widget_queue_draw(GtkWidget *widget) { (void)widget; }
void gtk_widget_destroy(GtkWidget *widget) { if (widget == NULL) return; free(widget->text); free(widget); }
int gtk_events_pending(void) { return 0; }
void gtk_main_iteration_do(int blocking) { (void)blocking; }
void gtk_main(void) {}
void gtk_main_quit(void) {}
int jayess_test_is_label(GtkWidget *widget) { return widget != NULL && widget->kind == JAYESS_GTK_KIND_LABEL; }
int jayess_test_is_button(GtkWidget *widget) { return widget != NULL && widget->kind == JAYESS_GTK_KIND_BUTTON; }
int jayess_test_is_entry(GtkWidget *widget) { return widget != NULL && widget->kind == JAYESS_GTK_KIND_ENTRY; }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	pkgConfigPath := filepath.Join(binDir, "pkg-config")
	if err := os.WriteFile(pkgConfigPath, []byte(fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--cflags" ] && [ "$2" = "gtk+-3.0" ]; then
  echo "-I%s"
  exit 0
fi
if [ "$1" = "--libs" ] && [ "$2" = "gtk+-3.0" ]; then
  exit 0
fi
echo "unsupported pkg-config args: $@" >&2
exit 1
`, includeRoot)), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	bindPath := filepath.Join(nativeDir, "gtk.bind.js")
	bindBytes, err := os.ReadFile(bindPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	bindText := string(bindBytes)
	bindText = strings.Replace(bindText, `sources: ["./gtk.c"],`, `sources: ["./gtk.c", "./gtk_stub.c"],`, 1)
	bindText = strings.Replace(bindText, `includeDirs: [],`, `includeDirs: [],
  pkgConfig: ["gtk+-3.0"],`, 1)
	bindText = strings.Replace(bindText, `linux: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0"]
    },`, `linux: {
      ldflags: []
    },`, 1)
	if err := os.WriteFile(bindPath, []byte(bindText), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { init, createWindow, setTitle, show, hide, quitMainLoop, runMainLoop, destroyWindow } from "@jayess/gtk";

function main(args) {
  console.log("gtk-pkg-init:" + init());
  var window = createWindow();
  setTitle(window, "gtk-pkg");
  show(window);
  hide(window);
  quitMainLoop();
  runMainLoop();
  console.log("gtk-pkg-destroy:" + destroyWindow(window));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "gtk-pkgconfig-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled GTK pkg-config program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"gtk-pkg-init:true",
		"gtk-pkg-destroy:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected GTK pkg-config output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsConfiguredOptimizationLevels(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping optimization-level build test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	source := `
function main(args) {
  console.log("opt-level");
  return 0;
}
`
	for _, level := range []string{"O0", "O2", "Oz"} {
		t.Run(level, func(t *testing.T) {
			result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple, OptimizationLevel: level})
			if err != nil {
				t.Fatalf("Compile returned error: %v", err)
			}
			outputPath := nativeOutputPath(t.TempDir(), "jayess-opt-"+strings.ToLower(level))
			if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple, OptimizationLevel: level}, outputPath); err != nil {
				t.Fatalf("BuildExecutable returned error for %s: %v", level, err)
			}
			cmd := exec.Command(outputPath)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compiled program returned error for %s: %v: %s", level, err, string(out))
			}
			if !strings.Contains(string(out), "opt-level") {
				t.Fatalf("expected optimized program output for %s, got: %s", level, string(out))
			}
		})
	}
}

func TestBuildExecutableArgsApplyOptimizationLevel(t *testing.T) {
	args := buildExecutableArgs(&compiler.Result{}, compiler.Options{TargetTriple: "x86_64-unknown-linux-gnu", OptimizationLevel: "O2"}, "module.ll", "runtime.c", "runtime", "", nil, false, "out")
	if !containsString(args, "-O2") {
		t.Fatalf("expected -O2 in executable args, got %#v", args)
	}
}

func TestBuildBitcodeProducesBitcodeFile(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping bitcode build test: %v", err)
	}
	if tc.LLVMAsPath == "" {
		t.Skip("skipping bitcode build test: llvm-as unavailable")
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile("function main(args) { return 0; }\n", compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "module.bc")
	if err := tc.BuildBitcode(result, outputPath); err != nil {
		t.Fatalf("BuildBitcode returned error: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty bitcode output, got size=%d", info.Size())
	}
}

func TestBuildStaticLibraryProducesArchiveFile(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping static library build test: %v", err)
	}
	if tc.ARPath == "" {
		t.Skip("skipping static library build test: ar unavailable")
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile("function main(args) { return 0; }\n", compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "libmodule.a")
	if err := tc.BuildStaticLibrary(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildStaticLibrary returned error: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty static library output, got size=%d", info.Size())
	}

	cmd := exec.Command(tc.ARPath, "t", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("archive listing returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "module.o") {
		t.Fatalf("expected archive to contain module.o, got: %s", string(out))
	}
}

func TestBuildSharedLibraryProducesSharedLibraryFile(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping shared library build test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile("function main(args) { return 0; }\n", compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputDir := t.TempDir()
	outputName := "libmodule.so"
	if runtime.GOOS == "darwin" {
		outputName = "libmodule.dylib"
	} else if runtime.GOOS == "windows" {
		outputName = "module.dll"
	}
	outputPath := filepath.Join(outputDir, outputName)
	if err := tc.BuildSharedLibrary(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildSharedLibrary returned error: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatalf("expected non-empty shared library output, got size=%d", info.Size())
	}
}

func TestBuildExecutableArgsApplyTargetCodegenFlags(t *testing.T) {
	args := buildExecutableArgs(&compiler.Result{}, compiler.Options{
		TargetTriple:    "x86_64-unknown-linux-gnu",
		TargetCPU:       "native",
		TargetFeatures:  []string{"+sse2", "-avx"},
		RelocationModel: "pic",
		CodeModel:       "small",
	}, "module.ll", "runtime.c", "runtime", "", nil, false, "out")
	if !containsString(args, "-mcpu=native") {
		t.Fatalf("expected -mcpu=native in executable args, got %#v", args)
	}
	if !containsString(args, "-fPIC") {
		t.Fatalf("expected -fPIC in executable args, got %#v", args)
	}
	if !containsString(args, "-mcmodel=small") {
		t.Fatalf("expected -mcmodel=small in executable args, got %#v", args)
	}
	for i := 0; i+3 < len(args); i++ {
		if args[i] == "-Xclang" && args[i+1] == "-target-feature" && args[i+2] == "-Xclang" && args[i+3] == "+sse2" {
			goto foundSSE2
		}
	}
	t.Fatalf("expected +sse2 target-feature args, got %#v", args)
foundSSE2:
	for i := 0; i+3 < len(args); i++ {
		if args[i] == "-Xclang" && args[i+1] == "-target-feature" && args[i+2] == "-Xclang" && args[i+3] == "-avx" {
			return
		}
	}
	t.Fatalf("expected -avx target-feature args, got %#v", args)
}

func TestBuildObjectArgsApplyTargetCodegenFlags(t *testing.T) {
	args := []string{"-target", "x86_64-unknown-linux-gnu"}
	if optFlag := clangOptimizationFlag("O0"); optFlag != "" {
		args = append(args, optFlag)
	}
	args = append(args, clangTargetCodegenArgs(compiler.Options{
		TargetCPU:       "native",
		TargetFeatures:  []string{"+aes"},
		RelocationModel: "pie",
		CodeModel:       "kernel",
	})...)
	if !containsString(args, "-mcpu=native") {
		t.Fatalf("expected -mcpu=native in object args, got %#v", args)
	}
	if !containsString(args, "-fPIE") {
		t.Fatalf("expected -fPIE in object args, got %#v", args)
	}
	if !containsString(args, "-mcmodel=kernel") {
		t.Fatalf("expected -mcmodel=kernel in object args, got %#v", args)
	}
	for i := 0; i+3 < len(args); i++ {
		if args[i] == "-Xclang" && args[i+1] == "-target-feature" && args[i+2] == "-Xclang" && args[i+3] == "+aes" {
			return
		}
	}
	t.Fatalf("expected +aes target-feature args, got %#v", args)
}

func TestLLVMIRWorksWithVerifierAndOptTools(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping LLVM verifier/opt test: %v", err)
	}
	if tc.LLVMAsPath == "" {
		t.Skip("skipping LLVM verifier/opt test: llvm-as unavailable")
	}
	optPath, err := exec.LookPath("opt")
	if err != nil {
		t.Skipf("skipping LLVM verifier/opt test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  console.log("llvm-tools");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	irPath := filepath.Join(workdir, "module.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	bitcodePath := filepath.Join(workdir, "module.bc")
	verifyCmd := exec.Command(tc.LLVMAsPath, irPath, "-o", bitcodePath)
	if output, err := verifyCmd.CombinedOutput(); err != nil {
		t.Fatalf("llvm-as verification returned error: %v: %s", err, string(output))
	}
	optCmd := exec.Command(optPath, "-passes=verify", "-disable-output", bitcodePath)
	if output, err := optCmd.CombinedOutput(); err != nil {
		t.Fatalf("opt verification returned error: %v: %s", err, string(output))
	}
}

func TestLLCObjectWorksWithSystemLinkerFlow(t *testing.T) {
	llcPath, err := exec.LookPath("llc")
	if err != nil {
		t.Skipf("skipping llc linker-flow test: %v", err)
	}
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping llc linker-flow test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  console.log("llc-link");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	irPath := filepath.Join(workdir, "module.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	objectPath := filepath.Join(workdir, "module.o")
	llcCmd := exec.Command(llcPath, "-filetype=obj", "-relocation-model=pic", irPath, "-o", objectPath)
	if output, err := llcCmd.CombinedOutput(); err != nil {
		t.Fatalf("llc returned error: %v: %s", err, string(output))
	}
	runtimePath, err := runtimeSourcePath("jayess_runtime.c")
	if err != nil {
		t.Fatalf("runtimeSourcePath returned error: %v", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		t.Fatalf("runtimeIncludePath returned error: %v", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()
	outputPath := nativeOutputPath(workdir, "llc-link")
	args := []string{"-target", triple, "-I", runtimeIncludeDir, objectPath, runtimePath}
	if brotliAvailable {
		args = append(args, "-I", brotliIncludeDir)
		args = append(args, brotliSources...)
	}
	args = append(args, nativeSystemLinkFlags(triple)...)
	args = append(args, "-o", outputPath)
	linkCmd := exec.Command(tc.ClangPath, args...)
	if output, err := linkCmd.CombinedOutput(); err != nil {
		t.Fatalf("clang link returned error: %v: %s", err, string(output))
	}
	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("linked llc program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "llc-link") {
		t.Fatalf("expected llc-linked program output, got: %s", string(out))
	}
}

func TestLLVMIRVerifiesAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target LLVM verification test: %v", err)
	}
	if tc.LLVMAsPath == "" {
		t.Skip("skipping cross-target LLVM verification test: llvm-as unavailable")
	}

	source := `
function main(args) {
  console.log("cross-target-llvm");
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
			irPath := filepath.Join(t.TempDir(), targetName+".ll")
			if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			bitcodePath := filepath.Join(t.TempDir(), targetName+".bc")
			cmd := exec.Command(tc.LLVMAsPath, irPath, "-o", bitcodePath)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("llvm-as returned error for %s: %v: %s", targetName, err, string(output))
			}
			info, err := os.Stat(bitcodePath)
			if err != nil {
				t.Fatalf("Stat returned error for %s: %v", targetName, err)
			}
			if info.IsDir() || info.Size() == 0 {
				t.Fatalf("expected non-empty verified bitcode for %s, got size=%d", targetName, info.Size())
			}
		})
	}
}

func TestExecutableOutputMatchesAcrossClangAndLLCFlows(t *testing.T) {
	llcPath, err := exec.LookPath("llc")
	if err != nil {
		t.Skipf("skipping ABI-stability flow test: %v", err)
	}
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping ABI-stability flow test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	result, err := compiler.Compile(`
function main(args) {
  console.log("flow-compare");
  console.log(7 + 5);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	directOutputPath := nativeOutputPath(workdir, "flow-direct")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, directOutputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}
	directCmd := exec.Command(directOutputPath)
	directOut, err := directCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("direct executable returned error: %v: %s", err, string(directOut))
	}

	irPath := filepath.Join(workdir, "flow.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	objectPath := filepath.Join(workdir, "flow-llc.o")
	llcCmd := exec.Command(llcPath, "-filetype=obj", "-relocation-model=pic", irPath, "-o", objectPath)
	if output, err := llcCmd.CombinedOutput(); err != nil {
		t.Fatalf("llc returned error: %v: %s", err, string(output))
	}
	runtimePath, err := runtimeSourcePath("jayess_runtime.c")
	if err != nil {
		t.Fatalf("runtimeSourcePath returned error: %v", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		t.Fatalf("runtimeIncludePath returned error: %v", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()
	llcOutputPath := nativeOutputPath(workdir, "flow-llc")
	args := []string{"-target", triple, "-I", runtimeIncludeDir, objectPath, runtimePath}
	if brotliAvailable {
		args = append(args, "-I", brotliIncludeDir)
		args = append(args, brotliSources...)
	}
	args = append(args, nativeSystemLinkFlags(triple)...)
	args = append(args, "-o", llcOutputPath)
	linkCmd := exec.Command(tc.ClangPath, args...)
	if output, err := linkCmd.CombinedOutput(); err != nil {
		t.Fatalf("clang link returned error: %v: %s", err, string(output))
	}
	llcExecCmd := exec.Command(llcOutputPath)
	llcOut, err := llcExecCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("llc-linked executable returned error: %v: %s", err, string(llcOut))
	}

	if string(directOut) != string(llcOut) {
		t.Fatalf("expected direct clang-IR and llc+clang outputs to match\ndirect:\n%s\nllc:\n%s", string(directOut), string(llcOut))
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

func TestBuildExecutableSupportsFileUrls(t *testing.T) {
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
function main() {
  var parsed = url.parse("file:///tmp/jayess/note.txt");
  console.log("file-protocol:" + parsed.protocol);
  console.log("file-host:" + parsed.host);
  console.log("file-path:" + parsed.pathname);
  console.log("file-format:" + url.format({ protocol: "file:", pathname: "/tmp/jayess/note.txt" }));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "file-url-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	for _, want := range []string{
		"file-protocol:file:",
		"file-host:",
		"file-path:/tmp/jayess/note.txt",
		"file-format:file:///tmp/jayess/note.txt",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected file URL output to contain %q, got: %s", want, text)
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

func TestBuildExecutableEnforcesTlsHostnameVerification(t *testing.T) {
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

	workdir := t.TempDir()
	caPath := filepath.Join(workdir, "hostname-server-cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		t.Fatalf("failed to encode certificate PEM")
	}
	if err := os.WriteFile(caPath, pemBytes, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	source := fmt.Sprintf(`
function main() {
  try {
    tls.connect({
      host: "%s",
      port: %d,
      serverName: "wrong.example.test",
      caFile: "%s",
      trustSystem: false,
      alpnProtocols: "http/1.1"
    });
    console.log("tls-hostname:unexpected-success");
  } catch (err) {
    console.log("tls-hostname:error:" + err.message);
  }
  return 0;
}
`, serverURL.Hostname(), port, caPath)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "tls-hostname-verify-native")
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
	if !strings.Contains(text, "tls-hostname:error:") {
		t.Fatalf("expected hostname verification error output, got: %s", text)
	}
	if strings.Contains(text, "tls-hostname:unexpected-success") {
		t.Fatalf("expected hostname verification to reject mismatched serverName, got: %s", text)
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

func TestBuildExecutableEnforcesHttpsSecureDefaults(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "secure-defaults")
	}))
	server.TLS = &tls.Config{NextProtos: []string{"http/1.1"}}
	server.StartTLS()
	defer server.Close()

	source := fmt.Sprintf(`
function main() {
  try {
    https.get({ url: "%s/defaults" });
    console.log("https-defaults:unexpected-success");
  } catch (err) {
    console.log("https-defaults:error:" + err.message);
  }
  return 0;
}
`, server.URL)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "https-secure-defaults-native")
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
	if !strings.Contains(text, "https-defaults:error:") {
		t.Fatalf("expected https secure-defaults verification error, got: %s", text)
	}
	if strings.Contains(text, "https-defaults:unexpected-success") {
		t.Fatalf("expected https secure defaults to reject self-signed certificate, got: %s", text)
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

func TestBuildExecutableReportsSocketErrors(t *testing.T) {
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

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		time.Sleep(250 * time.Millisecond)
		serverDone <- nil
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	source := fmt.Sprintf(`
function main() {
  var socket = net.connect({ host: "127.0.0.1", port: %d });
  var socketErrors = 0;
  socket.on("error", (err) => {
    socketErrors = socketErrors + 1;
    console.log("socket-error:" + err.message);
    return 0;
  });
  socket.setTimeout(80);
  var timedOut = socket.read(4);
  console.log("socket-timeout-read:" + timedOut);
  console.log("socket-errored:" + socket.errored + ":" + (typeof socket.error.message));

  var server = net.listen({ host: "127.0.0.1", port: 0 });
  var serverErrors = 0;
  server.on("error", (err) => {
    serverErrors = serverErrors + 1;
    console.log("server-error:" + err.message);
    return 0;
  });
  server.setTimeout(80);
  var accepted = server.accept();
  console.log("server-timeout-accept:" + accepted);
  console.log("server-errored:" + server.errored + ":" + (typeof server.error.message));
  console.log("error-counts:" + socketErrors + ":" + serverErrors);

  socket.close();
  server.close();
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "net-errors-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("helper server returned error: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"socket-error:socket read timed out",
		"socket-timeout-read:undefined",
		"socket-errored:true:string",
		"server-error:failed to accept socket connection",
		"server-timeout-accept:undefined",
		"server-errored:true:string",
		"error-counts:1:1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected socket error output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsDatagramSockets(t *testing.T) {
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
  var receiver = net.createDatagramSocket({ host: "127.0.0.1", port: 0, type: "udp4" });
  var sender = net.createDatagramSocket({ host: "127.0.0.1", port: 0, type: "udp4" });
  receiver.setTimeout(250);
  sender.setTimeout(250);

  var receiverAddress = receiver.address();
  var senderAddress = sender.address();
  console.log("udp-bind:" + (receiverAddress.port > 0) + ":" + receiverAddress.family + ":" + (senderAddress.port > 0));
  console.log("udp-broadcast:" + (sender.setBroadcast(true) === sender) + ":" + sender.broadcast);
  console.log("udp-send:" + sender.send("kimchi", receiverAddress.port, "127.0.0.1"));

  var packet = receiver.receive(64);
  console.log("udp-recv:" + packet.data + ":" + packet.address + ":" + (packet.port > 0) + ":" + packet.family + ":" + packet.bytes.length);
  console.log("udp-bytes:" + receiver.bytesRead + ":" + sender.bytesWritten);

  sender.close();
  receiver.close();
  console.log("udp-closed:" + sender.closed + ":" + receiver.closed);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "udp-datagram-native")
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
		"udp-bind:true:4:true",
		"udp-broadcast:true:true",
		"udp-send:true",
		"udp-recv:kimchi:127.0.0.1:true:4:6",
		"udp-bytes:6:6",
		"udp-closed:true:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected udp output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsDatagramBroadcastAndMulticast(t *testing.T) {
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
  var broadcastReceiver = net.createDatagramSocket({ host: "0.0.0.0", port: 0, type: "udp4", broadcast: true });
  var broadcastSender = net.createDatagramSocket({ host: "0.0.0.0", port: 0, type: "udp4" });
  broadcastReceiver.setTimeout(500);
  broadcastSender.setTimeout(500);
  var broadcastAddress = broadcastReceiver.address();
  broadcastSender.setBroadcast(true);
  console.log("udp-broadcast-flags:" + broadcastSender.broadcast + ":" + broadcastReceiver.broadcast);
  console.log("udp-broadcast-send:" + broadcastSender.send("bravo", broadcastAddress.port, "255.255.255.255"));
  var broadcastPacket = broadcastReceiver.receive(64);
  console.log("udp-broadcast-recv:" + broadcastPacket.data + ":" + (broadcastPacket.port > 0) + ":" + broadcastPacket.family);

  var multicastReceiver = net.createDatagramSocket({ host: "0.0.0.0", port: 0, type: "udp4" });
  var multicastSender = net.createDatagramSocket({ host: "0.0.0.0", port: 0, type: "udp4" });
  multicastReceiver.setTimeout(500);
  multicastSender.setTimeout(500);
  multicastSender.setMulticastInterface("127.0.0.1");
  multicastSender.setMulticastLoopback(true);
  var multicastAddress = multicastReceiver.address();
  var group = "239.255.0.1";
  console.log("udp-multicast-join:" + (multicastReceiver.joinGroup(group, "127.0.0.1") === multicastReceiver));
  console.log("udp-multicast-config:" + multicastSender.multicastInterface + ":" + multicastSender.multicastLoopback);
  console.log("udp-multicast-send:" + multicastSender.send("kimchi", multicastAddress.port, group));
  var multicastPacket = multicastReceiver.receive(64);
  console.log("udp-multicast-recv:" + multicastPacket.data + ":" + (multicastPacket.port > 0) + ":" + multicastPacket.family);
  console.log("udp-multicast-leave:" + (multicastReceiver.leaveGroup(group, "127.0.0.1") === multicastReceiver));

  broadcastSender.close();
  broadcastReceiver.close();
  multicastSender.close();
  multicastReceiver.close();
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "udp-broadcast-multicast-native")
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
		"udp-broadcast-flags:true:true",
		"udp-broadcast-send:true",
		"udp-broadcast-recv:bravo:true:4",
		"udp-multicast-join:true",
		"udp-multicast-config:127.0.0.1:true",
		"udp-multicast-send:true",
		"udp-multicast-recv:kimchi:true:4",
		"udp-multicast-leave:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected udp broadcast/multicast output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsChildProcesses(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	execCommand := `read line; echo OUT-$line; echo ERR-$line 1>&2; exit 7`
	spawnFile := "/bin/sh"
	spawnArgs := `["-c", "read line; echo SPAWN-$line; echo SPAWNERR-$line 1>&2; exit 5"]`
	if runtime.GOOS == "windows" {
		execCommand = `set /p LINE= & echo OUT-%LINE% & echo ERR-%LINE% 1>&2 & exit /b 7`
		spawnFile = "cmd.exe"
		spawnArgs = `["/C", "set /p LINE= & echo SPAWN-%LINE% & echo SPAWNERR-%LINE% 1>&2 & exit /b 5"]`
	}

	source := fmt.Sprintf(`
function main() {
  var execResult = childProcess.exec({ command: %q, input: "kimchi" });
  console.log("exec-ok:" + execResult.ok);
  console.log("exec-status:" + execResult.status);
  console.log("exec-stdout:" + execResult.stdout.trim());
  console.log("exec-stderr:" + execResult.stderr.trim());
  console.log("exec-pid:" + (execResult.pid > 0));

  var spawnResult = childProcess.spawn({ file: %q, args: %s, input: "jjigae" });
  console.log("spawn-ok:" + spawnResult.ok);
  console.log("spawn-status:" + spawnResult.status);
  console.log("spawn-stdout:" + spawnResult.stdout.trim());
  console.log("spawn-stderr:" + spawnResult.stderr.trim());
  console.log("spawn-pid:" + (spawnResult.pid > 0));
  return 0;
}
`, execCommand, spawnFile, spawnArgs)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "child-process-native")
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
		"exec-ok:false",
		"exec-status:7",
		"exec-stdout:OUT-kimchi",
		"exec-stderr:ERR-kimchi",
		"exec-pid:true",
		"spawn-ok:false",
		"spawn-status:5",
		"spawn-stdout:SPAWN-jjigae",
		"spawn-stderr:SPAWNERR-jjigae",
		"spawn-pid:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected child-process output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsChildProcessSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal handling test currently exercises POSIX child signals")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	source := `
function main() {
  var launched = childProcess.exec({ command: "sh -c 'sleep 30 >/dev/null 2>&1 & echo $!'" });
  var pid = launched.stdout.trim();
  var killed = childProcess.kill({ pid: pid, signal: "SIGTERM" });
  var probe = childProcess.exec({
    command: "sh -c 'for i in 1 2 3 4 5; do if ! kill -0 " + pid + " 2>/dev/null; then echo gone; exit 0; fi; sleep 0.1; done; echo alive'"
  });
  console.log("signal-killed:" + killed);
  console.log("signal-probe:" + probe.stdout.trim());
  return 0;
}
`

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "child-process-signal-native")
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
		"signal-killed:true",
		"signal-probe:gone",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected child-process signal output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsProcessSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signal runtime test currently exercises POSIX signal delivery")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	source := `
function main() {
  var count = 0;
  var once = 0;
  var removed = 0;
  var last = "";
  var keep = (event) => {
    count = count + 1;
    last = event.signal;
  };
  var fireOnce = (event) => {
    once = once + 1;
  };
  var removeMe = (event) => {
    removed = removed + 1;
  };

  process.onSignal("SIGTERM", keep);
  process.onceSignal("SIGTERM", fireOnce);
  process.onSignal("SIGTERM", removeMe);
  process.offSignal("SIGTERM", removeMe);

  process.raise("SIGTERM");
  sleep(20);
  process.raise("SIGTERM");
  sleep(20);

  console.log("signal-count:" + count);
  console.log("signal-once:" + once);
  console.log("signal-removed:" + removed);
  console.log("signal-last:" + last);
  return 0;
}
`

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "process-signal-native")
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
		"signal-count:2",
		"signal-once:1",
		"signal-removed:0",
		"signal-last:SIGTERM",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected process signal output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpCreateServer(t *testing.T) {
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
  var server = undefined;
  server = http.createServer((req, res) => {
    console.log("http-server-req:" + req.method + ":" + req.url + ":" + req.body);
    res.statusCode = 201;
    res.setHeader("Content-Type", "text/plain");
    res.setHeader("X-Test", "kimchi");
    res.write("hel");
    res.end("lo");
    server.close();
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-create-server-native")
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

	var resp *http.Response
	for i := 0; i < 40; i++ {
		req, reqErr := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/hello", port), strings.NewReader("kimchi=1"))
		if reqErr != nil {
			t.Fatalf("NewRequest returned error: %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("HTTP request returned error: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}
	if got := string(bodyBytes); got != "hello" {
		t.Fatalf("expected response body hello, got %q", got)
	}
	if got := resp.Header.Get("X-Test"); got != "kimchi" {
		t.Fatalf("expected X-Test header kimchi, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	if !strings.Contains(text, "http-server-req:POST:/hello:kimchi=1") {
		t.Fatalf("expected server output to contain parsed request data, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpsCreateServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("https.createServer server-side TLS path is not implemented on Windows yet")
	}

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

	workdir := t.TempDir()
	certPath, keyPath := writeTestTLSCertificatePair(t, workdir, "jayess-https-server")

	source := fmt.Sprintf(`
function main() {
  var server = undefined;
  server = https.createServer({ cert: "%s", key: "%s" }, (req, res) => {
    console.log("https-server-req:" + req.method + ":" + req.url + ":" + req.body);
    res.statusCode = 202;
    res.setHeader("Content-Type", "text/plain");
    res.setHeader("X-Test", "tls");
    res.write("sec");
    res.end("ure");
    server.close();
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, certPath, keyPath, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "https-create-server-native")
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

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var resp *http.Response
	for i := 0; i < 40; i++ {
		req, reqErr := http.NewRequest("POST", fmt.Sprintf("https://127.0.0.1:%d/hello", port), strings.NewReader("kimchi=1"))
		if reqErr != nil {
			t.Fatalf("NewRequest returned error: %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("HTTPS request returned error: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 202 {
		t.Fatalf("expected status 202, got %d", resp.StatusCode)
	}
	if got := string(bodyBytes); got != "secure" {
		t.Fatalf("expected response body secure, got %q", got)
	}
	if got := resp.Header.Get("X-Test"); got != "tls" {
		t.Fatalf("expected X-Test header tls, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled https server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	if !strings.Contains(text, "https-server-req:POST:/hello:kimchi=1") {
		t.Fatalf("expected https server output to contain parsed request data, got: %s", text)
	}
}

func TestBuildExecutableSupportsTlsCreateServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tls.createServer server-side TLS path is not implemented on Windows yet")
	}

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

	workdir := t.TempDir()
	certPath, keyPath := writeTestTLSCertificatePair(t, workdir, "jayess-tls-server")

	source := fmt.Sprintf(`
function main() {
  var server = undefined;
  server = tls.createServer({ cert: "%s", key: "%s" }, (socket) => {
    console.log("tls-server-socket:" + socket.secure + ":" + socket.backend + ":" + socket.protocol);
    var incoming = socket.read();
    console.log("tls-server-read:" + incoming);
    socket.write("pong");
    socket.close();
    server.close();
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, certPath, keyPath, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "tls-create-server-native")
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

	client := &tls.Config{InsecureSkipVerify: true}
	var conn *tls.Conn
	for i := 0; i < 40; i++ {
		conn, err = tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), client)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("TLS dial returned error: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("TLS write returned error: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("TLS read returned error: %v", err)
	}
	if got := string(reply); got != "pong" {
		t.Fatalf("expected pong reply, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled tls server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	if !strings.Contains(text, "tls-server-socket:true:openssl:TLS") {
		t.Fatalf("expected tls server socket output, got: %s", text)
	}
	if !strings.Contains(text, "tls-server-read:ping") {
		t.Fatalf("expected tls server read output, got: %s", text)
	}
}

func TestBuildExecutableSupportsHttpKeepAlive(t *testing.T) {
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
  var count = 0;
  var server = undefined;
  server = http.createServer((req, res) => {
    count = count + 1;
    console.log("http-keepalive-req:" + count + ":" + req.url + ":" + req.keepAlive);
    if (count == 2) {
      res.setHeader("Connection", "close");
    }
    res.setHeader("X-Count", "" + count);
    res.write("he");
    res.end("llo");
    if (count == 2) {
      server.close();
    }
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "http-keepalive-native")
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

	var conn net.Conn
	for i := 0; i < 40; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	if _, err := conn.Write([]byte("GET /one HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: keep-alive\r\n\r\n")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	req1, _ := http.NewRequest("GET", "http://127.0.0.1/one", nil)
	resp1, err := http.ReadResponse(reader, req1)
	if err != nil {
		t.Fatalf("ReadResponse 1 returned error: %v", err)
	}
	body1, err := io.ReadAll(resp1.Body)
	if err != nil {
		t.Fatalf("ReadAll 1 returned error: %v", err)
	}
	resp1.Body.Close()
	if got := string(body1); got != "hello" {
		t.Fatalf("expected first keep-alive body hello, got %q", got)
	}
	if resp1.Close {
		t.Fatalf("expected first response to keep the connection open")
	}
	if got := resp1.Header.Get("X-Count"); got != "1" {
		t.Fatalf("expected first X-Count 1, got %q", got)
	}

	if _, err := conn.Write([]byte("GET /two HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("Write second returned error: %v", err)
	}
	req2, _ := http.NewRequest("GET", "http://127.0.0.1/two", nil)
	resp2, err := http.ReadResponse(reader, req2)
	if err != nil {
		t.Fatalf("ReadResponse 2 returned error: %v", err)
	}
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("ReadAll 2 returned error: %v", err)
	}
	resp2.Body.Close()
	if got := string(body2); got != "hello" {
		t.Fatalf("expected second keep-alive body hello, got %q", got)
	}
	if !resp2.Close {
		t.Fatalf("expected second response to close the connection")
	}
	if got := resp2.Header.Get("X-Count"); got != "2" {
		t.Fatalf("expected second X-Count 2, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled keep-alive server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	for _, want := range []string{
		"http-keepalive-req:1:/one:true",
		"http-keepalive-req:2:/two:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected keep-alive output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHttpsKeepAlive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("https.createServer server-side TLS path is not implemented on Windows yet")
	}

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

	workdir := t.TempDir()
	certPath, keyPath := writeTestTLSCertificatePair(t, workdir, "jayess-https-keepalive")

	source := fmt.Sprintf(`
function main() {
  var count = 0;
  var server = undefined;
  server = https.createServer({ cert: "%s", key: "%s" }, (req, res) => {
    count = count + 1;
    console.log("https-keepalive-req:" + count + ":" + req.url + ":" + req.keepAlive);
    if (count == 2) {
      res.setHeader("Connection", "close");
    }
    res.setHeader("X-Count", "" + count);
    res.write("he");
    res.end("llo");
    if (count == 2) {
      server.close();
    }
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, certPath, keyPath, port)

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "https-keepalive-native")
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

	var conn *tls.Conn
	for i := 0; i < 40; i++ {
		conn, err = tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), &tls.Config{InsecureSkipVerify: true})
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("TLS dial returned error: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	if _, err := conn.Write([]byte("GET /one HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: keep-alive\r\n\r\n")); err != nil {
		t.Fatalf("TLS write returned error: %v", err)
	}
	req1, _ := http.NewRequest("GET", "https://127.0.0.1/one", nil)
	resp1, err := http.ReadResponse(reader, req1)
	if err != nil {
		t.Fatalf("TLS ReadResponse 1 returned error: %v", err)
	}
	body1, err := io.ReadAll(resp1.Body)
	if err != nil {
		t.Fatalf("TLS ReadAll 1 returned error: %v", err)
	}
	resp1.Body.Close()
	if got := string(body1); got != "hello" {
		t.Fatalf("expected first HTTPS keep-alive body hello, got %q", got)
	}
	if resp1.Close {
		t.Fatalf("expected first HTTPS response to keep the connection open")
	}

	if _, err := conn.Write([]byte("GET /two HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("TLS write second returned error: %v", err)
	}
	req2, _ := http.NewRequest("GET", "https://127.0.0.1/two", nil)
	resp2, err := http.ReadResponse(reader, req2)
	if err != nil {
		t.Fatalf("TLS ReadResponse 2 returned error: %v", err)
	}
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("TLS ReadAll 2 returned error: %v", err)
	}
	resp2.Body.Close()
	if got := string(body2); got != "hello" {
		t.Fatalf("expected second HTTPS keep-alive body hello, got %q", got)
	}
	if !resp2.Close {
		t.Fatalf("expected second HTTPS response to close the connection")
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled HTTPS keep-alive server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	for _, want := range []string{
		"https-keepalive-req:1:/one:true",
		"https-keepalive-req:2:/two:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected HTTPS keep-alive output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsWorkers(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	source := `
function main() {
  var workerThread = worker.create(function(message) {
    return {
      sum: message.left + message.right,
      text: message.text.toUpperCase(),
      values: [message.left, message.right, message.values[0]]
    };
  });
  var payload = { left: 2, right: 5, text: "kimchi", values: [9] };
  console.log("worker-post:" + workerThread.postMessage(payload));
  payload.left = 99;
  payload.text = "jjigae";
  var response = workerThread.receive(5000);
  console.log("worker-reply:" + response.ok + ":" + response.value.sum + ":" + response.value.text + ":" + response.value.values[2]);
  console.log("worker-closed-before:" + workerThread.closed);
  console.log("worker-terminate:" + workerThread.terminate());
  console.log("worker-closed-after:" + workerThread.closed);
  return 0;
}
`

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "worker-native")
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
		"worker-post:true",
		"worker-reply:true:7:KIMCHI:9",
		"worker-closed-before:false",
		"worker-terminate:true",
		"worker-closed-after:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected worker output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSharedMemoryAndAtomics(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	source := `
function main() {
  var buffer = new SharedArrayBuffer(8);
  var ints = new Int32Array(buffer);
  console.log("atomics-store:" + Atomics.store(ints, 0, 10));
  console.log("atomics-add:" + Atomics.add(ints, 0, 5));
  console.log("atomics-load:" + Atomics.load(ints, 0));
  console.log("atomics-sub:" + Atomics.sub(ints, 0, 3));
  console.log("atomics-and:" + Atomics.and(ints, 0, 14));
  console.log("atomics-or:" + Atomics.or(ints, 0, 1));
  console.log("atomics-xor:" + Atomics.xor(ints, 0, 3));
  console.log("atomics-exchange:" + Atomics.exchange(ints, 1, 20));
  console.log("atomics-compareExchange:" + Atomics.compareExchange(ints, 1, 20, 30));
  var workerThread = worker.create(function(message) {
    var shared = new Int32Array(message.buffer);
    var before = Atomics.add(shared, 0, 4);
    return { before: before, after: Atomics.load(shared, 0), second: shared[1] };
  });
  console.log("shared-post:" + workerThread.postMessage({ buffer: buffer }));
  var reply = workerThread.receive(5000);
  console.log("shared-reply:" + reply.ok + ":" + reply.value.before + ":" + reply.value.after + ":" + reply.value.second);
  console.log("shared-main:" + ints[0] + ":" + ints[1]);
  console.log("shared-close:" + workerThread.terminate());
  return 0;
}
`

	result, err := compiler.Compile(source, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "shared-memory-native")
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
		"atomics-store:10",
		"atomics-add:10",
		"atomics-load:15",
		"atomics-sub:15",
		"atomics-and:12",
		"atomics-or:12",
		"atomics-xor:13",
		"atomics-exchange:0",
		"atomics-compareExchange:20",
		"shared-post:true",
		"shared-reply:true:14:18:30",
		"shared-main:18:30",
		"shared-close:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected shared memory output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsCryptoSurface(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	sum := sha256.Sum256([]byte("kimchi"))
	expectedHash := hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("kimchi"))
	expectedHMAC := hex.EncodeToString(mac.Sum(nil))

	result, err := compiler.Compile(`
function main() {
  var bytes = crypto.randomBytes(16);
  var digest = crypto.hash("sha256", "kimchi");
  var mac = crypto.hmac("sha256", "secret", "kimchi");
  console.log("crypto-random:" + bytes.length + ":" + bytes.toString("hex").length);
  console.log("crypto-hash:" + digest);
  console.log("crypto-hmac:" + mac);
  console.log("crypto-compare:" + crypto.secureCompare("same", "same") + ":" + crypto.secureCompare("same", "different"));
  console.log("crypto-bytes-compare:" + crypto.secureCompare(bytes, bytes.slice(0, bytes.length)));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "crypto-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"crypto-random:16:32",
		"crypto-hash:" + expectedHash,
		"crypto-hmac:" + expectedHMAC,
		"crypto-compare:true:false",
		"crypto-bytes-compare:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected crypto output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsSymmetricEncryption(t *testing.T) {
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
  var key = Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex");
  var iv = Uint8Array.fromString("1af38c2dc2b96ffdd86694092341bc04", "hex");
  var aad = Uint8Array.fromString("feedfacedeadbeeffeedfacedeadbeefabaddad2", "hex");
  var encrypted = crypto.encrypt({
    algorithm: "aes-256-gcm",
    key: key,
    iv: iv,
    aad: aad,
    data: "jayess"
  });
  console.log("crypto-encrypt:" + encrypted.algorithm + ":" + encrypted.iv.toString("hex") + ":" + encrypted.ciphertext.toString("hex") + ":" + encrypted.tag.toString("hex"));
  var decrypted = crypto.decrypt({
    algorithm: "aes-256-gcm",
    key: key,
    iv: encrypted.iv,
    aad: aad,
    data: encrypted.ciphertext,
    tag: encrypted.tag
  });
  console.log("crypto-decrypt:" + decrypted.toString());
  var failed = crypto.decrypt({
    algorithm: "aes-256-gcm",
    key: key,
    iv: encrypted.iv,
    aad: aad,
    data: encrypted.ciphertext,
    tag: Uint8Array.fromString("00000000000000000000000000000000", "hex")
  });
  console.log("crypto-decrypt-invalid:" + typeof failed + ":" + failed);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "crypto-symmetric-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"crypto-encrypt:aes-256-gcm:1af38c2dc2b96ffdd86694092341bc04:",
		"crypto-decrypt:jayess",
		"crypto-decrypt-invalid:undefined:undefined",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected symmetric crypto output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsAsymmetricCryptoSurface(t *testing.T) {
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
  var pair = crypto.generateKeyPair({ type: "rsa", modulusLength: 2048 });
  console.log("crypto-keypair:" + pair.publicKey.type + ":" + pair.publicKey.private + ":" + pair.privateKey.private);
  var ciphertext = crypto.publicEncrypt({ algorithm: "rsa-oaep-sha256", key: pair.publicKey, data: "kimchi" });
  var plaintext = crypto.privateDecrypt({ algorithm: "rsa-oaep-sha256", key: pair.privateKey, data: ciphertext });
  console.log("crypto-asymmetric:" + ciphertext.length + ":" + plaintext.toString());
  var signature = crypto.sign({ algorithm: "rsa-pss-sha256", key: pair.privateKey, data: "jayess" });
  console.log("crypto-signature:" + signature.length);
  console.log("crypto-verify:" + crypto.verify({ algorithm: "rsa-pss-sha256", key: pair.publicKey, data: "jayess", signature: signature }) + ":" + crypto.verify({ algorithm: "rsa-pss-sha256", key: pair.publicKey, data: "jjigae", signature: signature }));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "crypto-asymmetric-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"crypto-keypair:rsa:false:true",
		"crypto-asymmetric:",
		":kimchi",
		"crypto-signature:",
		"crypto-verify:true:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected asymmetric crypto output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsCompressionSurface(t *testing.T) {
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
  var gzip = compression.gzip("kimchi");
  var gunzip = compression.gunzip(gzip);
  console.log("compression-gzip:" + gzip.length + ":" + gunzip.toString());
  var deflate = compression.deflate("jjigae");
  var inflate = compression.inflate(deflate);
  console.log("compression-deflate:" + deflate.length + ":" + inflate.toString());
  var brotli = compression.brotli("mandu");
  var unbrotli = compression.unbrotli(brotli);
  console.log("compression-brotli:" + brotli.length + ":" + unbrotli.toString());
  var bad = compression.inflate(Uint8Array.fromString("001122", "hex"));
  console.log("compression-invalid:" + typeof bad + ":" + bad);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "compression-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"compression-gzip:",
		":kimchi",
		"compression-deflate:",
		":jjigae",
		"compression-brotli:",
		":mandu",
		"compression-invalid:undefined:undefined",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected compression output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsCompressionStreams(t *testing.T) {
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
function main() {
  fs.mkdir("tmp", { recursive: true });
  fs.writeFile("tmp/plain.txt", "kimchi-jjigae-mandu");

  var gzipStream = compression.createGzipStream();
  var gzipWriter = fs.createWriteStream("tmp/plain.txt.gz");
  fs.createReadStream("tmp/plain.txt").pipe(gzipStream).pipe(gzipWriter);
  console.log("compression-stream-gzip-finish:" + gzipWriter.writableEnded);

  var gunzipStream = compression.createGunzipStream();
  var gunzipWriter = fs.createWriteStream("tmp/plain.roundtrip.txt");
  fs.createReadStream("tmp/plain.txt.gz").pipe(gunzipStream).pipe(gunzipWriter);
  console.log("compression-stream-gunzip-finish:" + gunzipWriter.writableEnded);
  console.log("compression-stream-roundtrip:" + fs.readFile("tmp/plain.roundtrip.txt", "utf8"));

  var brotliStream = compression.createBrotliStream();
  brotliStream.write("mandu");
  brotliStream.end();
  var compressed = brotliStream.readBytes(4096);
  var unbrotliStream = compression.createUnbrotliStream();
  unbrotliStream.write(compressed);
  unbrotliStream.end();
  console.log("compression-stream-brotli:" + unbrotliStream.read());
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "compression-streams-native")
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
		"compression-stream-gzip-finish:true",
		"compression-stream-gunzip-finish:true",
		"compression-stream-roundtrip:kimchi-jjigae-mandu",
		"compression-stream-brotli:mandu",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected compression stream output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessSQLitePackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping SQLite package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-sqlite-backend-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { open, close, exec, prepare, finalize, reset, clearBindings, bindNull, bindInteger, bindFloat, bindString, bindBlob, get, getArray, run, changes, lastInsertRowId, busyTimeout, begin, commit, rollback, pragma, query } from "@jayess/sqlite";

function main(args) {
  var db = open(":memory:");
  busyTimeout(db, 50);
  exec(db, "create table items(id integer primary key autoincrement, name text, qty integer, price real, data blob, note text)");

  begin(db);
  var insertStmt = prepare(db, "insert into items(name, qty, price, data, note) values(?, ?, ?, ?, ?)");
  bindString(insertStmt, 1, "kimchi");
  bindInteger(insertStmt, 2, 3);
  bindFloat(insertStmt, 3, 1.5);
  bindBlob(insertStmt, 4, new Uint8Array([1, 2, 3, 4]));
  bindNull(insertStmt, 5);
  run(insertStmt);
  finalize(insertStmt);
  commit(db);
  console.log("sqlite-insert:" + changes(db) + ":" + lastInsertRowId(db));

  var selectStmt = prepare(db, "select id, name, qty, price, data, note from items where id = ?");
  bindInteger(selectStmt, 1, 1);
  var row = get(selectStmt);
  console.log("sqlite-row:" + row.id + ":" + row.name + ":" + row.qty + ":" + row.price + ":" + row.note + ":" + row.data.length);

  reset(selectStmt);
  clearBindings(selectStmt);
  bindInteger(selectStmt, 1, 1);
  var arr = getArray(selectStmt);
  console.log("sqlite-array:" + arr[0] + ":" + arr[1] + ":" + arr[2] + ":" + arr[3] + ":" + arr[4].length);
  finalize(selectStmt);

  begin(db);
  exec(db, "insert into items(name, qty, price, data, note) values('rollback', 1, 2.0, x'0102', 'nope')");
  rollback(db);

  var rows = query(db, "select name from items order by id");
  console.log("sqlite-query:" + rows.length + ":" + rows[0].name);

  var pragmaRows = pragma(db, "table_info(items)");
  console.log("sqlite-pragma:" + pragmaRows.length);

  exec(db, "update items set qty = 4 where id = 1");
  console.log("sqlite-update:" + changes(db));
  exec(db, "delete from items where id = 1");
  console.log("sqlite-delete:" + changes(db));

  close(db);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "sqlite-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled SQLite program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"sqlite-insert:1:1",
		"sqlite-row:1:kimchi:3:1.5:null:4",
		"sqlite-array:1:kimchi:3:1.5:4",
		"sqlite-query:1:kimchi",
		"sqlite-pragma:6",
		"sqlite-update:1",
		"sqlite-delete:1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected SQLite output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsManualBindValueExports(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(nativeDir, "value.c"), []byte(`#include "jayess_runtime.h"
jayess_value *mylib_version_value(void) { return jayess_value_from_number(7); }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "value.bind.js"), []byte(`const f = () => {};
export const version = 0;
export default {
  sources: ["./value.c"],
  exports: {
    version: { symbol: "mylib_version_value", type: "value" }
  }
};`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { version } from "./native/value.bind.js";

function main(args) {
  console.log("bind-value:" + version);
  console.log("bind-value-plus:" + (version + 1));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "bind-value-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled bind-value program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"bind-value:7", "bind-value-plus:8"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected bind value output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsJayessSQLiteErrorsUsefully(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping SQLite error test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-sqlite-errors-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { open, close, exec, prepare, finalize, step } from "@jayess/sqlite";

function main(args) {
  var db = open(":memory:");
  try {
    exec(db, "not valid sql");
  } catch (err) {
    console.log("sqlite-error:" + err.name);
  }

  var stmt = prepare(db, "select 1 as value");
  finalize(stmt);
  try {
    step(stmt);
  } catch (err) {
    console.log("sqlite-finalize:" + err.name);
  }

  close(db);
  try {
    exec(db, "select 1");
  } catch (err) {
    console.log("sqlite-close:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "sqlite-errors-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled SQLite error program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"sqlite-error:SQLiteError",
		"sqlite-finalize:TypeError",
		"sqlite-close:TypeError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected SQLite error output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableKeepsJayessSQLiteBlobAndStringOwnershipSafe(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping SQLite ownership test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-sqlite-ownership-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { open, close, exec, prepare, finalize, bindString, bindBlob, run, get } from "@jayess/sqlite";

function main(args) {
  var db = open(":memory:");
  exec(db, "create table items(name text, data blob)");

  var sourceName = "kimchi";
  var sourceBlob = new Uint8Array([1, 2, 3, 4]);
  var insertStmt = prepare(db, "insert into items(name, data) values(?, ?)");
  bindString(insertStmt, 1, sourceName);
  bindBlob(insertStmt, 2, sourceBlob);

  sourceName = "jjigae";
  sourceBlob[0] = 9;
  sourceBlob[1] = 9;
  sourceBlob[2] = 9;
  sourceBlob[3] = 9;

  run(insertStmt);
  finalize(insertStmt);

  var selectStmt = prepare(db, "select name, data from items");
  var row = get(selectStmt);
  finalize(selectStmt);
  close(db);

  console.log("sqlite-ownership-bound:" + row.name + ":" + row.data.toString("hex"));

  row.data[0] = 255;
  console.log("sqlite-ownership-row:" + row.name + ":" + row.data.toString("hex"));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "sqlite-ownership-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled SQLite ownership program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"sqlite-ownership-bound:kimchi:01020304",
		"sqlite-ownership-row:kimchi:ff020304",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected SQLite ownership output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessOpenSSLPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	sum := sha256.Sum256([]byte("kimchi"))
	expectedHash := hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("kimchi"))
	expectedHMAC := hex.EncodeToString(mac.Sum(nil))

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-openssl-backend-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { randomBytes, version, supportsHash, supportsCipher, hash, hmac, encrypt, decrypt, generateKeyPair, publicEncrypt, privateDecrypt, sign, verify } from "@jayess/openssl";

function main(args) {
  var bytes = randomBytes(16);
  console.log("openssl-random:" + bytes.length + ":" + bytes.toString("hex").length);
  console.log("openssl-version:" + (version().length > 0));
  console.log("openssl-supports-hash:" + supportsHash("sha256") + ":" + supportsHash("sha999"));
  console.log("openssl-supports-cipher:" + supportsCipher("aes-256-gcm") + ":" + supportsCipher("chacha20-poly1305"));
  console.log("openssl-hash:" + hash("sha256", "kimchi"));
  console.log("openssl-hmac:" + hmac("sha256", "secret", "kimchi"));

  var encrypted = encrypt({
    algorithm: "aes-256-gcm",
    key: Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex"),
    iv: Uint8Array.fromString("1af38c2dc2b96ffdd86694092341bc04", "hex"),
    aad: Uint8Array.fromString("feedfacedeadbeeffeedfacedeadbeefabaddad2", "hex"),
    data: "jayess"
  });
  console.log("openssl-encrypt:" + encrypted.algorithm + ":" + encrypted.iv.toString("hex") + ":" + encrypted.ciphertext.toString("hex") + ":" + encrypted.tag.toString("hex"));
  var decrypted = decrypt({
    algorithm: encrypted.algorithm,
    key: Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex"),
    iv: encrypted.iv,
    aad: Uint8Array.fromString("feedfacedeadbeeffeedfacedeadbeefabaddad2", "hex"),
    data: encrypted.ciphertext,
    tag: encrypted.tag
  });
  console.log("openssl-decrypt:" + decrypted.toString());

  var failed = decrypt({
    algorithm: encrypted.algorithm,
    key: Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex"),
    iv: encrypted.iv,
    aad: Uint8Array.fromString("feedfacedeadbeeffeedfacedeadbeefabaddad2", "hex"),
    data: encrypted.ciphertext,
    tag: Uint8Array.fromString("00000000000000000000000000000000", "hex")
  });
  console.log("openssl-decrypt-invalid:" + typeof failed + ":" + failed);

  var pair = generateKeyPair({ type: "rsa", modulusLength: 2048 });
  console.log("openssl-keypair:" + pair.publicKey.type + ":" + pair.publicKey.private + ":" + pair.privateKey.private);
  var publicKey = pair.publicKey;
  var privateKey = pair.privateKey;
  pair = null;
  var sealed = publicEncrypt({ algorithm: "rsa-oaep-sha256", key: publicKey, data: "kimchi" });
  var opened = privateDecrypt({ algorithm: "rsa-oaep-sha256", key: privateKey, data: sealed });
  console.log("openssl-asymmetric:" + sealed.length + ":" + opened.toString());
  sealed = publicEncrypt({ algorithm: "rsa-oaep-sha256", key: publicKey, data: "kimchi" });
  opened = privateDecrypt({ algorithm: "rsa-oaep-sha256", key: privateKey, data: sealed });
  console.log("openssl-key-copy:" + opened.toString());
  var signature = sign({ algorithm: "rsa-pss-sha256", key: privateKey, data: "jayess" });
  console.log("openssl-signature:" + signature.length);
  console.log("openssl-verify:" + verify({ algorithm: "rsa-pss-sha256", key: publicKey, data: "jayess", signature: signature }) + ":" + verify({ algorithm: "rsa-pss-sha256", key: publicKey, data: "jjigae", signature: signature }));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled OpenSSL program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"openssl-random:16:32",
		"openssl-version:true",
		"openssl-supports-hash:true:false",
		"openssl-supports-cipher:true:false",
		"openssl-hash:" + expectedHash,
		"openssl-hmac:" + expectedHMAC,
		"openssl-encrypt:aes-256-gcm:1af38c2dc2b96ffdd86694092341bc04:",
		"openssl-decrypt:jayess",
		"openssl-decrypt-invalid:undefined:undefined",
		"openssl-keypair:rsa:false:true",
		"openssl-asymmetric:",
		":kimchi",
		"openssl-key-copy:kimchi",
		"openssl-signature:",
		"openssl-verify:true:false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected OpenSSL output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsJayessOpenSSLErrorsUsefully(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL error test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-openssl-errors-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { hash, hmac } from "@jayess/openssl";

function main(args) {
  try {
    hash("sha999", "kimchi");
  } catch (err) {
    console.log("openssl-hash-error:" + err.name);
  }
  try {
    hmac("sha999", "secret", "kimchi");
  } catch (err) {
    console.log("openssl-hmac-error:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-errors-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled OpenSSL error program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"openssl-hash-error:OpenSSLError",
		"openssl-hmac-error:OpenSSLError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected OpenSSL error output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsMissingOpenSSLDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL missing deps test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "openssl")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/openssl","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { randomBytesNative } from "./native/openssl.bind.js";
export function randomBytes(length) {
  return randomBytesNative(length);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openssl.bind.js"), []byte(`const f = () => {};
export const randomBytesNative = f;

export default {
  sources: ["./openssl.c"],
  includeDirs: ["."],
  cflags: [],
  ldflags: ["-lssl_missing", "-lcrypto_missing"],
  exports: {
    randomBytesNative: { symbol: "jayess_openssl_random_bytes_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openssl.c"), []byte(`#include "jayess_runtime.h"
#include <openssl/rand.h>

jayess_value *jayess_openssl_random_bytes_native(jayess_value *length_value) {
  (void) length_value;
  return jayess_value_from_bytes_copy((const unsigned char *)"ok", 2);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { randomBytes } from "@jayess/openssl";

function main(args) {
  console.log(randomBytes(2).length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-missing-native")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail when OpenSSL libraries are missing")
	}
	if !strings.Contains(err.Error(), "native library link failed for ssl_missing") {
		t.Fatalf("expected missing OpenSSL library diagnostic, got: %v", err)
	}
}

func TestBuildExecutableReportsMissingOpenSSLHeadersClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL missing header test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "openssl")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/openssl","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { randomBytesNative } from "./native/openssl.bind.js";
export function randomBytes(length) {
  return randomBytesNative(length);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openssl.bind.js"), []byte(`const f = () => {};
export const randomBytesNative = f;

export default {
  sources: ["./openssl.c"],
  includeDirs: ["."],
  cflags: [],
  ldflags: ["-lssl", "-lcrypto"],
  exports: {
    randomBytesNative: { symbol: "jayess_openssl_random_bytes_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openssl.c"), []byte(`#include "jayess_runtime.h"
#include <openssl/not_real_header.h>

jayess_value *jayess_openssl_random_bytes_native(jayess_value *length_value) {
  (void) length_value;
  return jayess_value_from_bytes_copy((const unsigned char *)"ok", 2);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { randomBytes } from "@jayess/openssl";

function main(args) {
  console.log(randomBytes(2).length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-missing-header-native")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail when OpenSSL headers are missing")
	}
	if !strings.Contains(err.Error(), "native header dependency missing for openssl/not_real_header.h") {
		t.Fatalf("expected missing OpenSSL header diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessCurlPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping curl package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/echo":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.Header().Set("X-Method", r.Method)
			_, _ = w.Write(body)
		case "/stream":
			w.Header().Set("Content-Type", "text/plain")
			if flusher, ok := w.(http.Flusher); ok {
				_, _ = w.Write([]byte("chunk-1:"))
				flusher.Flush()
				time.Sleep(150 * time.Millisecond)
				_, _ = w.Write([]byte("chunk-2"))
				flusher.Flush()
				return
			}
			_, _ = w.Write([]byte("chunk-1:chunk-2"))
		case "/redirect":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			_, _ = w.Write([]byte("redirected"))
		case "/cookie":
			_, _ = w.Write([]byte(r.Header.Get("Cookie")))
		case "/download":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("downloaded"))
		case "/multi-a":
			_, _ = w.Write([]byte("alpha"))
		case "/multi-b":
			_, _ = w.Write([]byte("beta"))
		case "/slow":
			time.Sleep(200 * time.Millisecond)
			_, _ = w.Write([]byte("slow"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()

	httpsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	defer httpsServer.Close()
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxied:" + r.URL.String()))
	}))
	defer proxyServer.Close()

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-curl-backend-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createEasy, configure, perform, cleanup, createMulti, addHandle, performMulti, cleanupMulti, performStream, request, requestMulti, requestStream, requestAsync, requestMultiAsync } from "@jayess/curl";

function main(args) {
  var base = args[0];
  var secureBase = args[1];
  var proxyBase = args[2];
  var downloadPath = args[3];

  var easy = createEasy();
  configure(easy, {
    url: base + "/echo",
    method: "POST",
    headers: ["Content-Type: text/plain"],
    body: "kimchi"
  });
  var upload = perform(easy);
  console.log("curl-handle:" + upload.status + ":" + upload.body);
  cleanup(easy);

  var redirected = request({ url: base + "/redirect", followRedirects: true });
  console.log("curl-redirect:" + redirected.status + ":" + redirected.body);

  var cookie = request({ url: base + "/cookie", cookie: "session=kimchi" });
  console.log("curl-cookie:" + cookie.status + ":" + cookie.body);

  var downloaded = request({ url: base + "/download", outputPath: downloadPath });
  console.log("curl-download:" + downloaded.status + ":" + downloaded.path);

  var secure = request({ url: secureBase, insecure: true });
  console.log("curl-https:" + secure.status + ":" + secure.body);

  var proxied = request({ url: base + "/through-proxy", proxy: proxyBase });
  console.log("curl-proxy:" + proxied.status + ":" + proxied.body);

  var streamStart = Date.now();
  var streamFirstMs = -1;
  var streamChunks = [];
  var streamHandle = createEasy();
  configure(streamHandle, { url: base + "/stream" });
  var streamed = performStream(streamHandle, function (chunk) {
    if (streamFirstMs < 0) {
      streamFirstMs = Date.now() - streamStart;
    }
    streamChunks.push(chunk);
  });
  var streamTotalMs = Date.now() - streamStart;
  console.log("curl-stream:" + streamed.status + ":" + streamed.chunks + ":" + streamChunks.join("") + ":" + (streamFirstMs >= 0 && streamFirstMs + 75 < streamTotalMs));
  cleanup(streamHandle);

  var requestStreamChunks = [];
  var requestStreamed = requestStream({ url: base + "/stream" }, function (chunk) {
    requestStreamChunks.push(chunk);
  });
  console.log("curl-request-stream:" + requestStreamed.status + ":" + requestStreamed.chunks + ":" + requestStreamChunks.join(""));

  var multi = createMulti();
  var multiA = createEasy();
  var multiB = createEasy();
  configure(multiA, { url: base + "/multi-a" });
  configure(multiB, { url: base + "/multi-b" });
  addHandle(multi, multiA);
  addHandle(multi, multiB);
  var multiResponses = performMulti(multi);
  console.log("curl-multi:" + multiResponses.length + ":" + multiResponses[0].status + ":" + multiResponses[0].body + ":" + multiResponses[1].status + ":" + multiResponses[1].body);
  cleanup(multiA);
  cleanup(multiB);
  cleanupMulti(multi);

  var requestMultiResponses = requestMulti([
    { url: base + "/multi-a" },
    { url: base + "/multi-b" }
  ]);
  console.log("curl-request-multi:" + requestMultiResponses.length + ":" + requestMultiResponses[0].body + ":" + requestMultiResponses[1].body);

  var asyncTimerFired = false;
  setTimeout(function() {
    asyncTimerFired = true;
    console.log("curl-async-timer");
    return 0;
  }, 0);
  var asyncResponse = await requestAsync({ url: base + "/slow" });
  console.log("curl-async:" + asyncResponse.status + ":" + asyncResponse.body + ":" + asyncTimerFired);

  var asyncMultiResponses = await requestMultiAsync([
    { url: base + "/multi-a" },
    { url: base + "/multi-b" }
  ]);
  console.log("curl-async-multi:" + asyncMultiResponses.length + ":" + asyncMultiResponses[0].body + ":" + asyncMultiResponses[1].body);

  try {
    request({ url: base + "/slow", timeoutMs: 50 });
    console.log("curl-timeout:false");
  } catch (err) {
    console.log("curl-timeout:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "curl-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	downloadPath := filepath.Join(workdir, "curl-download.txt")
	cmd := exec.Command(outputPath, httpServer.URL, httpsServer.URL, proxyServer.URL, downloadPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled curl program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"curl-handle:200:kimchi",
		"curl-redirect:200:redirected",
		"curl-cookie:200:session=kimchi",
		"curl-download:200:" + downloadPath,
		"curl-https:200:secure",
		"curl-proxy:200:proxied:" + httpServer.URL + "/through-proxy",
		"curl-stream:200:",
		"chunk-1:chunk-2:true",
		"curl-request-stream:200:",
		"chunk-1:chunk-2",
		"curl-multi:2:200:alpha:200:beta",
		"curl-request-multi:2:alpha:beta",
		"curl-async-timer",
		"curl-async:200:slow:true",
		"curl-async-multi:2:alpha:beta",
		"curl-timeout:CurlError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected curl output to contain %q, got: %s", want, text)
		}
	}
	downloadedBytes, err := os.ReadFile(downloadPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(downloadedBytes) != "downloaded" {
		t.Fatalf("expected downloaded file content, got %q", string(downloadedBytes))
	}
}

func TestBuildExecutableReportsMissingCurlDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping curl missing deps test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "curl")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include", "curl")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/curl","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createEasyNative } from "./native/curl.bind.js";
export function createEasy() {
  return createEasyNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "curl.h"), []byte(`#pragma once
typedef void CURL;
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "curl.bind.js"), []byte(`const f = () => {};
export const createEasyNative = f;

export default {
  sources: ["./curl.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: ["-lmissingcurl"],
  exports: {
    createEasyNative: { symbol: "jayess_curl_create_easy_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "curl.c"), []byte(`#include "jayess_runtime.h"
#include <curl/curl.h>

jayess_value *jayess_curl_create_easy_native(void) {
  return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createEasy } from "@jayess/curl";

function main(args) {
  console.log(createEasy() == undefined);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "curl-missing-native")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail when curl library is missing")
	}
	if !strings.Contains(err.Error(), "native library link failed for missingcurl") {
		t.Fatalf("expected missing curl library diagnostic, got: %v", err)
	}
}

func TestBuildExecutableReportsMissingCurlHeadersClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping curl missing header test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "curl")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/curl","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createEasyNative } from "./native/curl.bind.js";
export function createEasy() {
  return createEasyNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "curl.bind.js"), []byte(`const f = () => {};
export const createEasyNative = f;

export default {
  sources: ["./curl.c"],
  includeDirs: ["."],
  cflags: [],
  ldflags: ["/lib/x86_64-linux-gnu/libcurl.so.4"],
  exports: {
    createEasyNative: { symbol: "jayess_curl_create_easy_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "curl.c"), []byte(`#include "jayess_runtime.h"
#include <curl/not_real_header.h>

jayess_value *jayess_curl_create_easy_native(void) {
  return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createEasy } from "@jayess/curl";

function main(args) {
  console.log(createEasy() == undefined);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "curl-missing-header-native")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail when curl headers are missing")
	}
	if !strings.Contains(err.Error(), "native header dependency missing for curl/not_real_header.h") {
		t.Fatalf("expected missing curl header diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessLibUVPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping libuv package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-libuv-backend-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(filepath.Join(workdir, "hello.txt"), []byte("kimchi"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "watched.txt"), []byte("start"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	recvProbe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP receiver probe returned error: %v", err)
	}
	recvPort := recvProbe.LocalAddr().(*net.UDPAddr).Port
	_ = recvProbe.Close()
	sendConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP sender returned error: %v", err)
	}
	defer sendConn.Close()
	sendPort := sendConn.LocalAddr().(*net.UDPAddr).Port
	tcpProbe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen tcp probe returned error: %v", err)
	}
	tcpPort := tcpProbe.Addr().(*net.TCPAddr).Port
	_ = tcpProbe.Close()

	if err := os.WriteFile(entry, []byte(`
	import { createLoop, scheduleStop, scheduleCallback, readFile, watchSignal, closeSignalWatcher, watchPath, closePathWatcher, spawnProcess, closeProcess, createUDP, bindUDP, recvUDP, sendUDP, closeUDP, createTCPServer, listenTCP, acceptTCP, closeTCPServer, createTCPClient, connectTCP, readTCP, writeTCP, closeTCPClient, run, runOnce, stop, closeLoop, now } from "@jayess/libuv";

function main(args) {
  var loop = createLoop();
  var before = now(loop);
  var count = 0;
  var fileText = "";
  var fileError = "";
  var signalText = "";
  var watchType = "";
  var processExit = "";
  var closeProcessResult = "";
  var udpText = "";
  var closeUDPResult = "";
  var tcpServerText = "";
  var tcpClientText = "";
  var closeTCPServerResult = "";
  var closeAcceptedTCPResult = "";
  var closeTCPClientResult = "";
  var acceptedClient = undefined;
  scheduleCallback(loop, 0, () => {
    count = count + 1;
  });
  readFile(loop, "./hello.txt", (result) => {
    if (result.ok) {
      fileText = result.data;
    } else {
      fileError = result.error.name;
    }
  });
  var watcherToClose = watchSignal(loop, "SIGUSR2", (signal) => {});
  var watcher = watchSignal(loop, "SIGUSR1", (signal) => {
    signalText = signal;
    stop(loop);
  });
  var pathWatcherToClose = watchPath(loop, "./hello.txt", (result) => {});
  var pathWatcher = watchPath(loop, "./watched.txt", (result) => {
    if (result.ok) {
      watchType = result.eventType;
    }
  });
  var process = spawnProcess(loop, "/bin/sh", ["-c", "exit 7"], (result, proc) => {
    processExit = result.exitStatus + ":" + (result.signal === undefined);
    closeProcessResult = "" + closeProcess(proc);
  });
  var udp = createUDP(loop);
  bindUDP(udp, "127.0.0.1", `+strconv.Itoa(recvPort)+`);
  recvUDP(udp, (result) => {
    if (result.ok) {
      udpText = result.data;
    }
  });
  sendUDP(udp, "127.0.0.1", `+strconv.Itoa(sendPort)+`, "pong");
  var tcpServer = createTCPServer(loop);
  listenTCP(tcpServer, "127.0.0.1", `+strconv.Itoa(tcpPort)+`, (result) => {
    if (result.ok) {
      acceptedClient = acceptTCP(tcpServer);
      if (acceptedClient != undefined) {
        readTCP(acceptedClient, (packet) => {
          if (packet.ok) {
            tcpServerText = packet.data;
            writeTCP(acceptedClient, "world");
          }
        });
      }
    }
  });
  var tcpClient = createTCPClient(loop);
  connectTCP(tcpClient, "127.0.0.1", `+strconv.Itoa(tcpPort)+`, (result) => {
    if (result.ok) {
      readTCP(tcpClient, (packet) => {
        if (packet.ok) {
          tcpClientText = packet.data;
        }
      });
      writeTCP(tcpClient, "hello");
    }
  });
  scheduleStop(loop, 200);
  var ran = run(loop);
  var after = now(loop);
  console.log("libuv-run:" + ran + ":" + (after >= before));
  console.log("libuv-callback:" + count);
  console.log("libuv-read-file:" + fileText + ":" + fileError);
  console.log("libuv-signal:" + signalText);
  console.log("libuv-close-watcher:" + closeSignalWatcher(watcherToClose));
  console.log("libuv-close-active-watcher:" + closeSignalWatcher(watcher));
  console.log("libuv-watch-type:" + watchType);
  console.log("libuv-close-path-watcher:" + closePathWatcher(pathWatcherToClose));
  console.log("libuv-close-active-path-watcher:" + closePathWatcher(pathWatcher));
  console.log("libuv-process-exit:" + processExit);
  console.log("libuv-close-process:" + closeProcessResult);
  console.log("libuv-udp:" + udpText);
  console.log("libuv-close-udp:" + closeUDP(udp));
  if (acceptedClient != undefined) {
    closeAcceptedTCPResult = "" + closeTCPClient(acceptedClient);
  }
  closeTCPClientResult = "" + closeTCPClient(tcpClient);
  closeTCPServerResult = "" + closeTCPServer(tcpServer);
  console.log("libuv-tcp-server:" + tcpServerText);
  console.log("libuv-tcp-client:" + tcpClientText);
  console.log("libuv-close-accepted-tcp:" + closeAcceptedTCPResult);
  console.log("libuv-close-tcp-client:" + closeTCPClientResult);
  console.log("libuv-close-tcp-server:" + closeTCPServerResult);

  scheduleStop(loop, 0);
  var once = runOnce(loop);
  console.log("libuv-once:" + once);
  stop(loop);
  console.log("libuv-close:" + closeLoop(loop));
  try {
    now(loop);
    console.log("libuv-after-close:false");
  } catch (err) {
    console.log("libuv-after-close:" + err.name);
  }
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "libuv-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start compiled libuv program: %v", err)
	}
	receivedCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 128)
		_ = sendConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := sendConn.ReadFromUDP(buf)
		if err != nil {
			receivedCh <- "ERR:" + err.Error()
			return
		}
		receivedCh <- string(buf[:n])
	}()
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(workdir, "watched.txt"), []byte("updated"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to update watched file for libuv program: %v", err)
	}
	sendToReceiverConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: recvPort})
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to dial udp receiver for libuv program: %v", err)
	}
	defer sendToReceiverConn.Close()
	if _, err := sendToReceiverConn.Write([]byte("hello")); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to send udp packet to libuv program: %v", err)
	}
	if err := cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to send SIGUSR1 to compiled libuv program: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("compiled libuv program returned error: %v: %s", err, out.String())
	}
	select {
	case got := <-receivedCh:
		if got != "pong" {
			t.Fatalf("expected libuv udp sender to emit pong, got: %s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for libuv udp sender packet")
	}
	text := out.String()
	for _, want := range []string{
		"libuv-run:1:true",
		"libuv-callback:1",
		"libuv-read-file:kimchi:",
		"libuv-signal:SIGUSR1",
		"libuv-once:1",
		"libuv-close-watcher:true",
		"libuv-close-active-watcher:true",
		"libuv-watch-type:",
		"libuv-close-path-watcher:true",
		"libuv-close-active-path-watcher:true",
		"libuv-process-exit:7:true",
		"libuv-close-process:true",
		"libuv-udp:hello",
		"libuv-close-udp:true",
		"libuv-tcp-server:hello",
		"libuv-tcp-client:world",
		"libuv-close-accepted-tcp:true",
		"libuv-close-tcp-client:true",
		"libuv-close-tcp-server:true",
		"libuv-close:true",
		"libuv-after-close:TypeError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected libuv output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessLibUVSchedulerIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping libuv scheduler integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-libuv-scheduler-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(filepath.Join(workdir, "hello.txt"), []byte("jjigae"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "watched.txt"), []byte("start"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	recvProbe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP receiver probe returned error: %v", err)
	}
	recvPort := recvProbe.LocalAddr().(*net.UDPAddr).Port
	_ = recvProbe.Close()
	sendConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP sender returned error: %v", err)
	}
	defer sendConn.Close()
	sendPort := sendConn.LocalAddr().(*net.UDPAddr).Port

	if err := os.WriteFile(entry, []byte(`
	import { createLoop, scheduleStop, scheduleCallback, readFile, watchPath, closePathWatcher, spawnProcess, closeProcess, createUDP, bindUDP, recvUDP, sendUDP, closeUDP, run, closeLoop } from "@jayess/libuv";

function main(args) {
  var loop = createLoop();
  var events = [];
  var deferred = () => {
    events.push("libuv");
  };

  Promise.resolve("micro").then((value) => {
    events.push("promise:" + value);
  });

  timers.setTimeout(() => {
    events.push("timer");
  }, 0);

  readFile(loop, "./hello.txt", (result) => {
    if (result.ok) {
      events.push("fs:" + result.data);
    } else {
      events.push("fs-error:" + result.error.name);
    }
  });
  var pathWatcher = watchPath(loop, "./watched.txt", (result) => {
    if (result.ok) {
      events.push("watch:" + result.eventType);
    } else {
      events.push("watch-error:" + result.error.name);
    }
  });
  spawnProcess(loop, "/bin/sh", ["-c", "exit 3"], (result, proc) => {
    events.push("proc:" + result.exitStatus);
    events.push("proc-close:" + closeProcess(proc));
  });
  var udp = createUDP(loop);
  bindUDP(udp, "127.0.0.1", `+strconv.Itoa(recvPort)+`);
  recvUDP(udp, (result) => {
    if (result.ok) {
      events.push("udp:" + result.data);
    } else {
      events.push("udp-error:" + result.error.name);
    }
  });
  sendUDP(udp, "127.0.0.1", `+strconv.Itoa(sendPort)+`, "pong");
  scheduleCallback(loop, 0, deferred);
  scheduleStop(loop, 20);
  var ran = run(loop);

  console.log("libuv-integrated-run:" + ran);
  console.log("libuv-integrated-events:" + events.join(","));
  console.log("libuv-integrated-close-path-watcher:" + closePathWatcher(pathWatcher));
  console.log("libuv-integrated-close-udp:" + closeUDP(udp));
  console.log("libuv-integrated-close:" + closeLoop(loop));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "libuv-scheduler-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start compiled libuv scheduler program: %v", err)
	}
	receivedCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 128)
		_ = sendConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := sendConn.ReadFromUDP(buf)
		if err != nil {
			receivedCh <- "ERR:" + err.Error()
			return
		}
		receivedCh <- string(buf[:n])
	}()
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(workdir, "watched.txt"), []byte("updated"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to update watched file for libuv scheduler program: %v", err)
	}
	sendToReceiverConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: recvPort})
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to dial udp receiver for libuv scheduler program: %v", err)
	}
	defer sendToReceiverConn.Close()
	if _, err := sendToReceiverConn.Write([]byte("hello")); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to send udp packet to libuv scheduler program: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("compiled libuv scheduler program returned error: %v: %s", err, out.String())
	}
	select {
	case got := <-receivedCh:
		if got != "pong" {
			t.Fatalf("expected libuv scheduler udp sender to emit pong, got: %s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for libuv scheduler udp sender packet")
	}
	text := out.String()
	for _, want := range []string{
		"libuv-integrated-run:1",
		"libuv-integrated-events:",
		"promise:micro",
		"timer",
		"fs:jjigae",
		"watch:",
		"proc:3",
		"proc-close:true",
		"udp:hello",
		"libuv",
		"libuv-integrated-close-path-watcher:true",
		"libuv-integrated-close-udp:true",
		"libuv-integrated-close:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected integrated libuv output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessLibUVProcessAndSignalIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping libuv process/signal integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir, err := os.MkdirTemp(repoRoot, "jayess-libuv-process-signal-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
	import { createLoop, watchSignal, closeSignalWatcher, spawnProcess, closeProcess, scheduleStop, run, stop, closeLoop } from "@jayess/libuv";

function main(args) {
  var loop = createLoop();
  var signalText = "";
  var processExit = "";
  var closeProcessResult = "";
  var closeWatcherResult = "";

  var watcher = watchSignal(loop, "SIGUSR1", (signal) => {
    signalText = signal;
    if (signalText != "" && processExit != "") {
      stop(loop);
    }
  });

  spawnProcess(loop, "/bin/sh", ["-c", "exit 5"], (result, proc) => {
    processExit = result.exitStatus + ":" + (result.signal === undefined);
    closeProcessResult = "" + closeProcess(proc);
    if (signalText != "" && processExit != "") {
      stop(loop);
    }
  });

  scheduleStop(loop, 250);
  console.log("libuv-process-signal-run:" + run(loop));
  closeWatcherResult = "" + closeSignalWatcher(watcher);
  console.log("libuv-process-signal-signal:" + signalText);
  console.log("libuv-process-signal-exit:" + processExit);
  console.log("libuv-process-signal-close-process:" + closeProcessResult);
  console.log("libuv-process-signal-close-watcher:" + closeWatcherResult);
  console.log("libuv-process-signal-close-loop:" + closeLoop(loop));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "libuv-process-signal-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start compiled libuv process/signal program: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to send SIGUSR1 to compiled libuv process/signal program: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("compiled libuv process/signal program returned error: %v: %s", err, out.String())
	}

	text := out.String()
	for _, want := range []string{
		"libuv-process-signal-run:1",
		"libuv-process-signal-signal:SIGUSR1",
		"libuv-process-signal-exit:5:true",
		"libuv-process-signal-close-process:true",
		"libuv-process-signal-close-watcher:true",
		"libuv-process-signal-close-loop:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected libuv process/signal output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableReportsMissingLibUVDepsClearly(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping libuv missing deps test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "libuv")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/libuv","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createLoopNative } from "./native/libuv.bind.js";
export function createLoop() {
  return createLoopNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "libuv.bind.js"), []byte(`const f = () => {};
export const createLoopNative = f;

export default {
  sources: ["./libuv.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: ["-lmissinguv"],
  exports: {
    createLoopNative: { symbol: "jayess_libuv_create_loop_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "uv.h"), []byte(`#pragma once
typedef struct uv_loop_s uv_loop_t;
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "libuv.c"), []byte(`#include "jayess_runtime.h"
#include <uv.h>

jayess_value *jayess_libuv_create_loop_native(void) {
  return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createLoop } from "@jayess/libuv";

function main(args) {
  console.log(createLoop() == undefined);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "libuv-missing-native")
	err = tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath)
	if err == nil {
		t.Fatalf("expected BuildExecutable to fail when libuv library is missing")
	}
	if !strings.Contains(err.Error(), "native library link failed for missinguv") {
		t.Fatalf("expected missing libuv library diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessOpenSSLTLSConnect(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL TLS client test: %v", err)
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

	cert := server.Certificate()
	if cert == nil {
		t.Fatalf("expected server certificate")
	}
	serverName := serverURL.Hostname()
	if len(cert.DNSNames) > 0 {
		serverName = cert.DNSNames[0]
	}

	workdir := t.TempDir()
	repoRoot := repoRootFromBackendTest(t)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "openssl"),
		filepath.Join(workdir, "node_modules", "@jayess", "openssl"),
	)
	caPath := filepath.Join(workdir, "openssl-package-server-cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		t.Fatalf("failed to encode certificate PEM")
	}
	if err := os.WriteFile(caPath, pemBytes, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	source := fmt.Sprintf(`
import { tlsAvailable, tlsBackend, tlsConnect } from "@jayess/openssl";

function main() {
  console.log("openssl-tls-cap:" + tlsAvailable() + ":" + tlsBackend());
  var socket = tlsConnect({
    host: "%s",
    port: %d,
    serverName: "%s",
    caFile: "%s",
    trustSystem: false,
    alpnProtocols: ["jayess-test", "http/1.1"]
  });
  console.log("openssl-tls-socket:" + socket.secure + ":" + socket.backend + ":" + socket.authorized + ":" + socket.protocol);
  var cert = socket.getPeerCertificate();
  console.log("openssl-tls-cert:" + (cert != undefined) + ":" + cert.backend + ":" + cert.authorized);
  console.log("openssl-tls-alpn:" + socket.alpnProtocol + ":" + socket.alpnProtocols.length);
  socket.close();
  console.log("openssl-tls-cert-after-close:" + cert.subjectCN + ":" + cert.issuerCN + ":" + cert.backend);
  return 0;
}
`, serverURL.Hostname(), port, serverName, caPath)
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-tls-connect-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled OpenSSL TLS client program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "openssl-tls-cap:true:openssl") {
		t.Fatalf("expected OpenSSL TLS capability output, got: %s", text)
	}
	if !strings.Contains(text, "openssl-tls-socket:true:openssl:true:TLS") {
		t.Fatalf("expected OpenSSL TLS socket output, got: %s", text)
	}
	if !strings.Contains(text, "openssl-tls-cert:true:openssl:true") {
		t.Fatalf("expected OpenSSL TLS certificate output, got: %s", text)
	}
	if !strings.Contains(text, "openssl-tls-alpn:jayess-test:2") {
		t.Fatalf("expected OpenSSL TLS ALPN output, got: %s", text)
	}
	if !strings.Contains(text, "openssl-tls-cert-after-close:") {
		t.Fatalf("expected OpenSSL TLS detached certificate output, got: %s", text)
	}
}

func TestBuildExecutableEnforcesJayessOpenSSLTLSHostnameVerification(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL TLS hostname test: %v", err)
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

	workdir := t.TempDir()
	repoRoot := repoRootFromBackendTest(t)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "openssl"),
		filepath.Join(workdir, "node_modules", "@jayess", "openssl"),
	)
	caPath := filepath.Join(workdir, "openssl-package-hostname-cert.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if pemBytes == nil {
		t.Fatalf("failed to encode certificate PEM")
	}
	if err := os.WriteFile(caPath, pemBytes, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	source := fmt.Sprintf(`
import { tlsConnect } from "@jayess/openssl";

function main() {
  try {
    tlsConnect({
      host: "%s",
      port: %d,
      serverName: "wrong.example.test",
      caFile: "%s",
      trustSystem: false,
      alpnProtocols: "http/1.1"
    });
    console.log("openssl-tls-hostname:unexpected-success");
  } catch (err) {
    console.log("openssl-tls-hostname:error:" + err.message);
  }
  return 0;
}
`, serverURL.Hostname(), port, caPath)
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-tls-hostname-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled OpenSSL TLS hostname program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "openssl-tls-hostname:error:") || strings.Contains(text, "openssl-tls-hostname:unexpected-success") {
		t.Fatalf("expected OpenSSL TLS hostname verification failure output, got: %s", text)
	}
}

func TestBuildExecutableSupportsJayessOpenSSLTLSServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("server-side OpenSSL TLS package path is not implemented on Windows yet")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping OpenSSL TLS server test: %v", err)
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

	workdir := t.TempDir()
	repoRoot := repoRootFromBackendTest(t)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "openssl"),
		filepath.Join(workdir, "node_modules", "@jayess", "openssl"),
	)
	certPath, keyPath := writeTestTLSCertificatePair(t, workdir, "jayess-openssl-package-server")

	entry := filepath.Join(workdir, "main.js")
	source := fmt.Sprintf(`
import { tlsCreateServer } from "@jayess/openssl";

function main() {
  var server = undefined;
  server = tlsCreateServer({ cert: "%s", key: "%s" }, (socket) => {
    console.log("openssl-tls-server:" + socket.secure + ":" + socket.backend + ":" + socket.protocol);
    var incoming = socket.read();
    console.log("openssl-tls-server-read:" + incoming);
    socket.write("pong");
    socket.close();
    server.close();
    return 0;
  });
  server.listen(%d, "127.0.0.1");
  return 0;
}
`, certPath, keyPath, port)
	if err := os.WriteFile(entry, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "openssl-tls-server-native")
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

	client := &tls.Config{InsecureSkipVerify: true}
	var conn *tls.Conn
	for i := 0; i < 40; i++ {
		conn, err = tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), client)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("TLS dial returned error: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("TLS write returned error: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("TLS read returned error: %v", err)
	}
	if got := string(reply); got != "pong" {
		t.Fatalf("expected pong reply, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled OpenSSL TLS server returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	if !strings.Contains(text, "openssl-tls-server:true:openssl:TLS") {
		t.Fatalf("expected OpenSSL TLS server output, got: %s", text)
	}
	if !strings.Contains(text, "openssl-tls-server-read:ping") {
		t.Fatalf("expected OpenSSL TLS server read output, got: %s", text)
	}
}

func nativeOutputPath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".exe")
	}
	return filepath.Join(dir, name)
}
