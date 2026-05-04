package webparse

import "strings"

type FormatOptions struct {
	Pretty           bool
	Minify           bool
	PreserveComments bool
}

func SerializeHTMLWithOptions(node *Node, options FormatOptions) string {
	return serializeNodeWithOptions(node, false, options, 0)
}

func SerializeXMLWithOptions(node *Node, options FormatOptions) string {
	return serializeNodeWithOptions(node, true, options, 0)
}

func SerializeCSSWithOptions(sheet Stylesheet, options FormatOptions) string {
	if options.Minify {
		return SerializeCSS(sheet)
	}
	var out strings.Builder
	for _, rule := range sheet.Rules {
		out.WriteString(rule.Prelude)
		out.WriteString(" {\n")
		for _, declaration := range rule.Declarations {
			out.WriteString("  ")
			out.WriteString(declaration.Property)
			out.WriteString(": ")
			out.WriteString(declaration.Value)
			out.WriteString(";\n")
		}
		out.WriteString("}\n")
	}
	return out.String()
}

func serializeNodeWithOptions(node *Node, xml bool, options FormatOptions, depth int) string {
	if node == nil {
		return ""
	}
	if node.Type == CommentNode && !options.PreserveComments {
		return ""
	}
	if !options.Pretty {
		return serializeNodeWithoutComments(node, xml, options.PreserveComments)
	}
	indent := strings.Repeat("  ", depth)
	switch node.Type {
	case DocumentNode:
		var out strings.Builder
		for _, child := range node.Children {
			out.WriteString(serializeNodeWithOptions(child, xml, options, depth))
		}
		return out.String()
	case ElementNode:
		var out strings.Builder
		out.WriteString(indent)
		out.WriteString(openTag(node, xml))
		if len(node.Children) == 0 {
			out.WriteString(closeTag(node, xml))
			out.WriteByte('\n')
			return out.String()
		}
		out.WriteByte('\n')
		for _, child := range node.Children {
			out.WriteString(serializeNodeWithOptions(child, xml, options, depth+1))
		}
		out.WriteString(indent)
		out.WriteString("</")
		out.WriteString(node.Name)
		out.WriteString(">\n")
		return out.String()
	case TextNode:
		return indent + node.Text + "\n"
	case CommentNode:
		return indent + "<!--" + node.Text + "-->\n"
	default:
		return indent + serializeNode(node, xml) + "\n"
	}
}

func serializeNodeWithoutComments(node *Node, xml bool, preserveComments bool) string {
	if node == nil || (node.Type == CommentNode && !preserveComments) {
		return ""
	}
	if node.Type != DocumentNode && node.Type != ElementNode {
		return serializeNode(node, xml)
	}
	var out strings.Builder
	if node.Type == ElementNode {
		out.WriteString(openTag(node, xml))
	}
	for _, child := range node.Children {
		out.WriteString(serializeNodeWithoutComments(child, xml, preserveComments))
	}
	if node.Type == ElementNode {
		out.WriteString("</")
		out.WriteString(node.Name)
		out.WriteString(">")
	}
	return out.String()
}

func openTag(node *Node, xml bool) string {
	var out strings.Builder
	out.WriteByte('<')
	out.WriteString(node.Name)
	for _, attr := range node.Attributes {
		out.WriteByte(' ')
		out.WriteString(attr.Name)
		if !attr.Boolean || xml {
			out.WriteString("=\"")
			out.WriteString(attr.Value)
			out.WriteByte('"')
		}
	}
	out.WriteByte('>')
	return out.String()
}

func closeTag(node *Node, xml bool) string {
	if xml {
		return ""
	}
	return "</" + node.Name + ">"
}
