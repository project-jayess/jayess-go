package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLILowersMainReturnExpressionToLLVMReturnCode(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-return-code-*")
	input := filepath.Join(dir, "return_code.js")
	output := filepath.Join(dir, "return_code.ll")
	if err := os.WriteFile(input, []byte("function main() { return 10; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, input)
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read LLVM output: %v", err)
	}
	for _, want := range []string{
		"define i32 @main()",
		"call %jayess.value @__jayess_user_main()",
		"call i32 @jayess_value_to_exit_code",
		"define %jayess.value @__jayess_user_main()",
	} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("expected lowered runtime main IR to contain %q, got:\n%s", want, string(content))
		}
	}
}
