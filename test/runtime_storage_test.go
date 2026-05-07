package test

import (
	"errors"
	"path/filepath"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeStorageCapabilitiesAreDeclared(t *testing.T) {
	for _, name := range []string{"open", "close", "get", "put", "delete", "scan"} {
		if !jayessruntime.HasStorageCapability(name) {
			t.Fatalf("expected storage runtime capability %s", name)
		}
	}
}

func TestRuntimeStoragePersistsReadsDeletesAndScans(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.store.json")
	store, err := jayessruntime.StorageOpen(path)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	if err := jayessruntime.StoragePut(store, "user:2", "Grace"); err != nil {
		t.Fatalf("put user:2: %v", err)
	}
	if err := jayessruntime.StoragePut(store, "user:1", "Ada"); err != nil {
		t.Fatalf("put user:1: %v", err)
	}
	if err := jayessruntime.StoragePut(store, "config:theme", "dark"); err != nil {
		t.Fatalf("put config:theme: %v", err)
	}

	value, ok, err := jayessruntime.StorageGet(store, "user:1")
	if err != nil || !ok || value != "Ada" {
		t.Fatalf("unexpected user:1 lookup value=%q ok=%v err=%v", value, ok, err)
	}

	entries, err := jayessruntime.StorageScan(store, "user:")
	if err != nil {
		t.Fatalf("scan user prefix: %v", err)
	}
	if len(entries) != 2 || entries[0].Key != "user:1" || entries[1].Key != "user:2" {
		t.Fatalf("unexpected scan result: %#v", entries)
	}
	if err := jayessruntime.StorageDelete(store, "user:1"); err != nil {
		t.Fatalf("delete user:1: %v", err)
	}
	if err := jayessruntime.StorageClose(store); err != nil {
		t.Fatalf("close storage: %v", err)
	}

	reopened, err := jayessruntime.StorageOpen(path)
	if err != nil {
		t.Fatalf("reopen storage: %v", err)
	}
	defer jayessruntime.StorageClose(reopened)
	if _, ok, err := jayessruntime.StorageGet(reopened, "user:1"); err != nil || ok {
		t.Fatalf("deleted key persisted unexpectedly ok=%v err=%v", ok, err)
	}
	value, ok, err = jayessruntime.StorageGet(reopened, "user:2")
	if err != nil || !ok || value != "Grace" {
		t.Fatalf("persisted key mismatch value=%q ok=%v err=%v", value, ok, err)
	}
}

func TestRuntimeStorageRejectsUseAfterClose(t *testing.T) {
	store, err := jayessruntime.StorageOpen(filepath.Join(t.TempDir(), "closed.json"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	if err := jayessruntime.StorageClose(store); err != nil {
		t.Fatalf("close storage: %v", err)
	}
	if err := jayessruntime.StoragePut(store, "key", "value"); !errors.Is(err, jayessruntime.ErrStorageClosed) {
		t.Fatalf("expected closed storage error, got %v", err)
	}
}

func TestSemanticAllowsStorageSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			const db = storage.open("./app.store.json");
			storage.put(db, "key", "value");
			const value = storage.get(db, "key");
			const rows = storage.scan(db, "");
			storage.delete(db, "key");
			storage.close(db);
			return value || rows;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsTopLevelStorageRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var storage = {};`)
	requireSemanticError(t, err, "duplicate declaration storage")
}
