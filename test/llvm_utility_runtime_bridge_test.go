package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectUtilityRuntimeCalls(t *testing.T) {
	source := `
		const raw = Buffer.create(8);
		const encoded = Buffer.fromString("hello", "utf8");
		const decoded = Buffer.toString(encoded, "utf8");
		const part = Buffer.slice(encoded, 0, 4);
		Buffer.copy(part, raw, 0);
		const value = Buffer.readUInt16LE(raw, 0);
		Buffer.writeUInt16LE(raw, value, 2);
		Buffer.typedArrayView(raw, "Uint8Array");
		Buffer.createReadStream(raw);
		Buffer.createWriteStream(raw);

		const parsed = url.parse("https://example.com/a?b=1");
		url.format(parsed);
		const query = url.parseQuery("a=1");
		url.stringifyQuery(query);
		const escaped = url.encode("a b");
		url.decode(escaped);
		const fileURL = url.pathToFileURL("tmp/file.txt");
		url.fileURLToPath(fileURL);

		util.format("value %s", "ok");
		util.inspect(raw);
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
		Name:       "utility-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_buffer_create",
		"@jayess_buffer_from_string",
		"@jayess_buffer_to_string",
		"@jayess_buffer_slice",
		"@jayess_buffer_copy",
		"@jayess_buffer_read_uint16_le",
		"@jayess_buffer_write_uint16_le",
		"@jayess_buffer_typed_array_view",
		"@jayess_buffer_create_read_stream",
		"@jayess_buffer_create_write_stream",
		"@jayess_url_parse",
		"@jayess_url_format",
		"@jayess_url_parse_query",
		"@jayess_url_stringify_query",
		"@jayess_url_encode",
		"@jayess_url_decode",
		"@jayess_url_path_to_file_url",
		"@jayess_url_file_url_to_path",
		"@jayess_util_format",
		"@jayess_util_inspect",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected utility runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
