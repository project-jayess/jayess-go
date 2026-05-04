package resolver

// ReachableModuleOrder returns modules reachable from an entry in initialization order.
func (g *ModuleGraph) ReachableModuleOrder(entry string) ([]string, error) {
	return g.InitializationOrder(entry)
}
