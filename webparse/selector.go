package webparse

import "strings"

func Query(root *Node, selector string) []*Node {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}
	if left, right, ok := strings.Cut(selector, ">"); ok {
		return queryChild(root, strings.TrimSpace(left), strings.TrimSpace(right))
	}
	parts := strings.Fields(selector)
	if len(parts) > 1 {
		return queryDescendant(root, parts)
	}
	var matches []*Node
	TraverseDFS(root, func(node *Node) {
		if MatchSimpleSelector(node, selector) {
			matches = append(matches, node)
		}
	})
	return matches
}

func MatchSimpleSelector(node *Node, selector string) bool {
	if node == nil || node.Type != ElementNode {
		return false
	}
	pseudo := ""
	if base, _, ok := strings.Cut(selector, ":"); ok {
		pseudo = strings.TrimPrefix(selector[len(base):], ":")
		selector = base
	}
	matches := false
	switch {
	case strings.HasPrefix(selector, "#"):
		matches = attrEquals(node, "id", selector[1:])
	case strings.HasPrefix(selector, "."):
		matches = hasClass(node, selector[1:])
	case strings.HasPrefix(selector, "[") && strings.HasSuffix(selector, "]"):
		matches = hasAttr(node, strings.TrimSuffix(strings.TrimPrefix(selector, "["), "]"))
	case strings.Contains(selector, "."):
		tag, class, _ := strings.Cut(selector, ".")
		matches = node.Name == tag && hasClass(node, class)
	default:
		matches = node.Name == selector
	}
	return matches && matchesPseudo(node, pseudo)
}

func matchesPseudo(node *Node, pseudo string) bool {
	if pseudo == "" {
		return true
	}
	if pseudo != "first-child" || node.Parent == nil {
		return false
	}
	for _, child := range node.Parent.Children {
		if child.Type == ElementNode {
			return child == node
		}
	}
	return false
}

func queryDescendant(root *Node, parts []string) []*Node {
	current := []*Node{root}
	for _, part := range parts {
		var next []*Node
		for _, node := range current {
			for _, child := range Query(node, part) {
				if child != node {
					next = append(next, child)
				}
			}
		}
		current = next
	}
	return current
}

func queryChild(root *Node, parentSelector string, childSelector string) []*Node {
	var matches []*Node
	for _, parent := range Query(root, parentSelector) {
		for _, child := range parent.Children {
			if MatchSimpleSelector(child, childSelector) {
				matches = append(matches, child)
			}
		}
	}
	return matches
}

func attrEquals(node *Node, name string, value string) bool {
	for _, attr := range node.Attributes {
		if attr.Name == name && attr.Value == value {
			return true
		}
	}
	return false
}

func hasAttr(node *Node, name string) bool {
	for _, attr := range node.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}

func hasClass(node *Node, class string) bool {
	for _, attr := range node.Attributes {
		if attr.Name != "class" {
			continue
		}
		for _, value := range strings.Fields(attr.Value) {
			if value == class {
				return true
			}
		}
	}
	return false
}
