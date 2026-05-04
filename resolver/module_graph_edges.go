package resolver

import "sort"

type ModuleImportEdge struct {
	From string
	To   string
}

func (g *ModuleGraph) ImportEdges() []ModuleImportEdge {
	if g.imports == nil {
		return nil
	}
	var edges []ModuleImportEdge
	for importer, imports := range g.imports {
		for _, imported := range imports {
			edges = append(edges, ModuleImportEdge{From: importer, To: imported})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	return edges
}
