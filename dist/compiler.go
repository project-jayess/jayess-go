package dist

import (
	"os"
	"os/exec"
)

func buildCompiler(config Config, platform Platform, outputPath string) error {
	if err := os.MkdirAll(parentDir(outputPath), 0o755); err != nil {
		return err
	}
	args := []string{"build"}
	if len(config.GoTags) > 0 {
		args = append(args, "-tags", joinTags(config.GoTags))
	}
	args = append(args, "-o", outputPath, "./cmd/jayess")
	command := exec.Command("go", args...)
	command.Dir = config.SourceRoot
	command.Env = append(os.Environ(), "GOOS="+platform.GOOS, "GOARCH="+platform.GOARCH)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := tags[0]
	for _, tag := range tags[1:] {
		result += " " + tag
	}
	return result
}
