package llvmbackend

type BackendInvariant string

const (
	RuntimeLinkInvariant       BackendInvariant = "runtime-link"
	NativeBindingLinkInvariant BackendInvariant = "native-binding-link"
	CCallConventionInvariant   BackendInvariant = "c-call-convention"
	ErrorPathInvariant         BackendInvariant = "error-path"
	DataLayoutInvariant        BackendInvariant = "data-layout"
)

func BackendInvariants() []BackendInvariant {
	return []BackendInvariant{
		RuntimeLinkInvariant,
		NativeBindingLinkInvariant,
		CCallConventionInvariant,
		ErrorPathInvariant,
		DataLayoutInvariant,
	}
}
