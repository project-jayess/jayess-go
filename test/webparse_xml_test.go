package test

import (
	"testing"

	"jayess-go/webparse"
)

func TestWebParseXMLDocument(t *testing.T) {
	doc := webparse.ParseXMLDocument(`<?xml version="1.0"?><book xmlns:ex="urn:x"><title>A</title><!--c--><![CDATA[raw]]></book>`)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("expected clean xml parse, got %#v", doc.Diagnostics)
	}
	if len(doc.Root.Children) != 2 || doc.Root.Children[0].Type != webparse.ProcessingNode {
		t.Fatalf("expected processing instruction before root, got %#v", doc.Root.Children)
	}
	book := doc.Root.Children[1]
	if book.Name != "book" || len(book.Attributes) != 1 || len(book.Children) != 3 {
		t.Fatalf("unexpected xml tree: %#v", book)
	}

	serialized := webparse.SerializeXML(book)
	if serialized != `<book xmlns:ex="urn:x"><title>A</title><!--c--><![CDATA[raw]]></book>` {
		t.Fatalf("unexpected xml serialization %q", serialized)
	}
}

func TestWebParseXMLReportsStrictErrors(t *testing.T) {
	doc := webparse.ParseXMLDocument(`<root><item></root>`)
	if len(doc.Diagnostics) == 0 {
		t.Fatal("expected strict xml diagnostics")
	}
	if doc.Diagnostics[0].Recoverable {
		t.Fatalf("expected non-recoverable xml diagnostic, got %#v", doc.Diagnostics[0])
	}
}
