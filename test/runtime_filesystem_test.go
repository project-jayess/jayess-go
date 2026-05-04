package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeFilesystemCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"readFile",
		"writeFile",
		"appendFile",
		"deleteFile",
		"rename",
		"copyFile",
		"stat",
		"chmod",
		"exists",
		"mkdir",
		"mkdirp",
		"rmdir",
		"readdir",
		"walkDir",
		"symlink",
		"watch",
		"createReadStream",
		"createWriteStream",
	}
	for _, name := range expected {
		if !jayessruntime.HasFilesystemCapability(name) {
			t.Fatalf("expected filesystem runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsFilesystemSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(path, next) {
			const text = fs.readFile(path);
			fs.writeFile(next, text);
			fs.appendFile(next, "\n");
			const info = fs.stat(next);
			if (fs.exists(path)) {
				fs.copyFile(path, next);
				fs.rename(next, path);
			}
			fs.chmod(path, 420);
			fs.mkdir("out");
			fs.mkdirp("out/nested");
			const entries = fs.readdir("out");
			const tree = fs.walkDir("out");
			const input = fs.createReadStream(path);
			const output = fs.createWriteStream(next);
			const watcher = fs.watch("out", () => {});
			fs.symlink(path, "out/link");
			fs.deleteFile("out/link");
			fs.rmdir("out/nested");
			return info || entries || tree || input || output || watcher;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeFilesystemCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.FilesystemCapabilities() {
		if capability.Name == "" {
			t.Fatalf("filesystem capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("filesystem capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("filesystem capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelFilesystemRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var fs = {};`)
	requireSemanticError(t, err, "duplicate declaration fs")
}
