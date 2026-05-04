package webparse

func ParseHTMLDocument(input string) Document {
	return parseHTML(input, "", false)
}

func ParseHTMLFragment(input string) Document {
	return parseHTML(input, "", true)
}

func ParseHTMLFile(input string, file string) Document {
	return parseHTML(input, file, false)
}

func parseHTML(input string, file string, fragment bool) Document {
	tokens, diagnostics := tokenizeMarkup(input, file, false)
	root := &Node{Type: DocumentNode, Name: "html-document", Span: Span{File: file, Line: 1, Column: 1}}
	if fragment {
		root.Name = "html-fragment"
	}
	stack := []*Node{root}
	for _, tok := range tokens {
		parent := stack[len(stack)-1]
		switch tok.typ {
		case tokenStart:
			node := &Node{Type: ElementNode, Name: tok.name, Attributes: tok.attrs, Span: tok.span}
			AppendChild(parent, node)
			if !tok.selfClosed && !isHTMLVoid(tok.name) {
				stack = append(stack, node)
			}
		case tokenEnd:
			found := -1
			for index := len(stack) - 1; index > 0; index-- {
				if stack[index].Name == tok.name {
					found = index
					break
				}
			}
			if found < 0 {
				diagnostics = append(diagnostics, Diagnostic{Message: "unmatched closing tag " + tok.name, Span: tok.span, Recoverable: true})
				continue
			}
			if found != len(stack)-1 {
				diagnostics = append(diagnostics, Diagnostic{Message: "html elements closed out of order before " + tok.name, Span: tok.span, Recoverable: true})
			}
			stack = stack[:found]
		case tokenText:
			if tok.raw != "" {
				AppendChild(parent, &Node{Type: TextNode, Text: tok.raw, Span: tok.span})
			}
		case tokenComment:
			AppendChild(parent, &Node{Type: CommentNode, Text: tok.raw, Span: tok.span})
		}
	}
	if len(stack) > 1 {
		diagnostics = append(diagnostics, Diagnostic{Message: "unclosed html elements", Span: stack[len(stack)-1].Span, Recoverable: true})
	}
	return Document{Kind: "html", Root: root, Diagnostics: diagnostics}
}

func isHTMLVoid(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
