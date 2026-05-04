package webparse

import "strings"

func SerializeHTML(node *Node) string {
	return serializeNode(node, false)
}

func SerializeXML(node *Node) string {
	return serializeNode(node, true)
}

func serializeNode(node *Node, xml bool) string {
	if node == nil {
		return ""
	}
	switch node.Type {
	case DocumentNode:
		var out strings.Builder
		for _, child := range node.Children {
			out.WriteString(serializeNode(child, xml))
		}
		return out.String()
	case TextNode:
		return node.Text
	case CommentNode:
		return "<!--" + node.Text + "-->"
	case ProcessingNode:
		return "<?" + node.Text + "?>"
	case CDATANode:
		return "<![CDATA[" + node.Text + "]]>"
	case ElementNode:
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
		if len(node.Children) == 0 && xml {
			out.WriteString("/>")
			return out.String()
		}
		out.WriteByte('>')
		for _, child := range node.Children {
			out.WriteString(serializeNode(child, xml))
		}
		out.WriteString("</")
		out.WriteString(node.Name)
		out.WriteByte('>')
		return out.String()
	default:
		return ""
	}
}
