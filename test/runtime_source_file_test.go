package test

import (
	"path/filepath"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeSourceFileReadWriteAndList(t *testing.T) {
	root := t.TempDir()
	mainPath := filepath.Join(root, "src", "main.js")
	helperPath := filepath.Join(root, "src", "helpers", "util.js")
	if err := jayessruntime.WriteSourceFile(mainPath, `import "./helpers/util.js";`); err != nil {
		t.Fatal(err)
	}
	if err := jayessruntime.WriteSourceFile(helperPath, `export const value = 1;`); err != nil {
		t.Fatal(err)
	}
	source, err := jayessruntime.ReadSourceFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if source.Path != mustAbs(t, mainPath) || source.Text == "" {
		t.Fatalf("expected normalized source file, got %#v", source)
	}
	files, err := jayessruntime.ListSourceFiles(filepath.Join(root, "src"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != mustAbs(t, helperPath) || files[1] != mustAbs(t, mainPath) {
		t.Fatalf("expected sorted source files, got %#v", files)
	}
}

func TestRuntimeSourcePathResolutionIsDeterministic(t *testing.T) {
	root := t.TempDir()
	importer := filepath.Join(root, "src", "main.js")
	resolved, err := jayessruntime.ResolveSourcePath(importer, "./../src/helpers/../util.js")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != mustAbs(t, filepath.Join(root, "src", "util.js")) {
		t.Fatalf("expected clean absolute path, got %q", resolved)
	}
}

func TestRuntimeSourcePathRejectsInvalidInput(t *testing.T) {
	if _, err := jayessruntime.NormalizeSourcePath(""); err == nil {
		t.Fatal("expected empty path to fail")
	}
	if _, err := jayessruntime.NormalizeSourcePath("bad\x00path.js"); err == nil {
		t.Fatal("expected NUL path to fail")
	}
	if _, err := jayessruntime.ResolveSourcePath("", "./module.js"); err == nil {
		t.Fatal("expected missing importer to fail")
	}
}
