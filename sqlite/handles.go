package sqlite

type HandleKind string

const (
	DatabaseHandle  HandleKind = "sqlite3"
	StatementHandle HandleKind = "sqlite3_stmt"
	BlobHandle      HandleKind = "sqlite3_blob"
)

type HandleRule struct {
	Kind     HandleKind
	Managed  bool
	Closable bool
	Nullable bool
}

func HandleRules() []HandleRule {
	return []HandleRule{
		{Kind: DatabaseHandle, Managed: true, Closable: true},
		{Kind: StatementHandle, Managed: true, Closable: true},
		{Kind: BlobHandle, Managed: true, Closable: true, Nullable: true},
	}
}

func SupportsHandle(kind HandleKind) bool {
	for _, rule := range HandleRules() {
		if rule.Kind == kind && rule.Managed {
			return true
		}
	}
	return false
}
