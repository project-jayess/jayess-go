package test

import "testing"

func TestSemanticAllowsTaggedTemplateWithDeclaredTag(t *testing.T) {
	err := analyzeSource(t, "function html(parts) { return parts; }\nconst name = \"Jayess\";\nconst view = html`<p>${name}</p>`;")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsTaggedTemplateWithUndeclaredTag(t *testing.T) {
	err := analyzeSource(t, "const view = html`<p></p>`;")
	requireSemanticError(t, err, "use of html before declaration")
}

func TestSemanticAllowsTaggedTemplateWithMemberTag(t *testing.T) {
	err := analyzeSource(t, "const html = { raw(parts) { return parts; } };\nconst view = html.raw`<p></p>`;")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
