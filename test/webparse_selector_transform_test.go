package test

import (
	"testing"

	"jayess-go/webparse"
)

func TestWebParseSelectorsAndQueries(t *testing.T) {
	doc := webparse.ParseHTMLDocument(`<main id="app"><section><p class="lead" data-x="1">Hi</p><p>Bye</p></section></main>`)
	if len(webparse.Query(doc.Root, "#app")) != 1 {
		t.Fatal("expected id selector match")
	}
	if len(webparse.Query(doc.Root, "section .lead")) != 1 {
		t.Fatal("expected descendant class selector match")
	}
	if len(webparse.Query(doc.Root, "section > p:first-child")) != 1 {
		t.Fatal("expected child selector with basic pseudo-class match")
	}
	if len(webparse.Query(doc.Root, "[data-x]")) != 1 {
		t.Fatal("expected attribute selector match")
	}
}

func TestWebParseTransformTreeSafely(t *testing.T) {
	root := webparse.NewElement("root")
	child := webparse.NewElement("child")
	webparse.SetAttribute(child, "id", "a")
	webparse.AppendChild(root, child)
	webparse.AppendChild(child, webparse.NewText("value"))

	clone := webparse.Clone(root)
	if clone == root || clone.Children[0] == child {
		t.Fatal("expected deep clone")
	}

	replacement := webparse.NewElement("replacement")
	if !webparse.ReplaceChild(root, child, replacement) {
		t.Fatal("expected child replacement")
	}
	if !webparse.RemoveChild(root, replacement) || len(root.Children) != 0 {
		t.Fatalf("expected child removal, got %#v", root.Children)
	}
}

func TestWebParseTraversals(t *testing.T) {
	doc := webparse.ParseHTMLDocument(`<a><b></b><c></c></a>`)
	var dfs []string
	webparse.TraverseDFS(doc.Root, func(node *webparse.Node) {
		if node.Type == webparse.ElementNode {
			dfs = append(dfs, node.Name)
		}
	})
	if len(dfs) != 3 || dfs[0] != "a" || dfs[1] != "b" || dfs[2] != "c" {
		t.Fatalf("unexpected DFS order: %#v", dfs)
	}

	count := 0
	webparse.TraverseBFS(doc.Root, func(node *webparse.Node) {
		count++
	})
	if count != 4 {
		t.Fatalf("unexpected BFS count %d", count)
	}
}
