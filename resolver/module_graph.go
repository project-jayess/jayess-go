package resolver

import (
	"fmt"
	"strings"
)

type ModuleGraph struct {
	imports map[string][]string
}

type ImportCycleError struct {
	Cycle []string
}

func NewModuleGraph() *ModuleGraph {
	return &ModuleGraph{imports: map[string][]string{}}
}

func (g *ModuleGraph) AddModule(module string, imports []string) {
	if g.imports == nil {
		g.imports = map[string][]string{}
	}
	g.imports[module] = append([]string(nil), imports...)
	for _, imported := range imports {
		if _, ok := g.imports[imported]; !ok {
			g.imports[imported] = nil
		}
	}
}

func (g *ModuleGraph) InitializationOrder(entry string) ([]string, error) {
	return g.InitializationOrderFor([]string{entry})
}

func (g *ModuleGraph) InitializationOrderFor(entries []string) ([]string, error) {
	if g.imports == nil {
		return nil, nil
	}
	visited := map[string]bool{}
	active := map[string]int{}
	var stack []string
	var order []string
	for _, entry := range entries {
		if err := g.visit(entry, visited, active, &stack, &order); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (g *ModuleGraph) InitializationOrderAll() ([]string, error) {
	return g.InitializationOrderFor(g.Modules())
}

func (g *ModuleGraph) visit(module string, visited map[string]bool, active map[string]int, stack *[]string, order *[]string) error {
	if visited[module] {
		return nil
	}
	if index, ok := active[module]; ok {
		cycle := append([]string(nil), (*stack)[index:]...)
		cycle = append(cycle, module)
		return &ImportCycleError{Cycle: cycle}
	}
	active[module] = len(*stack)
	*stack = append(*stack, module)
	for _, imported := range g.imports[module] {
		if err := g.visit(imported, visited, active, stack, order); err != nil {
			return err
		}
	}
	*stack = (*stack)[:len(*stack)-1]
	delete(active, module)
	visited[module] = true
	*order = append(*order, module)
	return nil
}

func (e *ImportCycleError) Error() string {
	if e == nil || len(e.Cycle) == 0 {
		return "import cycle detected"
	}
	return fmt.Sprintf("import cycle detected: %s", strings.Join(e.Cycle, " -> "))
}
