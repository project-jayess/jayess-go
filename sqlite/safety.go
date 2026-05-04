package sqlite

type SafetyRule string

const (
	StatementLifetimeSafe   SafetyRule = "statement-lifetime-safe"
	DatabaseLifetimeSafe    SafetyRule = "database-lifetime-safe"
	BlobStringOwnershipSafe SafetyRule = "blob-string-ownership-safe"
	ErrorDiagnostics        SafetyRule = "sqlite-error-diagnostics"
)

func SafetyRules() []SafetyRule {
	return []SafetyRule{
		StatementLifetimeSafe,
		DatabaseLifetimeSafe,
		BlobStringOwnershipSafe,
		ErrorDiagnostics,
	}
}
