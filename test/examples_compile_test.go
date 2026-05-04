package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestExamplesParseAsJayessPrograms(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join("..", "examples", "**", "*.js"))
	if err != nil {
		t.Fatal(err)
	}
	rootExamples, err := filepath.Glob(filepath.Join("..", "examples", "*.js"))
	if err != nil {
		t.Fatal(err)
	}
	examples = append(rootExamples, examples...)
	if len(examples) == 0 {
		t.Fatal("expected example .js files")
	}
	for _, path := range examples {
		path := path
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.New(lexer.New(string(source))).ParseProgram(); err != nil {
				t.Fatalf("example should parse: %v", err)
			}
		})
	}
}

func TestRootExamplesCompileToLLVM(t *testing.T) {
	root := cliRepoRoot(t)
	outputDir := cliTempDir(t, root, "examples-llvm-*")
	examples, err := filepath.Glob(filepath.Join(root, "examples", "*.js"))
	if err != nil {
		t.Fatal(err)
	}
	if len(examples) == 0 {
		t.Fatal("expected root example .js files")
	}
	for _, path := range examples {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".ll"
			output := filepath.Join(outputDir, name)
			runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, path)
			content, err := os.ReadFile(output)
			if err != nil {
				t.Fatalf("read LLVM output: %v", err)
			}
			text := string(content)
			if !strings.Contains(text, `target triple = "x86_64-pc-linux-gnu"`) {
				t.Fatalf("expected linux target triple in %s, got:\n%s", output, text)
			}
			if !strings.Contains(text, "define i32 @main()") {
				t.Fatalf("expected main function in %s, got:\n%s", output, text)
			}
		})
	}
}
