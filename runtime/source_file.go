package runtime

import (
	"os"
	"path/filepath"
	"sort"
)

type SourceFile struct {
	Path string
	Text string
}

func ReadSourceFile(path string) (SourceFile, error) {
	normalized, err := NormalizeSourcePath(path)
	if err != nil {
		return SourceFile{}, err
	}
	source, err := os.ReadFile(normalized)
	if err != nil {
		return SourceFile{}, err
	}
	return SourceFile{Path: normalized, Text: string(source)}, nil
}

func WriteSourceFile(path string, text string) error {
	normalized, err := NormalizeSourcePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(normalized), 0o755); err != nil {
		return err
	}
	return os.WriteFile(normalized, []byte(text), 0o644)
}

func ListSourceFiles(root string) ([]string, error) {
	normalizedRoot, err := NormalizeSourcePath(root)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.WalkDir(normalizedRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".js" {
			return nil
		}
		normalized, err := NormalizeSourcePath(path)
		if err != nil {
			return err
		}
		files = append(files, normalized)
		return nil
	})
	sort.Strings(files)
	return files, err
}
