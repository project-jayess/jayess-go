package runtime

type ModuleBinding struct {
	name        string
	value       Value
	initialized bool
}

func NewModuleBinding(name string) *ModuleBinding {
	return &ModuleBinding{name: name, value: Undefined()}
}

func (binding *ModuleBinding) Name() string {
	if binding == nil {
		return ""
	}
	return binding.name
}

func (binding *ModuleBinding) Set(value Value) {
	if binding == nil {
		return
	}
	binding.value = value
	binding.initialized = true
}

func (binding *ModuleBinding) Value() (Value, bool) {
	if binding == nil || !binding.initialized {
		return Undefined(), false
	}
	return binding.value, true
}

type ModuleBindings struct {
	locals      map[string]*ModuleBinding
	imports     map[string]*ModuleBinding
	exports     map[string]*ModuleBinding
	localOrder  []string
	importOrder []string
	exportOrder []string
}

func NewModuleBindings() *ModuleBindings {
	return &ModuleBindings{
		locals:  map[string]*ModuleBinding{},
		imports: map[string]*ModuleBinding{},
		exports: map[string]*ModuleBinding{},
	}
}

func (bindings *ModuleBindings) DefineLocal(name string, value Value) *ModuleBinding {
	if bindings == nil || name == "" {
		return nil
	}
	bindings.ensureMaps()
	binding, exists := bindings.locals[name]
	if !exists {
		binding = NewModuleBinding(name)
		bindings.locals[name] = binding
		bindings.localOrder = append(bindings.localOrder, name)
	}
	binding.Set(value)
	return binding
}

func (bindings *ModuleBindings) BindImport(local string, imported *ModuleBinding) bool {
	if bindings == nil || local == "" || imported == nil {
		return false
	}
	bindings.ensureMaps()
	if _, exists := bindings.imports[local]; !exists {
		bindings.importOrder = append(bindings.importOrder, local)
	}
	bindings.imports[local] = imported
	return true
}

func (bindings *ModuleBindings) BindExport(exported string, local *ModuleBinding) bool {
	if bindings == nil || exported == "" || local == nil {
		return false
	}
	bindings.ensureMaps()
	if _, exists := bindings.exports[exported]; !exists {
		bindings.exportOrder = append(bindings.exportOrder, exported)
	}
	bindings.exports[exported] = local
	return true
}

func (bindings *ModuleBindings) Local(name string) (*ModuleBinding, bool) {
	if bindings == nil || bindings.locals == nil {
		return nil, false
	}
	binding, exists := bindings.locals[name]
	return binding, exists
}

func (bindings *ModuleBindings) Import(name string) (*ModuleBinding, bool) {
	if bindings == nil || bindings.imports == nil {
		return nil, false
	}
	binding, exists := bindings.imports[name]
	return binding, exists
}

func (bindings *ModuleBindings) Export(name string) (*ModuleBinding, bool) {
	if bindings == nil || bindings.exports == nil {
		return nil, false
	}
	binding, exists := bindings.exports[name]
	return binding, exists
}

func (bindings *ModuleBindings) LocalNames() []string {
	return orderedBindingNames(bindings, bindings.localOrder, bindings.locals)
}

func (bindings *ModuleBindings) ImportNames() []string {
	return orderedBindingNames(bindings, bindings.importOrder, bindings.imports)
}

func (bindings *ModuleBindings) ExportNames() []string {
	return orderedBindingNames(bindings, bindings.exportOrder, bindings.exports)
}

func (bindings *ModuleBindings) NamespaceObject() *Object {
	namespace := NewObject()
	for _, name := range bindings.ExportNames() {
		binding, _ := bindings.Export(name)
		value, ok := binding.Value()
		if !ok {
			value = Undefined()
		}
		namespace.SetNamedProperty(name, value)
	}
	return namespace
}

func (bindings *ModuleBindings) ensureMaps() {
	if bindings.locals == nil {
		bindings.locals = map[string]*ModuleBinding{}
	}
	if bindings.imports == nil {
		bindings.imports = map[string]*ModuleBinding{}
	}
	if bindings.exports == nil {
		bindings.exports = map[string]*ModuleBinding{}
	}
}

func orderedBindingNames(bindings *ModuleBindings, order []string, source map[string]*ModuleBinding) []string {
	if bindings == nil || len(order) == 0 {
		return nil
	}
	names := make([]string, 0, len(order))
	for _, name := range order {
		if _, exists := source[name]; exists {
			names = append(names, name)
		}
	}
	return names
}
