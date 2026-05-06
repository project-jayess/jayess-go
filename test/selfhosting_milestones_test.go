package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfHostingMilestoneUtilitiesCompile(t *testing.T) {
	root := cliRepoRoot(t)
	workDir := cliTempDir(t, root, "selfhost-tools-*")
	cases := []struct {
		name   string
		source string
	}{
		{name: "lexer", source: selfhostLexerSource()},
		{name: "parser", source: selfhostParserSource()},
		{name: "semantic", source: selfhostSemanticSource()},
		{name: "backend", source: selfhostBackendSource()},
		{name: "compiler", source: selfhostCompilerSource()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(workDir, tc.name+".js")
			output := filepath.Join(workDir, tc.name+".ll")
			if err := os.WriteFile(input, []byte(tc.source), 0o644); err != nil {
				t.Fatal(err)
			}
			runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, input)
			ir, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(ir), "define i32 @main()") {
				t.Fatalf("expected compiled Jayess tool IR, got:\n%s", string(ir))
			}
		})
	}
}

func selfhostLexerSource() string {
	return `
function classify(code) {
  if (code === 10) {
    return 1;
  }
  if (code === 32) {
    return 2;
  }
  return 3;
}

function main(args) {
  return classify(10) === 1 ? 0 : 1;
}
`
}

func selfhostParserSource() string {
	return `
function acceptToken(kind, expected) {
  return kind === expected;
}

function main(args) {
  return acceptToken(1, 1) ? 0 : 1;
}
`
}

func selfhostSemanticSource() string {
	return `
function sameName(left, right) {
  return left === right;
}

function main(args) {
  return sameName("value", "value") ? 0 : 1;
}
`
}

func selfhostBackendSource() string {
	return `
function emitReturnCode(value) {
  return value;
}

function main(args) {
  return emitReturnCode(0);
}
`
}

func selfhostCompilerSource() string {
	return `
function compileStage(ok) {
  if (ok) {
    return 0;
  }
  return 1;
}

function main(args) {
  return compileStage(true);
}
`
}
