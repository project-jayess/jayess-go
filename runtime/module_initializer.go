package runtime

type ModuleInitializerFunc func(*ModuleBindings) Value

type RuntimeModule struct {
	name         string
	bindings     *ModuleBindings
	initializer  ModuleInitializerFunc
	initializing bool
	initialized  bool
}

func NewRuntimeModule(name string, initializer ModuleInitializerFunc) *RuntimeModule {
	return &RuntimeModule{
		name:        name,
		bindings:    NewModuleBindings(),
		initializer: initializer,
	}
}

func (module *RuntimeModule) Name() string {
	if module == nil {
		return ""
	}
	return module.name
}

func (module *RuntimeModule) Bindings() *ModuleBindings {
	if module == nil {
		return nil
	}
	if module.bindings == nil {
		module.bindings = NewModuleBindings()
	}
	return module.bindings
}

func (module *RuntimeModule) Initialized() bool {
	return module != nil && module.initialized
}

type ModuleRegistry struct {
	modules map[string]*RuntimeModule
	order   []string
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{modules: map[string]*RuntimeModule{}}
}

func (registry *ModuleRegistry) Register(name string, initializer ModuleInitializerFunc) *RuntimeModule {
	if registry == nil || name == "" {
		return nil
	}
	registry.ensureModules()
	module, exists := registry.modules[name]
	if !exists {
		module = NewRuntimeModule(name, initializer)
		registry.modules[name] = module
		registry.order = append(registry.order, name)
		return module
	}
	module.initializer = initializer
	return module
}

func (registry *ModuleRegistry) Module(name string) (*RuntimeModule, bool) {
	if registry == nil || registry.modules == nil {
		return nil, false
	}
	module, exists := registry.modules[name]
	return module, exists
}

func (registry *ModuleRegistry) InitializeModule(name string) bool {
	module, ok := registry.Module(name)
	if !ok || module.initialized || module.initializing {
		return false
	}
	module.initializing = true
	if module.initializer != nil {
		module.initializer(module.Bindings())
	}
	module.initializing = false
	module.initialized = true
	return true
}

func (registry *ModuleRegistry) InitializeModules(order []string) []string {
	initialized := make([]string, 0, len(order))
	for _, name := range order {
		if registry.InitializeModule(name) {
			initialized = append(initialized, name)
		}
	}
	return initialized
}

func (registry *ModuleRegistry) ModuleNames() []string {
	if registry == nil {
		return nil
	}
	names := make([]string, 0, len(registry.order))
	for _, name := range registry.order {
		if _, exists := registry.modules[name]; exists {
			names = append(names, name)
		}
	}
	return names
}

func (registry *ModuleRegistry) ensureModules() {
	if registry.modules == nil {
		registry.modules = map[string]*RuntimeModule{}
	}
}
