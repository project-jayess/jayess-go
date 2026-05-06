package lowering

import "jayess-go/ast"

func cleanupBindingNames(pattern ast.BindingPattern) []string {
	switch pat := pattern.(type) {
	case *ast.BindingName:
		if pat.Name == "" {
			return nil
		}
		return []string{pat.Name}
	case *ast.BindingDefault:
		return cleanupBindingNames(pat.Pattern)
	case *ast.BindingRest:
		return cleanupBindingNames(pat.Pattern)
	case *ast.ArrayBindingPattern:
		var names []string
		for _, element := range pat.Elements {
			names = append(names, cleanupBindingNames(element)...)
		}
		return names
	case *ast.ObjectBindingPattern:
		var names []string
		for _, property := range pat.Properties {
			names = append(names, cleanupBindingNames(property.Pattern)...)
		}
		return names
	default:
		return nil
	}
}
