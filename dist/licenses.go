package dist

import (
	"os"
	"path/filepath"
)

type licenseFile struct {
	Source string
	Output string
}

func copyLicenses(sourceRoot string, outputDir string) ([]string, []string, error) {
	files := llvmLicenseFiles()
	var copied []string
	var diagnostics []string
	for _, file := range files {
		sourcePath := filepath.Join(sourceRoot, file.Source)
		if _, err := os.Stat(sourcePath); err != nil {
			diagnostics = append(diagnostics, "missing license file: "+sourcePath)
			continue
		}
		outputPath := filepath.Join(outputDir, file.Output)
		if err := copyFile(sourcePath, outputPath, 0o644); err != nil {
			return copied, diagnostics, err
		}
		copied = append(copied, file.Output)
	}
	if err := writeLicenseIndex(outputDir, copied); err != nil {
		return copied, diagnostics, err
	}
	return copied, diagnostics, nil
}

func llvmLicenseFiles() []licenseFile {
	return []licenseFile{
		{Source: filepath.Join("refs", "llvm-project", "LICENSE.TXT"), Output: "llvm-project-LICENSE.TXT"},
		{Source: filepath.Join("refs", "llvm-project", "llvm", "LICENSE.TXT"), Output: "llvm-LICENSE.TXT"},
		{Source: filepath.Join("refs", "llvm-project", "clang", "LICENSE.TXT"), Output: "clang-LICENSE.TXT"},
		{Source: filepath.Join("refs", "llvm-project", "lld", "LICENSE.TXT"), Output: "lld-LICENSE.TXT"},
		{Source: filepath.Join("refs", "llvm-project", "third-party", "README.md"), Output: "llvm-third-party-README.md"},
	}
}

func writeLicenseIndex(outputDir string, copied []string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	content := "Jayess distribution notices\n\n"
	content += "Bundled LLVM, Clang, and lld files are covered by the license files in this directory.\n"
	content += "Keep this directory with the distributed compiler package.\n\n"
	content += "Included notice files:\n"
	if len(copied) == 0 {
		content += "- none\n"
	} else {
		for _, name := range copied {
			content += "- " + name + "\n"
		}
	}
	return os.WriteFile(filepath.Join(outputDir, "README.txt"), []byte(content), 0o644)
}
