package backend

import (
	"fmt"
	"regexp"
	"strings"

	"jayess-go/compiler"
)

func nativeSystemLinkFlags(targetTriple string) []string {
	if strings.Contains(targetTriple, "windows") {
		return []string{"-lws2_32", "-lwinhttp", "-lsecur32", "-lcrypt32", "-lbcrypt"}
	}
	if strings.Contains(targetTriple, "linux") || strings.Contains(targetTriple, "darwin") {
		return []string{"-lssl", "-lcrypto", "-lz", "-lm"}
	}
	return nil
}

var undefinedSymbolPattern = regexp.MustCompile(`undefined reference to [` + "`" + `']([^` + "`" + `']+)[` + "`" + `']`)
var missingLibraryPattern = regexp.MustCompile(`cannot find -l([A-Za-z0-9_+.-]+)|library ['"]([^'"]+)['"] not found|unable to find library -l([A-Za-z0-9_+.-]+)`)
var missingHeaderPattern = regexp.MustCompile(`fatal error: ['"]([^'"]+\.(?:h|hh|hpp|hxx))['"] file not found|fatal error: ([^:\n]+?\.(?:h|hh|hpp|hxx)): No such file or directory`)
var missingTargetSDKHeaderPattern = regexp.MustCompile(`fatal error: ['"]((?:stdio|stdlib|string|stdint|stddef|stdbool|math|time|errno|signal|ctype|unistd|sys/types|sys/socket|netinet/in|arpa/inet|winsock2|windows)\.h)['"] file not found|fatal error: ((?:bits/[^:\n]+|sys/[^:\n]+|machine/[^:\n]+)\.h): No such file or directory`)

func formatNativeBuildError(err error, output string) error {
	return formatNativeBuildErrorForTarget(err, output, "")
}

func formatNativeBuildErrorForTarget(err error, output string, targetTriple string) error {
	if match := undefinedSymbolPattern.FindStringSubmatch(output); len(match) == 2 {
		return fmt.Errorf("native symbol resolution failed for %s: %w: %s", match[1], err, output)
	}
	if match := missingLibraryPattern.FindStringSubmatch(output); len(match) > 0 {
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				return fmt.Errorf("native library link failed for %s: %w: %s", candidate, err, output)
			}
		}
	}
	if strings.Contains(output, "Undefined symbols for architecture") {
		return fmt.Errorf("native symbol resolution failed: %w: %s", err, output)
	}
	if match := missingTargetSDKHeaderPattern.FindStringSubmatch(output); len(match) > 0 {
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				message := fmt.Sprintf("native target SDK or C runtime headers missing for %s", candidate)
				if hint := targetSDKInstallHint(targetTriple); hint != "" {
					message += ": " + hint
				}
				return fmt.Errorf("%s: %w: %s", message, err, output)
			}
		}
	}
	if match := missingHeaderPattern.FindStringSubmatch(output); len(match) > 0 {
		for _, candidate := range match[1:] {
			if strings.TrimSpace(candidate) != "" {
				return fmt.Errorf("native header dependency missing for %s: %w: %s", candidate, err, output)
			}
		}
	}
	return fmt.Errorf("clang native build failed: %w: %s", err, output)
}

func targetSDKInstallHint(targetTriple string) string {
	switch {
	case strings.Contains(targetTriple, "apple-darwin"), strings.Contains(targetTriple, "darwin"):
		return "darwin executable builds require an Apple SDK/sysroot (for example Xcode Command Line Tools via xcrun/SDKROOT or an equivalent osxcross-style sysroot)"
	case strings.Contains(targetTriple, "windows"), strings.Contains(targetTriple, "pc-windows"):
		return "windows executable builds require a Windows SDK plus C runtime headers/libs (for example an MSVC/clang-cl environment or a MinGW-style sysroot)"
	case strings.Contains(targetTriple, "linux"):
		return "cross-target executable builds require the target libc/sysroot headers and libraries"
	default:
		return ""
	}
}

func sharedLibraryModeArgs(targetTriple string) []string {
	if strings.Contains(targetTriple, "darwin") {
		return []string{"-dynamiclib", "-fPIC"}
	}
	return []string{"-shared", "-fPIC"}
}

func clangOptimizationFlag(level string) string {
	switch level {
	case "", "O0":
		return "-O0"
	case "O1":
		return "-O1"
	case "O2":
		return "-O2"
	case "O3":
		return "-O3"
	case "Oz":
		return "-Oz"
	default:
		return ""
	}
}

func clangTargetCodegenArgs(opts compiler.Options) []string {
	var args []string
	if opts.TargetCPU != "" {
		args = append(args, "-mcpu="+opts.TargetCPU)
	}
	for _, feature := range opts.TargetFeatures {
		feature = strings.TrimSpace(feature)
		if feature == "" {
			continue
		}
		args = append(args, "-Xclang", "-target-feature", "-Xclang", feature)
	}
	switch opts.RelocationModel {
	case "pic":
		args = append(args, "-fPIC")
	case "pie":
		args = append(args, "-fPIE")
	case "static":
		args = append(args, "-fno-pic")
	}
	if opts.CodeModel != "" {
		args = append(args, "-mcmodel="+opts.CodeModel)
	}
	return args
}
