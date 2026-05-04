package test

import (
	"testing"

	"jayess-go/webparse"
)

func TestWebParseCSSStylesheet(t *testing.T) {
	sheet := webparse.ParseCSSStylesheet(`/* note */ @import "base.css"; #app, main.card { color: red; margin: 10px } @media screen { display: block }`)
	if len(sheet.Diagnostics) != 0 {
		t.Fatalf("expected clean css parse, got %#v", sheet.Diagnostics)
	}
	if len(sheet.Rules) != 3 {
		t.Fatalf("expected three css rules, got %#v", sheet.Rules)
	}
	if !sheet.Rules[0].AtRule || sheet.Rules[0].Prelude != `@import "base.css"` {
		t.Fatalf("expected import at-rule, got %#v", sheet.Rules[0])
	}
	if sheet.Rules[1].Selectors[0] != "#app" || sheet.Rules[1].Declarations[1].Value != "10px" {
		t.Fatalf("unexpected css rule: %#v", sheet.Rules[0])
	}
	if !sheet.Rules[2].AtRule || sheet.Rules[2].Prelude != "@media screen" {
		t.Fatalf("expected media at-rule, got %#v", sheet.Rules[2])
	}
}

func TestWebParseCSSMalformedAndSerialization(t *testing.T) {
	sheet := webparse.ParseCSSStylesheet(`a { color: blue`)
	if len(sheet.Diagnostics) == 0 {
		t.Fatal("expected unterminated css rule diagnostic")
	}
	if !sheet.Diagnostics[0].Recoverable {
		t.Fatalf("expected recoverable css diagnostic, got %#v", sheet.Diagnostics[0])
	}

	serialized := webparse.SerializeCSS(webparse.ParseCSSStylesheet(`p { font-size: 12px; color: black }`))
	if serialized != `p{font-size:12px;color:black}` {
		t.Fatalf("unexpected css serialization %q", serialized)
	}

	pretty := webparse.SerializeCSSWithOptions(webparse.ParseCSSStylesheet(`p { color: black }`), webparse.FormatOptions{})
	if pretty != "p {\n  color: black;\n}\n" {
		t.Fatalf("unexpected pretty css serialization %q", pretty)
	}
}
