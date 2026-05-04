package test

import (
	"strings"
	"testing"
)

func testPathName(t *testing.T) string {
	t.Helper()
	return testPathNameForValue(t.Name())
}

func testPathNameForValue(value string) string {
	if value == "" {
		return "empty"
	}
	return strings.NewReplacer("/", "_", " ", "_", ".", "_", "@", "at", ":", "_").Replace(value)
}
