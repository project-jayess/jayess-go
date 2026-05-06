package runtime

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileInfo struct {
	Path    string
	Size    int64
	Mode    os.FileMode
	IsDir   bool
	ModTime time.Time
}

func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func WriteFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func AppendFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func DeleteFile(path string) error {
	return os.Remove(path)
}

func RenamePath(from string, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

func CopyFile(from string, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	target, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		return err
	}
	return target.Close()
}

func StatPath(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{Path: path, Size: info.Size(), Mode: info.Mode(), IsDir: info.IsDir(), ModTime: info.ModTime()}, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func MakeDir(path string) error {
	return os.Mkdir(path, 0o755)
}

func MakeDirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func RemoveDir(path string) error {
	return os.Remove(path)
}

func ReadDirNames(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func WalkDirPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func ChangeMode(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

func SymlinkPath(target string, link string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	return os.Symlink(target, link)
}

func CreateReadStream(path string) (*IOStream, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stream := NewReadableStream(path, file)
	stream.closer = file
	return stream, nil
}

func CreateWriteStream(path string) (*IOStream, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	stream := NewWritableStream(path, file)
	stream.closer = file
	return stream, nil
}
