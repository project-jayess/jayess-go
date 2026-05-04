package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/sqlite"
)

func TestSQLiteBindingModuleCanImportBindJS(t *testing.T) {
	module := sqlite.BindingModule{
		Path: "./native/sqlite.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./sqlite.c"},
			Exports: []binding.Export{
				{Name: "open", Symbol: "jayess_sqlite_open", Kind: binding.FunctionExport},
			},
		},
		APIs:    []sqlite.APIKind{sqlite.DatabaseAPI, sqlite.StatementAPI},
		Handles: []sqlite.HandleKind{sqlite.DatabaseHandle, sqlite.StatementHandle},
	}

	if diagnostics := sqlite.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid SQLite binding module, got %#v", diagnostics)
	}
	if !sqlite.SupportsAPI(module, sqlite.StatementAPI) {
		t.Fatal("expected SQLite statement API support")
	}
}

func TestSQLiteBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := sqlite.BindingModule{
		Path: "./native/sqlite.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "open", Symbol: "sqlite_open", Kind: binding.FunctionExport}},
		},
		Handles: []sqlite.HandleKind{sqlite.DatabaseHandle},
	}

	diagnostics := sqlite.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestSQLiteBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := sqlite.BindingModule{
		Path: "./native/sqlite.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DSQLITE_THREADSAFE=1"},
			Exports:     []binding.Export{{Name: "open", Symbol: "sqlite_open", Kind: binding.FunctionExport}},
		},
		Handles:        []sqlite.HandleKind{sqlite.DatabaseHandle},
		VendoredSource: true,
	}

	plan := sqlite.PlanBuild([]sqlite.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean SQLite build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one SQLite compile unit, got %#v", plan.CompileUnits)
	}
	if plan.CompileUnits[0].Source != "./sqlite3.c" {
		t.Fatalf("expected vendored sqlite3.c source, got %s", plan.CompileUnits[0].Source)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DSQLITE_THREADSAFE=1"})
}

func TestSQLiteHandlesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []sqlite.HandleKind{
		sqlite.DatabaseHandle,
		sqlite.StatementHandle,
		sqlite.BlobHandle,
	} {
		if !sqlite.SupportsHandle(kind) {
			t.Fatalf("expected SQLite handle support for %s", kind)
		}
	}
}
