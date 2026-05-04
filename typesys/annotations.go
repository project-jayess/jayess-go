package typesys

type AnnotationTarget string

const (
	VariableAnnotation  AnnotationTarget = "variable"
	ParameterAnnotation AnnotationTarget = "parameter"
	ReturnAnnotation    AnnotationTarget = "return"
	PropertyAnnotation  AnnotationTarget = "property"
)

type Annotation struct {
	Target   AnnotationTarget
	Name     string
	TypeName string
	Optional bool
	Readonly bool
}

func VariableType(name string, typeName string) Annotation {
	return Annotation{Target: VariableAnnotation, Name: name, TypeName: typeName}
}

func ParameterType(name string, typeName string, optional bool) Annotation {
	return Annotation{Target: ParameterAnnotation, Name: name, TypeName: typeName, Optional: optional}
}

func ReturnType(typeName string) Annotation {
	return Annotation{Target: ReturnAnnotation, TypeName: typeName}
}

func PropertyType(name string, typeName string, optional bool, readonly bool) Annotation {
	return Annotation{Target: PropertyAnnotation, Name: name, TypeName: typeName, Optional: optional, Readonly: readonly}
}

func ValidateAnnotation(annotation Annotation) bool {
	if annotation.Target == "" || annotation.TypeName == "" {
		return false
	}
	if annotation.Target == ReturnAnnotation {
		return true
	}
	return annotation.Name != ""
}

func AnnotationTargets() []AnnotationTarget {
	return []AnnotationTarget{
		VariableAnnotation,
		ParameterAnnotation,
		ReturnAnnotation,
		PropertyAnnotation,
	}
}
