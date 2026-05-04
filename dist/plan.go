package dist

import "path/filepath"

type Plan struct {
	Platform     Platform
	Version      string
	Root         string
	CompilerPath string
	ToolBinDir   string
	ToolLibDir   string
	LicenseDir   string
	ArchivePath  string
	ChecksumPath string
}

func BuildPlan(config Config) (Plan, error) {
	normalized, platform, err := NormalizeConfig(config)
	if err != nil {
		return Plan{}, err
	}
	name := PackageName(normalized.Version, platform.Name)
	root := filepath.Join(normalized.OutDir, platform.Name, name)
	archivePath := filepath.Join(normalized.OutDir, platform.Name, name+archiveExtension(platform))
	return Plan{
		Platform:     platform,
		Version:      normalized.Version,
		Root:         root,
		CompilerPath: filepath.Join(root, compilerExecutable(platform)),
		ToolBinDir:   filepath.Join(root, "tools", "bin"),
		ToolLibDir:   filepath.Join(root, "tools", "lib"),
		LicenseDir:   filepath.Join(root, "licenses"),
		ArchivePath:  archivePath,
		ChecksumPath: archivePath + ".sha256",
	}, nil
}

func PackageName(version string, platform string) string {
	return "jayess-" + version + "-" + platform
}

func compilerExecutable(platform Platform) string {
	if platform.GOOS == "windows" {
		return "jayess.exe"
	}
	return "jayess"
}

func archiveExtension(platform Platform) string {
	if platform.GOOS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
