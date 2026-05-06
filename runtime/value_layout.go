package runtime

type PayloadSlot string

const (
	NoPayload      PayloadSlot = "none"
	BoolPayload    PayloadSlot = "bool"
	Float64Payload PayloadSlot = "float64"
	PointerPayload PayloadSlot = "pointer"
	NativePayload  PayloadSlot = "native-pointer"
)

type ValueLayout struct {
	Kind          ValueKind
	TagName       string
	Payload       PayloadSlot
	HeapAllocated bool
	Managed       bool
}

func DynamicValueLayouts() []ValueLayout {
	return []ValueLayout{
		{Kind: UndefinedValue, TagName: "JAYESS_VALUE_UNDEFINED", Payload: NoPayload},
		{Kind: NullValue, TagName: "JAYESS_VALUE_NULL", Payload: NoPayload},
		{Kind: BooleanValue, TagName: "JAYESS_VALUE_BOOLEAN", Payload: BoolPayload},
		{Kind: NumberValue, TagName: "JAYESS_VALUE_NUMBER", Payload: Float64Payload},
		{Kind: BigIntValue, TagName: "JAYESS_VALUE_BIGINT", Payload: PointerPayload, HeapAllocated: true, Managed: true},
		{Kind: StringValue, TagName: "JAYESS_VALUE_STRING", Payload: PointerPayload, HeapAllocated: true, Managed: true},
		{Kind: ObjectValue, TagName: "JAYESS_VALUE_OBJECT", Payload: PointerPayload, HeapAllocated: true, Managed: true},
		{Kind: ArrayValue, TagName: "JAYESS_VALUE_ARRAY", Payload: PointerPayload, HeapAllocated: true, Managed: true},
		{Kind: FunctionValue, TagName: "JAYESS_VALUE_FUNCTION", Payload: PointerPayload, HeapAllocated: true, Managed: true},
		{Kind: NativeValue, TagName: "JAYESS_VALUE_NATIVE", Payload: NativePayload, HeapAllocated: true},
	}
}

func LayoutForKind(kind ValueKind) (ValueLayout, bool) {
	for _, layout := range DynamicValueLayouts() {
		if layout.Kind == kind {
			return layout, true
		}
	}
	return ValueLayout{}, false
}

func ValidateDynamicValueLayouts() []string {
	var diagnostics []string
	seenKinds := map[ValueKind]struct{}{}
	seenTags := map[string]struct{}{}
	for _, layout := range DynamicValueLayouts() {
		if !IsValueKind(layout.Kind) {
			diagnostics = append(diagnostics, "unknown value kind "+string(layout.Kind))
		}
		if layout.TagName == "" {
			diagnostics = append(diagnostics, "missing tag for "+string(layout.Kind))
		}
		if layout.Payload == "" {
			diagnostics = append(diagnostics, "missing payload for "+string(layout.Kind))
		}
		if _, ok := seenKinds[layout.Kind]; ok {
			diagnostics = append(diagnostics, "duplicate value kind "+string(layout.Kind))
		}
		if _, ok := seenTags[layout.TagName]; ok {
			diagnostics = append(diagnostics, "duplicate value tag "+layout.TagName)
		}
		seenKinds[layout.Kind] = struct{}{}
		seenTags[layout.TagName] = struct{}{}
	}
	for _, kind := range ValueKinds() {
		if _, ok := seenKinds[kind]; !ok {
			diagnostics = append(diagnostics, "missing layout for "+string(kind))
		}
	}
	return diagnostics
}
