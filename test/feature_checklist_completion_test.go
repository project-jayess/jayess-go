package test

import (
	"os"
	"strings"
	"testing"
)

func TestFeatureChecklistHasNoUncheckedItems(t *testing.T) {
	content, err := os.ReadFile("../jayess-feature-checklist.md")
	if err != nil {
		t.Fatalf("failed to read feature checklist: %v", err)
	}
	inRemainingWork := false
	for index, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "## 33. Remaining compiler completion work") {
			inRemainingWork = true
		}
		if strings.HasPrefix(line, "- [ ]") && !inRemainingWork {
			t.Fatalf("feature checklist has unchecked item at line %d: %s", index+1, line)
		}
	}
}
