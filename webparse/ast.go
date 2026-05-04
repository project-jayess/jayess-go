package webparse

type Span struct {
	File   string
	Line   int
	Column int
	Offset int
}

type Diagnostic struct {
	Message     string
	Span        Span
	Recoverable bool
}

type NodeType string

const (
	DocumentNode   NodeType = "document"
	ElementNode    NodeType = "element"
	TextNode       NodeType = "text"
	CommentNode    NodeType = "comment"
	ProcessingNode NodeType = "processing-instruction"
	CDATANode      NodeType = "cdata"
	StylesheetNode NodeType = "stylesheet"
)

type Attribute struct {
	Name    string
	Value   string
	Boolean bool
	Span    Span
}

type Node struct {
	Type       NodeType
	Name       string
	Text       string
	Attributes []Attribute
	Children   []*Node
	Parent     *Node
	Span       Span
}

type Document struct {
	Kind        string
	Root        *Node
	Diagnostics []Diagnostic
}

func NewElement(name string) *Node {
	return &Node{Type: ElementNode, Name: name}
}

func NewText(text string) *Node {
	return &Node{Type: TextNode, Text: text}
}

func AppendChild(parent *Node, child *Node) {
	if parent == nil || child == nil {
		return
	}
	child.Parent = parent
	parent.Children = append(parent.Children, child)
}

func SetAttribute(node *Node, name string, value string) {
	if node == nil {
		return
	}
	for index := range node.Attributes {
		if node.Attributes[index].Name == name {
			node.Attributes[index].Value = value
			node.Attributes[index].Boolean = false
			return
		}
	}
	node.Attributes = append(node.Attributes, Attribute{Name: name, Value: value})
}

func RemoveChild(parent *Node, child *Node) bool {
	if parent == nil || child == nil {
		return false
	}
	for index, candidate := range parent.Children {
		if candidate == child {
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
			child.Parent = nil
			return true
		}
	}
	return false
}

func ReplaceChild(parent *Node, oldChild *Node, newChild *Node) bool {
	if parent == nil || oldChild == nil || newChild == nil {
		return false
	}
	for index, candidate := range parent.Children {
		if candidate == oldChild {
			newChild.Parent = parent
			oldChild.Parent = nil
			parent.Children[index] = newChild
			return true
		}
	}
	return false
}

func Clone(node *Node) *Node {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Parent = nil
	clone.Attributes = append([]Attribute{}, node.Attributes...)
	clone.Children = nil
	for _, child := range node.Children {
		AppendChild(&clone, Clone(child))
	}
	return &clone
}
