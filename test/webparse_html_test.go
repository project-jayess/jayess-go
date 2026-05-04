package test

import (
	"strings"
	"testing"

	"jayess-go/webparse"
)

func TestWebParseHTMLDocumentAndFragment(t *testing.T) {
	doc := webparse.ParseHTMLDocument(`<main id="app"><img src="logo.png"><button disabled>Go</button><!--ok--></main>`)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("expected clean html parse, got %#v", doc.Diagnostics)
	}
	if doc.Root.Name != "html-document" || len(doc.Root.Children) != 1 {
		t.Fatalf("unexpected html root: %#v", doc.Root)
	}
	main := doc.Root.Children[0]
	if main.Name != "main" || len(main.Children) != 3 {
		t.Fatalf("expected ordered DOM-like children, got %#v", main)
	}
	button := main.Children[1]
	if button.Attributes[0].Name != "disabled" || !button.Attributes[0].Boolean {
		t.Fatalf("expected boolean disabled attribute, got %#v", button.Attributes)
	}

	fragment := webparse.ParseHTMLFragment(`<span class="a">x</span>`)
	if fragment.Root.Name != "html-fragment" {
		t.Fatalf("expected html fragment root, got %s", fragment.Root.Name)
	}
}

func TestWebParseHTMLMalformedAndSerialization(t *testing.T) {
	doc := webparse.ParseHTMLDocument(`<section><p>one</section>`)
	if len(doc.Diagnostics) == 0 {
		t.Fatal("expected recoverable malformed html diagnostic")
	}
	if !doc.Diagnostics[0].Recoverable {
		t.Fatalf("expected recoverable diagnostic, got %#v", doc.Diagnostics[0])
	}

	roundTrip := webparse.SerializeHTML(webparse.ParseHTMLDocument(`<p data-x="1">hi</p>`).Root)
	if roundTrip != `<p data-x="1">hi</p>` {
		t.Fatalf("unexpected html serialization %q", roundTrip)
	}

	stripped := webparse.SerializeHTMLWithOptions(webparse.ParseHTMLDocument(`<p>hi<!--x--></p>`).Root, webparse.FormatOptions{})
	if stripped != `<p>hi</p>` {
		t.Fatalf("unexpected comment stripping serialization %q", stripped)
	}
}

func TestWebParseLargeHTMLAndSourceSpan(t *testing.T) {
	source := "<root>\n" + strings.Repeat("<item>x</item>", 128) + "\n</root>"
	doc := webparse.ParseHTMLFile(source, "page.html")
	if len(doc.Root.Children) != 1 {
		t.Fatalf("expected one root element, got %#v", doc.Root.Children)
	}
	items := webparse.Query(doc.Root, "item")
	if len(items) != 128 {
		t.Fatalf("expected large-file item nodes, got %d", len(items))
	}
	item := items[0]
	if item.Span.File != "page.html" || item.Span.Line != 2 || item.Span.Column != 1 {
		t.Fatalf("unexpected item span: %#v", item.Span)
	}
}
