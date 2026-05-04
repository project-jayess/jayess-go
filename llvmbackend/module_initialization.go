package llvmbackend

import "jayess-go/resolver"

type ModuleInitializer struct {
	Module string
	Symbol string
}

type ModuleInitializationPlan struct {
	Initializers []ModuleInitializer
}

func PlanModuleInitialization(graph *resolver.ModuleGraph, entries []string) (ModuleInitializationPlan, error) {
	if graph == nil {
		return ModuleInitializationPlan{}, nil
	}
	order, err := graph.InitializationOrderFor(entries)
	if err != nil {
		return ModuleInitializationPlan{}, err
	}
	initializers := make([]ModuleInitializer, 0, len(order))
	for _, module := range order {
		initializers = append(initializers, ModuleInitializer{
			Module: module,
			Symbol: ModuleInitializationSymbol(module),
		})
	}
	return ModuleInitializationPlan{Initializers: initializers}, nil
}

func ModuleInitializationSymbol(module string) string {
	return "__jayess_init_module_" + symbolSuffix(module)
}

func moduleInitializationDeclarations(plan ModuleInitializationPlan) []Declaration {
	declarations := make([]Declaration, 0, len(plan.Initializers))
	for _, initializer := range plan.Initializers {
		declarations = append(declarations, Declaration{Name: initializer.Symbol, IRType: "void ()"})
	}
	return declarations
}

func moduleInitializationCalls(plan ModuleInitializationPlan) []string {
	calls := make([]string, 0, len(plan.Initializers))
	for _, initializer := range plan.Initializers {
		calls = append(calls, "call void @"+initializer.Symbol+"()")
	}
	return calls
}

func symbolSuffix(value string) string {
	if value == "" {
		return "anonymous"
	}
	suffix := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if isSymbolByte(character) {
			suffix = append(suffix, character)
			continue
		}
		suffix = append(suffix, '_')
	}
	return string(suffix)
}

func isSymbolByte(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		character == '_'
}
