package typesys

import "testing"

func TestParseAndStringStructuredTypes(t *testing.T) {
	expr, err := Parse(`{readonly tag:"ok",value:number,[key:string]:boolean}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	got := expr.String()
	want := `{readonly tag:"ok",value:number,[key:string]:boolean}`
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestNormalizeCanonicalizesAliases(t *testing.T) {
	if got := Normalize("bool"); got != "boolean" {
		t.Fatalf("Normalize(bool) = %q, want %q", got, "boolean")
	}
	if got := Normalize("dynamic"); got != "" {
		t.Fatalf("Normalize(dynamic) = %q, want empty string", got)
	}
}

func TestRewriteAliasesRewritesNestedExpressions(t *testing.T) {
	got, err := RewriteAliases(`Result|bool`, func(name string) (string, error) {
		switch name {
		case "Result":
			return `{ok:boolean,value:string}`, nil
		case "bool":
			return "boolean", nil
		default:
			return name, nil
		}
	})
	if err != nil {
		t.Fatalf("RewriteAliases returned error: %v", err)
	}
	want := `{ok:boolean,value:string}|boolean`
	if got != want {
		t.Fatalf("RewriteAliases() = %q, want %q", got, want)
	}
}

func TestParseRejectsMalformedType(t *testing.T) {
	if _, err := Parse(`{value:number`); err == nil {
		t.Fatal("Parse malformed type succeeded, want error")
	}
}
