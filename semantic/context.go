package semantic

type controlContext struct {
	inFunction         bool
	inAsyncFunction    bool
	inGenerator        bool
	ownsArguments      bool
	allowThis          bool
	inMethod           bool
	inClassMethod      bool
	inClassField       bool
	inClassStaticBlock bool
	inConstructor      bool
	allowNewTarget     bool
	hasSuperClass      bool
	privateMembers     map[string]bool
	inLoop             bool
	inSwitch           bool
	labels             []labelTarget
}

type labelTarget struct {
	name           string
	allowsContinue bool
}

func rootContext() controlContext {
	return controlContext{}
}

func (ctx controlContext) enterFunction(isAsync bool, isGenerator bool) controlContext {
	return controlContext{
		inFunction:      true,
		inAsyncFunction: isAsync,
		inGenerator:     isGenerator,
		ownsArguments:   true,
		allowThis:       true,
		allowNewTarget:  true,
	}
}

func (ctx controlContext) enterArrowFunction(isAsync bool, isGenerator bool) controlContext {
	return controlContext{
		inFunction:         true,
		inAsyncFunction:    isAsync,
		inGenerator:        isGenerator,
		allowThis:          ctx.allowThis,
		inMethod:           ctx.inMethod,
		inClassMethod:      ctx.inClassMethod,
		inClassField:       ctx.inClassField,
		inClassStaticBlock: ctx.inClassStaticBlock,
		inConstructor:      ctx.inConstructor,
		allowNewTarget:     ctx.allowNewTarget,
		hasSuperClass:      ctx.hasSuperClass,
		privateMembers:     ctx.privateMembers,
	}
}

func (ctx controlContext) enterClassMethod(hasSuperClass bool, privateMembers map[string]bool, isAsync bool, isGenerator bool, isConstructor bool) controlContext {
	ctx.inFunction = true
	ctx.inAsyncFunction = isAsync
	ctx.inGenerator = isGenerator
	ctx.ownsArguments = true
	ctx.allowThis = true
	ctx.inMethod = true
	ctx.inClassMethod = true
	ctx.inConstructor = isConstructor
	ctx.allowNewTarget = true
	ctx.hasSuperClass = hasSuperClass
	ctx.privateMembers = privateMembers
	return ctx
}

func (ctx controlContext) enterClassField(hasSuperClass bool, privateMembers map[string]bool) controlContext {
	ctx.inMethod = true
	ctx.allowThis = true
	ctx.inClassField = true
	ctx.hasSuperClass = hasSuperClass
	ctx.privateMembers = privateMembers
	return ctx
}

func (ctx controlContext) enterClassStaticBlock(hasSuperClass bool, privateMembers map[string]bool) controlContext {
	ctx.inMethod = true
	ctx.allowThis = true
	ctx.inClassStaticBlock = true
	ctx.hasSuperClass = hasSuperClass
	ctx.privateMembers = privateMembers
	return ctx
}

func (ctx controlContext) enterObjectMethod(isAsync bool, isGenerator bool) controlContext {
	ctx.inFunction = true
	ctx.inAsyncFunction = isAsync
	ctx.inGenerator = isGenerator
	ctx.ownsArguments = true
	ctx.allowThis = true
	ctx.inMethod = true
	ctx.allowNewTarget = true
	return ctx
}

func (ctx controlContext) enterLoop() controlContext {
	ctx.inLoop = true
	return ctx
}

func (ctx controlContext) enterSwitch() controlContext {
	ctx.inSwitch = true
	return ctx
}

func (ctx controlContext) enterLabel(name string, statement any) (controlContext, bool) {
	if _, exists := ctx.findLabel(name); exists {
		return ctx, false
	}
	ctx.labels = append(ctx.labels, labelTarget{
		name:           name,
		allowsContinue: isLoopStatement(statement),
	})
	return ctx, true
}

func (ctx controlContext) findLabel(name string) (labelTarget, bool) {
	for i := len(ctx.labels) - 1; i >= 0; i-- {
		if ctx.labels[i].name == name {
			return ctx.labels[i], true
		}
	}
	return labelTarget{}, false
}
