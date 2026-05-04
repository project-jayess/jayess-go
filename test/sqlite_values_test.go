package test

import (
	"testing"

	"jayess-go/sqlite"
)

func TestSQLiteValueBindingKinds(t *testing.T) {
	for _, value := range []sqlite.ValueKind{
		sqlite.NullValue,
		sqlite.IntegerValue,
		sqlite.FloatValue,
		sqlite.StringValue,
		sqlite.BlobValue,
	} {
		if !sqlite.SupportsBindableValue(value) {
			t.Fatalf("expected SQLite bindable value %s", value)
		}
	}
}

func TestSQLiteRowAccessKinds(t *testing.T) {
	kinds := sqlite.RowAccessKinds()
	for _, want := range []sqlite.RowAccessKind{
		sqlite.ColumnByIndex,
		sqlite.ColumnByName,
		sqlite.RowIterator,
	} {
		if !hasSQLiteRowAccessKind(kinds, want) {
			t.Fatalf("expected SQLite row access kind %s in %#v", want, kinds)
		}
	}
}

func hasSQLiteRowAccessKind(kinds []sqlite.RowAccessKind, want sqlite.RowAccessKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
