package sqlite

type ValueKind string

const (
	NullValue    ValueKind = "null"
	IntegerValue ValueKind = "integer"
	FloatValue   ValueKind = "float"
	StringValue  ValueKind = "string"
	BlobValue    ValueKind = "blob"
)

type RowAccessKind string

const (
	ColumnByIndex RowAccessKind = "column-by-index"
	ColumnByName  RowAccessKind = "column-by-name"
	RowIterator   RowAccessKind = "row-iterator"
)

func BindableValues() []ValueKind {
	return []ValueKind{NullValue, IntegerValue, FloatValue, StringValue, BlobValue}
}

func RowAccessKinds() []RowAccessKind {
	return []RowAccessKind{ColumnByIndex, ColumnByName, RowIterator}
}

func SupportsBindableValue(kind ValueKind) bool {
	for _, value := range BindableValues() {
		if value == kind {
			return true
		}
	}
	return false
}
