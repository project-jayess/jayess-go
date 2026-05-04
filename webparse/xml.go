package webparse

func ParseXMLDocument(input string) Document {
	tokens, diagnostics := tokenizeMarkup(input, "", true)
	root := &Node{Type: DocumentNode, Name: "xml-document", Span: Span{Line: 1, Column: 1}}
	stack := []*Node{root}
	for _, tok := range tokens {
		parent := stack[len(stack)-1]
		switch tok.typ {
		case tokenStart:
			node := &Node{Type: ElementNode, Name: tok.name, Attributes: tok.attrs, Span: tok.span}
			AppendChild(parent, node)
			if !tok.selfClosed {
				stack = append(stack, node)
			}
		case tokenEnd:
			if len(stack) == 1 || stack[len(stack)-1].Name != tok.name {
				diagnostics = append(diagnostics, Diagnostic{Message: "mismatched xml closing tag " + tok.name, Span: tok.span})
				continue
			}
			stack = stack[:len(stack)-1]
		case tokenText:
			if tok.raw != "" {
				AppendChild(parent, &Node{Type: TextNode, Text: tok.raw, Span: tok.span})
			}
		case tokenComment:
			AppendChild(parent, &Node{Type: CommentNode, Text: tok.raw, Span: tok.span})
		case tokenPI:
			AppendChild(parent, &Node{Type: ProcessingNode, Text: tok.raw, Span: tok.span})
		case tokenCDATA:
			AppendChild(parent, &Node{Type: CDATANode, Text: tok.raw, Span: tok.span})
		}
	}
	if len(stack) > 1 {
		diagnostics = append(diagnostics, Diagnostic{Message: "unclosed xml element " + stack[len(stack)-1].Name, Span: stack[len(stack)-1].Span})
	}
	elementCount := 0
	for _, child := range root.Children {
		if child.Type == ElementNode {
			elementCount++
		}
	}
	if elementCount != 1 {
		diagnostics = append(diagnostics, Diagnostic{Message: "xml document must contain exactly one root element", Span: root.Span})
	}
	return Document{Kind: "xml", Root: root, Diagnostics: diagnostics}
}
