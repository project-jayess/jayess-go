package test

import (
	"testing"

	"jayess-go/sqlite"
)

func TestSQLiteCoreDatabaseAPI(t *testing.T) {
	features := sqlite.CoreFeatures()
	for _, want := range []sqlite.CoreFeature{
		sqlite.OpenDatabase,
		sqlite.CloseDatabase,
		sqlite.ExecuteSQL,
		sqlite.PrepareStatement,
		sqlite.FinalizeStatement,
		sqlite.ResetStatement,
		sqlite.ClearStatementBindings,
	} {
		if !hasSQLiteCoreFeature(features, want) {
			t.Fatalf("expected SQLite core feature %s in %#v", want, features)
		}
	}
}

func hasSQLiteCoreFeature(features []sqlite.CoreFeature, want sqlite.CoreFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
