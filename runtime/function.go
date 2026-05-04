package runtime

type FunctionBody func(CallFrame) Value

type Function struct {
	name        string
	body        FunctionBody
	environment *ClosureEnvironment
	lexicalThis Value
	boundThis   Value
	boundArgs   []Value
	arrow       bool
	bound       bool
}

func NewFunction(name string, body FunctionBody) *Function {
	return &Function{name: name, body: body}
}

func NewClosureFunction(name string, environment *ClosureEnvironment, body FunctionBody) *Function {
	return &Function{name: name, body: body, environment: environment}
}

func NewArrowFunction(name string, lexicalThis Value, environment *ClosureEnvironment, body FunctionBody) *Function {
	return &Function{name: name, body: body, environment: environment, lexicalThis: lexicalThis, arrow: true}
}

func NewBoundFunction(target *Function, this Value, arguments ...Value) *Function {
	if target == nil {
		return NewFunction("", nil)
	}
	boundArgs := append(target.boundArguments(), arguments...)
	return &Function{
		name:        target.name,
		body:        target.body,
		environment: target.environment,
		lexicalThis: target.lexicalThis,
		boundThis:   this,
		boundArgs:   boundArgs,
		arrow:       target.arrow,
		bound:       true,
	}
}

func (function *Function) Name() string {
	return function.name
}

func (function *Function) IsArrow() bool {
	return function != nil && function.arrow
}

func (function *Function) IsBound() bool {
	return function != nil && function.bound
}

func (function *Function) LexicalThis() (Value, bool) {
	if function == nil || !function.arrow {
		return Undefined(), false
	}
	return function.lexicalThis, true
}

func (function *Function) Closure() (*ClosureEnvironment, bool) {
	if function == nil || function.environment == nil {
		return nil, false
	}
	return function.environment, true
}

func (function *Function) Call(this Value, arguments ...Value) Value {
	if function == nil || function.body == nil {
		return Undefined()
	}
	if function.bound {
		this = function.boundThis
		arguments = append(function.boundArguments(), arguments...)
	}
	if function.arrow {
		this = function.lexicalThis
	}
	return function.body(NewCallFrameWithClosure(this, function.environment, arguments...))
}

func (function *Function) boundArguments() []Value {
	copied := make([]Value, len(function.boundArgs))
	copy(copied, function.boundArgs)
	return copied
}

func CallFunction(value Value, this Value, arguments ...Value) Value {
	function, ok := value.Function()
	if !ok {
		return Undefined()
	}
	return function.Call(this, arguments...)
}
