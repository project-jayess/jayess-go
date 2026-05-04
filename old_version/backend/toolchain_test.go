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

func copyFileForTest(t *testing.T, srcPath, dstPath string) {
	t.Helper()

	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", srcPath, err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", dstPath, err)
	}
}

func rewriteAudioPackageToUseCubebStub(t *testing.T, repoRoot, pkgDir string) {
	t.Helper()

	nativeDir := filepath.Join(pkgDir, "native")
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "cubeb", "include"),
		filepath.Join(nativeDir, "cubeb_include"),
	)
	data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "miniaudio", "miniaudio.h"))
	if err != nil {
		t.Fatalf("ReadFile(miniaudio.h) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio.h"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(miniaudio.h) returned error: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(repoRoot, "refs", "miniaudio", "miniaudio.c"))
	if err != nil {
		t.Fatalf("ReadFile(miniaudio.c) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio.c"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(miniaudio.c) returned error: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(repoRoot, "refs", "miniaudio", "extras", "stb_vorbis.c"))
	if err != nil {
		t.Fatalf("ReadFile(stb_vorbis.c) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "stb_vorbis.c"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(stb_vorbis.c) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "cubeb_include", "cubeb", "cubeb_export.h"), []byte(`#ifndef CUBEB_EXPORT
#define CUBEB_EXPORT
#endif
`), 0o644); err != nil {
		t.Fatalf("WriteFile(cubeb_export.h) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "audio.bind.js"), []byte(`const f = () => {};
export const createContextNative = f;
export const destroyContextNative = f;
export const backendIdNative = f;
export const maxChannelCountNative = f;
export const listOutputDevicesNative = f;
export const listInputDevicesNative = f;
export const preferredSampleRateNative = f;
export const minLatencyNative = f;
export const createPlaybackStreamNative = f;
export const startPlaybackStreamNative = f;
export const pausePlaybackStreamNative = f;
export const stopPlaybackStreamNative = f;
export const submitPlaybackSamplesNative = f;
export const playbackStatsNative = f;
export const closePlaybackStreamNative = f;
export const nextStreamEventNative = f;
export const createCaptureStreamNative = f;
export const startCaptureStreamNative = f;
export const stopCaptureStreamNative = f;
export const readCapturedSamplesNative = f;
export const captureStatsNative = f;
export const closeCaptureStreamNative = f;
export const loadWavNative = f;
export const loadOggNative = f;
export const loadMp3Native = f;
export const loadFlacNative = f;

export default {
  sources: ["./audio.c", "./cubeb_stub.c", "./miniaudio.c"],
  includeDirs: [".", "./cubeb_include"],
  cflags: ["-DMA_NO_DEVICE_IO", "-DMA_NO_THREADING"],
  ldflags: ["-pthread", "-ldl", "-lm"],
  exports: {
    createContextNative: { symbol: "jayess_audio_create_context", type: "function" },
    destroyContextNative: { symbol: "jayess_audio_destroy_context", type: "function" },
    backendIdNative: { symbol: "jayess_audio_backend_id", type: "function" },
    maxChannelCountNative: { symbol: "jayess_audio_max_channel_count", type: "function" },
    listOutputDevicesNative: { symbol: "jayess_audio_list_output_devices", type: "function" },
    listInputDevicesNative: { symbol: "jayess_audio_list_input_devices", type: "function" },
    preferredSampleRateNative: { symbol: "jayess_audio_preferred_sample_rate", type: "function" },
    minLatencyNative: { symbol: "jayess_audio_min_latency", type: "function" },
    createPlaybackStreamNative: { symbol: "jayess_audio_create_playback_stream", type: "function" },
    startPlaybackStreamNative: { symbol: "jayess_audio_start_playback_stream", type: "function" },
    pausePlaybackStreamNative: { symbol: "jayess_audio_pause_playback_stream", type: "function" },
    stopPlaybackStreamNative: { symbol: "jayess_audio_stop_playback_stream", type: "function" },
    submitPlaybackSamplesNative: { symbol: "jayess_audio_submit_playback_samples", type: "function" },
    playbackStatsNative: { symbol: "jayess_audio_playback_stats", type: "function" },
    closePlaybackStreamNative: { symbol: "jayess_audio_close_playback_stream", type: "function" },
    nextStreamEventNative: { symbol: "jayess_audio_next_stream_event", type: "function" },
    createCaptureStreamNative: { symbol: "jayess_audio_create_capture_stream", type: "function" },
    startCaptureStreamNative: { symbol: "jayess_audio_start_capture_stream", type: "function" },
    stopCaptureStreamNative: { symbol: "jayess_audio_stop_capture_stream", type: "function" },
    readCapturedSamplesNative: { symbol: "jayess_audio_read_captured_samples", type: "function" },
    captureStatsNative: { symbol: "jayess_audio_capture_stats", type: "function" },
    closeCaptureStreamNative: { symbol: "jayess_audio_close_capture_stream", type: "function" },
    loadWavNative: { symbol: "jayess_audio_load_wav", type: "function" },
    loadOggNative: { symbol: "jayess_audio_load_ogg", type: "function" },
    loadMp3Native: { symbol: "jayess_audio_load_mp3", type: "function" },
    loadFlacNative: { symbol: "jayess_audio_load_flac", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile(audio.bind.js) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "cubeb_stub.c"), []byte(`#include <cubeb/cubeb.h>
#include <pthread.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

struct cubeb {
  int dummy;
};

struct cubeb_stream {
  cubeb_stream_params input_params;
  cubeb_stream_params output_params;
  uint32_t latency_frames;
  cubeb_data_callback data_cb;
  cubeb_state_callback state_cb;
  void * user_ptr;
  pthread_t thread;
  int running;
  int thread_started;
  int capture_mode;
  int emit_error_once;
};

static void * cubeb_stub_playback_thread(void * ptr) {
  cubeb_stream * stream = (cubeb_stream *) ptr;
  long frames = stream->latency_frames > 0 ? (long) stream->latency_frames : 64;
  long samples = frames * (long)(stream->capture_mode ? stream->input_params.channels : stream->output_params.channels);
  while (stream->running) {
    if (stream->emit_error_once) {
      stream->emit_error_once = 0;
      stream->running = 0;
      if (stream->state_cb != NULL) {
        stream->state_cb(stream, stream->user_ptr, CUBEB_STATE_ERROR);
      }
      break;
    }
    if (stream->data_cb != NULL) {
      if (stream->capture_mode) {
        long i = 0;
        if (stream->input_params.format == CUBEB_SAMPLE_FLOAT32NE) {
          float * in = (float *) calloc((size_t)samples, sizeof(float));
          for (i = 0; i < samples; i++) {
            in[i] = (i % 2 == 0) ? 0.25f : -0.25f;
          }
          stream->data_cb(stream, stream->user_ptr, in, NULL, frames);
          free(in);
        } else {
          int16_t * in = (int16_t *) calloc((size_t)samples, sizeof(int16_t));
          for (i = 0; i < samples; i++) {
            in[i] = (i % 2 == 0) ? 8192 : -8192;
          }
          stream->data_cb(stream, stream->user_ptr, in, NULL, frames);
          free(in);
        }
      } else if (stream->output_params.format == CUBEB_SAMPLE_FLOAT32NE) {
        float * out = (float *) calloc((size_t)samples, sizeof(float));
        stream->data_cb(stream, stream->user_ptr, NULL, out, frames);
        free(out);
      } else {
        int16_t * out = (int16_t *) calloc((size_t)samples, sizeof(int16_t));
        stream->data_cb(stream, stream->user_ptr, NULL, out, frames);
        free(out);
      }
    }
    usleep(1000);
  }
  return NULL;
}

int cubeb_init(cubeb ** context, char const * context_name, char const * backend_name) {
  (void) context_name;
  (void) backend_name;
  if (context == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  *context = (cubeb *) calloc(1, sizeof(cubeb));
  return *context != NULL ? CUBEB_OK : CUBEB_ERROR;
}

char const * cubeb_get_backend_id(cubeb * context) {
  (void) context;
  return "stub-cubeb";
}

int cubeb_get_max_channel_count(cubeb * context, uint32_t * max_channels) {
  (void) context;
  if (max_channels == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  *max_channels = 2;
  return CUBEB_OK;
}

int cubeb_get_preferred_sample_rate(cubeb * context, uint32_t * rate) {
  (void) context;
  if (rate == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  *rate = 48000;
  return CUBEB_OK;
}

int cubeb_get_min_latency(cubeb * context, cubeb_stream_params * params, uint32_t * latency_frames) {
  (void) context;
  (void) params;
  if (latency_frames == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  *latency_frames = 64;
  return CUBEB_OK;
}

int cubeb_stream_init(cubeb * context, cubeb_stream ** stream, char const * stream_name,
                      cubeb_devid input_device, cubeb_stream_params * input_stream_params,
                      cubeb_devid output_device, cubeb_stream_params * output_stream_params,
                      uint32_t latency_frames, cubeb_data_callback data_callback,
                      cubeb_state_callback state_callback, void * user_ptr) {
  cubeb_stream * created = NULL;
  (void) context;
  (void) stream_name;
  (void) input_device;
  (void) output_device;
  if (stream == NULL || (output_stream_params == NULL && input_stream_params == NULL)) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  created = (cubeb_stream *) calloc(1, sizeof(cubeb_stream));
  if (created == NULL) {
    return CUBEB_ERROR;
  }
  if (input_stream_params != NULL) {
    created->input_params = *input_stream_params;
  }
  if (output_stream_params != NULL) {
    created->output_params = *output_stream_params;
  }
  created->capture_mode = output_stream_params == NULL ? 1 : 0;
  created->emit_error_once = stream_name != NULL && strstr(stream_name, "error") != NULL;
  created->latency_frames = latency_frames;
  created->data_cb = data_callback;
  created->state_cb = state_callback;
  created->user_ptr = user_ptr;
  *stream = created;
  return CUBEB_OK;
}

void cubeb_stream_destroy(cubeb_stream * stream) {
  if (stream == NULL) {
    return;
  }
  if (stream->running) {
    stream->running = 0;
  }
  if (stream->thread_started) {
    pthread_join(stream->thread, NULL);
    stream->thread_started = 0;
  }
  free(stream);
}

int cubeb_stream_start(cubeb_stream * stream) {
  if (stream == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  if (stream->running) {
    return CUBEB_OK;
  }
  stream->running = 1;
  if (stream->state_cb != NULL) {
    stream->state_cb(stream, stream->user_ptr, CUBEB_STATE_STARTED);
  }
  if (pthread_create(&stream->thread, NULL, cubeb_stub_playback_thread, stream) != 0) {
    stream->running = 0;
    return CUBEB_ERROR;
  }
  stream->thread_started = 1;
  return CUBEB_OK;
}

int cubeb_stream_stop(cubeb_stream * stream) {
  if (stream == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  if (stream->running) {
    stream->running = 0;
  }
  if (stream->thread_started) {
    pthread_join(stream->thread, NULL);
    stream->thread_started = 0;
  }
  if (stream->state_cb != NULL) {
    stream->state_cb(stream, stream->user_ptr, CUBEB_STATE_STOPPED);
  }
  return CUBEB_OK;
}

int cubeb_enumerate_devices(cubeb * context, cubeb_device_type devtype, cubeb_device_collection * collection) {
  cubeb_device_info * devices = NULL;
  (void) context;
  if (collection == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  collection->count = devtype == (CUBEB_DEVICE_TYPE_INPUT | CUBEB_DEVICE_TYPE_OUTPUT) ? 2 : 1;
  devices = (cubeb_device_info *) calloc(collection->count, sizeof(cubeb_device_info));
  if (devices == NULL) {
    return CUBEB_ERROR;
  }
  devices[0].device_id = devtype == CUBEB_DEVICE_TYPE_INPUT ? "stub-input-0" : "stub-output-0";
  devices[0].friendly_name = devtype == CUBEB_DEVICE_TYPE_INPUT ? "Stub Input" : "Stub Output";
  devices[0].group_id = "stub-group";
  devices[0].vendor_name = "jayess";
  devices[0].type = devtype == CUBEB_DEVICE_TYPE_INPUT ? CUBEB_DEVICE_TYPE_INPUT : CUBEB_DEVICE_TYPE_OUTPUT;
  devices[0].state = CUBEB_DEVICE_STATE_ENABLED;
  devices[0].preferred = CUBEB_DEVICE_PREF_MULTIMEDIA;
  devices[0].format = CUBEB_DEVICE_FMT_F32NE;
  devices[0].default_format = CUBEB_DEVICE_FMT_F32NE;
  devices[0].max_channels = 2;
  devices[0].default_rate = 48000;
  devices[0].max_rate = 48000;
  devices[0].min_rate = 8000;
  devices[0].latency_lo = 64;
  devices[0].latency_hi = 256;
  if (collection->count > 1) {
    devices[1] = devices[0];
    devices[1].device_id = "stub-output-0";
    devices[1].friendly_name = "Stub Output";
    devices[1].type = CUBEB_DEVICE_TYPE_OUTPUT;
  }
  collection->device = devices;
  return CUBEB_OK;
}

int cubeb_device_collection_destroy(cubeb * context, cubeb_device_collection * collection) {
  (void) context;
  if (collection == NULL) {
    return CUBEB_ERROR_INVALID_PARAMETER;
  }
  free(collection->device);
  collection->device = NULL;
  collection->count = 0;
  return CUBEB_OK;
}

void cubeb_destroy(cubeb * context) {
  free(context);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile(cubeb_stub.c) returned error: %v", err)
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

func TestFormatNativeBuildErrorReportsMissingTargetSDKHeadersClearly(t *testing.T) {
	err := errors.New("compile failed")

	clangStyle := formatNativeBuildError(err, "fatal error: 'stdio.h' file not found")
	if !strings.Contains(clangStyle.Error(), "native target SDK or C runtime headers missing for stdio.h") {
		t.Fatalf("expected stdio.h SDK diagnostic, got: %v", clangStyle)
	}

	libcStyle := formatNativeBuildError(err, "fatal error: bits/libc-header-start.h: No such file or directory")
	if !strings.Contains(libcStyle.Error(), "native target SDK or C runtime headers missing for bits/libc-header-start.h") {
		t.Fatalf("expected libc header SDK diagnostic, got: %v", libcStyle)
	}

	windowsStyle := formatNativeBuildError(err, "fatal error: 'windows.h' file not found")
	if !strings.Contains(windowsStyle.Error(), "native target SDK or C runtime headers missing for windows.h") {
		t.Fatalf("expected windows.h SDK diagnostic, got: %v", windowsStyle)
	}
}

func TestFormatNativeBuildErrorReportsTargetSpecificSDKHintsClearly(t *testing.T) {
	err := errors.New("compile failed")

	darwinStyle := formatNativeBuildErrorForTarget(err, "fatal error: 'stdio.h' file not found", "arm64-apple-darwin")
	if !strings.Contains(darwinStyle.Error(), "Apple SDK/sysroot") {
		t.Fatalf("expected darwin SDK hint, got: %v", darwinStyle)
	}
	if !strings.Contains(darwinStyle.Error(), "xcrun/SDKROOT") {
		t.Fatalf("expected darwin xcrun/SDKROOT hint, got: %v", darwinStyle)
	}

	windowsStyle := formatNativeBuildErrorForTarget(err, "fatal error: 'windows.h' file not found", "x86_64-pc-windows-msvc")
	if !strings.Contains(windowsStyle.Error(), "Windows SDK") {
		t.Fatalf("expected windows SDK hint, got: %v", windowsStyle)
	}
	if !strings.Contains(windowsStyle.Error(), "MSVC/clang-cl environment") {
		t.Fatalf("expected windows MSVC hint, got: %v", windowsStyle)
	}

	linuxStyle := formatNativeBuildErrorForTarget(err, "fatal error: 'stdio.h' file not found", "aarch64-unknown-linux-gnu")
	if !strings.Contains(linuxStyle.Error(), "target libc/sysroot") {
		t.Fatalf("expected linux sysroot hint, got: %v", linuxStyle)
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

func TestBuildExecutablePreservesModuleStateAcrossLocalScopeExit(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
var savedObject = undefined;
var savedArray = undefined;

function seedState() {
  var localObject = { label: "module-object", count: 2 };
  var localArray = [localObject, 7];
  savedObject = localObject;
  savedArray = localArray;
}

function main(args) {
  seedState();
  savedObject.count = savedObject.count + 5;
  console.log("module-object:" + savedObject.label + ":" + savedObject.count);
  console.log("module-array:" + savedArray.length + ":" + savedArray[0].count + ":" + savedArray[1]);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "module-state-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"module-object:module-object:7", "module-array:2:7:7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected module-state lifetime output to contain %q, got: %s", want, text)
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

func TestBuildExecutablePreservesPreviousValuesAcrossContainerReplacement(t *testing.T) {
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
  var oldObject = { count: 2 };
  var oldArrayValue = { count: 4 };
  var box = { value: oldObject };
  var items = [oldArrayValue];

  box.value = { count: 9 };
  items[0] = { count: 8 };

  oldObject.count = oldObject.count + 5;
  oldArrayValue.count = oldArrayValue.count + 3;

  console.log("old-object:" + oldObject.count);
  console.log("new-object:" + box.value.count);
  console.log("old-array:" + oldArrayValue.count);
  console.log("new-array:" + items[0].count);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "container-replacement-lifetime-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{"old-object:7", "new-object:9", "old-array:7", "new-array:8"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected replacement lifetime output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableContainerReplacementDoesNotFinalizeExternallyAliasedValues(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, closeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  const oldObject = makeProbe("old-object");
  const oldArrayValue = makeProbe("old-array");
  const box = { value: oldObject };
  const items = [oldArrayValue];

  box.value = { replacement: true };
  items[0] = { replacement: true };

  console.log("after-replace:" + cleanupLog() + ":" + (oldObject != null) + ":" + (oldArrayValue != null));
  console.log("close-object:" + closeProbe(oldObject) + ":" + cleanupLog());
  console.log("close-array:" + closeProbe(oldArrayValue) + ":" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "container-replacement-alias-cleanup-native")
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
		"after-replace::true:true",
		"close-object:true:old-object;",
		"close-array:true:old-object;old-array;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected container replacement alias-cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableContainerRemovalDoesNotFinalizeExternallyAliasedValues(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, closeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  const removedObject = makeProbe("removed-object");
  const removedArray = makeProbe("removed-array");
  const box = { value: removedObject };
  const items = [removedArray];
  const alias = items[0];

  delete box.value;
  items.pop();

  console.log("after-remove:" + cleanupLog() + ":" + (removedObject != null) + ":" + (alias != null));
  console.log("close-object:" + closeProbe(removedObject) + ":" + cleanupLog());
  console.log("close-array:" + closeProbe(alias) + ":" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "container-removal-alias-cleanup-native")
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
		"after-remove::true:true",
		"close-object:true:removed-object;",
		"close-array:true:removed-object;removed-array;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected container removal alias-cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableDestructuredAliasesSurviveContainerRemoval(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, closeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  const objectProbe = makeProbe("destructured-object");
  const arrayProbe = makeProbe("destructured-array");
  const box = { value: objectProbe };
  const items = [arrayProbe];
  const { value: objectAlias } = box;
  const [arrayAlias] = items;

  delete box.value;
  items.pop();

  console.log("after-remove:" + cleanupLog() + ":" + (objectAlias != null) + ":" + (arrayAlias != null));
  console.log("close-object:" + closeProbe(objectAlias) + ":" + cleanupLog());
  console.log("close-array:" + closeProbe(arrayAlias) + ":" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructured-container-removal-native")
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
		"after-remove::true:true",
		"close-object:true:destructured-object;",
		"close-array:true:destructured-object;destructured-array;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected destructured removal output to contain %q, got: %s", want, text)
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

func TestBuildExecutableReleasesCapturedValuesWhenClosureEnvironmentDies(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function exerciseClosureCleanup() {
  var closure = (() => {
    const captured = makeProbe("captured-closure");
    return () => captured;
  })();

  const alias = closure();
  console.log("during-closure:" + cleanupLog() + ":" + (alias != null));
}

function main(args) {
  resetCleanupLog();
  console.log("before-closure:" + cleanupLog());
  exerciseClosureCleanup();
  console.log("after-closure:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "closure-environment-cleanup-native")
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
		"before-closure:",
		"during-closure::true",
		"after-closure:captured-closure;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected closure-environment cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func prepareCleanupProbePackage(t *testing.T, workdir string) {
	t.Helper()

	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "cleanupprobe")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/cleanupprobe","main":"./index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { makeProbeNative, closeProbeNative, cleanupLogNative, resetCleanupLogNative } from "./native/cleanupprobe.bind.js";

export function makeProbe(label) {
  return makeProbeNative(label);
}

export function closeProbe(value) {
  return closeProbeNative(value);
}

export function cleanupLog() {
  return cleanupLogNative();
}

export function resetCleanupLog() {
  return resetCleanupLogNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "cleanupprobe.bind.js"), []byte(`
export const makeProbeNative = f;
export const closeProbeNative = f;
export const cleanupLogNative = f;
export const resetCleanupLogNative = f;

export default {
  sources: ["./cleanupprobe.c"],
  exports: {
    makeProbeNative: { symbol: "jayess_cleanup_probe_make_native", type: "function", borrowsArgs: true },
    closeProbeNative: { symbol: "jayess_cleanup_probe_close_native", type: "function" },
    cleanupLogNative: { symbol: "jayess_cleanup_probe_log_native", type: "function" },
    resetCleanupLogNative: { symbol: "jayess_cleanup_probe_reset_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "cleanupprobe.c"), []byte(`
#include "jayess_runtime.h"

#include <stdlib.h>
#include <string.h>

struct jayess_cleanup_probe {
  char *label;
};

static char jayess_cleanup_probe_log[256];

static char *jayess_cleanup_probe_dup(const char *text) {
  size_t length;
  char *copy;
  if (text == NULL) {
    text = "";
  }
  length = strlen(text);
  copy = (char *)malloc(length + 1);
  if (copy == NULL) {
    return NULL;
  }
  memcpy(copy, text, length + 1);
  return copy;
}

static void jayess_cleanup_probe_append(const char *label) {
  size_t current = strlen(jayess_cleanup_probe_log);
  size_t extra = label != NULL ? strlen(label) : 0;
  if (current + extra + 2 >= sizeof(jayess_cleanup_probe_log)) {
    return;
  }
  if (extra > 0) {
    memcpy(jayess_cleanup_probe_log + current, label, extra);
    current += extra;
  }
  jayess_cleanup_probe_log[current++] = ';';
  jayess_cleanup_probe_log[current] = '\0';
}

static void jayess_cleanup_probe_finalizer(void *handle) {
  struct jayess_cleanup_probe *probe = (struct jayess_cleanup_probe *)handle;
  if (probe == NULL) {
    return;
  }
  jayess_cleanup_probe_append(probe->label);
  free(probe->label);
  free(probe);
}

jayess_value *jayess_cleanup_probe_make_native(jayess_value *label) {
  const char *text = jayess_expect_string(label, "cleanup probe label");
  struct jayess_cleanup_probe *probe;
  if (jayess_has_exception()) {
    return jayess_value_undefined();
  }
  probe = (struct jayess_cleanup_probe *)calloc(1, sizeof(struct jayess_cleanup_probe));
  if (probe == NULL) {
    return jayess_value_undefined();
  }
  probe->label = jayess_cleanup_probe_dup(text);
  if (probe->label == NULL) {
    free(probe);
    return jayess_value_undefined();
  }
  return jayess_value_from_managed_native_handle("CleanupProbe", probe, jayess_cleanup_probe_finalizer);
}

jayess_value *jayess_cleanup_probe_close_native(jayess_value *value) {
  return jayess_value_from_bool(jayess_value_close_native_handle(value));
}

jayess_value *jayess_cleanup_probe_log_native(void) {
  return jayess_value_from_string(jayess_cleanup_probe_log);
}

jayess_value *jayess_cleanup_probe_reset_native(void) {
  jayess_cleanup_probe_log[0] = '\0';
  return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func TestBuildExecutableDestroysValuesAtLexicalScopeExitByDefault(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  console.log("before-scope:" + cleanupLog());
  {
    const scoped = makeProbe("lexical-scope");
    console.log("during-scope:" + cleanupLog() + ":" + (scoped != null));
  }
  console.log("after-scope:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "lexical-scope-cleanup-native")
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
		"before-scope:",
		"during-scope::true",
		"after-scope:lexical-scope;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected lexical-scope cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableCleansUpEligibleDynamicLocalsOnScopeExit(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, closeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

class FreshBox {}
class PlainCtorBox {
  constructor() {
    this.kind = "plain";
  }
}

class FreshReturnCtorBox {
  constructor() {
    return { kind: "alt-fresh" };
  }
}

function innerReturn() {
  const value = makeProbe("return");
  return 7;
}

function freshObjectTemp() {
  return { kind: "fresh-call" };
}

function freshInvokeObject() {
  return { kind: "invoke-fresh" };
}

function freshBox() {
  return new FreshBox();
}

function freshSwitchCase() {
  switch ("case-" + "a") {
    case "case-a":
      break;
    case "case-b":
      break;
  }
}

function boundOffset(offset, x) {
  return x + offset;
}

function largeOffset(x) {
  return x + 20;
}

function boundGreaterThan(min, x) {
  return x > min;
}

function boundEquals(expected, x) {
  return x == expected;
}

function boundPairSum(a, b, x) {
  return x + a + b;
}

function boundBetween(min, max, x) {
  return x > min && x < max;
}

function boundTripleEquals(a, b, x) {
  return x == a + b;
}

function boundTripleSum(a, b, c, x) {
  return x + a + b + c;
}

function boundWindow(min, mid, max, x) {
  return x > min && x < max && x != mid;
}

function boundQuadEquals(a, b, c, x) {
  return x == a + b + c;
}

function boundQuadSum(a, b, c, d, x) {
  return x + a + b + c + d;
}

function boundOuterWindow(min, low, high, max, x) {
  return x > min && x >= low && x < max && x != high;
}

function boundQuintEquals(a, b, c, d, x) {
  return x == a + b + c + d;
}

function boundQuintSum(a, b, c, d, e, x) {
  return x + a + b + c + d + e;
}

function boundSextEquals(a, b, c, d, e, x) {
  return x == a + b + c + d + e;
}

function boundSextSum(a, b, c, d, e, f, x) {
  return x + a + b + c + d + e + f;
}

function boundSeptEquals(a, b, c, d, e, f, x) {
  return x == a + b + c + d + e + f;
}

function boundSeptSum(a, b, c, d, e, f, g, x) {
  return x + a + b + c + d + e + f + g;
}

function boundOctEquals(a, b, c, d, e, f, g, x) {
  return x == a + b + c + d + e + f + g;
}

function boundOctSum(a, b, c, d, e, f, g, h, x) {
  return x + a + b + c + d + e + f + g + h;
}

function boundNonetEquals(a, b, c, d, e, f, g, h, x) {
  return x == a + b + c + d + e + f + g + h;
}

function boundNonetSum(a, b, c, d, e, f, g, h, i, x) {
  return x + a + b + c + d + e + f + g + h + i;
}

function boundDecetEquals(a, b, c, d, e, f, g, h, i, x) {
  return x == a + b + c + d + e + f + g + h + i;
}

function boundDecetSum(a, b, c, d, e, f, g, h, i, j, x) {
  return x + a + b + c + d + e + f + g + h + i + j;
}

function boundUndecEquals(a, b, c, d, e, f, g, h, i, j, x) {
  return x == a + b + c + d + e + f + g + h + i + j;
}

function boundUndecSum(a, b, c, d, e, f, g, h, i, j, k, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k;
}

function boundDuodecEquals(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x == a + b + c + d + e + f + g + h + i + j + k + l;
}

function boundDuodecSum(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k + l;
}

function boundTridecSum(a, b, c, d, e, f, g, h, i, j, k, l, m, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k + l + m;
}

function numericSlowPathMap() {
  const items = [1, 2];
  const mapped = items.map(boundTridecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1));
  console.log("slow-numeric-map:" + mapped[0] + "," + mapped[1]);
}


function functionScopedCleanup() {
  var scoped = makeProbe("function-var");
  console.log("during-function-var:" + cleanupLog());
  return 11;
}

function loopVarFunctionScopedCleanup() {
  for (var n = 0; n < 3; n = n + 1) {
    var scoped = makeProbe("loop-var" + n);
  }
  return 13;
}

function discardedFreshTemporaries() {
  freshObjectTemp();
  freshSwitchCase();
  (() => "fresh-fn");
  new PlainCtorBox();
  new FreshReturnCtorBox();
  ({ name: "kimchi" });
  ({ answer: 41 }).answer;
  ({ label: "index" })["label"];
  ({ maybe: "opt-member" })?.maybe;
  ({ maybe: "opt-index" })?.["maybe"];
  "soup".length;
  [1, 2, 3];
  `+"`soup${1}`"+`;
  "left" + "right";
  ~1;
  1n & 3n;
  1n === 1n;
  ("cmp-left" + "x") === ("cmp-right" + "y");
  !("not-left" + "right");
  ("and-left" + "x") && ("and-right" + "y");
  ("or-left" + "x") || ("or-right" + "y");
  typeof ("type" + "of");
  freshBox() instanceof FreshBox;
  ("ok" is "ok" | "error");
  ([1, "ok"] is [number, string]);
  ({ kind: "ok", value: 3 } is { kind: "ok", value: number } | { kind: "error", message: string });
  true ? ({ kind: "conditional" }) : ({ kind: "fallback" });
  null ?? ({ kind: "nullish" });
  (({ kind: "comma-left" }), ({ kind: "comma-right" }));
  freshInvokeObject.bind(null);
  freshInvokeObject.call(null);
  freshInvokeObject.apply(null, []);
  [1, 2].forEach((x) => 0);
  [1, 2].map((x) => x + 1);
  [1, 2].filter((x) => x > 0);
  [1, 2].find((x) => false);
  [1, 2].forEach(boundOffset.bind(null, 1));
  [1, 2].forEach(boundOffset.bind(null, 20));
  [1, 2].forEach(largeOffset);
  [1, 2].map(boundOffset.bind(null, 1));
  [1, 2].map(boundOffset.bind(null, 20));
  [1, 2].filter(largeOffset);
  [1, 2].filter(boundGreaterThan.bind(null, 0));
  [1, 2].filter(boundOffset.bind(null, 20));
  [1, 2].find(largeOffset);
  [1, 2].find(boundEquals.bind(null, 9));
  [1, 2].find(boundOffset.bind(null, 20));
  [1, 2].forEach(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 10, 10));
  [1, 2].filter(boundBetween.bind(null, 0, 3));
  [1, 2].find(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleEquals.bind(null, 4, 5));
  [1, 2].forEach(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].forEach(boundPairSum.bind(null, 10, 10));
  [1, 2].map(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].map(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundWindow.bind(null, 0, 1, 3));
  [1, 2].filter(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].find(boundQuadEquals.bind(null, 3, 4, 5));
  [1, 2].forEach(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].forEach(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].forEach(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].map(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].filter(boundOuterWindow.bind(null, 0, 1, 4, 3));
  [1, 2].filter(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].find(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].forEach(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].map(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].filter(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintEquals.bind(null, 30, 30, 30, 30, 30));
  [1, 2].find(boundQuintEquals.bind(null, 3, 4, 5, 6));
  [1, 2].forEach(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextEquals.bind(null, 40, 40, 40, 40, 40, 40));
  [1, 2].forEach(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptEquals.bind(null, 50, 50, 50, 50, 50, 50, 50));
  [1, 2].forEach(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctEquals.bind(null, 60, 60, 60, 60, 60, 60, 60, 60));
  [1, 2].forEach(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetEquals.bind(null, 70, 70, 70, 70, 70, 70, 70, 70, 70));
  [1, 2].forEach(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetEquals.bind(null, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80));
  [1, 2].forEach(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecEquals.bind(null, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90));
  [1, 2].forEach(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecEquals.bind(null, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100));
}

function main(args) {
  resetCleanupLog();
  {
    const scoped = makeProbe("block");
    console.log("during-block:" + cleanupLog());
  }
  console.log("after-block:" + cleanupLog());
  resetCleanupLog();
  {
    const closed = makeProbe("manual-close");
    console.log("manual-close-call:" + closeProbe(closed) + ":" + cleanupLog());
  }
  console.log("after-manual-close:" + cleanupLog());
  resetCleanupLog();
  console.log("before-function-var:" + cleanupLog());
  functionScopedCleanup();
  console.log("after-function-var:" + cleanupLog());
  resetCleanupLog();
  loopVarFunctionScopedCleanup();
  console.log("after-loop-var-function:" + cleanupLog());
  discardedFreshTemporaries();
  numericSlowPathMap();
  resetCleanupLog();
  console.log("before-return:" + cleanupLog());
  innerReturn();
  console.log("after-return:" + cleanupLog());
  resetCleanupLog();
  try {
    const thrown = makeProbe("throw");
    throw "boom";
  } catch (err) {
    console.log("after-throw:" + cleanupLog() + ":" + err);
  }
  resetCleanupLog();
  for (var i = 0; i < 4; i = i + 1) {
    const loopScoped = makeProbe("continue" + i);
    if (i == 1) {
      continue;
    }
    if (i == 2) {
      break;
    }
  }
  console.log("after-loop:" + cleanupLog());
  resetCleanupLog();
  for (var j = 0; j < 1; j = j + 1) {
    const outerBreak = makeProbe("outer-break");
    {
      const innerBreak = makeProbe("inner-break");
      break;
    }
  }
  console.log("after-complex-break:" + cleanupLog());
  resetCleanupLog();
  for (var k = 0; k < 1; k = k + 1) {
    const outerContinue = makeProbe("outer-continue");
    {
      const innerContinue = makeProbe("inner-continue");
      continue;
    }
  }
  console.log("after-complex-continue:" + cleanupLog());
  resetCleanupLog();
  for (var m = 0; m < 1; m = m + 1) {
    const outerThrow = makeProbe("outer-throw");
    try {
      const innerThrow = makeProbe("inner-throw");
      throw "nested";
    } catch (err) {
      console.log("during-complex-throw:" + cleanupLog() + ":" + err);
    }
  }
  console.log("after-complex-throw:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "scope-cleanup-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled scope cleanup program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"during-block:",
		"after-block:block;",
		"manual-close-call:true:manual-close;",
		"after-manual-close:manual-close;",
		"before-function-var:",
		"during-function-var:",
		"after-function-var:function-var;",
		"after-loop-var-function:loop-var0;loop-var1;loop-var2;",
		"slow-numeric-map:14,15",
		"before-return:",
		"after-return:return;",
		"after-throw:throw;:boom",
		"after-loop:continue0;continue1;continue2;",
		"after-complex-break:inner-break;outer-break;",
		"after-complex-continue:inner-continue;outer-continue;",
		"during-complex-throw:inner-throw;:nested",
		"after-complex-throw:inner-throw;outer-throw;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected scope cleanup output to contain %q, got: %s", want, text)
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

func TestBuildExecutablePreservesEscapingLocalsAcrossReturnsContainersAndClosures(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
var globalBox = undefined;

function makeReturnedObject() {
  var local = { label: "object-ok", nested: { value: 41 } };
  return local;
}

function makeReturnedArray() {
  var local = [3, 4, 5];
  return local;
}

function makeStoredObject() {
  var local = { value: "stored-object" };
  var holder = {};
  holder.item = local;
  return holder;
}

function makeStoredArray() {
  var local = { value: "stored-array" };
  var holder = [];
  holder[0] = local;
  return holder;
}

function makeClosure() {
  var local = { value: 9 };
  return () => local.value + 1;
}

function seedGlobal() {
  var local = { value: "global-object" };
  globalBox = local;
}

function main(args) {
  var returnedObject = makeReturnedObject();
  var returnedArray = makeReturnedArray();
  var storedObject = makeStoredObject();
  var storedArray = makeStoredArray();
  var closure = makeClosure();
  seedGlobal();

  console.log("escape-object:" + returnedObject.label + ":" + returnedObject.nested.value);
  console.log("escape-array:" + returnedArray.length + ":" + returnedArray[0] + ":" + returnedArray[2]);
  console.log("escape-stored-object:" + storedObject.item.value);
  console.log("escape-stored-array:" + storedArray[0].value);
  console.log("escape-closure:" + closure());
  console.log("escape-global:" + globalBox.value);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "escaping-locals-native")
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
		"escape-object:object-ok:41",
		"escape-array:3:3:5",
		"escape-stored-object:stored-object",
		"escape-stored-array:stored-array",
		"escape-closure:10",
		"escape-global:global-object",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected escaping-locals output to contain %q, got: %s", want, text)
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
  var intsBuffer = ints.buffer;
  var viaBuffer = new Int32Array(intsBuffer);
  viaBuffer[1] = 1077936128;
  console.log("buffer-alias:" + intsBuffer.byteLength + ":" + ints[1] + ":" + floats[1]);

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
		"buffer-alias:8:1.07794e+09:3",
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

func TestBuildExecutableSupportsFunctionBindCallApplyMethods(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function greet(name) {
  return "hi:" + name;
}

function join(prefix, value) {
  return prefix + ":" + value;
}

function main(args) {
  const bound = greet.bind(null, "kimchi");
  console.log("bind:" + bound());
  console.log("call:" + greet.call(null, "mandu"));
  console.log("apply:" + greet.apply(null, ["bibim"]));
  const inc = join.bind(null, "left");
  console.log("partial:" + inc("right"));
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "function-bind-call-apply-native")
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
		"bind:hi:kimchi",
		"call:hi:mandu",
		"apply:hi:bibim",
		"partial:left:right",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected function bind/call/apply output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsHighArityBoundArrayMapAndFind(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
function boundTridecSum(a, b, c, d, e, f, g, h, i, j, k, l, m, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k + l + m;
}

function boundTridecEquals(a, b, c, d, e, f, g, h, i, j, k, l, expected, x) {
  return x == expected;
}

function main(args) {
  const items = [1, 2];
  const mapped = items.map(boundTridecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1));
  const found = items.find(boundTridecEquals.bind(null, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2));
  console.log("mapped:" + mapped[0] + "," + mapped[1]);
  console.log("found:" + found);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "high-arity-array-callbacks-native")
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
		"mapped:14,15",
		"found:2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected high-arity bound array callback output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableConstructorCanReturnAlternateObject(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	result, err := compiler.Compile(`
class Box {
  constructor() {
    return { aliased: "yes" };
  }
}

function main(args) {
  const value = new Box();
  console.log("ctor-alias:" + value.aliased);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "constructor-alias-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "ctor-alias:yes") {
		t.Fatalf("expected constructor alternate return output, got: %s", string(out))
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
  var beforeMemory = process.memoryInfo();
  var objectValue = { answer: "ok" };
  var arrayValue = [objectValue, "value"];
  var functionValue = function() { return arrayValue; };
  var memory = process.memoryInfo();
  var user = process.userInfo();
  console.log("uptime:" + (uptime >= 0));
  console.log("hrtime:" + (start > 0) + ":" + (finish > start));
  console.log("cpu:" + (cpu.count >= 1) + ":" + (typeof cpu.arch));
  console.log("memory:" + (memory.total >= 0) + ":" + (memory.available >= 0));
  console.log("jayess-memory:" +
    (typeof memory.jayess.boxedValues) + ":" +
    (memory.jayess.boxedValues > beforeMemory.jayess.boxedValues) + ":" +
    (memory.jayess.objects > beforeMemory.jayess.objects) + ":" +
    (memory.jayess.objectEntries > beforeMemory.jayess.objectEntries) + ":" +
    (memory.jayess.arrays > beforeMemory.jayess.arrays) + ":" +
    (memory.jayess.arraySlots > beforeMemory.jayess.arraySlots) + ":" +
    (memory.jayess.functions > beforeMemory.jayess.functions) + ":" +
    (memory.jayess.strings >= beforeMemory.jayess.strings) + ":" +
    (memory.jayess.nativeHandleWrappers >= 0) + ":" +
    (typeof functionValue));
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
		"jayess-memory:number:true:true:true:true:true:true:true:true:function",
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
	perms := strings.TrimPrefix(text, "perms:")
	if perms == "" {
		t.Fatalf("expected non-empty permissions text, got: %s", text)
	}
	if runtime.GOOS == "windows" {
		if perms != "rwx" {
			t.Fatalf("expected normalized windows permissions to be %q, got %q", "rwx", perms)
		}
	} else {
		if len(perms) != 9 {
			t.Fatalf("expected POSIX-style permission text length 9, got %q", perms)
		}
		for i, ch := range perms {
			valid := ch == 'r' || ch == 'w' || ch == 'x' || ch == '-'
			if !valid {
				t.Fatalf("unexpected permission character %q at index %d in %q", ch, i, perms)
			}
		}
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
  var viewAlias = new Uint8Array(view.buffer);
  viewAlias[8] = 9;
  console.log("data-view-bytes:" + viewBytes.toString("hex"));
  console.log("data-view-read:" + view.getUint8(0) + ":" + view.getUint16(1, false) + ":" + view.getUint16(3, true) + ":" + view.getUint32(5, false) + ":" + viewAlias[8] + ":" + view.buffer.byteLength);
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
	for _, want := range []string{"true", "kim", "chi", "null", "finish-count:3", "finish-event:true", "finished", "finished-two", "finished-once", "finished-late", "finished-once-late", "data:kimchi", "ended", "once-data:kimchi", "once-end-count:1", "once-ended", "once-end-after:0", "read-error:", "read-error-two:", "read-error-once:", "error-count:2", "error-event:true", "bytes-write:true", "bytes-read:4:65:0:255:66", "bytes-slice:2:0:255", "bytes-includes:true:false", "bytes-text:AB", "bytes-end:null", "from-string:6:106:jayess:jayess", "hex-bytes:3:65:66:255:4142ff", "utf8-bytes:kimchi", "base64-bytes:6:kimchi:a2ltY2hp", "base64-pad:A:AB", "bytes-concat:6:kimchi", "bytes-concat-method:jayess", "bytes-equals-same", "bytes-equals-diff", "bytes-equals-static", "bytes-compare-equal:0", "bytes-compare-less:-1", "bytes-compare-greater:1", "bytes-index-of-byte:3:-1", "bytes-index-of-seq:3:-1", "bytes-prefix-suffix:true:true", "bytes-prefix-suffix-byte:true:true", "bytes-set:6:006b696d4142", "bytes-copy-within:ababcf", "bytes-copy-overlap:aabcde", "data-view-bytes:ff1234341200010209", "data-view-read:255:4660:4660:66051:9:9", "data-view-signed-bytes:fffffefdfffe1dc0", "data-view-signed-read:-1:-2:-3:-123456", "data-view-float-bytes:3fc0000000000000000004c0", "data-view-float-read:1.5:-2.5", "write-error:"} {
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

func TestBuildExecutableSupportsManualSDLAudioBindFiles(t *testing.T) {
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
	nativeDir := filepath.Join(workdir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "SDL", "include"),
		filepath.Join(nativeDir, "sdl_include"),
	)
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio.c"), []byte(`#include "jayess_runtime.h"
#include <SDL3/SDL.h>
#include <string.h>
#include <stdlib.h>

typedef struct jayess_sdl_audio_device {
    SDL_AudioDeviceID id;
} jayess_sdl_audio_device;

static void jayess_sdl_audio_device_finalize(void *handle) {
    jayess_sdl_audio_device *device = (jayess_sdl_audio_device *)handle;
    if (device == NULL) {
        return;
    }
    if (device->id != 0) {
        SDL_CloseAudioDevice(device->id);
        device->id = 0;
    }
    free(device);
}

static const char *jayess_sdl_audio_format_text(SDL_AudioFormat format) {
    if (format == SDL_AUDIO_F32) {
        return "f32";
    }
    if (format == SDL_AUDIO_S16) {
        return "s16";
    }
    return "unknown";
}

jayess_value *jayess_sdl_audio_init(void) {
    return jayess_value_from_bool(SDL_InitSubSystem(SDL_INIT_AUDIO));
}

jayess_value *jayess_sdl_audio_quit(void) {
    SDL_QuitSubSystem(SDL_INIT_AUDIO);
    return jayess_value_from_bool(1);
}

jayess_value *jayess_sdl_audio_driver_count(void) {
    return jayess_value_from_number((double)SDL_GetNumAudioDrivers());
}

jayess_value *jayess_sdl_audio_driver_name(jayess_value *index_value) {
    int index = (int)jayess_value_to_number(index_value);
    const char *name = SDL_GetAudioDriver(index);
    if (name == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(name);
}

jayess_value *jayess_sdl_audio_current_driver(void) {
    const char *name = SDL_GetCurrentAudioDriver();
    if (name == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(name);
}

jayess_value *jayess_sdl_audio_open_default_playback(jayess_value *rate_value, jayess_value *channels_value) {
    SDL_AudioSpec spec;
    SDL_AudioDeviceID id = 0;
    jayess_sdl_audio_device *device = NULL;
    memset(&spec, 0, sizeof(spec));
    spec.format = SDL_AUDIO_F32;
    spec.freq = (int)jayess_value_to_number(rate_value);
    spec.channels = (int)jayess_value_to_number(channels_value);
    if (spec.freq <= 0 || spec.channels <= 0) {
        jayess_throw_named_error("SDLAudioError", "SDL playback rate and channels must be positive");
        return jayess_value_undefined();
    }
    id = SDL_OpenAudioDevice(SDL_AUDIO_DEVICE_DEFAULT_PLAYBACK, &spec);
    if (id == 0) {
        return jayess_value_undefined();
    }
    device = (jayess_sdl_audio_device *)calloc(1, sizeof(jayess_sdl_audio_device));
    if (device == NULL) {
        SDL_CloseAudioDevice(id);
        return jayess_value_undefined();
    }
    device->id = id;
    return jayess_value_from_managed_native_handle("SDLAudioDevice", device, jayess_sdl_audio_device_finalize);
}

jayess_value *jayess_sdl_audio_describe_playback(jayess_value *device_value) {
    jayess_sdl_audio_device *device = (jayess_sdl_audio_device *)jayess_expect_native_handle(device_value, "SDLAudioDevice", "jayess_sdl_audio_describe_playback");
    SDL_AudioSpec spec;
    int sample_frames = 0;
    jayess_object *object = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    memset(&spec, 0, sizeof(spec));
    if (!SDL_GetAudioDeviceFormat(device->id, &spec, &sample_frames)) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "format", jayess_value_from_string(jayess_sdl_audio_format_text(spec.format)));
    jayess_object_set_value(object, "channels", jayess_value_from_number((double)spec.channels));
    jayess_object_set_value(object, "freq", jayess_value_from_number((double)spec.freq));
    jayess_object_set_value(object, "sampleFrames", jayess_value_from_number((double)sample_frames));
    return jayess_value_from_object(object);
}

jayess_value *jayess_sdl_audio_close_playback(jayess_value *device_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(device_value));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio_stub.c"), []byte(`#include <SDL3/SDL.h>

static const char *jayess_sdl_current_audio_driver = NULL;
static SDL_AudioDeviceID jayess_sdl_open_audio_device = 0;
static SDL_AudioSpec jayess_sdl_last_spec;

bool SDL_InitSubSystem(SDL_InitFlags flags) {
    if ((flags & SDL_INIT_AUDIO) == 0) {
        return false;
    }
    jayess_sdl_current_audio_driver = "stub-sdl-audio";
    return true;
}

void SDL_QuitSubSystem(SDL_InitFlags flags) {
    (void)flags;
    jayess_sdl_current_audio_driver = NULL;
    jayess_sdl_open_audio_device = 0;
}

int SDL_GetNumAudioDrivers(void) {
    return 1;
}

const char *SDL_GetAudioDriver(int index) {
    return index == 0 ? "stub-sdl-audio" : NULL;
}

const char *SDL_GetCurrentAudioDriver(void) {
    return jayess_sdl_current_audio_driver;
}

SDL_AudioDeviceID SDL_OpenAudioDevice(SDL_AudioDeviceID devid, const SDL_AudioSpec *spec) {
    if (jayess_sdl_current_audio_driver == NULL || spec == NULL || devid != SDL_AUDIO_DEVICE_DEFAULT_PLAYBACK) {
        return 0;
    }
    jayess_sdl_last_spec = *spec;
    jayess_sdl_open_audio_device = 7;
    return jayess_sdl_open_audio_device;
}

bool SDL_GetAudioDeviceFormat(SDL_AudioDeviceID devid, SDL_AudioSpec *spec, int *sample_frames) {
    if (devid != jayess_sdl_open_audio_device || spec == NULL || sample_frames == NULL) {
        return false;
    }
    *spec = jayess_sdl_last_spec;
    *sample_frames = 256;
    return true;
}

void SDL_CloseAudioDevice(SDL_AudioDeviceID devid) {
    if (devid == jayess_sdl_open_audio_device) {
        jayess_sdl_open_audio_device = 0;
    }
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio.bind.js"), []byte(`const f = () => {};
export const initAudio = f;
export const quitAudio = f;
export const getNumAudioDrivers = f;
export const getAudioDriver = f;
export const getCurrentAudioDriver = f;
export const openDefaultPlayback = f;
export const describePlayback = f;
export const closePlayback = f;

export default {
  sources: ["./sdl_audio.c", "./sdl_audio_stub.c"],
  includeDirs: ["./sdl_include"],
  cflags: [],
  ldflags: [],
  exports: {
    initAudio: { symbol: "jayess_sdl_audio_init", type: "function" },
    quitAudio: { symbol: "jayess_sdl_audio_quit", type: "function" },
    getNumAudioDrivers: { symbol: "jayess_sdl_audio_driver_count", type: "function" },
    getAudioDriver: { symbol: "jayess_sdl_audio_driver_name", type: "function" },
    getCurrentAudioDriver: { symbol: "jayess_sdl_audio_current_driver", type: "function" },
    openDefaultPlayback: { symbol: "jayess_sdl_audio_open_default_playback", type: "function" },
    describePlayback: { symbol: "jayess_sdl_audio_describe_playback", type: "function" },
    closePlayback: { symbol: "jayess_sdl_audio_close_playback", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { initAudio, quitAudio, getNumAudioDrivers, getAudioDriver, getCurrentAudioDriver, openDefaultPlayback, describePlayback, closePlayback } from "./native/sdl_audio.bind.js";

function main(args) {
  console.log("sdl-audio-init:" + initAudio());
  console.log("sdl-audio-driver-count:" + getNumAudioDrivers());
  console.log("sdl-audio-driver-0:" + getAudioDriver(0));
  console.log("sdl-audio-current-driver:" + getCurrentAudioDriver());
  var device = openDefaultPlayback(48000, 2);
  console.log("sdl-audio-open:" + (device != undefined));
  var info = describePlayback(device);
  console.log("sdl-audio-format:" + info.format);
  console.log("sdl-audio-freq:" + info.freq);
  console.log("sdl-audio-channels:" + info.channels);
  console.log("sdl-audio-sample-frames:" + info.sampleFrames);
  console.log("sdl-audio-close:" + closePlayback(device));
  try {
    describePlayback(device);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("sdl-audio-quit:" + quitAudio());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "manual-sdl-audio-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled SDL audio bind program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"sdl-audio-init:true",
		"sdl-audio-driver-count:1",
		"sdl-audio-driver-0:stub-sdl-audio",
		"sdl-audio-current-driver:stub-sdl-audio",
		"sdl-audio-open:true",
		"sdl-audio-format:f32",
		"sdl-audio-freq:48000",
		"sdl-audio-channels:2",
		"sdl-audio-sample-frames:256",
		"sdl-audio-close:true",
		"TypeError:jayess_sdl_audio_describe_playback expects a SDLAudioDevice native handle",
		"sdl-audio-quit:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected SDL audio bind output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsManualOpenALBindFiles(t *testing.T) {
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
	includeDir := filepath.Join(nativeDir, "include", "AL")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "al.h"), []byte(`#pragma once
typedef char ALchar;
typedef int ALenum;
#define AL_VENDOR 0xB001
const ALchar *alGetString(ALenum param);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "alc.h"), []byte(`#pragma once
typedef char ALCchar;
typedef int ALCenum;
typedef int ALCint;
typedef struct ALCdevice_struct ALCdevice;
typedef struct ALCcontext_struct ALCcontext;
#define ALC_DEFAULT_DEVICE_SPECIFIER 0x1004
#define ALC_FREQUENCY 0x1007
#define ALC_MONO_SOURCES 0x1010
#define ALC_STEREO_SOURCES 0x1011
ALCdevice *alcOpenDevice(const ALCchar *devicename);
int alcCloseDevice(ALCdevice *device);
const ALCchar *alcGetString(ALCdevice *device, ALCenum param);
ALCcontext *alcCreateContext(ALCdevice *device, const ALCint *attrlist);
void alcDestroyContext(ALCcontext *context);
int alcMakeContextCurrent(ALCcontext *context);
void alcGetIntegerv(ALCdevice *device, ALCenum param, ALCint size, ALCint *values);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal.c"), []byte(`#include "jayess_runtime.h"
#include <AL/al.h>
#include <AL/alc.h>
#include <stdlib.h>

typedef struct jayess_openal_device {
    ALCdevice *device;
} jayess_openal_device;

typedef struct jayess_openal_context {
    ALCcontext *context;
    ALCdevice *device;
} jayess_openal_context;

static void jayess_openal_device_finalize(void *handle) {
    jayess_openal_device *device = (jayess_openal_device *)handle;
    if (device == NULL) {
        return;
    }
    if (device->device != NULL) {
        alcCloseDevice(device->device);
        device->device = NULL;
    }
    free(device);
}

static void jayess_openal_context_finalize(void *handle) {
    jayess_openal_context *context = (jayess_openal_context *)handle;
    if (context == NULL) {
        return;
    }
    if (context->context != NULL) {
        alcMakeContextCurrent(NULL);
        alcDestroyContext(context->context);
        context->context = NULL;
    }
    context->device = NULL;
    free(context);
}

jayess_value *jayess_openal_default_device_name(void) {
    const ALCchar *name = alcGetString(NULL, ALC_DEFAULT_DEVICE_SPECIFIER);
    if (name == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(name);
}

jayess_value *jayess_openal_open_default_device(void) {
    jayess_openal_device *device = NULL;
    ALCdevice *opened = alcOpenDevice(NULL);
    if (opened == NULL) {
        return jayess_value_undefined();
    }
    device = (jayess_openal_device *)calloc(1, sizeof(jayess_openal_device));
    if (device == NULL) {
        alcCloseDevice(opened);
        return jayess_value_undefined();
    }
    device->device = opened;
    return jayess_value_from_managed_native_handle("OpenALDevice", device, jayess_openal_device_finalize);
}

jayess_value *jayess_openal_create_context(jayess_value *device_value) {
    jayess_openal_device *device = (jayess_openal_device *)jayess_expect_native_handle(device_value, "OpenALDevice", "jayess_openal_create_context");
    jayess_openal_context *context = NULL;
    ALCcontext *created = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    created = alcCreateContext(device->device, NULL);
    if (created == NULL) {
        return jayess_value_undefined();
    }
    context = (jayess_openal_context *)calloc(1, sizeof(jayess_openal_context));
    if (context == NULL) {
        alcDestroyContext(created);
        return jayess_value_undefined();
    }
    context->context = created;
    context->device = device->device;
    return jayess_value_from_managed_native_handle("OpenALContext", context, jayess_openal_context_finalize);
}

jayess_value *jayess_openal_make_context_current(jayess_value *context_value) {
    jayess_openal_context *context = (jayess_openal_context *)jayess_expect_native_handle(context_value, "OpenALContext", "jayess_openal_make_context_current");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(alcMakeContextCurrent(context->context));
}

jayess_value *jayess_openal_vendor_name(void) {
    const ALchar *vendor = alGetString(AL_VENDOR);
    if (vendor == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(vendor);
}

jayess_value *jayess_openal_describe_context(jayess_value *context_value) {
    jayess_openal_context *context = (jayess_openal_context *)jayess_expect_native_handle(context_value, "OpenALContext", "jayess_openal_describe_context");
    ALCint frequency = 0;
    ALCint mono_sources = 0;
    ALCint stereo_sources = 0;
    jayess_object *object = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    alcGetIntegerv(context->device, ALC_FREQUENCY, 1, &frequency);
    alcGetIntegerv(context->device, ALC_MONO_SOURCES, 1, &mono_sources);
    alcGetIntegerv(context->device, ALC_STEREO_SOURCES, 1, &stereo_sources);
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "frequency", jayess_value_from_number((double)frequency));
    jayess_object_set_value(object, "monoSources", jayess_value_from_number((double)mono_sources));
    jayess_object_set_value(object, "stereoSources", jayess_value_from_number((double)stereo_sources));
    return jayess_value_from_object(object);
}

jayess_value *jayess_openal_close_context(jayess_value *context_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(context_value));
}

jayess_value *jayess_openal_close_device(jayess_value *device_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(device_value));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal_stub.c"), []byte(`#include <AL/al.h>
#include <AL/alc.h>
#include <stdlib.h>

struct ALCdevice_struct {
    int open;
};

struct ALCcontext_struct {
    ALCdevice *device;
};

static ALCcontext *jayess_openal_current_context = NULL;

ALCdevice *alcOpenDevice(const ALCchar *devicename) {
    ALCdevice *device = NULL;
    if (devicename != NULL && devicename[0] != '\0') {
        return NULL;
    }
    device = (ALCdevice *)calloc(1, sizeof(ALCdevice));
    if (device != NULL) {
        device->open = 1;
    }
    return device;
}

int alcCloseDevice(ALCdevice *device) {
    if (device == NULL || !device->open) {
        return 0;
    }
    if (jayess_openal_current_context != NULL && jayess_openal_current_context->device == device) {
        jayess_openal_current_context = NULL;
    }
    device->open = 0;
    free(device);
    return 1;
}

const ALCchar *alcGetString(ALCdevice *device, ALCenum param) {
    (void)device;
    return param == ALC_DEFAULT_DEVICE_SPECIFIER ? "stub-openal-device" : NULL;
}

ALCcontext *alcCreateContext(ALCdevice *device, const ALCint *attrlist) {
    ALCcontext *context = NULL;
    (void)attrlist;
    if (device == NULL || !device->open) {
        return NULL;
    }
    context = (ALCcontext *)calloc(1, sizeof(ALCcontext));
    if (context != NULL) {
        context->device = device;
    }
    return context;
}

void alcDestroyContext(ALCcontext *context) {
    if (jayess_openal_current_context == context) {
        jayess_openal_current_context = NULL;
    }
    free(context);
}

int alcMakeContextCurrent(ALCcontext *context) {
    jayess_openal_current_context = context;
    return 1;
}

void alcGetIntegerv(ALCdevice *device, ALCenum param, ALCint size, ALCint *values) {
    (void)size;
    if (device == NULL || values == NULL || !device->open) {
        return;
    }
    if (param == ALC_FREQUENCY) {
        *values = 48000;
    } else if (param == ALC_MONO_SOURCES) {
        *values = 32;
    } else if (param == ALC_STEREO_SOURCES) {
        *values = 16;
    }
}

const ALchar *alGetString(ALenum param) {
    if (jayess_openal_current_context == NULL) {
        return NULL;
    }
    return param == AL_VENDOR ? "stub-openal" : NULL;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal.bind.js"), []byte(`const f = () => {};
export const getDefaultDeviceName = f;
export const openDefaultDevice = f;
export const createContext = f;
export const makeContextCurrent = f;
export const getVendorName = f;
export const describeContext = f;
export const closeContext = f;
export const closeDevice = f;

export default {
  sources: ["./openal.c", "./openal_stub.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    getDefaultDeviceName: { symbol: "jayess_openal_default_device_name", type: "function" },
    openDefaultDevice: { symbol: "jayess_openal_open_default_device", type: "function" },
    createContext: { symbol: "jayess_openal_create_context", type: "function" },
    makeContextCurrent: { symbol: "jayess_openal_make_context_current", type: "function" },
    getVendorName: { symbol: "jayess_openal_vendor_name", type: "function" },
    describeContext: { symbol: "jayess_openal_describe_context", type: "function" },
    closeContext: { symbol: "jayess_openal_close_context", type: "function" },
    closeDevice: { symbol: "jayess_openal_close_device", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { getDefaultDeviceName, openDefaultDevice, createContext, makeContextCurrent, getVendorName, describeContext, closeContext, closeDevice } from "./native/openal.bind.js";

function main(args) {
  console.log("openal-default-device:" + getDefaultDeviceName());
  var device = openDefaultDevice();
  console.log("openal-open-device:" + (device != undefined));
  var context = createContext(device);
  console.log("openal-create-context:" + (context != undefined));
  console.log("openal-make-current:" + makeContextCurrent(context));
  console.log("openal-vendor:" + getVendorName());
  var info = describeContext(context);
  console.log("openal-frequency:" + info.frequency);
  console.log("openal-mono-sources:" + info.monoSources);
  console.log("openal-stereo-sources:" + info.stereoSources);
  console.log("openal-close-context:" + closeContext(context));
  try {
    describeContext(context);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("openal-close-device:" + closeDevice(device));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "manual-openal-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled OpenAL bind program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"openal-default-device:stub-openal-device",
		"openal-open-device:true",
		"openal-create-context:true",
		"openal-make-current:true",
		"openal-vendor:stub-openal",
		"openal-frequency:48000",
		"openal-mono-sources:32",
		"openal-stereo-sources:16",
		"openal-close-context:true",
		"TypeError:jayess_openal_describe_context expects a OpenALContext native handle",
		"openal-close-device:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected OpenAL bind output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsManualMiniaudioBindFiles(t *testing.T) {
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
	nativeDir := filepath.Join(workdir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, name := range []string{"miniaudio.h", "miniaudio.c"} {
		data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "miniaudio", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(nativeDir, name), data, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio_wrapper.c"), []byte(`#include "jayess_runtime.h"
#include "miniaudio.h"
#include <stdlib.h>

typedef struct jayess_miniaudio_context {
    ma_context context;
} jayess_miniaudio_context;

typedef struct jayess_miniaudio_device {
    ma_device device;
} jayess_miniaudio_device;

static void jayess_miniaudio_data_callback(ma_device *pDevice, void *pOutput, const void *pInput, ma_uint32 frameCount) {
    (void)pDevice;
    (void)pInput;
    ma_silence_pcm_frames(pOutput, frameCount, ma_format_f32, 2);
}

static void jayess_miniaudio_context_finalize(void *handle) {
    jayess_miniaudio_context *context = (jayess_miniaudio_context *)handle;
    if (context == NULL) {
        return;
    }
    ma_context_uninit(&context->context);
    free(context);
}

static void jayess_miniaudio_device_finalize(void *handle) {
    jayess_miniaudio_device *device = (jayess_miniaudio_device *)handle;
    if (device == NULL) {
        return;
    }
    ma_device_uninit(&device->device);
    free(device);
}

jayess_value *jayess_miniaudio_create_context(void) {
    ma_backend backends[] = { ma_backend_null };
    jayess_miniaudio_context *context = (jayess_miniaudio_context *)calloc(1, sizeof(jayess_miniaudio_context));
    if (context == NULL) {
        return jayess_value_undefined();
    }
    if (ma_context_init(backends, 1, NULL, &context->context) != MA_SUCCESS) {
        free(context);
        return jayess_value_undefined();
    }
    return jayess_value_from_managed_native_handle("MiniAudioContext", context, jayess_miniaudio_context_finalize);
}

jayess_value *jayess_miniaudio_get_backend_name(jayess_value *context_value) {
    jayess_miniaudio_context *context = (jayess_miniaudio_context *)jayess_expect_native_handle(context_value, "MiniAudioContext", "jayess_miniaudio_get_backend_name");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(ma_get_backend_name(context->context.backend));
}

jayess_value *jayess_miniaudio_describe_playback_devices(jayess_value *context_value) {
    jayess_miniaudio_context *context = (jayess_miniaudio_context *)jayess_expect_native_handle(context_value, "MiniAudioContext", "jayess_miniaudio_describe_playback_devices");
    ma_device_info *playbackInfos = NULL;
    ma_uint32 playbackCount = 0;
    jayess_array *array = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    if (ma_context_get_devices(&context->context, &playbackInfos, &playbackCount, NULL, NULL) != MA_SUCCESS) {
        return jayess_value_undefined();
    }
    array = jayess_array_new();
    if (array == NULL) {
        return jayess_value_undefined();
    }
    for (ma_uint32 i = 0; i < playbackCount; ++i) {
        jayess_object *entry = jayess_object_new();
        if (entry == NULL) {
            return jayess_value_undefined();
        }
        jayess_object_set_value(entry, "name", jayess_value_from_string(playbackInfos[i].name));
        jayess_object_set_value(entry, "isDefault", jayess_value_from_bool(playbackInfos[i].isDefault));
        jayess_array_push_value(array, jayess_value_from_object(entry));
    }
    return jayess_value_from_array(array);
}

jayess_value *jayess_miniaudio_open_playback_device(jayess_value *context_value) {
    jayess_miniaudio_context *context = (jayess_miniaudio_context *)jayess_expect_native_handle(context_value, "MiniAudioContext", "jayess_miniaudio_open_playback_device");
    jayess_miniaudio_device *device = NULL;
    ma_device_config config;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    device = (jayess_miniaudio_device *)calloc(1, sizeof(jayess_miniaudio_device));
    if (device == NULL) {
        return jayess_value_undefined();
    }
    config = ma_device_config_init(ma_device_type_playback);
    config.playback.format = ma_format_f32;
    config.playback.channels = 2;
    config.sampleRate = 48000;
    config.dataCallback = jayess_miniaudio_data_callback;
    if (ma_device_init(&context->context, &config, &device->device) != MA_SUCCESS) {
        free(device);
        return jayess_value_undefined();
    }
    return jayess_value_from_managed_native_handle("MiniAudioDevice", device, jayess_miniaudio_device_finalize);
}

jayess_value *jayess_miniaudio_describe_playback_device(jayess_value *device_value) {
    jayess_miniaudio_device *device = (jayess_miniaudio_device *)jayess_expect_native_handle(device_value, "MiniAudioDevice", "jayess_miniaudio_describe_playback_device");
    jayess_object *object = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "name", jayess_value_from_string(device->device.playback.name));
    jayess_object_set_value(object, "format", jayess_value_from_string(device->device.playback.format == ma_format_f32 ? "f32" : "unknown"));
    jayess_object_set_value(object, "channels", jayess_value_from_number((double)device->device.playback.channels));
    jayess_object_set_value(object, "sampleRate", jayess_value_from_number((double)device->device.sampleRate));
    return jayess_value_from_object(object);
}

jayess_value *jayess_miniaudio_start_playback_device(jayess_value *device_value) {
    jayess_miniaudio_device *device = (jayess_miniaudio_device *)jayess_expect_native_handle(device_value, "MiniAudioDevice", "jayess_miniaudio_start_playback_device");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(ma_device_start(&device->device) == MA_SUCCESS);
}

jayess_value *jayess_miniaudio_stop_playback_device(jayess_value *device_value) {
    jayess_miniaudio_device *device = (jayess_miniaudio_device *)jayess_expect_native_handle(device_value, "MiniAudioDevice", "jayess_miniaudio_stop_playback_device");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(ma_device_stop(&device->device) == MA_SUCCESS);
}

jayess_value *jayess_miniaudio_close_playback_device(jayess_value *device_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(device_value));
}

jayess_value *jayess_miniaudio_destroy_context(jayess_value *context_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(context_value));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio.bind.js"), []byte(`const f = () => {};
export const createContext = f;
export const getBackendName = f;
export const describePlaybackDevices = f;
export const openPlaybackDevice = f;
export const describePlaybackDevice = f;
export const startPlaybackDevice = f;
export const stopPlaybackDevice = f;
export const closePlaybackDevice = f;
export const destroyContext = f;

export default {
  sources: ["./miniaudio_wrapper.c", "./miniaudio.c"],
  includeDirs: ["."],
  cflags: ["-DMA_ENABLE_ONLY_NULL"],
  ldflags: ["-pthread", "-ldl", "-lm"],
  exports: {
    createContext: { symbol: "jayess_miniaudio_create_context", type: "function" },
    getBackendName: { symbol: "jayess_miniaudio_get_backend_name", type: "function" },
    describePlaybackDevices: { symbol: "jayess_miniaudio_describe_playback_devices", type: "function" },
    openPlaybackDevice: { symbol: "jayess_miniaudio_open_playback_device", type: "function" },
    describePlaybackDevice: { symbol: "jayess_miniaudio_describe_playback_device", type: "function" },
    startPlaybackDevice: { symbol: "jayess_miniaudio_start_playback_device", type: "function" },
    stopPlaybackDevice: { symbol: "jayess_miniaudio_stop_playback_device", type: "function" },
    closePlaybackDevice: { symbol: "jayess_miniaudio_close_playback_device", type: "function" },
    destroyContext: { symbol: "jayess_miniaudio_destroy_context", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { createContext, getBackendName, describePlaybackDevices, openPlaybackDevice, describePlaybackDevice, startPlaybackDevice, stopPlaybackDevice, closePlaybackDevice, destroyContext } from "./native/miniaudio.bind.js";

function main(args) {
  var context = createContext();
  console.log("miniaudio-context:" + (context != undefined));
  console.log("miniaudio-backend:" + getBackendName(context));
  var devices = describePlaybackDevices(context);
  console.log("miniaudio-device-count:" + devices.length);
  console.log("miniaudio-device-0:" + devices[0].name + ":" + devices[0].isDefault);
  var device = openPlaybackDevice(context);
  console.log("miniaudio-open:" + (device != undefined));
  var info = describePlaybackDevice(device);
  console.log("miniaudio-format:" + info.format);
  console.log("miniaudio-channels:" + info.channels);
  console.log("miniaudio-sample-rate:" + info.sampleRate);
  console.log("miniaudio-start:" + startPlaybackDevice(device));
  console.log("miniaudio-stop:" + stopPlaybackDevice(device));
  console.log("miniaudio-close:" + closePlaybackDevice(device));
  try {
    describePlaybackDevice(device);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("miniaudio-destroy:" + destroyContext(context));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "manual-miniaudio-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled miniaudio bind program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"miniaudio-context:true",
		"miniaudio-backend:Null",
		"miniaudio-device-count:1",
		"miniaudio-device-0:NULL Playback Device:true",
		"miniaudio-open:true",
		"miniaudio-format:f32",
		"miniaudio-channels:2",
		"miniaudio-sample-rate:48000",
		"miniaudio-start:true",
		"miniaudio-stop:true",
		"miniaudio-close:true",
		"TypeError:jayess_miniaudio_describe_playback_device expects a MiniAudioDevice native handle",
		"miniaudio-destroy:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected miniaudio bind output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsManualPortAudioBindFiles(t *testing.T) {
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
	nativeDir := filepath.Join(workdir, "native")
	includeDir := filepath.Join(nativeDir, "portaudio_include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "refs", "portaudio", "include", "portaudio.h"))
	if err != nil {
		t.Fatalf("ReadFile(portaudio.h) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "portaudio.h"), data, 0o644); err != nil {
		t.Fatalf("WriteFile(portaudio.h) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio_wrapper.c"), []byte(`#include "jayess_runtime.h"
#include <portaudio.h>
#include <stdlib.h>

typedef struct jayess_portaudio_stream {
    PaStream *stream;
} jayess_portaudio_stream;

static int jayess_portaudio_callback(const void *input, void *output, unsigned long frameCount,
                                     const PaStreamCallbackTimeInfo *timeInfo, PaStreamCallbackFlags statusFlags,
                                     void *userData) {
    (void)input;
    (void)output;
    (void)frameCount;
    (void)timeInfo;
    (void)statusFlags;
    (void)userData;
    return paContinue;
}

static void jayess_portaudio_stream_finalize(void *handle) {
    jayess_portaudio_stream *stream = (jayess_portaudio_stream *)handle;
    if (stream == NULL) {
        return;
    }
    if (stream->stream != NULL) {
        Pa_CloseStream(stream->stream);
        stream->stream = NULL;
    }
    free(stream);
}

static const char *jayess_portaudio_format_name(PaSampleFormat format) {
    if (format == paFloat32) {
        return "f32";
    }
    return "unknown";
}

jayess_value *jayess_portaudio_init(void) {
    return jayess_value_from_bool(Pa_Initialize() == paNoError);
}

jayess_value *jayess_portaudio_terminate(void) {
    return jayess_value_from_bool(Pa_Terminate() == paNoError);
}

jayess_value *jayess_portaudio_version_text(void) {
    const char *text = Pa_GetVersionText();
    if (text == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(text);
}

jayess_value *jayess_portaudio_device_count(void) {
    return jayess_value_from_number((double)Pa_GetDeviceCount());
}

jayess_value *jayess_portaudio_default_output_device(void) {
    PaDeviceIndex index = Pa_GetDefaultOutputDevice();
    const PaDeviceInfo *info = NULL;
    jayess_object *object = NULL;
    if (index < 0) {
        return jayess_value_undefined();
    }
    info = Pa_GetDeviceInfo(index);
    if (info == NULL) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "name", jayess_value_from_string(info->name));
    jayess_object_set_value(object, "maxOutputChannels", jayess_value_from_number((double)info->maxOutputChannels));
    jayess_object_set_value(object, "defaultSampleRate", jayess_value_from_number(info->defaultSampleRate));
    return jayess_value_from_object(object);
}

jayess_value *jayess_portaudio_open_default_playback(jayess_value *sample_rate_value, jayess_value *frames_per_buffer_value) {
    jayess_portaudio_stream *stream = NULL;
    PaStream *opened = NULL;
    double sample_rate = jayess_value_to_number(sample_rate_value);
    unsigned long frames_per_buffer = (unsigned long)jayess_value_to_number(frames_per_buffer_value);
    if (sample_rate <= 0 || frames_per_buffer == 0) {
        jayess_throw_named_error("PortAudioError", "PortAudio sample rate and framesPerBuffer must be positive");
        return jayess_value_undefined();
    }
    if (Pa_OpenDefaultStream(&opened, 0, 2, paFloat32, sample_rate, frames_per_buffer, jayess_portaudio_callback, NULL) != paNoError) {
        return jayess_value_undefined();
    }
    stream = (jayess_portaudio_stream *)calloc(1, sizeof(jayess_portaudio_stream));
    if (stream == NULL) {
        Pa_CloseStream(opened);
        return jayess_value_undefined();
    }
    stream->stream = opened;
    return jayess_value_from_managed_native_handle("PortAudioStream", stream, jayess_portaudio_stream_finalize);
}

jayess_value *jayess_portaudio_describe_stream(jayess_value *stream_value) {
    jayess_portaudio_stream *stream = (jayess_portaudio_stream *)jayess_expect_native_handle(stream_value, "PortAudioStream", "jayess_portaudio_describe_stream");
    const PaStreamInfo *info = NULL;
    jayess_object *object = NULL;
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    info = Pa_GetStreamInfo(stream->stream);
    if (info == NULL) {
        return jayess_value_undefined();
    }
    object = jayess_object_new();
    if (object == NULL) {
        return jayess_value_undefined();
    }
    jayess_object_set_value(object, "sampleRate", jayess_value_from_number(info->sampleRate));
    jayess_object_set_value(object, "outputLatency", jayess_value_from_number(info->outputLatency));
    jayess_object_set_value(object, "active", jayess_value_from_bool(Pa_IsStreamActive(stream->stream) == 1));
    jayess_object_set_value(object, "format", jayess_value_from_string(jayess_portaudio_format_name(paFloat32)));
    return jayess_value_from_object(object);
}

jayess_value *jayess_portaudio_start_stream(jayess_value *stream_value) {
    jayess_portaudio_stream *stream = (jayess_portaudio_stream *)jayess_expect_native_handle(stream_value, "PortAudioStream", "jayess_portaudio_start_stream");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(Pa_StartStream(stream->stream) == paNoError);
}

jayess_value *jayess_portaudio_stop_stream(jayess_value *stream_value) {
    jayess_portaudio_stream *stream = (jayess_portaudio_stream *)jayess_expect_native_handle(stream_value, "PortAudioStream", "jayess_portaudio_stop_stream");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bool(Pa_StopStream(stream->stream) == paNoError);
}

jayess_value *jayess_portaudio_close_stream(jayess_value *stream_value) {
    return jayess_value_from_bool(jayess_value_close_native_handle(stream_value));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio_stub.c"), []byte(`#include <portaudio.h>
#include <stdlib.h>

typedef struct jayess_portaudio_stream_stub {
    int active;
    double sampleRate;
    unsigned long framesPerBuffer;
} jayess_portaudio_stream_stub;

static int jayess_portaudio_initialized = 0;
static PaDeviceInfo jayess_portaudio_device_info = {
    2,
    "stub-portaudio-output",
    0,
    0,
    2,
    0.01,
    0.02,
    0.10,
    0.20,
    48000.0
};

static PaStreamInfo jayess_portaudio_stream_info = {
    1,
    0.0,
    0.02,
    48000.0
};

PaError Pa_Initialize(void) {
    jayess_portaudio_initialized = 1;
    return paNoError;
}

PaError Pa_Terminate(void) {
    jayess_portaudio_initialized = 0;
    return paNoError;
}

const char* Pa_GetVersionText(void) {
    return "stub-portaudio";
}

PaDeviceIndex Pa_GetDeviceCount(void) {
    return jayess_portaudio_initialized ? 1 : 0;
}

PaDeviceIndex Pa_GetDefaultOutputDevice(void) {
    return jayess_portaudio_initialized ? 0 : paNoDevice;
}

const PaDeviceInfo* Pa_GetDeviceInfo(PaDeviceIndex device) {
    if (!jayess_portaudio_initialized || device != 0) {
        return NULL;
    }
    return &jayess_portaudio_device_info;
}

PaError Pa_OpenDefaultStream(PaStream** stream, int numInputChannels, int numOutputChannels, PaSampleFormat sampleFormat, double sampleRate, unsigned long framesPerBuffer, PaStreamCallback *streamCallback, void *userData) {
    jayess_portaudio_stream_stub *opened = NULL;
    (void)numInputChannels;
    (void)streamCallback;
    (void)userData;
    if (!jayess_portaudio_initialized || stream == NULL || numOutputChannels != 2 || sampleFormat != paFloat32) {
        return paInvalidDevice;
    }
    opened = (jayess_portaudio_stream_stub *)calloc(1, sizeof(jayess_portaudio_stream_stub));
    if (opened == NULL) {
        return paInsufficientMemory;
    }
    opened->sampleRate = sampleRate;
    opened->framesPerBuffer = framesPerBuffer;
    jayess_portaudio_stream_info.sampleRate = sampleRate;
    *stream = (PaStream *)opened;
    return paNoError;
}

PaError Pa_CloseStream(PaStream *stream) {
    jayess_portaudio_stream_stub *state = (jayess_portaudio_stream_stub *)stream;
    if (stream == NULL) {
        return paBadStreamPtr;
    }
    free(state);
    return paNoError;
}

PaError Pa_StartStream(PaStream *stream) {
    jayess_portaudio_stream_stub *state = (jayess_portaudio_stream_stub *)stream;
    if (stream == NULL) {
        return paBadStreamPtr;
    }
    state->active = 1;
    return paNoError;
}

PaError Pa_StopStream(PaStream *stream) {
    jayess_portaudio_stream_stub *state = (jayess_portaudio_stream_stub *)stream;
    if (stream == NULL) {
        return paBadStreamPtr;
    }
    state->active = 0;
    return paNoError;
}

PaError Pa_IsStreamActive(PaStream *stream) {
    jayess_portaudio_stream_stub *state = (jayess_portaudio_stream_stub *)stream;
    if (stream == NULL) {
        return paBadStreamPtr;
    }
    return state->active ? 1 : 0;
}

const PaStreamInfo* Pa_GetStreamInfo(PaStream *stream) {
    jayess_portaudio_stream_stub *state = (jayess_portaudio_stream_stub *)stream;
    if (stream == NULL) {
        return NULL;
    }
    jayess_portaudio_stream_info.sampleRate = state->sampleRate;
    return &jayess_portaudio_stream_info;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio.bind.js"), []byte(`const f = () => {};
export const initAudio = f;
export const terminateAudio = f;
export const getVersionText = f;
export const getDeviceCount = f;
export const getDefaultOutputDevice = f;
export const openDefaultPlayback = f;
export const describeStream = f;
export const startStream = f;
export const stopStream = f;
export const closeStream = f;

export default {
  sources: ["./portaudio_wrapper.c", "./portaudio_stub.c"],
  includeDirs: ["./portaudio_include"],
  cflags: [],
  ldflags: [],
  exports: {
    initAudio: { symbol: "jayess_portaudio_init", type: "function" },
    terminateAudio: { symbol: "jayess_portaudio_terminate", type: "function" },
    getVersionText: { symbol: "jayess_portaudio_version_text", type: "function" },
    getDeviceCount: { symbol: "jayess_portaudio_device_count", type: "function" },
    getDefaultOutputDevice: { symbol: "jayess_portaudio_default_output_device", type: "function" },
    openDefaultPlayback: { symbol: "jayess_portaudio_open_default_playback", type: "function" },
    describeStream: { symbol: "jayess_portaudio_describe_stream", type: "function" },
    startStream: { symbol: "jayess_portaudio_start_stream", type: "function" },
    stopStream: { symbol: "jayess_portaudio_stop_stream", type: "function" },
    closeStream: { symbol: "jayess_portaudio_close_stream", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(`
import { initAudio, terminateAudio, getVersionText, getDeviceCount, getDefaultOutputDevice, openDefaultPlayback, describeStream, startStream, stopStream, closeStream } from "./native/portaudio.bind.js";

function main(args) {
  console.log("portaudio-init:" + initAudio());
  console.log("portaudio-version:" + getVersionText());
  console.log("portaudio-device-count:" + getDeviceCount());
  var device = getDefaultOutputDevice();
  console.log("portaudio-device:" + device.name + ":" + device.maxOutputChannels + ":" + device.defaultSampleRate);
  var stream = openDefaultPlayback(48000, 256);
  console.log("portaudio-open:" + (stream != undefined));
  var info = describeStream(stream);
  console.log("portaudio-format:" + info.format);
  console.log("portaudio-sample-rate:" + info.sampleRate);
  console.log("portaudio-output-latency:" + info.outputLatency);
  console.log("portaudio-active-before-start:" + info.active);
  console.log("portaudio-start:" + startStream(stream));
  console.log("portaudio-active-after-start:" + describeStream(stream).active);
  console.log("portaudio-stop:" + stopStream(stream));
  console.log("portaudio-close:" + closeStream(stream));
  try {
    describeStream(stream);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  console.log("portaudio-terminate:" + terminateAudio());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "manual-portaudio-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled PortAudio bind program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"portaudio-init:true",
		"portaudio-version:stub-portaudio",
		"portaudio-device-count:1",
		"portaudio-device:stub-portaudio-output:2:48000",
		"portaudio-open:true",
		"portaudio-format:f32",
		"portaudio-sample-rate:48000",
		"portaudio-output-latency:0.02",
		"portaudio-active-before-start:false",
		"portaudio-start:true",
		"portaudio-active-after-start:true",
		"portaudio-stop:true",
		"portaudio-close:true",
		"TypeError:jayess_portaudio_describe_stream expects a PortAudioStream native handle",
		"portaudio-terminate:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected PortAudio bind output to contain %q, got: %s", want, text)
		}
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

func TestBuildExecutableReassignmentReleasesPreviousOwnedLocalValue(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  var current = makeProbe("first-owned");
  console.log("after-first:" + cleanupLog());
  current = makeProbe("second-owned");
  console.log("after-second:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "reassignment-owned-cleanup-native")
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
		"after-first:",
		"after-second:first-owned;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected reassignment-owned cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableDestructuredOwnedLocalReassignmentReleasesPreviousValue(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  var [fromArray] = [makeProbe("array-first")];
  console.log("after-array-first:" + cleanupLog());
  fromArray = makeProbe("array-second");
  console.log("after-array-second:" + cleanupLog());

  resetCleanupLog();
  var { value: fromObject } = { value: makeProbe("object-first") };
  console.log("after-object-first:" + cleanupLog());
  fromObject = makeProbe("object-second");
  console.log("after-object-second:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructured-reassignment-owned-cleanup-native")
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
		"after-array-first:",
		"after-array-second:array-first;",
		"after-object-first:",
		"after-object-second:object-first;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected destructured reassignment cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableDestructuringAssignmentFromLiteralReleasesPreviousOwnedValue(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  var fromArray = makeProbe("array-seed");
  console.log("after-array-seed:" + cleanupLog());
  [fromArray] = [makeProbe("array-next")];
  console.log("after-array-assign:" + cleanupLog());

  resetCleanupLog();
  var fromObject = makeProbe("object-seed");
  console.log("after-object-seed:" + cleanupLog());
  [{ value: fromObject }] = [{ value: makeProbe("object-next") }];
  console.log("after-object-assign:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-assignment-owned-cleanup-native")
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
		"after-array-seed:",
		"after-array-assign:array-seed;",
		"after-object-seed:",
		"after-object-assign:object-seed;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected destructuring-assignment cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableDestructuringWithRestKeepsNamedLiteralBindingsOwned(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

function main(args) {
  resetCleanupLog();
  var [fromArray, ...tail] = [makeProbe("array-first"), 2, 3];
  console.log("after-array-first:" + cleanupLog() + ":" + tail.length);
  fromArray = makeProbe("array-second");
  console.log("after-array-second:" + cleanupLog() + ":" + tail.length);

  resetCleanupLog();
  var { value: fromObject, ...rest } = { value: makeProbe("object-first"), extra: 1 };
  console.log("after-object-first:" + cleanupLog() + ":" + rest.extra);
  fromObject = makeProbe("object-second");
  console.log("after-object-second:" + cleanupLog() + ":" + rest.extra);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-rest-owned-cleanup-native")
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
		"after-array-first::2",
		"after-array-second:array-first;:2",
		"after-object-first::1",
		"after-object-second:object-first;:1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected destructuring-with-rest cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableDestructuringWithRestDoesNotReevaluateLiteralEntries(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var calls = 0;
  const mark = (label) => {
    calls = calls + 1;
    console.log("mark:" + label + ":" + calls);
    return label + "-" + calls;
  };

  var [head, ...tail] = [mark("array-head"), mark("array-tail-1"), mark("array-tail-2")];
  console.log("array:" + head + ":" + tail.length + ":" + tail[0] + ":" + tail[1] + ":" + calls);

  calls = 0;
  var { value, ...rest } = { value: mark("object-value"), extra: mark("object-extra") };
  console.log("object:" + value + ":" + rest.extra + ":" + calls);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-rest-single-eval-native")
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
		"mark:array-head:1",
		"mark:array-tail-1:2",
		"mark:array-tail-2:3",
		"array:array-head-1:2:array-tail-1-2:array-tail-2-3:3",
		"mark:object-value:1",
		"mark:object-extra:2",
		"object:object-value-1:object-extra-2:2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected destructuring-rest single-eval output to contain %q, got: %s", want, text)
		}
	}
	if strings.Count(text, "mark:array-head:") != 1 || strings.Count(text, "mark:object-value:") != 1 {
		t.Fatalf("expected fresh literal entries to be evaluated exactly once, got: %s", text)
	}
}

func TestBuildExecutableObjectLiteralDestructuringFallsBackOnDuplicateKeys(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var calls = 0;
  const mark = (label) => {
    calls = calls + 1;
    console.log("mark:" + label + ":" + calls);
    return label + "-" + calls;
  };

  var { value, ...rest } = {
    value: mark("first"),
    value: mark("second"),
    extra: mark("extra")
  };

  console.log("result:" + value + ":" + rest.extra + ":" + calls);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-duplicate-object-keys-native")
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
		"mark:first:1",
		"mark:second:2",
		"mark:extra:3",
		"result:second-2:extra-3:3",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected duplicate-key destructuring output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableArrayLiteralDestructuringFallsBackOnSourceSpreadEffects(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var calls = 0;
  const mark = (label) => {
    calls = calls + 1;
    console.log("mark:" + label + ":" + calls);
    return label + "-" + calls;
  };
  const buildTail = () => {
    console.log("tail-build:" + calls);
    calls = calls + 1;
    console.log("tail-item-1:" + calls);
    const first = "tail-1-" + calls;
    calls = calls + 1;
    console.log("tail-item-2:" + calls);
    const second = "tail-2-" + calls;
    return [first, second];
  };

  var [first] = [mark("head"), ...buildTail()];
  console.log("result:" + first + ":" + calls);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-array-source-spread-native")
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
		"mark:head:1",
		"tail-build:1",
		"tail-item-1:2",
		"tail-item-2:3",
		"result:head-1:3",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected array-source-spread destructuring output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableObjectLiteralDestructuringFallsBackOnSourceSpreadEffects(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var calls = 0;
  const mark = (label) => {
    calls = calls + 1;
    console.log("mark:" + label + ":" + calls);
    return label + "-" + calls;
  };
  const buildExtra = () => {
    console.log("extra-build:" + calls);
    calls = calls + 1;
    console.log("extra-item:" + calls);
    return { extra: "extra-" + calls };
  };

  var { value } = { value: mark("value"), ...buildExtra() };
  console.log("result:" + value + ":" + calls);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-object-source-spread-native")
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
		"mark:value:1",
		"extra-build:1",
		"extra-item:2",
		"result:value-1:2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected object-source-spread destructuring output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableNestedLiteralFallbackMaterializesSourceOnce(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var calls = 0;
  const buildInner = () => {
    calls = calls + 1;
    console.log("build:" + calls);
    return { a: "a-" + calls, b: "b-" + calls };
  };

  var { inner: { a: declA, b: declB } } = { inner: buildInner() };
  console.log("decl:" + declA + ":" + declB + ":" + calls);

  calls = 0;
  var assignA = "";
  var assignB = "";
  [{ a: assignA, b: assignB }] = [buildInner()];
  console.log("assign:" + assignA + ":" + assignB + ":" + calls);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-nested-literal-fallback-native")
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
		"build:1",
		"decl:a-1:b-1:1",
		"assign:a-1:b-1:1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected nested literal fallback output to contain %q, got: %s", want, text)
		}
	}
	if strings.Count(text, "build:1") != 2 {
		t.Fatalf("expected nested fallback source to be materialized exactly once per destructuring site, got: %s", text)
	}
}

func TestBuildExecutableSupportsDestructuringInForLoopInit(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  for (var [i, limit] = [0, 3]; i < limit; i = i + 1) {
    total = total + i;
  }
  console.log("total:" + total);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-for-init-native")
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
	if !strings.Contains(text, "total:3") {
		t.Fatalf("expected for-loop init destructuring output to contain total:3, got: %s", text)
	}
}

func TestBuildExecutableSupportsDestructuringInForLoopUpdate(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  var i = 0;
  for (; i < 4; [i] = [i + 1]) {
    total = total + i;
  }
  console.log("total:" + total + ":" + i);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-for-update-native")
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
	if !strings.Contains(text, "total:6:4") {
		t.Fatalf("expected for-loop update destructuring output to contain total:6:4, got: %s", text)
	}
}

func TestBuildExecutableSupportsCompoundAssignmentInForLoopUpdate(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  for (var i = 0; i < 4; i += 1) {
    total = total + i;
  }
  console.log("total:" + total);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "compound-for-update-native")
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
	if !strings.Contains(text, "total:6") {
		t.Fatalf("expected compound for-loop update output to contain total:6, got: %s", text)
	}
}

func TestBuildExecutableSupportsDestructuringInForOfBinding(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  for (var [a, b] of [[1, 2], [3, 4]]) {
    total = total + a + b;
  }
  console.log("total:" + total);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-for-of-native")
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
	if !strings.Contains(text, "total:10") {
		t.Fatalf("expected for...of destructuring output to contain total:10, got: %s", text)
	}
}

func TestBuildExecutableSupportsObjectDestructuringInForOfBinding(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  for (var { value = 1, ...rest } of [{ extra: 2 }, { value: 3, extra: 4 }]) {
    total = total + value + rest.extra;
  }
  console.log("total:" + total);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-for-of-object-native")
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
	if !strings.Contains(text, "total:10") {
		t.Fatalf("expected for...of object destructuring output to contain total:10, got: %s", text)
	}
}

func TestBuildExecutableForOfDestructuringTempDoesNotCollideWithUserName(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native e2e test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	workdir := t.TempDir()

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
function main(args) {
  var total = 0;
  for (var [value] of [[1], [2], [3]]) {
    const __jayess_foreach_0 = 100;
    total = total + value;
    console.log("shadow:" + __jayess_foreach_0 + ":" + value);
  }
  console.log("total:" + total);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "destructuring-for-of-temp-collision-native")
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
		"shadow:100:1",
		"shadow:100:2",
		"shadow:100:3",
		"total:6",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected for...of temp-collision output to contain %q, got: %s", want, text)
		}
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

jayess_value *jayess_require_greeting(jayess_value *value) {
    const char *text = jayess_expect_string(value, "jayess_require_greeting");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(text);
}

jayess_value *jayess_require_items_length(jayess_value *value) {
    jayess_array *items = jayess_expect_array(value, "jayess_require_items_length");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_number((double)jayess_array_length(items));
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
export const requireGreeting = f;
export const requireItemsLength = f;
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
    requireGreeting: { symbol: "jayess_require_greeting", type: "function" },
    requireItemsLength: { symbol: "jayess_require_items_length", type: "function" },
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
import { requireName, requireGreeting, requireItemsLength, requireBytesSum, createCounter, counterValue } from "./native/safe.bind.js";

function main(args) {
  try {
    requireName(1);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    requireGreeting(1);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    requireItemsLength({});
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
		"TypeError:jayess_require_greeting expects a string",
		"TypeError:jayess_require_items_length expects an array",
		"TypeError:jayess_require_bytes_sum expects a Uint8Array or byte buffer value",
		"TypeError:jayess_counter_checked_value expects a CounterHandle native handle",
		"counter-safe:5",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected native type-safety output to contain %q, got: %s", want, text)
		}
	}
}

func TestCompileRejectsUsingBlockScopedValueOutsideItsLexicalScope(t *testing.T) {
	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	_, err = compiler.Compile(`
function main(args) {
  if (true) {
    const scoped = 7;
  }
  return scoped;
}
`, compiler.Options{TargetTriple: triple})
	if err == nil {
		t.Fatalf("expected compiler diagnostic for out-of-scope value access")
	}

	var compileErr *compiler.CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected CompileError, got %T: %v", err, err)
	}
	if compileErr.Diagnostic.Line == 0 || compileErr.Diagnostic.Column == 0 {
		t.Fatalf("expected compiler diagnostic location, got %#v", compileErr.Diagnostic)
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
  var alias = counter;
  var box = { value: counter };
  var items = [counter];
  var captured = function() {
    return counter;
  };
  var boxed = function() {
    return box.value;
  };
  var indexed = function() {
    return items[0];
  };
  console.log("owned-before:" + counterValue(counter) + ":" + counterClosed(counter));
  console.log("owned-structured-before:" + counterClosed(box.value) + ":" + counterClosed(items[0]) + ":" + counterClosed(captured()));
  console.log("owned-close:" + closeCounter(counter) + ":" + counterClosed(counter) + ":" + finalizeTotal());
  console.log("owned-alias-closed:" + counterClosed(alias));
  console.log("owned-structured-closed:" + counterClosed(box.value) + ":" + counterClosed(items[0]) + ":" + counterClosed(captured()) + ":" + counterClosed(boxed()) + ":" + counterClosed(indexed()));
  console.log("owned-close-again:" + closeCounter(counter) + ":" + counterClosed(counter) + ":" + finalizeTotal());
  console.log("owned-alias-close-again:" + closeCounter(alias) + ":" + counterClosed(alias) + ":" + finalizeTotal());
  console.log("owned-box-close-again:" + closeCounter(box.value) + ":" + counterClosed(box.value) + ":" + finalizeTotal());
  console.log("owned-array-close-again:" + closeCounter(items[0]) + ":" + counterClosed(items[0]) + ":" + finalizeTotal());
  try {
    counterValue(counter);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    counterValue(alias);
  } catch (err) {
    console.log("alias-" + err.name + ":" + err.message);
  }
  try {
    counterValue(box.value);
  } catch (err) {
    console.log("box-" + err.name + ":" + err.message);
  }
  try {
    counterValue(items[0]);
  } catch (err) {
    console.log("array-" + err.name + ":" + err.message);
  }
  try {
    counterValue(captured());
  } catch (err) {
    console.log("closure-" + err.name + ":" + err.message);
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
		"owned-structured-before:false:false:false",
		"owned-close:true:true:1",
		"owned-alias-closed:true",
		"owned-structured-closed:true:true:true:true:true",
		"owned-close-again:false:true:1",
		"owned-alias-close-again:false:true:1",
		"owned-box-close-again:false:true:1",
		"owned-array-close-again:false:true:1",
		"TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
		"alias-TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
		"box-TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
		"array-TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
		"closure-TypeError:jayess_owned_counter_value expects a OwnedCounter native handle",
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

typedef struct jayess_bytes_box {
    unsigned char *bytes;
    size_t length;
} jayess_bytes_box;

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

static void jayess_bytes_box_finalize(void *handle) {
    jayess_bytes_box *box = (jayess_bytes_box *)handle;
    if (box != NULL) {
        jayess_bytes_free(box->bytes);
        free(box);
    }
}

jayess_value *jayess_bytes_box_new(jayess_value *value) {
    jayess_bytes_box *box = (jayess_bytes_box *)malloc(sizeof(jayess_bytes_box));
    if (box == NULL) {
        return jayess_value_undefined();
    }
    box->bytes = jayess_expect_bytes_copy(value, &box->length, "jayess_bytes_box_new");
    if (jayess_has_exception()) {
        free(box);
        return jayess_value_undefined();
    }
    return jayess_value_from_managed_native_handle("BytesBox", box, jayess_bytes_box_finalize);
}

jayess_value *jayess_bytes_box_get(jayess_value *value) {
    jayess_bytes_box *box = (jayess_bytes_box *)jayess_expect_native_handle(value, "BytesBox", "jayess_bytes_box_get");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_value_from_bytes_copy(box->bytes != NULL ? box->bytes : (const unsigned char *)"", box->length);
}

jayess_value *jayess_bytes_box_close(jayess_value *value) {
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
export const createBytesBox = f;
export const readBytesBox = f;
export const closeBytesBox = f;

export default {
  sources: ["./lifetime.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    createBox: { symbol: "jayess_name_box_new", type: "function" },
    readBox: { symbol: "jayess_name_box_get", type: "function" },
    closeBox: { symbol: "jayess_name_box_close", type: "function" },
    createBytesBox: { symbol: "jayess_bytes_box_new", type: "function" },
    readBytesBox: { symbol: "jayess_bytes_box_get", type: "function" },
    closeBytesBox: { symbol: "jayess_bytes_box_close", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createBox, readBox, closeBox, createBytesBox, readBytesBox, closeBytesBox } from "./native/lifetime.bind.js";

function makeBox() {
  var name = "Kimchi";
  var box = createBox(name);
  name = "Soup";
  return box;
}

function makeBytesBox() {
  var bytes = Uint8Array.fromString("kimchi");
  var box = createBytesBox(bytes);
  bytes[0] = 106;
  return box;
}

function main(args) {
  var box = makeBox();
  var bytesBox = makeBytesBox();
  console.log("lifetime-copy:" + readBox(box));
  console.log("lifetime-bytes-copy:" + readBytesBox(bytesBox).toString());
  console.log("lifetime-close:" + closeBox(box));
  console.log("lifetime-bytes-close:" + closeBytesBox(bytesBox));
  try {
    readBox(box);
  } catch (err) {
    console.log(err.name + ":" + err.message);
  }
  try {
    readBytesBox(bytesBox);
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
		"lifetime-bytes-copy:kimchi",
		"lifetime-close:true",
		"lifetime-bytes-close:true",
		"TypeError:jayess_name_box_get expects a NameBox native handle",
		"TypeError:jayess_bytes_box_get expects a BytesBox native handle",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected lifetime-safe wrapper output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableRetainsJayessValuesOnWrapperObjectInsteadOfBorrowedNativePointers(t *testing.T) {
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

typedef struct jayess_retained_box {
    int marker;
} jayess_retained_box;

static void jayess_retained_box_finalize(void *handle) {
    if (handle != NULL) {
        free(handle);
    }
}

jayess_value *jayess_retained_box_new(jayess_value *value) {
    jayess_retained_box *box_state = (jayess_retained_box *)malloc(sizeof(jayess_retained_box));
    jayess_value *box;
    jayess_object *box_object;

    if (box_state == NULL) {
        return jayess_value_undefined();
    }
    box_state->marker = 1;
    box = jayess_value_from_managed_native_handle("RetainedValueBox", box_state, jayess_retained_box_finalize);
    box_object = jayess_value_as_object(box);
    if (box_object == NULL) {
        jayess_value_close_native_handle(box);
        return jayess_value_undefined();
    }
    /*
     * Keep the Jayess-managed value reachable from the wrapper object itself
     * instead of storing a borrowed jayess_value* in native long-lived state.
     */
    jayess_object_set_value(box_object, "__jayess_retained", value);
    return box;
}

static jayess_value *jayess_retained_box_value(jayess_value *box, const char *context) {
    jayess_object *box_object;
    jayess_retained_box *box_state = (jayess_retained_box *)jayess_expect_native_handle(box, "RetainedValueBox", context);
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    if (box_state == NULL || box_state->marker != 1) {
        return jayess_value_undefined();
    }
    box_object = jayess_value_as_object(box);
    if (box_object == NULL) {
        return jayess_value_undefined();
    }
    return jayess_object_get(box_object, "__jayess_retained");
}

jayess_value *jayess_retained_box_count(jayess_value *box) {
    jayess_value *value = jayess_retained_box_value(box, "jayess_retained_box_count");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    jayess_object *object = jayess_expect_object(value, "jayess_retained_box_count");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_object_get(object, "count");
}

jayess_value *jayess_retained_box_label(jayess_value *box) {
    jayess_value *value = jayess_retained_box_value(box, "jayess_retained_box_label");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    jayess_object *object = jayess_expect_object(value, "jayess_retained_box_label");
    if (jayess_has_exception()) {
        return jayess_value_undefined();
    }
    return jayess_object_get(object, "label");
}

jayess_value *jayess_retained_box_object(jayess_value *box) {
    return jayess_retained_box_value(box, "jayess_retained_box_object");
}

jayess_value *jayess_retained_box_close(jayess_value *box) {
    return jayess_value_from_bool(jayess_value_close_native_handle(box));
}
`
	if err := os.WriteFile(filepath.Join(nativeDir, "retained.c"), []byte(nativeSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "retained.bind.js"), []byte(`const f = () => {};
export const makeRetainedBox = f;
export const retainedCount = f;
export const retainedLabel = f;
export const retainedObject = f;
export const closeRetainedBox = f;

export default {
  sources: ["./retained.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    makeRetainedBox: { symbol: "jayess_retained_box_new", type: "function" },
    retainedCount: { symbol: "jayess_retained_box_count", type: "function" },
    retainedLabel: { symbol: "jayess_retained_box_label", type: "function" },
    retainedObject: { symbol: "jayess_retained_box_object", type: "function" },
    closeRetainedBox: { symbol: "jayess_retained_box_close", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { makeRetainedBox, retainedCount, retainedLabel, retainedObject, closeRetainedBox } from "./native/retained.bind.js";

function makeBox() {
  var local = { count: 2, label: "kept" };
  return makeRetainedBox(local);
}

function main(args) {
  var box = makeBox();
  console.log("retained-before:" + retainedCount(box) + ":" + retainedLabel(box));
  var alias = retainedObject(box);
  alias.count = alias.count + 5;
  console.log("retained-after:" + retainedCount(box) + ":" + alias.count);
  console.log("retained-close:" + closeRetainedBox(box));
  try {
    retainedCount(box);
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

	outputPath := nativeOutputPath(workdir, "ffi-native-retained-value-box")
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
		"retained-before:2:kept",
		"retained-after:7:7",
		"retained-close:true",
		"TypeError:jayess_retained_box_count expects a RetainedValueBox native handle",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected retained-value native wrapper output to contain %q, got: %s", want, text)
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
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var resp1 *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case runErr := <-done:
			t.Fatalf("mongoose embedded app program exited before serving requests: %v\noutput: %s", runErr, stdout.String())
		default:
		}
		resp1, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatalf("Get(/) returned error: %v\noutput: %s", err, stdout.String())
		}
		time.Sleep(50 * time.Millisecond)
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
		_ = cmd.Process.Kill()
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
		_ = cmd.Process.Kill()
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

	if err := <-done; err != nil {
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
	rewriteAudioPackageToUseCubebStub(t, repoRoot, filepath.Join(workdir, "node_modules", "@jayess", "audio"))
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
import { init, terminate, createWindow, createOpenGLWindow, destroyWindow, pollEvents, swapBuffers, makeContextCurrent, isContextCurrent, swapInterval, getProcAddress, hasProcAddress, isVulkanSupported, getRequiredVulkanInstanceExtensions, getVulkanInstanceProcAddress, windowShouldClose, getTime, setTime, getWindowSize, setWindowSize, getFramebufferSize, setKeyCallback, setMouseButtonCallback, setCursorPosCallback, setScrollCallback, simulateKeyEvent, simulateMouseButtonEvent, simulateCursorPosEvent, simulateScrollEvent, setWindowFullscreen, setWindowWindowed, isJoystickPresent, isJoystickGamepad, getJoystickName, getGamepadName, getGamepadButton } from "@jayess/glfw";
import { createContext, destroyContext, preferredSampleRate, minLatency, createPlaybackStream, startPlaybackStream, submitPlaybackSamples, playbackStats, nextStreamEvent, stopPlaybackStream, closePlaybackStream } from "@jayess/audio";

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
  var glWindow = createOpenGLWindow(32, 24, "jayess-glfw-gl");
  if (glWindow === undefined) {
    console.log("glfw-worker-terminate:" + workerThread.terminate());
    console.log("glfw-headless");
    terminate();
    return 0;
  }
  var window = createWindow(64, 64, "jayess-glfw-test");
  if (window === undefined) {
    console.log("glfw-destroy-gl:" + destroyWindow(glWindow));
    console.log("glfw-worker-terminate:" + workerThread.terminate());
    console.log("glfw-headless");
    terminate();
    return 0;
  }
  makeContextCurrent(glWindow);
  swapInterval(1);
  var glClearPtr = getProcAddress("glClear");
  var glGetStringPtr = getProcAddress("glGetString");
  var vulkanSupported = isVulkanSupported();
  var vulkanExtensions = getRequiredVulkanInstanceExtensions();
  var vkCreateInstancePtr = getVulkanInstanceProcAddress("vkCreateInstance");
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
  var audioCtx = createContext("jayess-glfw-audio", null);
  var audioRate = preferredSampleRate(audioCtx);
  var audioLatency = minLatency(audioCtx, { sampleRate: audioRate, channels: 2, format: "f32" });
  var audioStream = createPlaybackStream(audioCtx, { name: "glfw-audio", sampleRate: audioRate, channels: 2, format: "f32", latencyFrames: audioLatency });
  var audioSamples = new Float32Array(16);
  audioSamples[0] = 0.2;
  audioSamples[1] = -0.2;
  audioSamples[2] = 0.15;
  audioSamples[3] = -0.15;
  audioSamples[4] = 0.1;
  audioSamples[5] = -0.1;
  audioSamples[6] = 0.05;
  audioSamples[7] = -0.05;
  console.log("glfw-audio-open:" + (audioStream != undefined));
  console.log("glfw-audio-submit:" + (submitPlaybackSamples(audioStream, audioSamples) > 0));
  console.log("glfw-audio-start:" + startPlaybackStream(audioStream));
  var audioEvent = nextStreamEvent(audioStream);
  pollEvents();
  swapBuffers(window);
  console.log("glfw-worker-post:" + workerThread.postMessage({ left: 5, right: 7, text: "kimchi" }));
  console.log("glfw-worker-loop:" + await sleepAsync(0, "tick"));
  var workerReply = workerThread.receive(5000);
  await sleepAsync(20, null);
  var audioStats = playbackStats(audioStream);
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
  console.log("glfw-context-current:" + isContextCurrent(glWindow));
  console.log("glfw-proc-clear:" + (glClearPtr != undefined));
  console.log("glfw-proc-getstring:" + (glGetStringPtr != undefined));
  console.log("glfw-has-proc-clear:" + hasProcAddress("glClear"));
  console.log("glfw-vulkan-supported:" + (vulkanSupported === false || vulkanSupported === true));
  console.log("glfw-vulkan-exts:" + (vulkanExtensions == undefined || vulkanExtensions.length >= 1));
  console.log("glfw-vulkan-proc:" + (vkCreateInstancePtr == undefined || typeof vkCreateInstancePtr === "bigint"));
  console.log("glfw-size:" + size.width + "x" + size.height);
  console.log("glfw-framebuffer-size:" + framebuffer.width + "x" + framebuffer.height);
  console.log("glfw-key-callback:" + keyCount);
  console.log("glfw-mouse-callback:" + mouseCount);
  console.log("glfw-cursor-callback:" + cursorCount);
  console.log("glfw-scroll-callback:" + scrollCount);
  console.log("glfw-worker-reply:" + workerReply.value.sum + ":" + workerReply.value.text);
  console.log("glfw-audio-event-started:" + (audioEvent != undefined && audioEvent.type === "started"));
  console.log("glfw-audio-callbacks:" + (audioStats.callbacks > 0));
  console.log("glfw-audio-consumed:" + (audioStats.consumedFrames > 0));
  console.log("glfw-audio-running:" + audioStats.running);
  console.log("glfw-fullscreen-size:" + fullscreenSize.width + "x" + fullscreenSize.height);
  console.log("glfw-windowed-size:" + windowedSize.width + "x" + windowedSize.height);
  console.log("glfw-joystick-present:" + (isJoystickPresent(0) === false || isJoystickPresent(0) === true));
  console.log("glfw-joystick-gamepad:" + (isJoystickGamepad(0) === false || isJoystickGamepad(0) === true));
  console.log("glfw-joystick-name:" + (joystickName == undefined || typeof joystickName === "string"));
  console.log("glfw-gamepad-name:" + (gamepadName == undefined || typeof gamepadName === "string"));
  console.log("glfw-gamepad-button:" + (getGamepadButton(0, 0) == undefined || getGamepadButton(0, 0) === false || getGamepadButton(0, 0) === true));
  console.log("glfw-window:" + windowShouldClose(window));
  console.log("glfw-audio-stop:" + stopPlaybackStream(audioStream));
  console.log("glfw-audio-close:" + closePlaybackStream(audioStream));
  console.log("glfw-audio-destroy:" + destroyContext(audioCtx));
  console.log("glfw-destroy:" + destroyWindow(window));
  console.log("glfw-destroy-gl:" + destroyWindow(glWindow));
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
		"glfw-audio-open:true",
		"glfw-audio-submit:true",
		"glfw-audio-start:true",
		"glfw-audio-event-started:true",
		"glfw-audio-callbacks:true",
		"glfw-audio-consumed:true",
		"glfw-audio-running:true",
		"glfw-vulkan-supported:true",
		"glfw-vulkan-exts:true",
		"glfw-vulkan-proc:true",
		"glfw-fullscreen-size:1920x1080",
		"glfw-windowed-size:80x60",
		"glfw-joystick-present:true",
		"glfw-joystick-gamepad:true",
		"glfw-joystick-name:true",
		"glfw-gamepad-name:true",
		"glfw-gamepad-button:true",
		"glfw-window:",
		"glfw-audio-stop:true",
		"glfw-audio-close:true",
		"glfw-audio-destroy:true",
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
		filepath.Join(repoRoot, "node_modules", "@jayess", "audio"),
		filepath.Join(workdir, "node_modules", "@jayess", "audio"),
	)
	rewriteAudioPackageToUseCubebStub(t, repoRoot, filepath.Join(workdir, "node_modules", "@jayess", "audio"))
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "raylib", "src"),
		filepath.Join(workdir, "refs", "raylib", "src"),
	)
	tinyPPM := []byte{
		'P', '6', '\n', '2', ' ', '2', '\n', '2', '5', '5', '\n',
		255, 0, 0,
		0, 255, 0,
		0, 0, 255,
		255, 255, 255,
	}
	if err := os.WriteFile(filepath.Join(workdir, "tiny.ppm"), tinyPPM, 0o644); err != nil {
		t.Fatalf("WriteFile tiny.ppm returned error: %v", err)
	}
	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { setTraceLogLevel, setTraceLogCallback, clearTraceLogCallback, emitTraceLog, setConfigFlags, initWindow, closeWindow, isWindowReady, windowShouldClose, setWindowTitle, setWindowSize, setWindowFullscreen, setWindowWindowed, getScreenWidth, getScreenHeight, beginDrawing, endDrawing, clearBackground, drawCircle, drawText, genImageColor, loadImage, loadImageFromBytes, unloadImage, isImageReady, getImageWidth, getImageHeight, loadTextureFromImage, unloadTexture, isTextureReady, getTextureWidth, getTextureHeight, drawTexture, isKeyPressed, isKeyDown, isMouseButtonDown, getMouseX, getMouseY, getMousePosition, isGamepadAvailable, getGamepadAxisCount, isGamepadButtonDown, getGamepadName, setTargetFPS, getFrameTime, getTime, setRandomSeed, getRandomValue } from "@jayess/raylib";
import { createContext, destroyContext, preferredSampleRate, minLatency, createPlaybackStream, startPlaybackStream, submitPlaybackSamples, playbackStats, nextStreamEvent, stopPlaybackStream, closePlaybackStream } from "@jayess/audio";

function main(args) {
  var black = { r: 0, g: 0, b: 0, a: 255 };
  var red = { r: 230, g: 41, b: 55, a: 255 };
  var white = { r: 255, g: 255, b: 255, a: 255 };
  var green = { r: 0, g: 228, b: 48, a: 255 };
  var mouse = null;
  var gamepadName = null;
  var image = null;
  var memoryImage = null;
  var fileImage = null;
  var bytesImage = null;
  var texture = null;
  var memoryTexture = null;
  var fileTexture = null;
  var bytesTexture = null;
  var traceCount = 0;
  setTraceLogLevel(7);
  {
    var prefix = "trace";
    console.log("raylib-trace-install:" + setTraceLogCallback((level, message) => {
      traceCount = traceCount + 1;
      console.log("raylib-trace-callback:" + prefix + ":" + level + ":" + message + ":" + traceCount);
      return 0;
    }));
  }
  console.log("raylib-trace-emit:" + emitTraceLog(3, "hello"));
  console.log("raylib-trace-clear:" + clearTraceLogCallback());
  console.log("raylib-trace-after-clear:" + emitTraceLog(4, "later"));
  console.log("raylib-trace-count:" + traceCount);
  setConfigFlags(128);
  setRandomSeed(123);
  console.log("raylib-rand:" + getRandomValue(1, 10));
  console.log("raylib-init:" + initWindow(64, 64, "jayess-raylib"));
  console.log("raylib-ready:" + isWindowReady());
  setWindowTitle("jayess-raylib-updated");
  setWindowSize(48, 40);
  console.log("raylib-size:" + getScreenWidth() + "x" + getScreenHeight());
  setWindowFullscreen();
  console.log("raylib-fullscreen-size:" + getScreenWidth() + "x" + getScreenHeight());
  setWindowWindowed(80, 60);
  console.log("raylib-windowed-size:" + getScreenWidth() + "x" + getScreenHeight());
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
  var audioCtx = createContext("jayess-raylib-audio", null);
  var audioRate = preferredSampleRate(audioCtx);
  var audioLatency = minLatency(audioCtx, { sampleRate: audioRate, channels: 2, format: "f32" });
  var audioStream = createPlaybackStream(audioCtx, { name: "raylib-audio", sampleRate: audioRate, channels: 2, format: "f32", latencyFrames: audioLatency });
  var audioSamples = new Float32Array(16);
  audioSamples[0] = 0.2;
  audioSamples[1] = -0.2;
  audioSamples[2] = 0.15;
  audioSamples[3] = -0.15;
  audioSamples[4] = 0.1;
  audioSamples[5] = -0.1;
  audioSamples[6] = 0.05;
  audioSamples[7] = -0.05;
  console.log("raylib-audio-open:" + (audioStream != undefined));
  console.log("raylib-audio-submit:" + (submitPlaybackSamples(audioStream, audioSamples) > 0));
  console.log("raylib-audio-start:" + startPlaybackStream(audioStream));
  var audioEvent = nextStreamEvent(audioStream);
  var assetPath = path.join(".", "tiny.ppm");
  var assetBytes = fs.createReadStream(assetPath).readBytes(23);
  image = genImageColor(4, 4, green);
  memoryImage = genImageColor(2, 2, white);
  fileImage = loadImage(assetPath);
  bytesImage = loadImageFromBytes(".ppm", assetBytes);
  texture = loadTextureFromImage(image);
  memoryTexture = loadTextureFromImage(memoryImage);
  fileTexture = loadTextureFromImage(fileImage);
  bytesTexture = loadTextureFromImage(bytesImage);
  console.log("raylib-image-ready:" + isImageReady(image));
  console.log("raylib-image-size:" + getImageWidth(image) + "x" + getImageHeight(image));
  console.log("raylib-second-image-size:" + getImageWidth(memoryImage) + "x" + getImageHeight(memoryImage));
  console.log("raylib-file-asset-path:" + assetPath);
  console.log("raylib-file-asset-bytes:" + assetBytes.length);
  console.log("raylib-file-image-size:" + getImageWidth(fileImage) + "x" + getImageHeight(fileImage));
  console.log("raylib-bytes-image-size:" + getImageWidth(bytesImage) + "x" + getImageHeight(bytesImage));
  console.log("raylib-texture-ready:" + isTextureReady(texture));
  console.log("raylib-texture-size:" + getTextureWidth(texture) + "x" + getTextureHeight(texture));
  console.log("raylib-second-texture-size:" + getTextureWidth(memoryTexture) + "x" + getTextureHeight(memoryTexture));
  console.log("raylib-file-texture-size:" + getTextureWidth(fileTexture) + "x" + getTextureHeight(fileTexture));
  console.log("raylib-bytes-texture-size:" + getTextureWidth(bytesTexture) + "x" + getTextureHeight(bytesTexture));
  setTargetFPS(60);
  beginDrawing();
  clearBackground(black);
  drawCircle(32, 32, 8, red);
  drawText("jayess", 4, 4, 10, white);
  drawTexture(texture, 12, 12, white);
  drawTexture(memoryTexture, 20, 20, white);
  drawTexture(fileTexture, 28, 28, white);
  drawTexture(bytesTexture, 36, 36, white);
  endDrawing();
  setTimeout(() => {
    console.log("raylib-timer");
    return 0;
  }, 0);
  console.log("raylib-async:" + await sleepAsync(0, "ok"));
  await sleepAsync(20, null);
  var audioStats = playbackStats(audioStream);
  beginDrawing();
  clearBackground(black);
  endDrawing();
  console.log("raylib-text:true");
  console.log("raylib-texture-draw:true");
  console.log("raylib-audio-event-started:" + (audioEvent != undefined && audioEvent.type === "started"));
  console.log("raylib-audio-callbacks:" + (audioStats.callbacks > 0));
  console.log("raylib-audio-consumed:" + (audioStats.consumedFrames > 0));
  console.log("raylib-audio-running:" + audioStats.running);
  console.log("raylib-unload-texture:" + unloadTexture(texture));
  console.log("raylib-unload-memory-texture:" + unloadTexture(memoryTexture));
  console.log("raylib-unload-file-texture:" + unloadTexture(fileTexture));
  console.log("raylib-unload-bytes-texture:" + unloadTexture(bytesTexture));
  console.log("raylib-unload-image:" + unloadImage(image));
  console.log("raylib-unload-memory-image:" + unloadImage(memoryImage));
  console.log("raylib-unload-file-image:" + unloadImage(fileImage));
  console.log("raylib-unload-bytes-image:" + unloadImage(bytesImage));
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
  console.log("raylib-audio-stop:" + stopPlaybackStream(audioStream));
  console.log("raylib-audio-close:" + closePlaybackStream(audioStream));
  console.log("raylib-audio-destroy:" + destroyContext(audioCtx));
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
		"raylib-trace-install:true",
		"raylib-trace-callback:trace:3:hello:1",
		"raylib-trace-emit:true",
		"raylib-trace-clear:true",
		"raylib-trace-after-clear:false",
		"raylib-trace-count:1",
		"raylib-rand:",
		"raylib-init:true",
		"raylib-ready:true",
		"raylib-size:48x40",
		"raylib-fullscreen-size:1920x1080",
		"raylib-windowed-size:80x60",
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
		"raylib-file-asset-path:./tiny.ppm",
		"raylib-file-asset-bytes:23",
		"raylib-file-image-size:2x2",
		"raylib-bytes-image-size:2x2",
		"raylib-texture-ready:true",
		"raylib-texture-size:4x4",
		"raylib-second-texture-size:2x2",
		"raylib-file-texture-size:2x2",
		"raylib-bytes-texture-size:2x2",
		"raylib-timer",
		"raylib-async:ok",
		"raylib-text:true",
		"raylib-texture-draw:true",
		"raylib-audio-open:true",
		"raylib-audio-submit:true",
		"raylib-audio-start:true",
		"raylib-audio-event-started:true",
		"raylib-audio-callbacks:true",
		"raylib-audio-consumed:true",
		"raylib-audio-running:true",
		"raylib-unload-texture:true",
		"raylib-unload-memory-texture:true",
		"raylib-unload-file-texture:true",
		"raylib-unload-bytes-texture:true",
		"raylib-unload-image:true",
		"raylib-unload-memory-image:true",
		"raylib-unload-file-image:true",
		"raylib-unload-bytes-image:true",
		"TypeError:jayess_raylib_get_texture_width expects a RaylibTexture native handle",
		"TypeError:jayess_raylib_get_image_width expects a RaylibImage native handle",
		"raylib-frame:true",
		"raylib-time:true",
		"raylib-audio-stop:true",
		"raylib-audio-close:true",
		"raylib-audio-destroy:true",
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
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "miniaudio"),
		filepath.Join(workdir, "refs", "miniaudio"),
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
import { createContext, backendId, maxChannelCount, listOutputDevices, listInputDevices, destroyContext } from "@jayess/audio";

function main(args) {
  var ctx = createContext("jayess-audio-test", null);
  if (ctx === undefined) {
    console.log("audio-init:undefined");
    return 0;
  }
  console.log("audio-backend:" + backendId(ctx));
  console.log("audio-max-channels:" + maxChannelCount(ctx));
  var outputs = listOutputDevices(ctx);
  var inputs = listInputDevices(ctx);
  console.log("audio-output-devices:" + outputs.length);
  console.log("audio-input-devices:" + inputs.length);
  if (outputs.length > 0) {
    console.log("audio-first-output-type:" + outputs[0].type);
  }
  if (inputs.length > 0) {
    console.log("audio-first-input-type:" + inputs[0].type);
  }
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
		text := string(out)
		for _, want := range []string{
			"audio-backend:",
			"audio-max-channels:",
			"audio-output-devices:",
			"audio-input-devices:",
			"audio-destroy:true",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("expected audio package smoke output to contain %q, got: %s", want, text)
			}
		}
		return
	}
	if !strings.Contains(err.Error(), "cubeb") || (!strings.Contains(err.Error(), "native library link failed") && !strings.Contains(err.Error(), "clang native build failed")) {
		t.Fatalf("expected clear Cubeb build diagnostic, got: %v", err)
	}
}

func TestBuildExecutableSupportsJayessAudioPlaybackSurface(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping native audio playback test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	pkgDir := filepath.Join(workdir, "node_modules", "@jayess", "audio")
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "audio"),
		pkgDir,
	)
	rewriteAudioPackageToUseCubebStub(t, repoRoot, pkgDir)

	mainPath := filepath.Join(workdir, "main.js")
	mainSource := `
import { createContext, destroyContext, preferredSampleRate, minLatency, createPlaybackStream, startPlaybackStream, pausePlaybackStream, stopPlaybackStream, submitPlaybackSamples, playbackStats, closePlaybackStream, nextStreamEvent, createCaptureStream, startCaptureStream, stopCaptureStream, readCapturedSamples, captureStats, closeCaptureStream, loadWav, loadOgg, loadMp3, loadFlac } from "@jayess/audio";

function main(args) {
  var ctx = createContext("jayess-audio-playback-test", null);
  var decoded = loadWav("sample.wav");
  var decodedOgg = loadOgg("sample.ogg");
  var decodedMp3 = loadMp3("sample.mp3");
  var decodedFlac = loadFlac("sample.flac");
  var rate = preferredSampleRate(ctx);
  var latency = minLatency(ctx, { sampleRate: rate, channels: 2, format: "f32" });
  var workerThread = worker.create(function(message) {
    return { count: message.count + 1, label: message.label.toUpperCase() };
  });
  var stream = createPlaybackStream(ctx, { name: "stub-playback", sampleRate: rate, channels: 2, format: "f32", latencyFrames: latency });
  var first = [];
  var second = [];
  var i = 0;
  for (i = 0; i < 256; i = i + 1) {
    first.push(0.15);
    first.push(-0.15);
  }
  for (i = 0; i < 128; i = i + 1) {
    second.push(0.05);
    second.push(-0.05);
  }
  console.log("audio-stream-open:" + (stream != undefined));
  console.log("audio-wav-rate:" + decoded.sampleRate);
  console.log("audio-wav-channels:" + decoded.channels);
  console.log("audio-wav-frames:" + decoded.frames);
  console.log("audio-wav-format:" + decoded.format);
  console.log("audio-wav-source-format:" + decoded.sourceFormat);
  console.log("audio-wav-samples-length:" + decoded.samples.length);
  console.log("audio-wav-first-sample:" + (decoded.samples[0] > 0.49));
  console.log("audio-wav-second-sample:" + (decoded.samples[1] < -0.49));
  console.log("audio-ogg-rate:" + decodedOgg.sampleRate);
  console.log("audio-ogg-channels:" + decodedOgg.channels);
  console.log("audio-ogg-format:" + decodedOgg.format);
  console.log("audio-ogg-source-format:" + decodedOgg.sourceFormat);
  console.log("audio-ogg-samples-length:" + (decodedOgg.samples.length > 0));
  console.log("audio-mp3-rate:" + (decodedMp3.sampleRate > 0));
  console.log("audio-mp3-channels:" + (decodedMp3.channels > 0));
  console.log("audio-mp3-format:" + decodedMp3.format);
  console.log("audio-mp3-source-format:" + decodedMp3.sourceFormat);
  console.log("audio-mp3-samples-length:" + (decodedMp3.samples.length > 0));
  console.log("audio-flac-rate:" + decodedFlac.sampleRate);
  console.log("audio-flac-channels:" + decodedFlac.channels);
  console.log("audio-flac-format:" + decodedFlac.format);
  console.log("audio-flac-source-format:" + decodedFlac.sourceFormat);
  console.log("audio-flac-samples-length:" + (decodedFlac.samples.length > 0));
  console.log("audio-stream-rate:" + rate);
  console.log("audio-stream-latency:" + latency);
  console.log("audio-stream-submit-a:" + submitPlaybackSamples(stream, first));
  console.log("audio-stream-start:" + startPlaybackStream(stream));
  var startedEvent = nextStreamEvent(stream);
  await sleepAsync(20, null);
  var statsA = playbackStats(stream);
  var typed = new Float32Array(8);
  typed[0] = 0.2;
  typed[1] = -0.2;
  typed[2] = 0.1;
  typed[3] = -0.1;
  typed[4] = 0.05;
  typed[5] = -0.05;
  typed[6] = 0.025;
  typed[7] = -0.025;
  console.log("audio-stream-format:" + statsA.format);
  console.log("audio-stream-channels:" + statsA.channels);
  console.log("audio-stream-callbacks:" + (statsA.callbacks > 0));
  console.log("audio-stream-consumed:" + (statsA.consumedFrames > 0));
  console.log("audio-stream-running:" + statsA.running);
  console.log("audio-stream-event-started:" + (startedEvent != undefined && startedEvent.type === "started"));
  console.log("audio-stream-submit-f32-buffer:" + submitPlaybackSamples(stream, typed));
  console.log("audio-stream-submit-b:" + submitPlaybackSamples(stream, second));
  console.log("audio-worker-post:" + workerThread.postMessage({ count: 4, label: "mix" }));
  await sleepAsync(20, null);
  var workerReply = workerThread.receive(5000);
  var statsB = playbackStats(stream);
  console.log("audio-stream-streaming:" + (statsB.submittedFrames > statsA.submittedFrames && statsB.consumedFrames > statsA.consumedFrames));
  console.log("audio-stream-pause:" + pausePlaybackStream(stream));
  var paused = playbackStats(stream);
  console.log("audio-stream-paused:" + (!paused.running));
  console.log("audio-stream-resume:" + startPlaybackStream(stream));
  await sleepAsync(20, null);
  var resumed = playbackStats(stream);
  console.log("audio-stream-resumed:" + (resumed.running && resumed.callbacks > paused.callbacks));
  console.log("audio-stream-stop:" + stopPlaybackStream(stream));
  var stopped = playbackStats(stream);
  console.log("audio-stream-stopped:" + (!stopped.running));
  var empty = createPlaybackStream(ctx, { name: "stub-empty", sampleRate: rate, channels: 2, format: "f32", latencyFrames: latency });
  console.log("audio-empty-open:" + (empty != undefined));
  console.log("audio-empty-start:" + startPlaybackStream(empty));
  var emptyStarted = nextStreamEvent(empty);
  await sleepAsync(20, null);
  var emptyStats = playbackStats(empty);
  var emptyEvent = nextStreamEvent(empty);
  console.log("audio-empty-underruns:" + (emptyStats.underruns > 0));
  console.log("audio-empty-event-started:" + (emptyStarted != undefined && emptyStarted.type === "started"));
  console.log("audio-empty-event-underrun:" + (emptyEvent != undefined && emptyEvent.type === "underrun"));
  console.log("audio-empty-stop:" + stopPlaybackStream(empty));
  console.log("audio-empty-close:" + closePlaybackStream(empty));
  var errorStream = createPlaybackStream(ctx, { name: "stub-error", sampleRate: rate, channels: 2, format: "f32", latencyFrames: latency });
  console.log("audio-error-open:" + (errorStream != undefined));
  console.log("audio-error-submit:" + submitPlaybackSamples(errorStream, [0.25, -0.25, 0.125, -0.125]));
  console.log("audio-error-start:" + startPlaybackStream(errorStream));
  var errorStarted = nextStreamEvent(errorStream);
  var errorEvent = undefined;
  var errorSeen = false;
  var errorPoll = 0;
  for (errorPoll = 0; errorPoll < 8; errorPoll = errorPoll + 1) {
    await sleepAsync(5, null);
    errorEvent = nextStreamEvent(errorStream);
    if (errorEvent != undefined && errorEvent.type === "error") {
      errorSeen = true;
      break;
    }
  }
  var errorStats = playbackStats(errorStream);
  console.log("audio-error-event-started:" + (errorStarted != undefined && errorStarted.type === "started"));
  console.log("audio-error-event-error:" + errorSeen);
  console.log("audio-error-last-state:" + errorStats.lastState);
  console.log("audio-error-running:" + errorStats.running);
  console.log("audio-error-close:" + closePlaybackStream(errorStream));
  var capture = createCaptureStream(ctx, { name: "stub-capture", sampleRate: rate, channels: 1, format: "f32", latencyFrames: latency });
  console.log("audio-capture-open:" + (capture != undefined));
  console.log("audio-capture-start:" + startCaptureStream(capture));
  var captureStarted = nextStreamEvent(capture);
  await sleepAsync(20, null);
  var captureA = captureStats(capture);
  var captured = readCapturedSamples(capture, 4);
  console.log("audio-capture-running:" + captureA.running);
  console.log("audio-capture-event-started:" + (captureStarted != undefined && captureStarted.type === "started"));
  console.log("audio-capture-callbacks:" + (captureA.callbacks > 0));
  console.log("audio-capture-frames:" + (captureA.capturedFrames > 0));
  console.log("audio-capture-buffer-length:" + captured.length);
  console.log("audio-capture-first-sample:" + (captured[0] > 0.24));
  console.log("audio-capture-second-sample:" + (captured[1] < -0.24));
  console.log("audio-capture-stop:" + stopCaptureStream(capture));
  var captureStopped = nextStreamEvent(capture);
  console.log("audio-capture-event-stopped:" + (captureStopped != undefined && captureStopped.type === "stopped"));
  console.log("audio-capture-close:" + closeCaptureStream(capture));
  var bytes = Uint8Array.fromString("ff7f0080004000c0", "hex");
  var s16stream = createPlaybackStream(ctx, { name: "stub-s16", sampleRate: rate, channels: 2, format: "s16", latencyFrames: latency });
  console.log("audio-s16-open:" + (s16stream != undefined));
  console.log("audio-s16-submit:" + submitPlaybackSamples(s16stream, bytes));
  console.log("audio-s16-start:" + startPlaybackStream(s16stream));
  await sleepAsync(20, null);
  var s16stats = playbackStats(s16stream);
  console.log("audio-s16-format:" + s16stats.format);
  console.log("audio-s16-consumed:" + (s16stats.consumedFrames > 0));
  console.log("audio-s16-stop:" + stopPlaybackStream(s16stream));
  console.log("audio-s16-close:" + closePlaybackStream(s16stream));
  console.log("audio-worker-reply:" + (workerReply != undefined));
  console.log("audio-worker-terminate:" + workerThread.terminate());
  console.log("audio-stream-close:" + closePlaybackStream(stream));
  console.log("audio-context-destroy:" + destroyContext(ctx));
  return 0;
}
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile(main.js) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "sample.wav"), []byte{
		'R', 'I', 'F', 'F',
		44, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ',
		16, 0, 0, 0,
		1, 0,
		2, 0,
		0x80, 0xbb, 0, 0,
		0x00, 0xee, 0x02, 0x00,
		4, 0,
		16, 0,
		'd', 'a', 't', 'a',
		8, 0, 0, 0,
		0xff, 0x3f,
		0x01, 0xc0,
		0x00, 0x20,
		0x00, 0xe0,
	}, 0o644); err != nil {
		t.Fatalf("WriteFile(sample.wav) returned error: %v", err)
	}
	copyFileForTest(
		t,
		filepath.Join(repoRoot, "refs", "miniaudio", "data", "48000-stereo.ogg"),
		filepath.Join(workdir, "sample.ogg"),
	)
	copyFileForTest(
		t,
		filepath.Join(repoRoot, "refs", "SDL_mixer", "examples", "music.mp3"),
		filepath.Join(workdir, "sample.mp3"),
	)
	copyFileForTest(
		t,
		filepath.Join(repoRoot, "refs", "miniaudio", "data", "16-44100-stereo.flac"),
		filepath.Join(workdir, "sample.flac"),
	)

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-audio-playback")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}
	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled audio playback program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"audio-stream-open:true",
		"audio-wav-rate:48000",
		"audio-wav-channels:2",
		"audio-wav-frames:2",
		"audio-wav-format:f32",
		"audio-wav-source-format:wav-s16",
		"audio-wav-samples-length:4",
		"audio-wav-first-sample:true",
		"audio-wav-second-sample:true",
		"audio-ogg-rate:48000",
		"audio-ogg-channels:2",
		"audio-ogg-format:f32",
		"audio-ogg-source-format:ogg-vorbis",
		"audio-ogg-samples-length:true",
		"audio-mp3-rate:true",
		"audio-mp3-channels:true",
		"audio-mp3-format:f32",
		"audio-mp3-source-format:mp3",
		"audio-mp3-samples-length:true",
		"audio-flac-rate:44100",
		"audio-flac-channels:2",
		"audio-flac-format:f32",
		"audio-flac-source-format:flac",
		"audio-flac-samples-length:true",
		"audio-stream-rate:48000",
		"audio-stream-latency:64",
		"audio-stream-submit-a:256",
		"audio-stream-start:true",
		"audio-stream-format:f32",
		"audio-stream-channels:2",
		"audio-stream-callbacks:true",
		"audio-stream-consumed:true",
		"audio-stream-running:true",
		"audio-stream-event-started:true",
		"audio-stream-submit-f32-buffer:4",
		"audio-stream-submit-b:128",
		"audio-worker-post:true",
		"audio-stream-streaming:true",
		"audio-stream-pause:true",
		"audio-stream-paused:true",
		"audio-stream-resume:true",
		"audio-stream-resumed:true",
		"audio-stream-stop:true",
		"audio-stream-stopped:true",
		"audio-empty-open:true",
		"audio-empty-start:true",
		"audio-empty-underruns:true",
		"audio-empty-event-started:true",
		"audio-empty-event-underrun:true",
		"audio-empty-stop:true",
		"audio-empty-close:true",
		"audio-error-open:true",
		"audio-error-submit:2",
		"audio-error-start:true",
		"audio-error-event-started:true",
		"audio-error-event-error:true",
		"audio-error-last-state:error",
		"audio-error-running:false",
		"audio-error-close:true",
		"audio-capture-open:true",
		"audio-capture-start:true",
		"audio-capture-running:true",
		"audio-capture-event-started:true",
		"audio-capture-callbacks:true",
		"audio-capture-frames:true",
		"audio-capture-buffer-length:4",
		"audio-capture-first-sample:true",
		"audio-capture-second-sample:true",
		"audio-capture-stop:true",
		"audio-capture-event-stopped:true",
		"audio-capture-close:true",
		"audio-s16-open:true",
		"audio-s16-submit:2",
		"audio-s16-start:true",
		"audio-s16-format:s16",
		"audio-s16-consumed:true",
		"audio-s16-stop:true",
		"audio-s16-close:true",
		"audio-worker-reply:true",
		"audio-worker-terminate:true",
		"audio-stream-close:true",
		"audio-context-destroy:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected audio playback output to contain %q, got: %s", want, text)
		}
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

func prepareStubWebviewPackage(t *testing.T, repoRoot string, workdir string) {
	t.Helper()

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
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "glfw"),
		filepath.Join(workdir, "refs", "glfw"),
	)
	prepareStubWebviewPackage(t, repoRoot, workdir)

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

func TestBuildExecutableSupportsJayessWebviewHTTPServerCoexistence(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping webview HTTP coexistence test: %v", err)
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

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	prepareStubWebviewPackage(t, repoRoot, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(fmt.Sprintf(`
import { createWindow, destroyWindow } from "@jayess/webview";

var serverRef = undefined;

function handleRequest(req, res) {
  console.log("webview-http-req:" + req.method + ":" + req.url);
  res.statusCode = 200;
  res.setHeader("Content-Type", "text/html; charset=utf-8");
  res.end("<!doctype html><h1>webview-http</h1>");
  serverRef.close();
  return 0;
}

function main() {
  var view = undefined;
  console.log("webview-http-imported:true");
  serverRef = http.createServer(handleRequest);
  console.log("webview-http-server:" + (serverRef != undefined));
  serverRef.listen(%d, "127.0.0.1");
  console.log("webview-http-view:" + (view == undefined));
  return 0;
}
`, port)), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "webview-http-coexistence-native")
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
		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		select {
		case resultRun := <-done:
			t.Fatalf("HTTP request returned error: %v\nchild error: %v\nchild output:\n%s", err, resultRun.err, string(resultRun.out))
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("HTTP request returned error: %v\nchild process did not exit within diagnostic timeout", err)
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := string(bodyBytes); got != "<!doctype html><h1>webview-http</h1>" {
		t.Fatalf("expected embedded HTML body, got %q", got)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected text/html content type, got %q", got)
	}

	resultRun := <-done
	if resultRun.err != nil {
		t.Fatalf("compiled webview HTTP coexistence program returned error: %v: %s", resultRun.err, string(resultRun.out))
	}
	text := string(resultRun.out)
	for _, want := range []string{
		"webview-http-imported:true",
		"webview-http-server:true",
		"webview-http-req:GET:/",
		"webview-http-view:true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected webview HTTP coexistence output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessHTMLPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping HTML package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "html"),
		filepath.Join(workdir, "node_modules", "@jayess", "html"),
	)

	sampleHTML := `<div id="a" disabled><span>hi</span><!--note--><br/></div>`
	if err := os.WriteFile(filepath.Join(workdir, "sample.html"), []byte(sampleHTML), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeHtml, parseHtml, parseHtmlFragment, serializeHtml, serializeHtmlWithOptions, createElement, createText, createComment, setAttribute, removeAttribute, appendChild, removeChild, replaceChild, cloneNode, walkDepthFirst, findByTag, matchesSelector, querySelectorAll } from "@jayess/html";

function main(args) {
  var tokens = tokenizeHtml("<!DOCTYPE html><div id=a disabled><span>hi</span><!--note--><br/></div>");
  console.log("html-tokens:" + tokens.length);
  console.log("html-token-doctype:" + tokens[0].type + ":" + tokens[0].value + ":" + tokens[0].span.end.offset);
  console.log("html-token-start:" + tokens[1].type + ":" + tokens[1].tagName + ":" + tokens[1].attributes.id + ":" + tokens[1].attributes.disabled + ":" + tokens[1].selfClosing);
  console.log("html-token-text:" + tokens[3].type + ":" + tokens[3].value + ":" + tokens[3].span.start.offset + ":" + tokens[3].span.end.offset);
  console.log("html-token-comment:" + tokens[5].type + ":" + tokens[5].value);
  console.log("html-token-self-close:" + tokens[6].type + ":" + tokens[6].tagName + ":" + tokens[6].selfClosing);
  console.log("html-token-end:" + tokens[7].type + ":" + tokens[7].tagName);
  try {
    tokenizeHtml("<!--broken");
    console.log("html-token-error:false");
  } catch (err) {
    console.log("html-token-error:" + err.name);
  }

  var doc = parseHtml(fs.readFile("./sample.html", "utf8"));
  var root = doc.children[0];
  var span = root.children[0];
  var comment = root.children[1];
  var br = root.children[2];
  console.log("html-doc:" + doc.type + ":" + doc.children.length);
  console.log("html-doc-span:" + doc.span.start.line + ":" + doc.span.start.column + ":" + doc.span.end.offset);
  console.log("html-tag:" + root.tagName);
  console.log("html-root-span:" + root.span.start.offset + ":" + root.span.end.offset);
  console.log("html-attr:" + root.attributes.id + ":" + root.attributes.disabled);
  console.log("html-span:" + span.tagName + ":" + span.children[0].value);
  console.log("html-span-text-span:" + span.children[0].span.start.offset + ":" + span.children[0].span.end.offset);
  console.log("html-comment:" + comment.value);
  console.log("html-comment-span:" + comment.span.start.offset + ":" + comment.span.end.offset);
  console.log("html-br:" + br.tagName + ":" + br.selfClosing);

  var frag = parseHtmlFragment("<p class=x>ok</p>tail");
  console.log("html-frag:" + frag.type + ":" + frag.children.length + ":" + frag.children[1].value);

  var malformed = parseHtmlFragment("<div><span>x</div>");
  var malformedDiv = malformed.children[0];
  var malformedSpan = malformedDiv.children[0];
  console.log("html-malformed:" + malformedDiv.children.length + ":" + malformedSpan.tagName + ":" + malformedSpan.children[0].value);

  var truncated = parseHtmlFragment("<div><span>x");
  var truncatedDiv = truncated.children[0];
  var truncatedSpan = truncatedDiv.children[0];
  console.log("html-truncated:" + truncatedDiv.children.length + ":" + truncatedSpan.tagName + ":" + truncatedSpan.children[0].value);

  try {
    parseHtml("<!--broken");
    console.log("html-error:false");
  } catch (err) {
  console.log("html-error:" + err.name);
  }

  console.log("html-serialize:" + serializeHtml(doc));
  console.log("html-serialize-no-comments:" + serializeHtmlWithOptions(doc, { comments: false }));
  console.log("html-serialize-pretty:" + serializeHtmlWithOptions(doc, { pretty: true }));
  console.log("html-serialize-minify:" + serializeHtmlWithOptions(doc, { minify: true }));

  var built = createElement("section", undefined, undefined);
  appendChild(built, createText("lead"));
  appendChild(built, createComment("keep"));
  setAttribute(built, "id", "root");
  setAttribute(built, "class", "hero");
  var inner = createElement("span", undefined, undefined);
  setAttribute(inner, "id", "child");
  appendChild(inner, createText("x"));
  appendChild(built, inner);
  replaceChild(built, 0, createText("intro"));
  removeChild(built, 1);
  removeAttribute(built, "id");
  var clone = cloneNode(built);
  console.log("html-built:" + serializeHtml(built));
  console.log("html-clone:" + serializeHtml(clone));
  console.log("html-walk:" + walkDepthFirst(clone).length);
  console.log("html-find-span:" + findByTag(doc, "span").length);
  console.log("html-match-tag:" + matchesSelector(root, "div"));
  console.log("html-match-id:" + matchesSelector(root, "#a"));
  console.log("html-match-class:" + matchesSelector(frag.children[0], ".x"));
  console.log("html-match-attr:" + matchesSelector(root, "[disabled]"));
  console.log("html-select-desc:" + querySelectorAll(doc, "div span").length);
  console.log("html-select-child:" + querySelectorAll(doc, "div > span").length);
  console.log("html-select-attr-value:" + querySelectorAll(doc, "div[id=a]").length);
  console.log("html-select-first-child:" + querySelectorAll(doc, "div > span:first-child").length);
  console.log("html-select-last-child:" + querySelectorAll(doc, "div > br:last-child").length);
  console.log("html-select-empty:" + querySelectorAll(doc, "div > br:empty").length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "html-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled HTML program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"html-tokens:8",
		"html-token-doctype:doctype:html:15",
		"html-token-start:startTag:div:a:true:false",
		"html-token-text:text:hi:40:42",
		"html-token-comment:comment:note",
		"html-token-self-close:startTag:br:true",
		"html-token-end:endTag:div",
		"html-token-error:HTMLParseError",
		"html-doc:document:1",
		"html-doc-span:1:1:58",
		"html-tag:div",
		"html-root-span:0:58",
		"html-attr:a:true",
		"html-span:span:hi",
		"html-span-text-span:27:29",
		"html-comment:note",
		"html-comment-span:36:47",
		"html-br:br:true",
		"html-frag:fragment:2:tail",
		"html-malformed:1:span:x",
		"html-truncated:1:span:x",
		"html-error:HTMLParseError",
		`html-serialize:<div id="a" disabled><span>hi</span><!--note--><br/></div>`,
		`html-serialize-no-comments:<div id="a" disabled><span>hi</span><br/></div>`,
		"html-serialize-pretty:<div id=\"a\" disabled>\n  <span>hi</span>\n  <!--note-->\n  <br/>\n</div>",
		`html-serialize-minify:<div id="a" disabled><span>hi</span><!--note--><br/></div>`,
		`html-built:<section class="hero">intro<span id="child">x</span></section>`,
		`html-clone:<section class="hero">intro<span id="child">x</span></section>`,
		"html-walk:4",
		"html-find-span:1",
		"html-match-tag:true",
		"html-match-id:true",
		"html-match-class:true",
		"html-match-attr:true",
		"html-select-desc:1",
		"html-select-child:1",
		"html-select-attr-value:1",
		"html-select-first-child:1",
		"html-select-last-child:1",
		"html-select-empty:1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected HTML output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessXMLPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping XML package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "xml"),
		filepath.Join(workdir, "node_modules", "@jayess", "xml"),
	)

	sampleXML := `<?xml version="1.0"?><!--lead--><root id="a"><child>hi</child><![CDATA[<raw>]]><empty flag="x"/></root>`
	if err := os.WriteFile(filepath.Join(workdir, "sample.xml"), []byte(sampleXML), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "ns.xml"), []byte(`<ns:root xmlns:ns="urn:test" xmlns="urn:default"><ns:item ns:id="7"/><child plain="x"/></ns:root>`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeXml, parseXml, serializeXml, serializeXmlWithOptions } from "@jayess/xml";

function main(args) {
  var tokens = tokenizeXml(fs.readFile("./sample.xml", "utf8"));
  console.log("xml-tokens:" + tokens.length);
  console.log("xml-token-pi:" + tokens[0].type + ":" + tokens[0].target + ":" + tokens[0].data);
  console.log("xml-token-comment:" + tokens[1].type + ":" + tokens[1].value);
  console.log("xml-token-start:" + tokens[2].type + ":" + tokens[2].tagName + ":" + tokens[2].attributes.id);
  console.log("xml-token-cdata:" + tokens[6].type + ":" + tokens[6].value);
  console.log("xml-token-empty:" + tokens[7].type + ":" + tokens[7].tagName + ":" + tokens[7].selfClosing);
  console.log("xml-token-end:" + tokens[8].type + ":" + tokens[8].tagName);
  console.log("xml-token-span:" + tokens[2].span.start.offset + ":" + tokens[2].span.end.offset);

  var doc = parseXml(fs.readFile("./sample.xml", "utf8"));
  console.log("xml-doc:" + doc.type + ":" + doc.children.length);
  console.log("xml-doc-span:" + doc.span.start.line + ":" + doc.span.end.offset);
  console.log("xml-pi:" + doc.children[0].target + ":" + doc.children[0].data);
  console.log("xml-comment:" + doc.children[1].value);
  var root = doc.children[2];
  console.log("xml-root:" + root.tagName + ":" + root.attributes.id + ":" + root.children.length);
  console.log("xml-child:" + root.children[0].tagName + ":" + root.children[0].children[0].value);
  console.log("xml-cdata:" + root.children[1].type + ":" + root.children[1].value);
  console.log("xml-empty:" + root.children[2].tagName + ":" + root.children[2].attributes.flag + ":" + root.children[2].selfClosing);
  console.log("xml-root-span:" + root.span.start.offset + ":" + root.span.end.offset);
  console.log("xml-serialize:" + serializeXml(doc));
  console.log("xml-serialize-no-comments:" + serializeXmlWithOptions(doc, { comments: false }));
  console.log("xml-serialize-pretty:" + serializeXmlWithOptions(doc, { pretty: true }));
  console.log("xml-serialize-minify:" + serializeXmlWithOptions(doc, { minify: true }));

  var nsDoc = parseXml(fs.readFile("./ns.xml", "utf8"));
  var nsRoot = nsDoc.children[0];
  var nsItem = nsRoot.children[0];
  var defaultChild = nsRoot.children[1];
  console.log("xml-ns-root:" + nsRoot.tagName + ":" + nsRoot.prefix + ":" + nsRoot.localName + ":" + nsRoot.namespaceURI);
  console.log("xml-ns-item:" + nsItem.tagName + ":" + nsItem.prefix + ":" + nsItem.localName + ":" + nsItem.namespaceURI);
  console.log("xml-ns-default:" + defaultChild.tagName + ":" + defaultChild.localName + ":" + defaultChild.namespaceURI);
  console.log("xml-ns-attr:" + nsItem.attributeDetails["ns:id"].prefix + ":" + nsItem.attributeDetails["ns:id"].localName + ":" + nsItem.attributeDetails["ns:id"].namespaceURI);
  console.log("xml-ns-xmlns:" + nsRoot.attributeDetails["xmlns:ns"].namespaceURI);
  console.log("xml-ns-plain-attr:" + defaultChild.attributeDetails["plain"].localName + ":" + defaultChild.attributeDetails["plain"].namespaceURI);

  try {
    parseXml("<root><a></root>");
    console.log("xml-error-mismatch:false");
  } catch (err) {
    console.log("xml-error-mismatch:" + err.name);
  }

  try {
    parseXml("<root a=1/>");
    console.log("xml-error-attr:false");
  } catch (err) {
    console.log("xml-error-attr:" + err.name);
  }

  try {
    tokenizeXml("<!--broken");
    console.log("xml-token-error:false");
  } catch (err) {
    console.log("xml-token-error:" + err.name);
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

	outputPath := nativeOutputPath(workdir, "xml-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled XML program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"xml-tokens:9",
		"xml-token-pi:processingInstruction:xml:version=\"1.0\"",
		"xml-token-comment:comment:lead",
		"xml-token-start:startTag:root:a",
		"xml-token-cdata:cdata:<raw>",
		"xml-token-empty:startTag:empty:true",
		"xml-token-end:endTag:root",
		"xml-token-span:32:45",
		"xml-doc:document:3",
		"xml-doc-span:1:103",
		"xml-pi:xml:version=\"1.0\"",
		"xml-comment:lead",
		"xml-root:root:a:3",
		"xml-child:child:hi",
		"xml-cdata:cdata:<raw>",
		"xml-empty:empty:x:true",
		"xml-root-span:32:103",
		"xml-serialize:<?xml version=\"1.0\"?><!--lead--><root id=\"a\"><child>hi</child><![CDATA[<raw>]]><empty flag=\"x\"/></root>",
		"xml-serialize-no-comments:<?xml version=\"1.0\"?><root id=\"a\"><child>hi</child><![CDATA[<raw>]]><empty flag=\"x\"/></root>",
		"xml-serialize-pretty:<?xml version=\"1.0\"?><!--lead--><root id=\"a\">\n  <child>hi</child>\n  <![CDATA[<raw>]]>\n  <empty flag=\"x\"/>\n</root>",
		"xml-serialize-minify:<?xml version=\"1.0\"?><!--lead--><root id=\"a\"><child>hi</child><![CDATA[<raw>]]><empty flag=\"x\"/></root>",
		"xml-ns-root:ns:root:ns:root:urn:test",
		"xml-ns-item:ns:item:ns:item:urn:test",
		"xml-ns-default:child:child:urn:default",
		"xml-ns-attr:ns:id:urn:test",
		"xml-ns-xmlns:http://www.w3.org/2000/xmlns/",
		"xml-ns-plain-attr:plain:undefined",
		"xml-error-mismatch:XMLParseError",
		"xml-error-attr:XMLParseError",
		"xml-token-error:XMLParseError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected XML output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessCSSPackage(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping CSS package test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "css"),
		filepath.Join(workdir, "node_modules", "@jayess", "css"),
	)

	sampleCSS := `@import "theme.css";
/*lead*/
@media screen and (min-width: 600px) { .card { padding: 8px; } }
.btn.primary, #app > .item { color: red; margin: 1.5rem; content: "hi"; }
#footer { padding: 8px; }`
	if err := os.WriteFile(filepath.Join(workdir, "sample.css"), []byte(sampleCSS), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeCss, parseCss, serializeCss, serializeCssWithOptions } from "@jayess/css";

function main(args) {
  var source = fs.readFile("./sample.css", "utf8");
  var tokens = tokenizeCss(source);
  console.log("css-tokens:" + tokens.length);
  console.log("css-token-at:" + tokens[0].type + ":" + tokens[0].value);
  console.log("css-token-import-string:" + tokens[1].type + ":" + tokens[1].value);
  console.log("css-token-comment:" + tokens[3].type + ":" + tokens[3].value);
  console.log("css-token-dimension:" + tokens[24].type + ":" + tokens[24].value);
  console.log("css-token-string:" + tokens[29].type + ":" + tokens[29].value);

  var sheet = parseCss(source);
  console.log("css-sheet:" + sheet.type + ":" + sheet.rules.length);
  console.log("css-sheet-span:" + sheet.span.start.line + ":" + sheet.span.end.offset);
  console.log("css-import:" + sheet.rules[0].type + ":" + sheet.rules[0].name + ":" + sheet.rules[0].prelude);
  console.log("css-comment:" + sheet.rules[1].type + ":" + sheet.rules[1].value);
  console.log("css-media:" + sheet.rules[2].type + ":" + sheet.rules[2].name + ":" + sheet.rules[2].prelude + ":" + sheet.rules[2].rules.length);
  var mediaRule = sheet.rules[2].rules[0];
  console.log("css-media-rule:" + mediaRule.selector + ":" + mediaRule.declarations[0].property + ":" + mediaRule.declarations[0].value);
  var rule = sheet.rules[3];
  console.log("css-rule:" + rule.type + ":" + rule.selector + ":" + rule.declarations.length);
  console.log("css-selector-tokens:" + rule.selectorTokens.length);
  console.log("css-decl-color:" + rule.declarations[0].property + ":" + rule.declarations[0].value + ":" + rule.declarations[0].valueParts[0].type);
  console.log("css-decl-margin:" + rule.declarations[1].property + ":" + rule.declarations[1].value + ":" + rule.declarations[1].valueParts[0].type);
  console.log("css-decl-content:" + rule.declarations[2].property + ":" + rule.declarations[2].value + ":" + rule.declarations[2].valueParts[0].type);
  console.log("css-rule-order:" + sheet.rules[4].selector);
  console.log("css-serialize:" + serializeCss(sheet));
  console.log("css-serialize-no-comments:" + serializeCssWithOptions(sheet, { comments: false }));
  console.log("css-serialize-pretty:" + serializeCssWithOptions(sheet, { pretty: true }));
  console.log("css-serialize-minify:" + serializeCssWithOptions(sheet, { minify: true }));

  try {
    parseCss("@supports (display: grid) {}");
    console.log("css-error-unsupported-at-rule:false");
  } catch (err) {
    console.log("css-error-unsupported-at-rule:" + err.name);
  }

  try {
    parseCss(".x { color red; }");
    console.log("css-error-decl:false");
  } catch (err) {
    console.log("css-error-decl:" + err.name);
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

	outputPath := nativeOutputPath(workdir, "css-native")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled CSS program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"css-token-at:atKeyword:import",
		"css-token-import-string:string:theme.css",
		"css-token-comment:comment:lead",
		"css-sheet:stylesheet:5",
		"css-import:atRule:import:\"theme.css\"",
		"css-comment:comment:lead",
		"css-media:atRule:media:screen and (min-width: 600px):1",
		"css-media-rule:.card:padding:8px",
		"css-rule:rule:.btn.primary, #app > .item:3",
		"css-decl-color:color:red:ident",
		"css-decl-margin:margin:1.5rem:dimension",
		"css-decl-content:content:\"hi\":string",
		"css-rule-order:#footer",
		`css-serialize:@import "theme.css";/*lead*/@media screen and (min-width: 600px){.card{padding:8px;}}.btn.primary, #app > .item{color:red;margin:1.5rem;content:"hi";}#footer{padding:8px;}`,
		`css-serialize-no-comments:@import "theme.css";@media screen and (min-width: 600px){.card{padding:8px;}}.btn.primary, #app > .item{color:red;margin:1.5rem;content:"hi";}#footer{padding:8px;}`,
		"css-serialize-pretty:@import \"theme.css\";/*lead*/@media screen and (min-width: 600px){\n  .card{\n    padding: 8px;\n  }\n}.btn.primary, #app > .item{\n  color: red;\n  margin: 1.5rem;\n  content: \"hi\";\n}#footer{\n  padding: 8px;\n}",
		`css-serialize-minify:@import "theme.css";/*lead*/@media screen and (min-width: 600px){.card{padding:8px;}}.btn.primary, #app > .item{color:red;margin:1.5rem;content:"hi";}#footer{padding:8px;}`,
		"css-error-unsupported-at-rule:CSSParseError",
		"css-error-decl:CSSParseError",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected CSS output to contain %q, got: %s", want, text)
		}
	}
}

func buildExecutableWithAddressSanitizer(t *testing.T, tc *Toolchain, result *compiler.Result, opts compiler.Options, outputPath string) error {
	t.Helper()

	tempDir := t.TempDir()
	irPath := filepath.Join(tempDir, "module.ll")
	if err := os.WriteFile(irPath, result.LLVMIR, 0o644); err != nil {
		return fmt.Errorf("write temporary LLVM IR: %w", err)
	}

	runtimePaths, err := runtimeSourcePaths()
	if err != nil {
		return fmt.Errorf("resolve runtime sources: %w", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		return fmt.Errorf("resolve runtime include directory: %w", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()

	sanitizeFlags := []string{"-O0", "-fsanitize=address", "-fno-omit-frame-pointer"}
	var objectPaths []string

	moduleObjectPath := filepath.Join(tempDir, "module.o")
	args := []string{"-target", opts.TargetTriple}
	args = append(args, sanitizeFlags...)
	args = append(args, "-c", irPath, "-o", moduleObjectPath)
	cmd := exec.Command(tc.ClangPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("asan clang object build failed: %w: %s", err, string(output))
	}
	objectPaths = append(objectPaths, moduleObjectPath)

	compileSource := func(sourcePath string, includeDirs, compileFlags []string, objectPath string) error {
		sourceArgs := []string{"-target", opts.TargetTriple}
		sourceArgs = append(sourceArgs, sanitizeFlags...)
		for _, includeDir := range includeDirs {
			sourceArgs = append(sourceArgs, "-I", includeDir)
		}
		sourceArgs = append(sourceArgs, compileFlags...)
		sourceArgs = append(sourceArgs, "-c", sourcePath, "-o", objectPath)
		sourceCmd := exec.Command(tc.ClangPath, sourceArgs...)
		if output, err := sourceCmd.CombinedOutput(); err != nil {
			return formatNativeBuildError(err, string(output))
		}
		return nil
	}

	runtimeIncludeDirs := []string{runtimeIncludeDir}
	if brotliAvailable {
		runtimeIncludeDirs = append(runtimeIncludeDirs, brotliIncludeDir)
	}
	for i, runtimePath := range runtimePaths {
		runtimeObjectPath := filepath.Join(tempDir, fmt.Sprintf("runtime-%d.o", i))
		if err := compileSource(runtimePath, runtimeIncludeDirs, nil, runtimeObjectPath); err != nil {
			return err
		}
		objectPaths = append(objectPaths, runtimeObjectPath)
	}

	if brotliAvailable {
		for i, source := range brotliSources {
			objectPath := filepath.Join(tempDir, fmt.Sprintf("brotli-%d.o", i))
			if err := compileSource(source, []string{brotliIncludeDir}, nil, objectPath); err != nil {
				return err
			}
			objectPaths = append(objectPaths, objectPath)
		}
	}

	nativeIncludeDirs := append([]string{runtimeIncludeDir}, result.NativeIncludeDirs...)
	for i, source := range result.NativeImports {
		objectPath := filepath.Join(tempDir, fmt.Sprintf("native-%d.o", i))
		if err := compileSource(source, nativeIncludeDirs, result.NativeCompileFlags, objectPath); err != nil {
			return err
		}
		objectPaths = append(objectPaths, objectPath)
	}

	linkArgs := []string{"-target", opts.TargetTriple}
	linkArgs = append(linkArgs, sanitizeFlags...)
	linkArgs = append(linkArgs, objectPaths...)
	linkArgs = append(linkArgs, nativeSystemLinkFlags(opts.TargetTriple)...)
	linkArgs = append(linkArgs, result.NativeLinkFlags...)
	linkArgs = append(linkArgs, "-o", outputPath)
	linkCmd := exec.Command(tc.ClangPath, linkArgs...)
	if output, err := linkCmd.CombinedOutput(); err != nil {
		return formatNativeBuildError(err, string(output))
	}

	return nil
}

func TestBuildExecutableParserPackagesAreLeakFreeUnderASAN(t *testing.T) {
	if os.Getenv("JAYESS_RUN_PARSER_ASAN_PROBE") != "1" {
		t.Skip("skipping parser ASAN leak probe; set JAYESS_RUN_PARSER_ASAN_PROBE=1 to run")
	}
	if runtime.GOOS != "linux" {
		t.Skip("ASAN/LSAN parser leak probe is only exercised on Linux")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping parser ASAN test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}
	if !strings.Contains(triple, "linux") {
		t.Skipf("skipping parser ASAN test for non-Linux triple %q", triple)
	}

	repoRoot := repoRootFromBackendTest(t)
	cases := []struct {
		name      string
		pkg       string
		sampleRel string
		sample    string
		source    string
		want      string
	}{
		{
			name:      "html",
			pkg:       "html",
			sampleRel: "sample.html",
			sample:    `<!doctype html><div id="a"><span>hi</span><!--note--><br/></div>`,
			source: `
import { parseHtml, serializeHtml } from "@jayess/html";

function main(args) {
  var doc = parseHtml(fs.readFile("./sample.html", "utf8"));
  console.log("asan-html:" + serializeHtml(doc));
  return 0;
}
`,
			want: `asan-html:<!doctype html><div id="a"><span>hi</span><!--note--><br/></div>`,
		},
		{
			name:      "xml",
			pkg:       "xml",
			sampleRel: "sample.xml",
			sample:    `<?xml version="1.0"?><root><child>hi</child><![CDATA[<raw>]]></root>`,
			source: `
import { parseXml, serializeXml } from "@jayess/xml";

function main(args) {
  var doc = parseXml(fs.readFile("./sample.xml", "utf8"));
  console.log("asan-xml:" + serializeXml(doc));
  return 0;
}
`,
			want: `asan-xml:<?xml version="1.0"?><root><child>hi</child><![CDATA[<raw>]]></root>`,
		},
		{
			name:      "css",
			pkg:       "css",
			sampleRel: "sample.css",
			sample:    `@import "theme.css"; .btn { color: red; margin: 1.5rem; }`,
			source: `
import { parseCss, serializeCss } from "@jayess/css";

function main(args) {
  var sheet = parseCss(fs.readFile("./sample.css", "utf8"));
  console.log("asan-css:" + serializeCss(sheet));
  return 0;
}
`,
			want: `asan-css:@import "theme.css";.btn{color:red;margin:1.5rem;}`,
		},
	}

	for _, tcCase := range cases {
		t.Run(tcCase.name, func(t *testing.T) {
			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", tcCase.pkg),
				filepath.Join(workdir, "node_modules", "@jayess", tcCase.pkg),
			)
			if err := os.WriteFile(filepath.Join(workdir, tcCase.sampleRel), []byte(tcCase.sample), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			entry := filepath.Join(workdir, "main.js")
			if err := os.WriteFile(entry, []byte(tcCase.source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := nativeOutputPath(workdir, "parser-asan-native")
			if err := buildExecutableWithAddressSanitizer(t, tc, result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Fatalf("buildExecutableWithAddressSanitizer returned error: %v", err)
			}

			cmd := exec.Command(outputPath)
			cmd.Dir = workdir
			cmd.Env = append(os.Environ(),
				"ASAN_OPTIONS=detect_leaks=1:halt_on_error=1:exitcode=66",
				"LSAN_OPTIONS=exitcode=66",
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compiled ASAN parser program returned error: %v: %s", err, string(out))
			}
			text := string(out)
			if !strings.Contains(text, tcCase.want) {
				t.Fatalf("expected ASAN parser output to contain %q, got: %s", tcCase.want, text)
			}
		})
	}
}

func TestBuildExecutableScopeCleanupStaysSafeUnderASAN(t *testing.T) {
	if os.Getenv("JAYESS_RUN_LIFETIME_ASAN_PROBE") != "1" {
		t.Skip("skipping lifetime ASAN probe; set JAYESS_RUN_LIFETIME_ASAN_PROBE=1 to run")
	}
	if runtime.GOOS != "linux" {
		t.Skip("ASAN/LSAN lifetime probe is only exercised on Linux")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping lifetime ASAN test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}
	if !strings.Contains(triple, "linux") {
		t.Skipf("skipping lifetime ASAN test for non-Linux triple %q", triple)
	}

	workdir := t.TempDir()
	prepareCleanupProbePackage(t, workdir)

	entry := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { makeProbe, closeProbe, cleanupLog, resetCleanupLog } from "@jayess/cleanupprobe";

class FreshBox {}
class PlainCtorBox {
  constructor() {
    this.kind = "plain";
  }
}

class FreshReturnCtorBox {
  constructor() {
    return { kind: "alt-fresh" };
  }
}

function functionScopedCleanup() {
  var scoped = makeProbe("function-var");
  return 11;
}

function innerReturn() {
  const value = makeProbe("return");
  return 7;
}

function freshObjectTemp() {
  return { kind: "fresh-call" };
}

function freshInvokeObject() {
  return { kind: "invoke-fresh" };
}

function freshBox() {
  return new FreshBox();
}

function freshSwitchCase() {
  switch ("case-" + "a") {
    case "case-a":
      break;
    case "case-b":
      break;
  }
}

function boundOffset(offset, x) {
  return x + offset;
}

function largeOffset(x) {
  return x + 20;
}

function boundGreaterThan(min, x) {
  return x > min;
}

function boundEquals(expected, x) {
  return x == expected;
}

function boundPairSum(a, b, x) {
  return x + a + b;
}

function boundBetween(min, max, x) {
  return x > min && x < max;
}

function boundTripleEquals(a, b, x) {
  return x == a + b;
}

function boundTripleSum(a, b, c, x) {
  return x + a + b + c;
}

function boundWindow(min, mid, max, x) {
  return x > min && x < max && x != mid;
}

function boundQuadEquals(a, b, c, x) {
  return x == a + b + c;
}

function boundQuadSum(a, b, c, d, x) {
  return x + a + b + c + d;
}

function boundOuterWindow(min, low, high, max, x) {
  return x > min && x >= low && x < max && x != high;
}

function boundQuintEquals(a, b, c, d, x) {
  return x == a + b + c + d;
}

function boundQuintSum(a, b, c, d, e, x) {
  return x + a + b + c + d + e;
}

function boundSextEquals(a, b, c, d, e, x) {
  return x == a + b + c + d + e;
}

function boundSextSum(a, b, c, d, e, f, x) {
  return x + a + b + c + d + e + f;
}

function boundSeptEquals(a, b, c, d, e, f, x) {
  return x == a + b + c + d + e + f;
}

function boundSeptSum(a, b, c, d, e, f, g, x) {
  return x + a + b + c + d + e + f + g;
}

function boundOctEquals(a, b, c, d, e, f, g, x) {
  return x == a + b + c + d + e + f + g;
}

function boundOctSum(a, b, c, d, e, f, g, h, x) {
  return x + a + b + c + d + e + f + g + h;
}

function boundNonetEquals(a, b, c, d, e, f, g, h, x) {
  return x == a + b + c + d + e + f + g + h;
}

function boundNonetSum(a, b, c, d, e, f, g, h, i, x) {
  return x + a + b + c + d + e + f + g + h + i;
}

function boundDecetEquals(a, b, c, d, e, f, g, h, i, x) {
  return x == a + b + c + d + e + f + g + h + i;
}

function boundDecetSum(a, b, c, d, e, f, g, h, i, j, x) {
  return x + a + b + c + d + e + f + g + h + i + j;
}

function boundUndecEquals(a, b, c, d, e, f, g, h, i, j, x) {
  return x == a + b + c + d + e + f + g + h + i + j;
}

function boundUndecSum(a, b, c, d, e, f, g, h, i, j, k, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k;
}

function boundDuodecEquals(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x == a + b + c + d + e + f + g + h + i + j + k + l;
}

function boundDuodecSum(a, b, c, d, e, f, g, h, i, j, k, l, x) {
  return x + a + b + c + d + e + f + g + h + i + j + k + l;
}


function discardedFreshTemporaries() {
  freshObjectTemp();
  freshSwitchCase();
  (() => "fresh-fn");
  new PlainCtorBox();
  new FreshReturnCtorBox();
  ({ name: "kimchi" });
  ({ answer: 41 }).answer;
  ({ label: "index" })["label"];
  ({ maybe: "opt-member" })?.maybe;
  ({ maybe: "opt-index" })?.["maybe"];
  "soup".length;
  [1, 2, 3];
  `+"`soup${1}`"+`;
  "left" + "right";
  ~1;
  1n & 3n;
  1n === 1n;
  ("cmp-left" + "x") === ("cmp-right" + "y");
  !("not-left" + "right");
  ("and-left" + "x") && ("and-right" + "y");
  ("or-left" + "x") || ("or-right" + "y");
  typeof ("type" + "of");
  freshBox() instanceof FreshBox;
  ("ok" is "ok" | "error");
  ([1, "ok"] is [number, string]);
  ({ kind: "ok", value: 3 } is { kind: "ok", value: number } | { kind: "error", message: string });
  true ? ({ kind: "conditional" }) : ({ kind: "fallback" });
  null ?? ({ kind: "nullish" });
  (({ kind: "comma-left" }), ({ kind: "comma-right" }));
  freshInvokeObject.bind(null);
  freshInvokeObject.call(null);
  freshInvokeObject.apply(null, []);
  [1, 2].forEach((x) => 0);
  [1, 2].map((x) => x + 1);
  [1, 2].filter((x) => x > 0);
  [1, 2].find((x) => false);
  [1, 2].forEach(boundOffset.bind(null, 1));
  [1, 2].forEach(boundOffset.bind(null, 20));
  [1, 2].forEach(largeOffset);
  [1, 2].map(boundOffset.bind(null, 1));
  [1, 2].map(boundOffset.bind(null, 20));
  [1, 2].filter(largeOffset);
  [1, 2].filter(boundGreaterThan.bind(null, 0));
  [1, 2].filter(boundOffset.bind(null, 20));
  [1, 2].find(largeOffset);
  [1, 2].find(boundEquals.bind(null, 9));
  [1, 2].find(boundOffset.bind(null, 20));
  [1, 2].forEach(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 1, 2));
  [1, 2].map(boundPairSum.bind(null, 10, 10));
  [1, 2].filter(boundBetween.bind(null, 0, 3));
  [1, 2].find(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleEquals.bind(null, 4, 5));
  [1, 2].forEach(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].forEach(boundPairSum.bind(null, 10, 10));
  [1, 2].map(boundTripleSum.bind(null, 1, 2, 3));
  [1, 2].map(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundWindow.bind(null, 0, 1, 3));
  [1, 2].filter(boundPairSum.bind(null, 10, 10));
  [1, 2].find(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].find(boundQuadEquals.bind(null, 3, 4, 5));
  [1, 2].forEach(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].forEach(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].forEach(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].map(boundQuadSum.bind(null, 1, 2, 3, 4));
  [1, 2].filter(boundOuterWindow.bind(null, 0, 1, 4, 3));
  [1, 2].filter(boundTripleSum.bind(null, 10, 10, 10));
  [1, 2].filter(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].find(boundQuadSum.bind(null, 4, 4, 4, 4));
  [1, 2].forEach(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].map(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].filter(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintSum.bind(null, 1, 1, 1, 1, 16));
  [1, 2].find(boundQuintEquals.bind(null, 30, 30, 30, 30, 30));
  [1, 2].find(boundQuintEquals.bind(null, 3, 4, 5, 6));
  [1, 2].forEach(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextSum.bind(null, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSextEquals.bind(null, 40, 40, 40, 40, 40, 40));
  [1, 2].forEach(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptSum.bind(null, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundSeptEquals.bind(null, 50, 50, 50, 50, 50, 50, 50));
  [1, 2].forEach(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundOctEquals.bind(null, 60, 60, 60, 60, 60, 60, 60, 60));
  [1, 2].forEach(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundNonetEquals.bind(null, 70, 70, 70, 70, 70, 70, 70, 70, 70));
  [1, 2].forEach(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDecetEquals.bind(null, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80));
  [1, 2].forEach(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundUndecEquals.bind(null, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90));
  [1, 2].forEach(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].map(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].filter(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecSum.bind(null, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 16));
  [1, 2].find(boundDuodecEquals.bind(null, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100));
}

function main(args) {
  resetCleanupLog();
  {
    const scoped = makeProbe("block");
  }
  {
    const closed = makeProbe("manual-close");
    closeProbe(closed);
  }
  discardedFreshTemporaries();
  functionScopedCleanup();
  innerReturn();
  try {
    const thrown = makeProbe("throw");
    throw "boom";
  } catch (err) {
    console.log("asan-catch:" + err);
  }
  for (var i = 0; i < 4; i = i + 1) {
    const loopScoped = makeProbe("continue" + i);
    if (i == 1) {
      continue;
    }
    if (i == 2) {
      break;
    }
  }
  for (var j = 0; j < 1; j = j + 1) {
    const outerBreak = makeProbe("outer-break");
    {
      const innerBreak = makeProbe("inner-break");
      break;
    }
  }
  for (var k = 0; k < 1; k = k + 1) {
    const outerContinue = makeProbe("outer-continue");
    {
      const innerContinue = makeProbe("inner-continue");
      continue;
    }
  }
  for (var m = 0; m < 1; m = m + 1) {
    const outerThrow = makeProbe("outer-throw");
    try {
      const innerThrow = makeProbe("inner-throw");
      throw "nested";
    } catch (err) {
      console.log("asan-nested-catch:" + err);
    }
  }
  console.log("asan-cleanup:" + cleanupLog());
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "scope-cleanup-asan-native")
	if err := buildExecutableWithAddressSanitizer(t, tc, result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("buildExecutableWithAddressSanitizer returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"ASAN_OPTIONS=detect_leaks=1:halt_on_error=1:exitcode=66",
		"LSAN_OPTIONS=exitcode=66",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled ASAN cleanup program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"block;",
		"manual-close;",
		"function-var;",
		"return;",
		"throw;",
		"continue0;",
		"continue1;",
		"continue2;",
		"inner-break;",
		"outer-break;",
		"inner-continue;",
		"outer-continue;",
		"inner-throw;",
		"outer-throw;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected ASAN cleanup output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableEscapingLocalsStaySafeUnderASAN(t *testing.T) {
	if os.Getenv("JAYESS_RUN_LIFETIME_ASAN_PROBE") != "1" {
		t.Skip("skipping lifetime ASAN probe; set JAYESS_RUN_LIFETIME_ASAN_PROBE=1 to run")
	}
	if runtime.GOOS != "linux" {
		t.Skip("ASAN/LSAN lifetime probe is only exercised on Linux")
	}

	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping lifetime ASAN test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}
	if !strings.Contains(triple, "linux") {
		t.Skipf("skipping lifetime ASAN test for non-Linux triple %q", triple)
	}

	result, err := compiler.Compile(`
var globalBox = undefined;

function makeReturnedObject() {
  var local = { label: "object-ok", nested: { value: 41 } };
  return local;
}

function makeReturnedArray() {
  var local = [3, 4, 5];
  return local;
}

function makeStoredObject() {
  var local = { value: "stored-object" };
  var holder = {};
  holder.item = local;
  return holder;
}

function makeStoredArray() {
  var local = { value: "stored-array" };
  var holder = [];
  holder[0] = local;
  return holder;
}

function makeClosure() {
  var local = { value: 9 };
  return () => local.value + 1;
}

function seedGlobal() {
  var local = { value: "global-object" };
  globalBox = local;
}

function main(args) {
  var returnedObject = makeReturnedObject();
  var returnedArray = makeReturnedArray();
  var storedObject = makeStoredObject();
  var storedArray = makeStoredArray();
  var closure = makeClosure();
  seedGlobal();

  console.log("asan-escape-object:" + returnedObject.label + ":" + returnedObject.nested.value);
  console.log("asan-escape-array:" + returnedArray.length + ":" + returnedArray[0] + ":" + returnedArray[2]);
  console.log("asan-escape-stored-object:" + storedObject.item.value);
  console.log("asan-escape-stored-array:" + storedArray[0].value);
  console.log("asan-escape-closure:" + closure());
  console.log("asan-escape-global:" + globalBox.value);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(t.TempDir(), "escaping-locals-asan-native")
	if err := buildExecutableWithAddressSanitizer(t, tc, result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("buildExecutableWithAddressSanitizer returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Env = append(os.Environ(),
		"ASAN_OPTIONS=detect_leaks=1:halt_on_error=1:exitcode=66",
		"LSAN_OPTIONS=exitcode=66",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled ASAN escaping-locals program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"asan-escape-object:object-ok:41",
		"asan-escape-array:3:3:5",
		"asan-escape-stored-object:stored-object",
		"asan-escape-stored-array:stored-array",
		"asan-escape-closure:10",
		"asan-escape-global:global-object",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected ASAN escaping-locals output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessParserPackagesThroughLocalModules(t *testing.T) {
	toolchain, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping parser module integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	cases := []struct {
		name         string
		pkg          string
		moduleSource string
		entrySource  string
		wants        []string
	}{
		{
			name: "html",
			pkg:  "html",
			moduleSource: `
import { parseHtmlFragment, querySelectorAll } from "@jayess/html";

export function htmlSpanCount(source) {
  var frag = parseHtmlFragment(source);
  return querySelectorAll(frag, "section > span").length + ":" + frag.children.length;
}
`,
			entrySource: `
import { htmlSpanCount } from "./lib/parsers.js";

function main(args) {
  console.log("parser-module-html:" + htmlSpanCount("<section><span>a</span><span>b</span></section>"));
  return 0;
}
`,
			wants: []string{"parser-module-html:2:1"},
		},
		{
			name: "xml",
			pkg:  "xml",
			moduleSource: `
import { parseXml, serializeXmlWithOptions } from "@jayess/xml";

export function xmlSummary(source) {
  var doc = parseXml(source);
  return doc.children[0].tagName + ":" + serializeXmlWithOptions(doc, { comments: false });
}
`,
			entrySource: `
import { xmlSummary } from "./lib/parsers.js";

function main(args) {
  console.log("parser-module-xml:" + xmlSummary("<root><!--note--><child/></root>"));
  return 0;
}
`,
			wants: []string{`parser-module-xml:root:<root><child/></root>`},
		},
		{
			name: "css",
			pkg:  "css",
			moduleSource: `
import { parseCss, serializeCssWithOptions } from "@jayess/css";

export function cssSummary(source) {
  var sheet = parseCss(source);
  return sheet.rules.length + ":" + serializeCssWithOptions(sheet, { minify: true });
}
`,
			entrySource: `
import { cssSummary } from "./lib/parsers.js";

function main(args) {
  console.log("parser-module-css:" + cssSummary("/*lead*/ .a { color: red; } .b { margin: 2px; }"));
  return 0;
}
`,
			wants: []string{"parser-module-css:3:/*lead*/.a{color:red;}.b{margin:2px;}"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", tc.pkg),
				filepath.Join(workdir, "node_modules", "@jayess", tc.pkg),
			)

			libDir := filepath.Join(workdir, "lib")
			if err := os.MkdirAll(libDir, 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			if err := os.WriteFile(filepath.Join(libDir, "parsers.js"), []byte(tc.moduleSource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			entry := filepath.Join(workdir, "main.js")
			if err := os.WriteFile(entry, []byte(tc.entrySource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := nativeOutputPath(workdir, "parser-modules-native")
			if err := toolchain.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Fatalf("BuildExecutable returned error: %v", err)
			}

			cmd := exec.Command(outputPath)
			cmd.Dir = workdir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compiled parser module program returned error: %v: %s", err, string(out))
			}
			text := string(out)
			for _, want := range tc.wants {
				if !strings.Contains(text, want) {
					t.Fatalf("expected parser module output to contain %q, got: %s", want, text)
				}
			}
		})
	}
}

func TestBuildExecutableSupportsJayessLargeParserInputs(t *testing.T) {
	toolchain, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping parser large-input test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	type parserCase struct {
		name        string
		pkg         string
		sampleName  string
		sample      string
		entrySource string
		want        string
	}

	var htmlBuilder strings.Builder
	htmlBuilder.WriteString("<div>")
	for i := 0; i < 3000; i++ {
		htmlBuilder.WriteString("<span class=\"x\">")
		htmlBuilder.WriteString(strconv.Itoa(i))
		htmlBuilder.WriteString("</span>")
	}
	htmlBuilder.WriteString("</div>")

	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<root>")
	for i := 0; i < 3000; i++ {
		xmlBuilder.WriteString("<item id=\"")
		xmlBuilder.WriteString(strconv.Itoa(i))
		xmlBuilder.WriteString("\">")
		xmlBuilder.WriteString(strconv.Itoa(i))
		xmlBuilder.WriteString("</item>")
	}
	xmlBuilder.WriteString("</root>")

	var cssBuilder strings.Builder
	for i := 0; i < 3000; i++ {
		cssBuilder.WriteString(".c")
		cssBuilder.WriteString(strconv.Itoa(i))
		cssBuilder.WriteString(" { width: ")
		cssBuilder.WriteString(strconv.Itoa(i))
		cssBuilder.WriteString("px; height: 1px; }\n")
	}

	cases := []parserCase{
		{
			name:       "html",
			pkg:        "html",
			sampleName: "large.html",
			sample:     htmlBuilder.String(),
			entrySource: `
import { parseHtml, querySelectorAll } from "@jayess/html";

function main(args) {
  var doc = parseHtml(fs.readFile("./large.html", "utf8"));
  var root = doc.children[0];
  console.log("html-large:" + root.children.length + ":" + querySelectorAll(doc, "div > span").length);
  return 0;
}
`,
			want: "html-large:3000:3000",
		},
		{
			name:       "xml",
			pkg:        "xml",
			sampleName: "large.xml",
			sample:     xmlBuilder.String(),
			entrySource: `
import { parseXml } from "@jayess/xml";

function main(args) {
  var doc = parseXml(fs.readFile("./large.xml", "utf8"));
  var root = doc.children[0];
  console.log("xml-large:" + root.children.length + ":" + root.children[2999].attributes.id);
  return 0;
}
`,
			want: "xml-large:3000:2999",
		},
		{
			name:       "css",
			pkg:        "css",
			sampleName: "large.css",
			sample:     cssBuilder.String(),
			entrySource: `
import { parseCss } from "@jayess/css";

function main(args) {
  var sheet = parseCss(fs.readFile("./large.css", "utf8"));
  console.log("css-large:" + sheet.rules.length + ":" + sheet.rules[2999].selector);
  return 0;
}
`,
			want: "css-large:3000:.c2999",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", tc.pkg),
				filepath.Join(workdir, "node_modules", "@jayess", tc.pkg),
			)
			if err := os.WriteFile(filepath.Join(workdir, tc.sampleName), []byte(tc.sample), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			entry := filepath.Join(workdir, "main.js")
			if err := os.WriteFile(entry, []byte(tc.entrySource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := nativeOutputPath(workdir, "parser-large-native")
			if err := toolchain.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Fatalf("BuildExecutable returned error: %v", err)
			}

			cmd := exec.Command(outputPath)
			cmd.Dir = workdir
			start := time.Now()
			out, err := cmd.CombinedOutput()
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("compiled large-parser program returned error: %v: %s", err, string(out))
			}
			if elapsed > 10*time.Second {
				t.Fatalf("expected large parser runtime to stay within 10s, got %v", elapsed)
			}
			text := string(out)
			if !strings.Contains(text, tc.want) {
				t.Fatalf("expected large parser output to contain %q, got: %s", tc.want, text)
			}
		})
	}
}

func TestBuildExecutableParserSpansAlignWithCompilerDiagnostics(t *testing.T) {
	toolchain, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping parser/compiler span alignment test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	type parserCase struct {
		name           string
		pkg            string
		sampleName     string
		sample         string
		entrySource    string
		want           string
		diagSource     string
		wantDiagLine   int
		wantDiagColumn int
	}

	cases := []parserCase{
		{
			name:       "html",
			pkg:        "html",
			sampleName: "sample.html",
			sample: `<div>
  <span>ok</span>
</div>`,
			entrySource: `
import { parseHtml } from "@jayess/html";

function main(args) {
  var doc = parseHtml(fs.readFile("./sample.html", "utf8"));
  var span = doc.children[0].children[1];
  console.log("parser-span:" + span.span.start.line + ":" + span.span.start.column);
  return 0;
}
`,
			want: "parser-span:2:3",
			diagSource: `
function main(args) {
  @;
}
`,
			wantDiagLine:   3,
			wantDiagColumn: 3,
		},
		{
			name:       "xml",
			pkg:        "xml",
			sampleName: "sample.xml",
			sample: `<root>
  <child>ok</child>
</root>`,
			entrySource: `
import { parseXml } from "@jayess/xml";

function main(args) {
  var doc = parseXml(fs.readFile("./sample.xml", "utf8"));
  var child = doc.children[0].children[1];
  console.log("parser-span:" + child.span.start.line + ":" + child.span.start.column);
  return 0;
}
`,
			want: "parser-span:2:3",
			diagSource: `
function main(args) {
  @;
}
`,
			wantDiagLine:   3,
			wantDiagColumn: 3,
		},
		{
			name:       "css",
			pkg:        "css",
			sampleName: "sample.css",
			sample: `

.rule { color: red; }
`,
			entrySource: `
import { parseCss } from "@jayess/css";

function main(args) {
  var sheet = parseCss(fs.readFile("./sample.css", "utf8"));
  var rule = sheet.rules[0];
  console.log("parser-span:" + rule.span.start.line + ":" + rule.span.start.column);
  return 0;
}
`,
			want: "parser-span:3:1",
			diagSource: `

@
function main(args) {
  return 0;
}
`,
			wantDiagLine:   3,
			wantDiagColumn: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workdir := t.TempDir()
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", tc.pkg),
				filepath.Join(workdir, "node_modules", "@jayess", tc.pkg),
			)
			if err := os.WriteFile(filepath.Join(workdir, tc.sampleName), []byte(tc.sample), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			entry := filepath.Join(workdir, "main.js")
			if err := os.WriteFile(entry, []byte(tc.entrySource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := compiler.CompilePath(entry, compiler.Options{TargetTriple: triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}

			outputPath := nativeOutputPath(workdir, "parser-span-native")
			if err := toolchain.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Fatalf("BuildExecutable returned error: %v", err)
			}

			cmd := exec.Command(outputPath)
			cmd.Dir = workdir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compiled parser span program returned error: %v: %s", err, string(out))
			}
			text := string(out)
			if !strings.Contains(text, tc.want) {
				t.Fatalf("expected parser span output to contain %q, got: %s", tc.want, text)
			}

			_, err = compiler.Compile(tc.diagSource, compiler.Options{TargetTriple: triple})
			if err == nil {
				t.Fatalf("expected compiler diagnostic")
			}
			var compileErr *compiler.CompileError
			if !errors.As(err, &compileErr) {
				t.Fatalf("expected CompileError, got %T: %v", err, err)
			}
			if compileErr.Diagnostic.Line != tc.wantDiagLine || compileErr.Diagnostic.Column != tc.wantDiagColumn {
				t.Fatalf("expected compiler diagnostic at %d:%d, got %d:%d", tc.wantDiagLine, tc.wantDiagColumn, compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
			}
		})
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

func TestBuildExecutableSupportsJayessGLFWImageLoadingIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping GLFW image integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)

	workdir := t.TempDir()
	copyDirRecursive(t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(t,
		filepath.Join(repoRoot, "refs", "glfw", "include"),
		filepath.Join(workdir, "refs", "glfw", "include"),
	)
	copyDirRecursive(t,
		filepath.Join(repoRoot, "refs", "glfw", "src"),
		filepath.Join(workdir, "refs", "glfw", "src"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "refs", "raylib", "src", "external"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	copyFileForTest(t,
		filepath.Join(repoRoot, "refs", "raylib", "src", "external", "stb_image.h"),
		filepath.Join(workdir, "refs", "raylib", "src", "external", "stb_image.h"),
	)
	if err := os.MkdirAll(filepath.Join(workdir, "bindings"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "bindings", "image.bind.js"), []byte(`const f = () => {};
export const loadImageInfoNative = f;

export default {
  sources: ["./image.c"],
  includeDirs: ["../refs/raylib/src/external"],
  exports: {
    loadImageInfoNative: { symbol: "jayess_image_load_info", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile image.bind.js returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "bindings", "image.c"), []byte(`#define STB_IMAGE_IMPLEMENTATION
#include "stb_image.h"
#include "jayess_runtime.h"

jayess_value *jayess_image_load_info(jayess_value *path_value) {
    const char *path = jayess_expect_string(path_value, "jayess_image_load_info");
    int width = 0;
    int height = 0;
    int channels = 0;
    jayess_object *result;
    if (jayess_has_exception()) return jayess_value_undefined();
    if (!stbi_info(path, &width, &height, &channels)) {
        const char *reason = stbi_failure_reason();
        jayess_throw_named_error("ImageError", reason != NULL ? reason : "failed to load image info");
        return jayess_value_undefined();
    }
    result = jayess_object_new();
    jayess_object_set_value(result, "width", jayess_value_from_number((double) width));
    jayess_object_set_value(result, "height", jayess_value_from_number((double) height));
    jayess_object_set_value(result, "channels", jayess_value_from_number((double) channels));
    return jayess_value_from_object(result);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile image.c returned error: %v", err)
	}

	tinyBMP := []byte{
		0x42, 0x4d, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00, 0x00, 0x00,
		0x28, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0x00, 0xff, 0x00, 0x00, 0x00,
		0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00,
	}
	if err := os.WriteFile(filepath.Join(workdir, "tiny.bmp"), tinyBMP, 0o644); err != nil {
		t.Fatalf("WriteFile tiny.bmp returned error: %v", err)
	}

	mainSource := `
import { init, terminate, createOpenGLWindow, makeContextCurrent, swapBuffers, pollEvents, destroyWindow } from "@jayess/glfw";
import { loadImageInfoNative } from "./bindings/image.bind.js";

function main(args) {
  init();
  var window = createOpenGLWindow(64, 64, "glfw-image");
  makeContextCurrent(window);
  pollEvents();
  swapBuffers(window);
  var info = loadImageInfoNative("tiny.bmp");
  console.log("glfw-image-open:" + (window != undefined));
  console.log("glfw-image-size:" + info.width + "x" + info.height);
  console.log("glfw-image-channels:" + info.channels);
  destroyWindow(window);
  terminate();
  return 0;
}
`
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-glfw-image")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled GLFW image program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"glfw-image-open:true",
		"glfw-image-size:2x2",
		"glfw-image-channels:3",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected GLFW image integration output to contain %q, got: %s", want, text)
		}
	}
}

func TestBuildExecutableSupportsJayessGLFWVulkanSurfaceIntegration(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping GLFW Vulkan surface integration test: %v", err)
	}

	triple, err := target.DefaultTriple()
	if err != nil {
		t.Fatalf("DefaultTriple returned error: %v", err)
	}

	repoRoot := repoRootFromBackendTest(t)
	workdir := t.TempDir()
	copyDirRecursive(t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "glfw"),
		filepath.Join(workdir, "node_modules", "@jayess", "glfw"),
	)
	copyDirRecursive(t,
		filepath.Join(repoRoot, "refs", "glfw", "include"),
		filepath.Join(workdir, "refs", "glfw", "include"),
	)
	copyDirRecursive(t,
		filepath.Join(repoRoot, "refs", "glfw", "src"),
		filepath.Join(workdir, "refs", "glfw", "src"),
	)

	vulkanStubPath := filepath.Join(workdir, "libvulkan.so.1")
	vulkanStubSourcePath := filepath.Join(workdir, "vulkan_stub.c")
	vulkanStubSource := `#include <stdint.h>
#include <string.h>

typedef uint32_t VkFlags;
typedef uint32_t VkBool32;
typedef uint64_t VkInstance;
typedef uint64_t VkSurfaceKHR;
typedef int32_t VkResult;
typedef void (*PFN_vkVoidFunction)(void);

#define VK_SUCCESS 0

typedef struct VkExtensionProperties {
    char extensionName[256];
    uint32_t specVersion;
} VkExtensionProperties;

static PFN_vkVoidFunction stub_vkGetInstanceProcAddr(VkInstance instance, const char* name);
static VkResult stub_vkEnumerateInstanceExtensionProperties(const char* layerName, uint32_t* propertyCount, VkExtensionProperties* properties);
static VkResult stub_vkCreateHeadlessSurfaceEXT(VkInstance instance, const void* createInfo, const void* allocator, VkSurfaceKHR* surface);

__attribute__((visibility("default")))
PFN_vkVoidFunction vkGetInstanceProcAddr(VkInstance instance, const char* name) {
    return stub_vkGetInstanceProcAddr(instance, name);
}

static PFN_vkVoidFunction stub_vkGetInstanceProcAddr(VkInstance instance, const char* name) {
    (void) instance;
    if (name == NULL) return (PFN_vkVoidFunction) 0;
    if (strcmp(name, "vkGetInstanceProcAddr") == 0) return (PFN_vkVoidFunction) vkGetInstanceProcAddr;
    if (strcmp(name, "vkEnumerateInstanceExtensionProperties") == 0) return (PFN_vkVoidFunction) stub_vkEnumerateInstanceExtensionProperties;
    if (strcmp(name, "vkCreateHeadlessSurfaceEXT") == 0) return (PFN_vkVoidFunction) stub_vkCreateHeadlessSurfaceEXT;
    return (PFN_vkVoidFunction) 0;
}

static VkResult stub_vkEnumerateInstanceExtensionProperties(const char* layerName, uint32_t* propertyCount, VkExtensionProperties* properties) {
    (void) layerName;
    if (propertyCount == NULL) return VK_SUCCESS;
    if (properties == NULL) {
        *propertyCount = 2;
        return VK_SUCCESS;
    }
    *propertyCount = 2;
    memset(properties, 0, sizeof(VkExtensionProperties) * 2);
    strncpy(properties[0].extensionName, "VK_KHR_surface", sizeof(properties[0].extensionName) - 1);
    strncpy(properties[1].extensionName, "VK_EXT_headless_surface", sizeof(properties[1].extensionName) - 1);
    properties[0].specVersion = 1;
    properties[1].specVersion = 1;
    return VK_SUCCESS;
}

static VkResult stub_vkCreateHeadlessSurfaceEXT(VkInstance instance, const void* createInfo, const void* allocator, VkSurfaceKHR* surface) {
    (void) instance;
    (void) createInfo;
    (void) allocator;
    if (surface != NULL) *surface = 0xFEEDBEEFull;
    return VK_SUCCESS;
}
`
	if err := os.WriteFile(vulkanStubSourcePath, []byte(vulkanStubSource), 0o644); err != nil {
		t.Fatalf("WriteFile vulkan_stub.c returned error: %v", err)
	}
	buildVulkanStubCmd := exec.Command(tc.ClangPath, "-shared", "-fPIC", vulkanStubSourcePath, "-o", vulkanStubPath)
	buildVulkanStubCmd.Dir = workdir
	if output, err := buildVulkanStubCmd.CombinedOutput(); err != nil {
		t.Fatalf("building fake Vulkan loader returned error: %v: %s", err, string(output))
	}

	mainSource := `
import { init, terminate, createWindow, destroyWindow, pollEvents, isVulkanSupported, getRequiredVulkanInstanceExtensions, createVulkanSurface } from "@jayess/glfw";

function main(args) {
  if (!init()) {
    console.log("glfw-vulkan-init:false");
    return 0;
  }
  var window = createWindow(64, 64, "glfw-vulkan");
  var supported = isVulkanSupported();
  var extensions = getRequiredVulkanInstanceExtensions();
  var surface = createVulkanSurface(window, 1n);
  pollEvents();
  console.log("glfw-vulkan-init:true");
  console.log("glfw-vulkan-supported:" + supported);
  console.log("glfw-vulkan-ext0:" + extensions[0]);
  console.log("glfw-vulkan-ext1:" + extensions[1]);
  console.log("glfw-vulkan-surface:" + (surface != undefined));
  console.log("glfw-vulkan-surface-type:" + typeof surface);
  destroyWindow(window);
  terminate();
  return 0;
}
`
	mainPath := filepath.Join(workdir, "main.js")
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("WriteFile main.js returned error: %v", err)
	}

	result, err := compiler.CompilePath(mainPath, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "jayess-glfw-vulkan")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+workdir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled GLFW Vulkan surface program returned error: %v: %s", err, string(out))
	}
	text := string(out)
	for _, want := range []string{
		"glfw-vulkan-init:true",
		"glfw-vulkan-supported:true",
		"glfw-vulkan-ext0:VK_KHR_surface",
		"glfw-vulkan-ext1:VK_EXT_headless_surface",
		"glfw-vulkan-surface:true",
		"glfw-vulkan-surface-type:bigint",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected GLFW Vulkan surface integration output to contain %q, got: %s", want, text)
		}
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

func TestBuildExecutableSupportsCrossOSPathSyntax(t *testing.T) {
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
  var file = "C:/tmp/nested/file.txt";
  var normalized = path.normalize("C:/tmp/nested/../file.txt");
  var parts = path.parse(file);
  console.log("abs:" + path.isAbsolute(file));
  console.log("base:" + path.basename(file));
  console.log("dir:" + path.dirname(file));
  console.log("ext:" + path.extname(file));
  console.log("norm:" + normalized);
  console.log("fmt:" + path.format(parts));
  console.log("root:" + parts.root);
  console.log("pdir:" + parts.dir);
  console.log("pbase:" + parts.base);
  console.log("pname:" + parts.name);
  console.log("pext:" + parts.ext);
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	outputPath := nativeOutputPath(workdir, "cross-os-path-native")
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
	expected := []string{
		"abs:true",
		"base:file.txt",
		"dir:C:/tmp/nested",
		"ext:.txt",
		"norm:C:/tmp/file.txt",
		"fmt:C:/tmp/nested/file.txt",
		"root:C:/",
		"pdir:C:/tmp/nested",
		"pbase:file.txt",
		"pname:file",
		"pext:.txt",
	}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d output lines, got %d: %q", len(expected), len(lines), string(out))
	}
	for i, want := range expected {
		if lines[i] != want {
			t.Fatalf("expected line %d to be %q, got %q", i, want, lines[i])
		}
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
	for _, targetName := range []string{"windows-x64", "linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64"} {
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

func TestBuildObjectEmitsDWARFDebugInfo(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping debug object test: %v", err)
	}
	dwarfdumpPath, err := exec.LookPath("llvm-dwarfdump")
	if err != nil {
		t.Skip("skipping debug object test: llvm-dwarfdump not found in PATH")
	}

	triple, err := target.FromName("linux-x64")
	if err != nil {
		t.Fatalf("FromName returned error: %v", err)
	}

	workdir := t.TempDir()
	sourcePath := filepath.Join(workdir, "debug-info.jy")
	source := `
function helper() {
  return 41;
}

function main(args) {
  return helper() + 1;
}
`
	if err := os.WriteFile(sourcePath, []byte(strings.TrimSpace(source)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := compiler.CompilePath(sourcePath, compiler.Options{TargetTriple: triple, OptimizationLevel: "O0"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	outputPath := filepath.Join(workdir, "debug-info.o")
	if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple, OptimizationLevel: "O0"}, outputPath); err != nil {
		t.Fatalf("BuildObject returned error: %v", err)
	}

	cmd := exec.Command(dwarfdumpPath, "--debug-info", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("llvm-dwarfdump returned error: %v: %s", err, string(out))
	}

	text := string(out)
	for _, fragment := range []string{
		"DW_TAG_compile_unit",
		`"debug-info.jy"`,
		"DW_TAG_subprogram",
		`"helper"`,
		`"main"`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected DWARF output to contain %q, got:\n%s", fragment, text)
		}
	}
}

func TestBuildExecutableSupportsLinuxX64Target(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping linux-x64 executable test: %v", err)
	}

	triple, err := target.FromName("linux-x64")
	if err != nil {
		t.Fatalf("FromName returned error: %v", err)
	}

	result, err := compiler.Compile(`
function main(args) {
  console.log("linux-x64-target");
  return 0;
}
`, compiler.Options{TargetTriple: triple})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	workdir := t.TempDir()
	outputPath := nativeOutputPath(workdir, "linux-x64-target")
	if err := tc.BuildExecutable(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
		t.Fatalf("BuildExecutable returned error: %v", err)
	}

	cmd := exec.Command(outputPath)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled linux-x64 target program returned error: %v: %s", err, string(out))
	}
	if !strings.Contains(string(out), "linux-x64-target") {
		t.Fatalf("expected linux-x64 target output, got: %s", string(out))
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

func TestBuildObjectSupportsJayessAudioAcrossConfiguredTargets(t *testing.T) {
	tc, err := DetectToolchain()
	if err != nil {
		t.Skipf("skipping cross-target audio build test: %v", err)
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
				filepath.Join(repoRoot, "node_modules", "@jayess", "audio"),
				filepath.Join(workdir, "node_modules", "@jayess", "audio"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "miniaudio"),
				filepath.Join(workdir, "refs", "miniaudio"),
			)
			copyDirRecursive(
				t,
				filepath.Join(repoRoot, "refs", "cubeb", "include"),
				filepath.Join(workdir, "refs", "cubeb", "include"),
			)

			mainPath := filepath.Join(workdir, "main.js")
			mainSource := `
import { createContext, backendId, maxChannelCount, listOutputDevices, listInputDevices, destroyContext } from "@jayess/audio";

function main(args) {
  var ctx = createContext("jayess-audio-cross-target", null);
  if (ctx !== undefined) {
    backendId(ctx);
    maxChannelCount(ctx);
    listOutputDevices(ctx);
    listInputDevices(ctx);
    destroyContext(ctx);
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

			outputPath := filepath.Join(workdir, targetName+".o")
			if err := tc.BuildObject(result, compiler.Options{TargetTriple: triple}, outputPath); err != nil {
				t.Skipf("cross-target audio object build unavailable for %s: %v", targetName, err)
			}
			if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
				t.Fatalf("expected built audio object file for %s, got err=%v", targetName, err)
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
	for _, level := range []string{"O0", "O2"} {
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
	for _, tc := range []struct {
		level string
		flag  string
	}{
		{level: "O0", flag: "-O0"},
		{level: "O2", flag: "-O2"},
		{level: "Oz", flag: "-Oz"},
	} {
		args := buildExecutableArgs(&compiler.Result{}, compiler.Options{TargetTriple: "x86_64-unknown-linux-gnu", OptimizationLevel: tc.level}, "module.ll", "runtime.c", "runtime", "", nil, false, "out")
		if !containsString(args, tc.flag) {
			t.Fatalf("expected %s in executable args for %s, got %#v", tc.flag, tc.level, args)
		}
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
	runtimePaths, err := runtimeSourcePaths()
	if err != nil {
		t.Fatalf("runtimeSourcePaths returned error: %v", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		t.Fatalf("runtimeIncludePath returned error: %v", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()
	outputPath := nativeOutputPath(workdir, "llc-link")
	args := []string{"-target", triple, "-I", runtimeIncludeDir, objectPath}
	args = append(args, runtimePaths...)
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
	runtimePaths, err := runtimeSourcePaths()
	if err != nil {
		t.Fatalf("runtimeSourcePaths returned error: %v", err)
	}
	runtimeIncludeDir, err := runtimeIncludePath()
	if err != nil {
		t.Fatalf("runtimeIncludePath returned error: %v", err)
	}
	brotliIncludeDir, brotliSources, brotliAvailable := brotliBuildInputs()
	llcOutputPath := nativeOutputPath(workdir, "flow-llc")
	args := []string{"-target", triple, "-I", runtimeIncludeDir, objectPath}
	args = append(args, runtimePaths...)
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
  console.log("dns-set-resolver:" + dns.setResolver({
    hosts: {
      "kimchi.local": "127.0.0.9",
      "jjigae.local": ["127.0.0.10", "::1"]
    },
    reverse: {
      "127.0.0.9": "kimchi.local"
    }
  }));
  var custom = dns.lookup("kimchi.local");
  var customAll = dns.lookupAll("jjigae.local");
  console.log("dns-custom:" + custom.host + ":" + custom.address + ":" + custom.family);
  console.log("dns-custom-all:" + customAll.length + ":" + customAll[0].address + ":" + customAll[1].family);
  console.log("dns-custom-reverse:" + dns.reverse("127.0.0.9"));
  console.log("dns-clear-resolver:" + dns.clearResolver());
  console.log("dns-custom-cleared:" + dns.lookup("kimchi.local"));
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
		"dns-set-resolver:true",
		"dns-custom:kimchi.local:127.0.0.9:4",
		"dns-custom-all:2:127.0.0.10:6",
		"dns-custom-reverse:kimchi.local",
		"dns-clear-resolver:true",
		"dns-custom-cleared:undefined",
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
		remaining := 9000
		buffer = make([]byte, 1024)
		for remaining > 0 {
			readCount, err := conn.Read(buffer)
			if err != nil {
				serverErr <- err
				return
			}
			remaining -= readCount
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
  var socketDrainCount = 0;
  socket.on("drain", () => {
    socketDrainCount = socketDrainCount + 1;
    return 0;
  });
  var large = "";
  var i = 0;
  while (i < 9000) {
    large = large + "x";
    i = i + 1;
  }
  console.log("socket-backpressure:" + socket.write(large) + ":" + socketDrainCount + ":" + socket.writableNeedDrain + ":" + (socket.writableLength == 0));
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
		"socket-backpressure:false:1:false:true",
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
  var serverDrainCount = 0;
  socket.on("drain", () => {
    serverDrainCount = serverDrainCount + 1;
    return 0;
  });
  var large = "";
  var i = 0;
  while (i < 9000) {
    large = large + "y";
    i = i + 1;
  }
  console.log("server-backpressure:" + socket.write(large) + ":" + serverDrainCount + ":" + socket.writableNeedDrain + ":" + (socket.writableLength == 0));
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
	largeBuffer := make([]byte, 9000)
	if _, err := io.ReadFull(client, largeBuffer); err != nil {
		t.Fatalf("client ReadFull(large) returned error: %v", err)
	}
	if string(largeBuffer) != strings.Repeat("y", 9000) {
		t.Fatalf("expected 9000-byte server payload, got prefix %q", string(largeBuffer[:16]))
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
		"server-backpressure:false:1:false:true",
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
  console.log("shared-buffer-bytes:" + ints.buffer.byteLength);
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
		"shared-buffer-bytes:8",
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

  var large = "";
  var i = 0;
  while (i < 9000) {
    large = large + "x";
    i = i + 1;
  }

  var fileDrainCount = 0;
  var fileWriter = fs.createWriteStream("tmp/backpressure.txt");
  fileWriter.on("drain", function () {
    fileDrainCount = fileDrainCount + 1;
    return 0;
  });
  var fileWriteOk = fileWriter.write(large);
  console.log("compression-stream-file-backpressure:" + fileWriteOk + ":" + fileDrainCount + ":" + fileWriter.writableNeedDrain + ":" + (fileWriter.writableLength == 0));
  fileWriter.end();
  console.log("compression-stream-file-finish:" + fileWriter.writableEnded + ":" + fs.readFile("tmp/backpressure.txt", "utf8").length);

  var packed = compression.gzip(large);
  var inflateDrainCount = 0;
  var inflateStream = compression.createGunzipStream();
  inflateStream.on("drain", function () {
    inflateDrainCount = inflateDrainCount + 1;
    return 0;
  });
  var inflateWriteOk = inflateStream.write(packed);
  var inflateNeedDrain = inflateStream.writableNeedDrain;
  var inflateOverHighWater = inflateStream.writableLength > inflateStream.writableHighWaterMark;
  var inflated = inflateStream.read(large.length);
  inflateStream.end();
  console.log("compression-stream-transform-backpressure:" + inflateWriteOk + ":" + inflateNeedDrain + ":" + inflateOverHighWater + ":" + inflateDrainCount + ":" + inflated.length);
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
		"compression-stream-file-backpressure:false:1:false:true",
		"compression-stream-file-finish:true:9000",
		"compression-stream-transform-backpressure:false:true:true:1:9000",
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
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "openssl", "include"),
		filepath.Join(workdir, "refs", "openssl", "include"),
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
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "openssl", "include"),
		filepath.Join(workdir, "refs", "openssl", "include"),
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
	copyDirRecursive(
		t,
		filepath.Join(repoRoot, "refs", "openssl", "include"),
		filepath.Join(workdir, "refs", "openssl", "include"),
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
