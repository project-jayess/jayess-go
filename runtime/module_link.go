package runtime

func (bindings *ModuleBindings) BindImportFrom(local string, source *ModuleBindings, imported string) bool {
	if bindings == nil || source == nil {
		return false
	}
	binding, ok := source.Export(imported)
	if !ok {
		return false
	}
	return bindings.BindImport(local, binding)
}

func (bindings *ModuleBindings) BindReExportFrom(exported string, source *ModuleBindings, imported string) bool {
	if bindings == nil || source == nil {
		return false
	}
	binding, ok := source.Export(imported)
	if !ok {
		return false
	}
	return bindings.BindExport(exported, binding)
}

func (bindings *ModuleBindings) BindNamespaceImport(local string, source *ModuleBindings) bool {
	if bindings == nil || source == nil || local == "" {
		return false
	}
	namespace := NewModuleBinding(local)
	namespace.Set(NewObjectValue(source.NamespaceObject()))
	return bindings.BindImport(local, namespace)
}

func (bindings *ModuleBindings) BindNamespaceExport(exported string, source *ModuleBindings) bool {
	if bindings == nil || source == nil || exported == "" {
		return false
	}
	namespace := NewModuleBinding(exported)
	namespace.Set(NewObjectValue(source.NamespaceObject()))
	return bindings.BindExport(exported, namespace)
}
