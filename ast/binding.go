package ast

type BindingPattern interface {
	bindingPattern()
}

type BindingName struct {
	Name string
}

func (*BindingName) bindingPattern() {}

type BindingDefault struct {
	Pattern BindingPattern
	Value   Expression
}

func (*BindingDefault) bindingPattern() {}

type BindingRest struct {
	Pattern BindingPattern
}

func (*BindingRest) bindingPattern() {}

type ArrayBindingPattern struct {
	Elements []BindingPattern
}

func (*ArrayBindingPattern) bindingPattern() {}

type ObjectBindingProperty struct {
	Key      string
	KeyExpr  Expression
	Pattern  BindingPattern
	Computed bool
	Rest     bool
}

type ObjectBindingPattern struct {
	Properties []ObjectBindingProperty
}

func (*ObjectBindingPattern) bindingPattern() {}
