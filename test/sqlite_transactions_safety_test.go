package test

import (
	"testing"

	"jayess-go/sqlite"
)

func TestSQLiteTransactionFeatures(t *testing.T) {
	features := sqlite.TransactionFeatures()
	for _, want := range []sqlite.TransactionFeature{
		sqlite.BeginTransaction,
		sqlite.CommitTransaction,
		sqlite.RollbackTransaction,
		sqlite.PragmaHelpers,
		sqlite.BusyTimeout,
	} {
		if !hasSQLiteTransactionFeature(features, want) {
			t.Fatalf("expected SQLite transaction feature %s in %#v", want, features)
		}
	}
}

func TestSQLiteSafetyRules(t *testing.T) {
	rules := sqlite.SafetyRules()
	for _, want := range []sqlite.SafetyRule{
		sqlite.StatementLifetimeSafe,
		sqlite.DatabaseLifetimeSafe,
		sqlite.BlobStringOwnershipSafe,
		sqlite.ErrorDiagnostics,
	} {
		if !hasSQLiteSafetyRule(rules, want) {
			t.Fatalf("expected SQLite safety rule %s in %#v", want, rules)
		}
	}
}

func hasSQLiteTransactionFeature(features []sqlite.TransactionFeature, want sqlite.TransactionFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasSQLiteSafetyRule(rules []sqlite.SafetyRule, want sqlite.SafetyRule) bool {
	for _, rule := range rules {
		if rule == want {
			return true
		}
	}
	return false
}
