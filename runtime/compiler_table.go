package runtime

type CompilerTable struct {
	values map[string]Value
	order  []string
}

func NewCompilerTable() *CompilerTable {
	return &CompilerTable{values: map[string]Value{}}
}

func (table *CompilerTable) Set(key string, value Value) {
	if table.values == nil {
		table.values = map[string]Value{}
	}
	if _, exists := table.values[key]; !exists {
		table.order = append(table.order, key)
	}
	table.values[key] = value
}

func (table *CompilerTable) Get(key string) (Value, bool) {
	if table == nil || table.values == nil {
		return Undefined(), false
	}
	value, exists := table.values[key]
	if !exists {
		return Undefined(), false
	}
	return value, true
}

func (table *CompilerTable) Has(key string) bool {
	if table == nil || table.values == nil {
		return false
	}
	_, exists := table.values[key]
	return exists
}

func (table *CompilerTable) Keys() []string {
	if table == nil {
		return nil
	}
	keys := make([]string, 0, len(table.order))
	for _, key := range table.order {
		if table.Has(key) {
			keys = append(keys, key)
		}
	}
	return keys
}
