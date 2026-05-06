package test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestFilesystemOperationsReadWriteMoveAndList(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "nested", "input.txt")
	copyPath := filepath.Join(root, "copy.txt")
	moved := filepath.Join(root, "moved.txt")

	if err := jayessruntime.WriteFile(source, "one"); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := jayessruntime.AppendFile(source, "\ntwo"); err != nil {
		t.Fatalf("AppendFile returned error: %v", err)
	}
	text, err := jayessruntime.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if text != "one\ntwo" {
		t.Fatalf("unexpected file content %q", text)
	}

	info, err := jayessruntime.StatPath(source)
	if err != nil {
		t.Fatalf("StatPath returned error: %v", err)
	}
	if info.Path != source || info.Size != int64(len(text)) || info.IsDir || info.ModTime.IsZero() {
		t.Fatalf("unexpected stat info: %#v", info)
	}

	if err := jayessruntime.CopyFile(source, copyPath); err != nil {
		t.Fatalf("CopyFile returned error: %v", err)
	}
	if err := jayessruntime.RenamePath(copyPath, moved); err != nil {
		t.Fatalf("RenamePath returned error: %v", err)
	}
	if !jayessruntime.Exists(moved) {
		t.Fatalf("expected renamed file to exist")
	}

	names, err := jayessruntime.ReadDirNames(root)
	if err != nil {
		t.Fatalf("ReadDirNames returned error: %v", err)
	}
	if strings.Join(names, ",") != "moved.txt,nested" {
		t.Fatalf("unexpected directory names %v", names)
	}

	paths, err := jayessruntime.WalkDirPaths(root)
	if err != nil {
		t.Fatalf("WalkDirPaths returned error: %v", err)
	}
	if len(paths) < 3 {
		t.Fatalf("expected recursive paths, got %v", paths)
	}

	if err := jayessruntime.DeleteFile(moved); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if jayessruntime.Exists(moved) {
		t.Fatalf("expected deleted file to be gone")
	}
}

func TestFilesystemFileStreamsAreBacked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream.txt")

	writer, err := jayessruntime.CreateWriteStream(path)
	if err != nil {
		t.Fatalf("CreateWriteStream returned error: %v", err)
	}
	if _, err := writer.WriteString("streamed"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer Close returned error: %v", err)
	}

	reader, err := jayessruntime.CreateReadStream(path)
	if err != nil {
		t.Fatalf("CreateReadStream returned error: %v", err)
	}
	defer reader.Close()
	data, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(data) != "streamed" {
		t.Fatalf("unexpected stream content %q", string(data))
	}
}

func TestFilesystemSymlinkWhenPlatformSupportsIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "links", "target.txt")
	if err := os.WriteFile(target, []byte("linked"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	if err := jayessruntime.SymlinkPath(target, link); err != nil {
		t.Fatalf("SymlinkPath returned error: %v", err)
	}
	text, err := jayessruntime.ReadFile(link)
	if err != nil {
		t.Fatalf("ReadFile through symlink returned error: %v", err)
	}
	if text != "linked" {
		t.Fatalf("unexpected symlink content %q", text)
	}
}

func TestFilesystemWatchPathReportsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatalf("seed watched file: %v", err)
	}

	events := make(chan jayessruntime.FileEvent, 2)
	watcher, err := jayessruntime.WatchPath(path, 10*time.Millisecond, func(event jayessruntime.FileEvent) {
		events <- event
	})
	if err != nil {
		t.Fatalf("WatchPath returned error: %v", err)
	}
	defer watcher.Close()

	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatalf("modify watched file: %v", err)
	}

	select {
	case event := <-events:
		if event.Path != path || event.Kind != "change" {
			t.Fatalf("unexpected watch event: %#v", event)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for watch event")
	}
}
