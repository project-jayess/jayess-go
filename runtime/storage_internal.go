package runtime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var ErrStorageClosed = errors.New("storage is closed")

type StorageStore struct {
	path   string
	values map[string]string
	closed bool
	mu     sync.Mutex
}

type StorageEntry struct {
	Key   string
	Value string
}

func StorageOpen(path string) (*StorageStore, error) {
	store := &StorageStore{path: path, values: map[string]string{}}
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) > 0 {
			if err := json.Unmarshal(data, &store.values); err != nil {
				return nil, err
			}
		}
		return store, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	return nil, err
}

func StorageClose(store *StorageStore) error {
	if store == nil {
		return ErrStorageClosed
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return ErrStorageClosed
	}
	if err := store.persistLocked(); err != nil {
		return err
	}
	store.closed = true
	return nil
}

func StorageGet(store *StorageStore, key string) (string, bool, error) {
	if err := ensureStorageOpen(store); err != nil {
		return "", false, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	value, ok := store.values[key]
	return value, ok, nil
}

func StoragePut(store *StorageStore, key string, value string) error {
	if err := ensureStorageOpen(store); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.values[key] = value
	return store.persistLocked()
}

func StorageDelete(store *StorageStore, key string) error {
	if err := ensureStorageOpen(store); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.values, key)
	return store.persistLocked()
}

func StorageScan(store *StorageStore, prefix string) ([]StorageEntry, error) {
	if err := ensureStorageOpen(store); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	keys := make([]string, 0, len(store.values))
	for key := range store.values {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	entries := make([]StorageEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, StorageEntry{Key: key, Value: store.values[key]})
	}
	return entries, nil
}

func ensureStorageOpen(store *StorageStore) error {
	if store == nil || store.closed {
		return ErrStorageClosed
	}
	return nil
}

func (store *StorageStore) persistLocked() error {
	if store.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store.values, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(store.path, append(data, '\n'), 0o644)
}
