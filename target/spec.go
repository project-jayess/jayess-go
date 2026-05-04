package target

type Spec struct {
	Name           string
	Aliases        []string
	GOOS           string
	GOARCH         string
	Triple         string
	RuntimeLinkage string
	PathSeparator  string
	PermissionMode string
	Networking     bool
}

func Supported() []Spec {
	return []Spec{
		{
			Name:           "linux-x64",
			GOOS:           "linux",
			GOARCH:         "amd64",
			Triple:         "x86_64-pc-linux-gnu",
			RuntimeLinkage: "elf",
			PathSeparator:  "/",
			PermissionMode: "posix",
			Networking:     true,
		},
		{
			Name:           "linux-arm64",
			GOOS:           "linux",
			GOARCH:         "arm64",
			Triple:         "aarch64-unknown-linux-gnu",
			RuntimeLinkage: "elf",
			PathSeparator:  "/",
			PermissionMode: "posix",
			Networking:     true,
		},
		{
			Name:           "macos-x64",
			Aliases:        []string{"darwin-x64"},
			GOOS:           "darwin",
			GOARCH:         "amd64",
			Triple:         "x86_64-apple-darwin",
			RuntimeLinkage: "mach-o",
			PathSeparator:  "/",
			PermissionMode: "posix",
			Networking:     true,
		},
		{
			Name:           "macos-arm64",
			Aliases:        []string{"darwin-arm64"},
			GOOS:           "darwin",
			GOARCH:         "arm64",
			Triple:         "arm64-apple-darwin",
			RuntimeLinkage: "mach-o",
			PathSeparator:  "/",
			PermissionMode: "posix",
			Networking:     true,
		},
		{
			Name:           "windows-x64",
			GOOS:           "windows",
			GOARCH:         "amd64",
			Triple:         "x86_64-pc-windows-msvc",
			RuntimeLinkage: "coff",
			PathSeparator:  "\\",
			PermissionMode: "windows",
			Networking:     true,
		},
	}
}
