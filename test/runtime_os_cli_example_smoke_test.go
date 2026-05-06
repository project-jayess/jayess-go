package test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeOSCLIExampleServicesSmoke(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "input.txt")
	outputPath := filepath.Join(root, "output.txt")
	if err := jayessruntime.WriteFile(inputPath, "hello"); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	input, err := jayessruntime.CreateReadStream(inputPath)
	if err != nil {
		t.Fatalf("CreateReadStream returned error: %v", err)
	}
	output, err := jayessruntime.CreateWriteStream(outputPath)
	if err != nil {
		t.Fatalf("CreateWriteStream returned error: %v", err)
	}
	if _, err := input.PipeTo(output); err != nil {
		t.Fatalf("PipeTo returned error: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("output Close returned error: %v", err)
	}
	copied, err := jayessruntime.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if copied != "hello" {
		t.Fatalf("unexpected copied content %q", copied)
	}

	process, stdout, _ := jayessruntime.NewBufferedProcessRuntime([]string{"example"}, map[string]string{"TERM": "dumb"}, "")
	if _, err := process.Stdout().WriteString("jayess\n"); err != nil {
		t.Fatalf("stdout WriteString returned error: %v", err)
	}
	if stdout.String() != "jayess\n" {
		t.Fatalf("unexpected stdout %q", stdout.String())
	}
	if color := jayessruntime.DetectTerminal(nil, process.Env()).SupportsColor; color {
		t.Fatalf("expected dumb terminal to disable color")
	}

	stream, buffer := jayessruntime.NewBufferStream("buffer")
	if _, err := stream.WriteString("child"); err != nil {
		t.Fatalf("buffer WriteString returned error: %v", err)
	}
	if buffer.String() != "child" {
		t.Fatalf("unexpected buffer stream content %q", buffer.String())
	}

	result, err := jayessruntime.ExecProcess("printf", []string{"cli-smoke"}, nil, "")
	if err != nil {
		t.Fatalf("ExecProcess returned error: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "cli-smoke" || result.ExitCode != 0 {
		t.Fatalf("unexpected exec result %#v", result)
	}

	var sink bytes.Buffer
	if _, err := jayessruntime.NewReadableStream("stdout", bytes.NewBufferString(result.Stdout)).PipeTo(jayessruntime.NewWritableStream("sink", &sink)); err != nil {
		t.Fatalf("pipe exec stdout returned error: %v", err)
	}
	if sink.String() != result.Stdout {
		t.Fatalf("unexpected piped stdout %q", sink.String())
	}
}
