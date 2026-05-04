package dist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeManifest(config Config, result Result) error {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Jayess %s for %s\n", config.Version, result.Plan.Platform.Name)
	fmt.Fprintf(&builder, "\nCompiler: %s\n", filepath.Base(result.Plan.CompilerPath))
	builder.WriteString("\nBundled LLVM tools:\n")
	if len(result.CopiedTools) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, tool := range result.CopiedTools {
			fmt.Fprintf(&builder, "- tools/bin/%s\n", tool)
		}
	}
	builder.WriteString("\nBundled LLVM runtime libraries:\n")
	if len(result.CopiedLibs) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, library := range result.CopiedLibs {
			fmt.Fprintf(&builder, "- tools/lib/%s\n", library)
		}
	}
	builder.WriteString("\nBundled license and notice files:\n")
	if len(result.CopiedLicenses) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, license := range result.CopiedLicenses {
			fmt.Fprintf(&builder, "- licenses/%s\n", license)
		}
		builder.WriteString("- licenses/README.txt\n")
	}
	if len(result.Diagnostics) > 0 {
		builder.WriteString("\nPackaging diagnostics:\n")
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprintf(&builder, "- %s\n", diagnostic)
		}
	}
	builder.WriteString("\nThe compiler searches this package's tools directory automatically.\n")
	return os.WriteFile(filepath.Join(result.Plan.Root, "README.txt"), []byte(builder.String()), 0o644)
}
