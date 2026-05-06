package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectOSCLIRuntimeCalls(t *testing.T) {
	source := `
		fs.writeFile("temp/out.txt", "hello");
		const input = fs.createReadStream("temp/out.txt");
		const output = fs.createWriteStream("temp/copy.txt");
		stream.pipe(input, output);
		const result = childProcess.exec("jayess", ["--version"]);
		terminal.supportsColor(process.stdout);
		process.stdout.write(result);
		process.cwd();
	`
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:       "os-cli",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_fs_write_file",
		"@jayess_fs_create_read_stream",
		"@jayess_fs_create_write_stream",
		"@jayess_stream_pipe",
		"@jayess_child_process_exec",
		"@jayess_terminal_supports_color",
		"@jayess_process_stdout",
		"@jayess_process_stdout_write",
		"@jayess_process_cwd",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected OS/CLI runtime IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestCLIOSRuntimeExampleCompilesToRuntimeCalls(t *testing.T) {
	root := cliRepoRoot(t)
	outputDir := cliTempDir(t, root, "os-cli-example-*")
	output := filepath.Join(outputDir, "16-cli-os-runtime.ll")
	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, filepath.Join(root, "examples", "16-cli-os-runtime.js"))

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read LLVM output: %v", err)
	}
	ir := string(content)
	for _, want := range []string{
		"@jayess_fs_mkdirp",
		"@jayess_fs_write_file",
		"@jayess_stream_pipe",
		"@jayess_child_process_exec",
		"@jayess_terminal_supports_color",
		"@jayess_process_stdout_write",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected compiled CLI example to contain %q:\n%s", want, ir)
		}
	}
}
