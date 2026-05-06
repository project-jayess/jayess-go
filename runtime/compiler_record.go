package runtime

type CompilerRecordShape struct {
	Name   string
	Fields []string
}

type CompilerRecord struct {
	shape  CompilerRecordShape
	values *CompilerTable
}

func NewCompilerRecord(shape CompilerRecordShape) *CompilerRecord {
	return &CompilerRecord{
		shape:  shape,
		values: NewCompilerTable(),
	}
}

func (record *CompilerRecord) Shape() CompilerRecordShape {
	if record == nil {
		return CompilerRecordShape{}
	}
	return record.shape
}

func (record *CompilerRecord) Set(field string, value Value) bool {
	if record == nil || !record.allows(field) {
		return false
	}
	record.values.Set(field, value)
	return true
}

func (record *CompilerRecord) Get(field string) (Value, bool) {
	if record == nil || !record.allows(field) {
		return Undefined(), false
	}
	return record.values.Get(field)
}

func (record *CompilerRecord) Fields() []string {
	if record == nil {
		return nil
	}
	return append([]string(nil), record.shape.Fields...)
}

func (record *CompilerRecord) allows(field string) bool {
	for _, allowed := range record.shape.Fields {
		if allowed == field {
			return true
		}
	}
	return false
}
