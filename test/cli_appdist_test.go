package test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCLIEmitsAppDistributionFromExecutable(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-dist-*")
	input := filepath.Join(dir, "main.js")
	executable := filepath.Join(dir, "build", "demo")
	output := filepath.Join(dir, "dist", "demo")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatalf("create executable dir: %v", err)
	}
	if err := os.WriteFile(executable, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=dist", "--executable", executable, "-o", output, input)
	requireFile(t, filepath.Join(output, "demo"))
}

func TestCLIPackageCommandUsesDistEmit(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-package-*")
	input := filepath.Join(dir, "main.js")
	executable := filepath.Join(dir, "demo")
	output := filepath.Join(dir, "package")
	if err := os.WriteFile(input, []byte("function main() { return 0; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(executable, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	runJayessCLI(t, root, "package", "--target=linux-x64", "--executable", executable, "-o", output, input)
	requireFile(t, filepath.Join(output, "demo"))
}

func TestCLIAppDistributionCopiesBindingSharedLibraries(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-dist-binding-*")
	input := filepath.Join(dir, "main.js")
	nativeModule := filepath.Join(dir, "native", "helper.js")
	sharedLibrary := filepath.Join(dir, "native", "lib", "libhelper.so")
	licenseFile := filepath.Join(dir, "native", "LICENSE.helper")
	executable := filepath.Join(dir, "build", "demo")
	output := filepath.Join(dir, "dist", "demo")
	if err := os.MkdirAll(filepath.Dir(nativeModule), 0o755); err != nil {
		t.Fatalf("create native dir: %v", err)
	}
	if err := os.WriteFile(input, []byte(`import { help } from "./native/helper.js"; function main() { return 0; }`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(nativeModule, []byte(`
		import { bind } from "ffi";
		const f = () => {};
		export const help = f;
		export default bind({
			sharedLibraries: ["./lib/libhelper.so"],
			licenseFiles: ["./LICENSE.helper"],
			exports: {
				help: { symbol: "helper_help", type: "function" }
			}
		});
	`), 0o644); err != nil {
		t.Fatalf("write binding module: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(sharedLibrary), 0o755); err != nil {
		t.Fatalf("create shared library dir: %v", err)
	}
	if err := os.WriteFile(sharedLibrary, []byte("fake helper"), 0o755); err != nil {
		t.Fatalf("write shared library: %v", err)
	}
	if err := os.WriteFile(licenseFile, []byte("helper license"), 0o644); err != nil {
		t.Fatalf("write license file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatalf("create executable dir: %v", err)
	}
	if err := os.WriteFile(executable, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=dist", "--executable", executable, "-o", output, input)
	requireFile(t, filepath.Join(output, "demo"))
	requireFile(t, filepath.Join(output, "libhelper.so"))
	requireFile(t, filepath.Join(output, "licenses", "LICENSE.helper"))
}

func TestCLIAppDistributionReportsMissingBindingSharedLibrary(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-dist-missing-binding-*")
	input := filepath.Join(dir, "main.js")
	nativeModule := filepath.Join(dir, "native", "helper.js")
	executable := filepath.Join(dir, "build", "demo")
	output := filepath.Join(dir, "dist", "demo")
	if err := os.MkdirAll(filepath.Dir(nativeModule), 0o755); err != nil {
		t.Fatalf("create native dir: %v", err)
	}
	if err := os.WriteFile(input, []byte(`import { help } from "./native/helper.js"; function main() { return 0; }`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(nativeModule, []byte(`
		import { bind } from "ffi";
		const f = () => {};
		export const help = f;
		export default bind({
			sharedLibraries: ["./lib/libmissing.so"],
			exports: {
				help: { symbol: "helper_help", type: "function" }
			}
		});
	`), 0o644); err != nil {
		t.Fatalf("write binding module: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatalf("create executable dir: %v", err)
	}
	if err := os.WriteFile(executable, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go executable: %v", err)
	}
	command := exec.Command(goPath, "run", "./cmd/jayess", "compile", "--target=linux-x64", "--emit=dist", "--executable", executable, "-o", output, input)
	command.Dir = root
	result, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected app distribution to fail for missing shared library, got:\n%s", string(result))
	}
	text := string(result)
	if !strings.Contains(text, "missing runtime shared library") || !strings.Contains(text, "libmissing.so") {
		t.Fatalf("expected missing runtime shared library diagnostic, got:\n%s", text)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("expected no app dist after diagnostic, stat error: %v", err)
	}
}

func TestCLIAppDistributionOutputLayout(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-dist-layout-*")
	input := filepath.Join(dir, "main.js")
	nativeModule := filepath.Join(dir, "native", "helper.js")
	sharedLibrary := filepath.Join(dir, "native", "lib", "libhelper.so")
	licenseFile := filepath.Join(dir, "native", "NOTICE.helper")
	executable := filepath.Join(dir, "build", "demo")
	output := filepath.Join(dir, "dist", "demo")
	if err := os.MkdirAll(filepath.Dir(sharedLibrary), 0o755); err != nil {
		t.Fatalf("create native lib dir: %v", err)
	}
	if err := os.WriteFile(input, []byte(`import { help } from "./native/helper.js"; function main() { return help(); }`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(nativeModule, []byte(`
		import { bind } from "ffi";
		const f = () => {};
		export const help = f;
		export default bind({
			sharedLibraries: ["./lib/libhelper.so"],
			licenseFiles: ["./NOTICE.helper"],
			exports: {
				help: { symbol: "helper_help", type: "function" }
			}
		});
	`), 0o644); err != nil {
		t.Fatalf("write binding module: %v", err)
	}
	if err := os.WriteFile(sharedLibrary, []byte("fake helper"), 0o755); err != nil {
		t.Fatalf("write shared library: %v", err)
	}
	if err := os.WriteFile(licenseFile, []byte("helper notice"), 0o644); err != nil {
		t.Fatalf("write license file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatalf("create executable dir: %v", err)
	}
	if err := os.WriteFile(executable, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	runJayessCLI(t, root, "package", "--target=linux-x64", "--executable", executable, "-o", output, input)
	requireStringSlice(t, appDistRelativeFiles(t, output), []string{
		"demo",
		"libhelper.so",
		filepath.Join("licenses", "NOTICE.helper"),
	})
}

func appDistRelativeFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relative)
		return nil
	})
	if err != nil {
		t.Fatalf("walk app distribution: %v", err)
	}
	sort.Strings(files)
	return files
}
