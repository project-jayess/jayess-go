package target

func Lookup(name string) (Spec, bool) {
	for _, spec := range Supported() {
		if spec.Name == name {
			return spec, true
		}
		for _, alias := range spec.Aliases {
			if alias == name {
				return spec, true
			}
		}
	}
	return Spec{}, false
}

func LookupOSArch(goos, goarch string) (Spec, bool) {
	for _, spec := range Supported() {
		if spec.GOOS == goos && spec.GOARCH == goarch {
			return spec, true
		}
	}
	return Spec{}, false
}

func Triple(name string) (string, bool) {
	spec, ok := Lookup(name)
	if !ok {
		return "", false
	}
	return spec.Triple, true
}
