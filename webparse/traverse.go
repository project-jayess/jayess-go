package webparse

func TraverseDFS(root *Node, visit func(*Node)) {
	if root == nil || visit == nil {
		return
	}
	visit(root)
	for _, child := range root.Children {
		TraverseDFS(child, visit)
	}
}

func TraverseBFS(root *Node, visit func(*Node)) {
	if root == nil || visit == nil {
		return
	}
	queue := []*Node{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visit(node)
		queue = append(queue, node.Children...)
	}
}
