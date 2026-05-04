package lldc

type LinkRequest struct {
	ObjectPath       string
	ExtraObjectFiles []string
	OutputPath       string
	TargetTriple     string
	Shared           bool
	LinkFlags        []string
}
