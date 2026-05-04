package test

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/dist"
)

func TestDistCreateCopiesBundledLLVMToolsAndArchives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", "clang"), "fake clang")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", "ld.lld"), "fake lld")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "lib", "libLLVM.so.23"), "fake llvm")
	writeFakeLLVMLicenses(t, root)

	result, err := dist.Create(dist.Config{
		Platform:      "linux-x64",
		Version:       "test",
		OutDir:        filepath.Join(root, "dist"),
		SourceRoot:    root,
		Archive:       true,
		BuildCompiler: false,
		StrictTools:   false,
		Tools:         []string{"clang", "ld.lld", "llvm-as"},
	})
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.Plan.Root, "tools", "bin", "clang"))
	requireFile(t, filepath.Join(result.Plan.Root, "tools", "bin", "ld.lld"))
	requireFile(t, filepath.Join(result.Plan.Root, "tools", "lib", "libLLVM.so.23"))
	requireFile(t, filepath.Join(result.Plan.Root, "licenses", "llvm-project-LICENSE.TXT"))
	requireFile(t, filepath.Join(result.Plan.Root, "licenses", "clang-LICENSE.TXT"))
	requireFile(t, filepath.Join(result.Plan.Root, "licenses", "lld-LICENSE.TXT"))
	requireFile(t, filepath.Join(result.Plan.Root, "licenses", "README.txt"))
	requireFile(t, filepath.Join(result.Plan.Root, "README.txt"))
	requireFile(t, result.ArchivePath)
	requireFile(t, result.ChecksumPath)
	if len(result.Diagnostics) != 1 || !strings.Contains(result.Diagnostics[0], "llvm-as") {
		t.Fatalf("expected missing llvm-as diagnostic, got %#v", result.Diagnostics)
	}
	entries := tarGzEntries(t, result.ArchivePath)
	requireArchiveEntry(t, entries, "jayess-test-linux-x64/tools/bin/clang")
	requireArchiveEntry(t, entries, "jayess-test-linux-x64/licenses/clang-LICENSE.TXT")
	requireArchiveEntry(t, entries, "jayess-test-linux-x64/README.txt")
}

func TestDistCreateCanRequireBundledLLVMTools(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", "clang"), "fake clang")
	writeFakeLLVMLicenses(t, root)

	result, err := dist.Create(dist.Config{
		Platform:      "linux-x64",
		Version:       "test",
		OutDir:        filepath.Join(root, "dist"),
		SourceRoot:    root,
		Archive:       false,
		BuildCompiler: false,
		StrictTools:   true,
		Tools:         []string{"clang", "ld.lld"},
	})
	if err == nil {
		t.Fatal("expected strict tool packaging to fail")
	}
	if len(result.Diagnostics) != 1 || !strings.Contains(result.Diagnostics[0], "ld.lld") {
		t.Fatalf("expected missing ld.lld diagnostic, got %#v", result.Diagnostics)
	}
}

func TestJayessDistStrictToolsSucceedsFromCleanCheckout(t *testing.T) {
	root := cliRepoRoot(t)
	sourceRoot := cliTempDir(t, root, "dist-clean-source-*")
	outDir := filepath.Join(cliTempDir(t, root, "dist-clean-out-*"), "dist")
	writeFakeLLVMTools(t, sourceRoot, dist.DefaultTools())
	writeFakeLLVMLicenses(t, sourceRoot)

	command := exec.Command(
		"go", "run", "./cmd/jayess-dist",
		"--platform=linux-x64",
		"--version=strict",
		"--source-root", sourceRoot,
		"--out", outDir,
		"--archive=false",
		"--build-compiler=false",
		"--strict-tools=true",
	)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("jayess-dist strict packaging failed: %v\n%s", err, string(output))
	}
	packageRoot := filepath.Join(outDir, "linux-x64", "jayess-strict-linux-x64")
	for _, tool := range dist.DefaultTools() {
		requireFile(t, filepath.Join(packageRoot, "tools", "bin", tool))
	}
	if strings.Contains(string(output), "warning:") {
		t.Fatalf("expected strict clean checkout packaging without warnings, got:\n%s", string(output))
	}
}

func TestDistCreateVerifiesPlatformSDKArchives(t *testing.T) {
	cases := []struct {
		platform string
		archive  string
		compiler string
	}{
		{platform: "linux-x64", archive: ".tar.gz", compiler: "jayess"},
		{platform: "macos-arm64", archive: ".tar.gz", compiler: "jayess"},
		{platform: "windows-x64", archive: ".zip", compiler: "jayess.exe"},
	}
	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", "clang"), "fake clang")
			writeFakeLLVMLicenses(t, root)

			result, err := dist.Create(dist.Config{
				Platform:      tc.platform,
				Version:       "test",
				OutDir:        filepath.Join(root, "dist"),
				SourceRoot:    root,
				Archive:       true,
				BuildCompiler: false,
				StrictTools:   true,
				Tools:         []string{"clang"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(result.ArchivePath, tc.archive) {
				t.Fatalf("expected %s archive, got %q", tc.archive, result.ArchivePath)
			}
			requireFile(t, result.ArchivePath)
			requireFile(t, result.ChecksumPath)
			packageName := dist.PackageName("test", tc.platform)
			entries := archiveEntries(t, result.ArchivePath)
			requireArchiveEntry(t, entries, packageName+"/tools/bin/clang")
			requireArchiveEntry(t, entries, packageName+"/licenses/README.txt")
			if result.CompilerBuilt {
				requireArchiveEntry(t, entries, packageName+"/"+tc.compiler)
			}
		})
	}
}

func TestDistCreateLeavesReferenceDirectoriesUnchanged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", "clang"), "fake clang")
	writeFakeLLVMLicenses(t, root)
	refSentinel := filepath.Join(root, "refs", "llvm-project", "SOURCE_SENTINEL.txt")
	oldSentinel := filepath.Join(root, "old_version", "SOURCE_SENTINEL.txt")
	writeFile(t, refSentinel, "do not modify refs")
	writeFile(t, oldSentinel, "do not modify old_version")
	beforeRefs := readFile(t, refSentinel)
	beforeOld := readFile(t, oldSentinel)

	_, err := dist.Create(dist.Config{
		Platform:      "linux-x64",
		Version:       "test",
		OutDir:        filepath.Join(root, "dist"),
		SourceRoot:    root,
		Archive:       true,
		BuildCompiler: false,
		StrictTools:   true,
		Tools:         []string{"clang"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, refSentinel); got != beforeRefs {
		t.Fatalf("expected refs sentinel to stay unchanged, got %q", got)
	}
	if got := readFile(t, oldSentinel); got != beforeOld {
		t.Fatalf("expected old_version sentinel to stay unchanged, got %q", got)
	}
}

func TestDistCreatePackagedCompilerCompilesExample(t *testing.T) {
	root := cliRepoRoot(t)
	workDir := cliTempDir(t, root, "dist-smoke-*")
	result, err := dist.Create(dist.Config{
		Platform:      "linux-x64",
		Version:       "smoke",
		OutDir:        filepath.Join(workDir, "dist"),
		SourceRoot:    root,
		Archive:       false,
		BuildCompiler: true,
		StrictTools:   false,
		GoTags:        []string{},
		Tools:         []string{"clang"},
	})
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, result.Plan.CompilerPath)
	output := filepath.Join(workDir, "out", "basic.ll")
	command := exec.Command(result.Plan.CompilerPath, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, filepath.Join(root, "examples", "01-basics.js"))
	command.Dir = result.Plan.Root
	combined, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("packaged jayess failed: %v\n%s", err, string(combined))
	}
	content := readFile(t, output)
	if !strings.Contains(content, `target triple = "x86_64-pc-linux-gnu"`) {
		t.Fatalf("expected linux target triple in smoke output, got:\n%s", content)
	}
	if !strings.Contains(content, "define i32 @main()") {
		t.Fatalf("expected main function in smoke output, got:\n%s", content)
	}
}

func writeFakeLLVMLicenses(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "LICENSE.TXT"), "llvm project license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "llvm", "LICENSE.TXT"), "llvm license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "clang", "LICENSE.TXT"), "clang license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "lld", "LICENSE.TXT"), "lld license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "third-party", "README.md"), "third party notices")
}

func writeFakeLLVMTools(t *testing.T, root string, tools []string) {
	t.Helper()
	for _, tool := range tools {
		writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", tool), "fake "+tool)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
}

func tarGzEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	entries := map[string]struct{}{}
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		entries[header.Name] = struct{}{}
	}
	return entries
}

func requireArchiveEntry(t *testing.T, entries map[string]struct{}, name string) {
	t.Helper()
	if _, ok := entries[name]; !ok {
		t.Fatalf("expected archive entry %q, got %#v", name, entries)
	}
}

func archiveEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	if strings.HasSuffix(path, ".zip") {
		return zipEntries(t, path)
	}
	return tarGzEntries(t, path)
}

func zipEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := map[string]struct{}{}
	for _, file := range reader.File {
		entries[file.Name] = struct{}{}
	}
	return entries
}
