package typesys

type Policy struct {
	OptionalOnly        bool
	ErasedAtCompile     bool
	TypedUntypedInterop bool
	CastSyntax          string
	RuntimeChecks       RuntimeCheckPolicy
}

type RuntimeCheckPolicy string

const (
	RuntimeChecksUnsupported RuntimeCheckPolicy = "unsupported"
	RuntimeChecksOptional    RuntimeCheckPolicy = "optional"
	RuntimeChecksRequired    RuntimeCheckPolicy = "required"
)

func DefaultPolicy() Policy {
	return Policy{
		OptionalOnly:        true,
		ErasedAtCompile:     true,
		TypedUntypedInterop: true,
		CastSyntax:          "assertion",
		RuntimeChecks:       RuntimeChecksUnsupported,
	}
}

func SupportsTypedUntypedInterop(policy Policy) bool {
	return policy.OptionalOnly && policy.TypedUntypedInterop
}

func ErasesTypes(policy Policy) bool {
	return policy.OptionalOnly && policy.ErasedAtCompile
}
