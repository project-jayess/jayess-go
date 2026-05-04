package test

import "testing"

func TestSemanticAllowsClassKeywordMembers(t *testing.T) {
	err := analyzeSource(t, `
		class Item {
			default = 1;
			class(value) {
				return value;
			}
			get import() {
				return this.default;
			}
		}
		const item = new Item();
		item.default;
		item.class(1);
		item.import;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
