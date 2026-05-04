package resolver

func (g *ModuleGraph) RemoveImport(module string, imported string) bool {
	if g.imports == nil {
		return false
	}
	imports, ok := g.imports[module]
	if !ok {
		return false
	}
	for index, current := range imports {
		if current != imported {
			continue
		}
		g.imports[module] = append(imports[:index], imports[index+1:]...)
		return true
	}
	return false
}

func (g *ModuleGraph) RemoveAllImports(module string, imported string) int {
	if g.imports == nil {
		return 0
	}
	imports, ok := g.imports[module]
	if !ok {
		return 0
	}
	kept := imports[:0]
	removed := 0
	for _, current := range imports {
		if current == imported {
			removed++
			continue
		}
		kept = append(kept, current)
	}
	g.imports[module] = kept
	return removed
}
