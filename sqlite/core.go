package sqlite

type CoreFeature string

const (
	OpenDatabase           CoreFeature = "open-database"
	CloseDatabase          CoreFeature = "close-database"
	ExecuteSQL             CoreFeature = "execute-sql"
	PrepareStatement       CoreFeature = "prepare-statement"
	FinalizeStatement      CoreFeature = "finalize-statement"
	ResetStatement         CoreFeature = "reset-statement"
	ClearStatementBindings CoreFeature = "clear-statement-bindings"
)

func CoreFeatures() []CoreFeature {
	return []CoreFeature{
		OpenDatabase,
		CloseDatabase,
		ExecuteSQL,
		PrepareStatement,
		FinalizeStatement,
		ResetStatement,
		ClearStatementBindings,
	}
}
