package test

import (
	"testing"

	"jayess-go/libuv"
)

func TestLibUVSafetyRules(t *testing.T) {
	rules := libuv.SafetyRules()
	for _, want := range []libuv.SafetyRule{
		libuv.HandleLifetimeSafe,
		libuv.CallbackLifetimeSafe,
		libuv.ThreadLoopOwnership,
		libuv.ErrorDiagnostics,
	} {
		if !hasLibUVSafetyRule(rules, want) {
			t.Fatalf("expected libuv safety rule %s in %#v", want, rules)
		}
	}
}

func hasLibUVSafetyRule(rules []libuv.SafetyRule, want libuv.SafetyRule) bool {
	for _, rule := range rules {
		if rule == want {
			return true
		}
	}
	return false
}
