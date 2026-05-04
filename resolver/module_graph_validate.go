package resolver

func (g *ModuleGraph) ValidateAcyclic() error {
	_, err := g.InitializationOrderAll()
	return err
}

func (g *ModuleGraph) IsAcyclic() bool {
	return g.ValidateAcyclic() == nil
}
