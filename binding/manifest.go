package binding

type Manifest struct {
	Sources            []string
	IncludeDirs        []string
	LibraryDirs        []string
	SharedLibraries    []string
	LicenseFiles       []string
	CFlags             []string
	LDFlags            []string
	Platforms          map[string]PlatformOptions
	Exports            []Export
	PlaceholderExports []string
}

type PlatformOptions struct {
	Sources         []string
	IncludeDirs     []string
	LibraryDirs     []string
	SharedLibraries []string
	LicenseFiles    []string
	CFlags          []string
	LDFlags         []string
}

type BuildInputs struct {
	Sources         []string
	IncludeDirs     []string
	LibraryDirs     []string
	SharedLibraries []string
	LicenseFiles    []string
	CFlags          []string
	LDFlags         []string
}

func (manifest Manifest) BuildInputsFor(platform string) BuildInputs {
	inputs := BuildInputs{
		Sources:         append([]string{}, manifest.Sources...),
		IncludeDirs:     append([]string{}, manifest.IncludeDirs...),
		LibraryDirs:     append([]string{}, manifest.LibraryDirs...),
		SharedLibraries: append([]string{}, manifest.SharedLibraries...),
		LicenseFiles:    append([]string{}, manifest.LicenseFiles...),
		CFlags:          append([]string{}, manifest.CFlags...),
		LDFlags:         append([]string{}, manifest.LDFlags...),
	}
	if override, ok := manifest.Platforms[platform]; ok {
		inputs.Sources = append(inputs.Sources, override.Sources...)
		inputs.IncludeDirs = append(inputs.IncludeDirs, override.IncludeDirs...)
		inputs.LibraryDirs = append(inputs.LibraryDirs, override.LibraryDirs...)
		inputs.SharedLibraries = append(inputs.SharedLibraries, override.SharedLibraries...)
		inputs.LicenseFiles = append(inputs.LicenseFiles, override.LicenseFiles...)
		inputs.CFlags = append(inputs.CFlags, override.CFlags...)
		inputs.LDFlags = append(inputs.LDFlags, override.LDFlags...)
	}
	return inputs
}
