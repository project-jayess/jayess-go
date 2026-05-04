package sqlite

type TransactionFeature string

const (
	BeginTransaction    TransactionFeature = "begin-transaction"
	CommitTransaction   TransactionFeature = "commit-transaction"
	RollbackTransaction TransactionFeature = "rollback-transaction"
	PragmaHelpers       TransactionFeature = "pragma-helpers"
	BusyTimeout         TransactionFeature = "busy-timeout"
)

func TransactionFeatures() []TransactionFeature {
	return []TransactionFeature{
		BeginTransaction,
		CommitTransaction,
		RollbackTransaction,
		PragmaHelpers,
		BusyTimeout,
	}
}
