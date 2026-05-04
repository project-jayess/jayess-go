package lifetime

import "jayess-go/ast"

func bindingNames(pattern ast.BindingPattern) []string {
	names := map[string]bool{}
	collectBindingNames(pattern, names)
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result
}

func collectBindingNames(pattern ast.BindingPattern, names map[string]bool) {
	switch pattern := pattern.(type) {
	case *ast.BindingName:
		if pattern.Name != "" {
			names[pattern.Name] = true
		}
	case *ast.BindingDefault:
		collectBindingNames(pattern.Pattern, names)
	case *ast.BindingRest:
		collectBindingNames(pattern.Pattern, names)
	case *ast.ArrayBindingPattern:
		for _, element := range pattern.Elements {
			collectBindingNames(element, names)
		}
	case *ast.ObjectBindingPattern:
		for _, property := range pattern.Properties {
			collectBindingNames(property.Pattern, names)
		}
	}
}
