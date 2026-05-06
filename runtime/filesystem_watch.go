package runtime

import (
	"os"
	"time"
)

type FileEvent struct {
	Path string
	Kind string
}

type FileWatcher struct {
	done chan struct{}
}

func WatchPath(path string, interval time.Duration, handler func(FileEvent)) (*FileWatcher, error) {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	watcher := &FileWatcher{done: make(chan struct{})}
	previous, previousErr := os.Stat(path)
	go watcher.watch(path, interval, previous, previousErr, handler)
	return watcher, nil
}

func (watcher *FileWatcher) Close() {
	if watcher == nil || watcher.done == nil {
		return
	}
	select {
	case <-watcher.done:
	default:
		close(watcher.done)
	}
}

func (watcher *FileWatcher) watch(path string, interval time.Duration, previous os.FileInfo, previousErr error, handler func(FileEvent)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-watcher.done:
			return
		case <-ticker.C:
			current, currentErr := os.Stat(path)
			kind := fileEventKind(previous, previousErr, current, currentErr)
			previous, previousErr = current, currentErr
			if kind != "" && handler != nil {
				handler(FileEvent{Path: path, Kind: kind})
			}
		}
	}
}

func fileEventKind(previous os.FileInfo, previousErr error, current os.FileInfo, currentErr error) string {
	previousMissing := os.IsNotExist(previousErr)
	currentMissing := os.IsNotExist(currentErr)
	if previousMissing && currentErr == nil {
		return "create"
	}
	if previousErr == nil && currentMissing {
		return "delete"
	}
	if previousErr == nil && currentErr == nil && fileChanged(previous, current) {
		return "change"
	}
	return ""
}

func fileChanged(previous os.FileInfo, current os.FileInfo) bool {
	if previous == nil || current == nil {
		return previous != current
	}
	return previous.Size() != current.Size() || !previous.ModTime().Equal(current.ModTime()) || previous.Mode() != current.Mode()
}
